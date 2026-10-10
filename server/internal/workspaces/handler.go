package workspaces

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/usetrim/trim/server/internal/billingsettings"
	"github.com/usetrim/trim/server/internal/emaildenylist"
	"github.com/usetrim/trim/server/internal/mailer"
	"github.com/usetrim/trim/server/internal/middleware"
	"github.com/usetrim/trim/server/internal/notifications"
	"github.com/usetrim/trim/server/internal/pagination"
	"github.com/usetrim/trim/server/internal/subscriptions"
)

type Handler struct {
	DB           *pgxpool.Pool
	ReadDB       *pgxpool.Pool
	AppPublicURL string
	Mailer       mailer.SMTPConfig
}

func NewHandler(db, readDB *pgxpool.Pool, appPublicURL string, smtp mailer.SMTPConfig) *Handler {
	return &Handler{
		DB:           db,
		ReadDB:       readDB,
		AppPublicURL: strings.TrimRight(strings.TrimSpace(appPublicURL), "/"),
		Mailer:       smtp,
	}
}

func (h *Handler) readPool() *pgxpool.Pool {
	if h.ReadDB != nil {
		return h.ReadDB
	}
	return h.DB
}

func writeJSONErr(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": subscriptions.MessageForCode(code),
	})
}

func escapeILikePattern(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

type workspaceRow struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	PlanTier       string `json:"plan_tier"`
	AllocatedSeats int    `json:"allocated_seats"`
	Role           string `json:"role"`
	CreditsLimit   int    `json:"monthly_shared_credits"`
	CreditsUsed    int    `json:"credits_consumed"`
	MemberCount    int    `json:"member_count"`
	CreatedAt      string `json:"created_at"`
	// SummaryLine is backend-owned row meta (role, members, credits, tier).
	SummaryLine string `json:"summary_line"`
}

type memberRow struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	Role      string `json:"role"`
	RoleLabel string `json:"role_label"`
	JoinedAt  string `json:"joined_at"`
}

type inviteRow struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	RoleLabel string `json:"role_label"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
	InviteURL string `json:"invite_url,omitempty"`
	// SummaryLine is backend-owned invite meta (role · expires …).
	SummaryLine string `json:"summary_line"`
}

func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}
	db := h.readPool()
	maxLimit, err := billingsettings.MaxPageSize(r.Context(), db)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	skipCap, err := billingsettings.SkipToMaxPages(r.Context(), db)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	params, err := pagination.Parse(r, maxLimit)
	if err != nil {
		writeJSONErr(w, http.StatusBadRequest, err.Error())
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 200 {
		q = q[:200]
	}

	where := `wm.user_id = $1`
	args := []interface{}{userID}
	argN := 2
	if q != "" {
		like := "%" + escapeILikePattern(q) + "%"
		where += fmt.Sprintf(` and (
			coalesce(w.name, '') ilike $%d escape '\'
			or coalesce(w.plan_tier, '') ilike $%d escape '\'
			or coalesce(wm.role, '') ilike $%d escape '\'
			or w.id::text ilike $%d escape '\'
		)`, argN, argN, argN, argN)
		args = append(args, like)
		argN++
	}

	var total int
	countSQL := `
		select count(*)
		from public.workspace_members wm
		join public.workspaces w on w.id = wm.workspace_id
		where ` + where
	if err := db.QueryRow(r.Context(), countSQL, args...).Scan(&total); err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_COUNT_FAILED")
		return
	}

	listSQL := fmt.Sprintf(`
		select w.id::text, w.name, w.plan_tier, w.allocated_seats, wm.role,
		       q.monthly_shared_credits, q.credits_consumed,
		       (select count(*) from public.workspace_members m where m.workspace_id = w.id),
		       w.created_at::text
		from public.workspaces w
		join public.workspace_members wm on wm.workspace_id = w.id
		inner join public.workspace_quotas q on q.workspace_id = w.id
		where %s
		order by w.created_at desc
		offset $%d limit $%d
	`, where, argN, argN+1)
	listArgs := append(append([]interface{}{}, args...), params.Skip, params.Limit)
	rows, err := db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_LIST_FAILED")
		return
	}
	defer rows.Close()

	items := make([]workspaceRow, 0)
	for rows.Next() {
		var row workspaceRow
		if err := rows.Scan(
			&row.ID, &row.Name, &row.PlanTier, &row.AllocatedSeats, &row.Role,
			&row.CreditsLimit, &row.CreditsUsed, &row.MemberCount, &row.CreatedAt,
		); err != nil {
			writeJSONErr(w, http.StatusInternalServerError, "WS_SCAN_FAILED")
			return
		}
		sep := subscriptions.MessageForCode("WORKSPACE_META_SEP")
		roleLabel := subscriptions.RoleLabel(row.Role)
		seatsFmt := subscriptions.MessageForCode("WORKSPACE_META_SEATS_FMT")
		seatsPart := ""
		if seatsFmt != "" {
			seatsPart = fmt.Sprintf(seatsFmt, row.MemberCount, row.AllocatedSeats)
		}
		row.SummaryLine = fmt.Sprintf(
			"%s%s%d%s%s%d/%d%s%s%s%s%s",
			roleLabel,
			sep,
			row.MemberCount,
			subscriptions.MessageForCode("WORKSPACE_META_MEMBERS_UNIT"),
			sep,
			row.CreditsUsed,
			row.CreditsLimit,
			subscriptions.MessageForCode("WORKSPACE_META_CREDITS_UNIT"),
			sep,
			seatsPart,
			sep,
			row.PlanTier,
		)
		items = append(items, row)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"items":                       items,
		"meta":                        pagination.BuildMeta(params, total, skipCap),
		"create_action_label":         subscriptions.ActionLabelForCode("WORKSPACE_CREATE"),
		"create_pending_label":        subscriptions.PendingLabelForCode("WORKSPACE_CREATE"),
		"invite_action_label":         subscriptions.ActionLabelForCode("WORKSPACE_INVITE"),
		"invite_pending_label":        subscriptions.PendingLabelForCode("WORKSPACE_INVITE"),
		"remove_action_label":         subscriptions.ActionLabelForCode("WORKSPACE_REMOVE"),
		"remove_pending_label":        subscriptions.PendingLabelForCode("WORKSPACE_REMOVE"),
		"remove_confirm_message":      subscriptions.MessageForCode("WORKSPACE_REMOVE_CONFIRM"),
		"leave_action_label":          subscriptions.ActionLabelForCode("WORKSPACE_LEAVE"),
		"leave_pending_label":         subscriptions.PendingLabelForCode("WORKSPACE_LEAVE"),
		"leave_confirm_message":       subscriptions.MessageForCode("WORKSPACE_LEAVE_CONFIRM"),
		"back_action_label":           subscriptions.MessageForCode("WORKSPACE_BACK_LABEL"),
		"back_href":                   subscriptions.MessageForCode("APP_PATH_DASHBOARD"),
		"page_eyebrow":                subscriptions.MessageForCode("WORKSPACE_PAGE_EYEBROW"),
		"page_title":                  subscriptions.MessageForCode("WORKSPACE_PAGE_TITLE"),
		"page_description":            subscriptions.MessageForCode("WORKSPACE_PAGE_DESCRIPTION"),
		"list_title":                  subscriptions.MessageForCode("WORKSPACE_LIST_TITLE"),
		"name_placeholder":            subscriptions.MessageForCode("WORKSPACE_NAME_PLACEHOLDER"),
		"name_label":                  subscriptions.MessageForCode("WORKSPACE_NAME_LABEL"),
		"name_description":            subscriptions.MessageForCode("WORKSPACE_NAME_DESC"),
		"members_title":               subscriptions.MessageForCode("WORKSPACE_MEMBERS_TITLE"),
		"invites_title":               subscriptions.MessageForCode("WORKSPACE_INVITES_TITLE"),
		"invite_email_placeholder":    subscriptions.MessageForCode("WORKSPACE_INVITE_EMAIL_PLACEHOLDER"),
		"invite_email_label":          subscriptions.MessageForCode("WORKSPACE_INVITE_EMAIL_LABEL"),
		"invite_email_description":    subscriptions.MessageForCode("WORKSPACE_INVITE_EMAIL_DESC"),
		"select_hint":                 subscriptions.MessageForCode("WORKSPACE_SELECT_HINT"),
		"empty_message":               subscriptions.MessageForCode("WORKSPACE_EMPTY"),
		"invite_url_label":            subscriptions.MessageForCode("WORKSPACE_INVITE_URL_LABEL"),
		"invites_empty_message":       subscriptions.MessageForCode("WORKSPACE_INVITES_EMPTY"),
		"revoke_action_label":         subscriptions.ActionLabelForCode("WORKSPACE_INVITE_REVOKE"),
		"revoke_pending_label":        subscriptions.PendingLabelForCode("WORKSPACE_INVITE_REVOKE"),
		"copy_invite_action_label":    subscriptions.ActionLabelForCode("WORKSPACE_INVITE_COPY"),
		"copy_invite_pending_label":   subscriptions.PendingLabelForCode("WORKSPACE_INVITE_COPY"),
		"copied_invite_action_label":  subscriptions.ActionLabelForCode("WORKSPACE_INVITE_COPIED"),
		"table_select_all":            subscriptions.MessageForCode("TABLE_SELECT_ALL"),
		"table_select_row":            subscriptions.MessageForCode("TABLE_SELECT_ROW"),
		"table_selected_fmt":          subscriptions.MessageForCode("TABLE_SELECTED_FMT"),
		"table_row_actions":           subscriptions.MessageForCode("TABLE_ROW_ACTIONS"),
		"table_bulk_revoke":           subscriptions.MessageForCode("TABLE_BULK_REVOKE"),
		"table_bulk_remove":           subscriptions.MessageForCode("TABLE_BULK_REMOVE"),
		"table_bulk_delete":           subscriptions.MessageForCode("TABLE_BULK_DELETE"),
		"table_clear_selection":       subscriptions.MessageForCode("TABLE_CLEAR_SELECTION"),
		"search_placeholder":          subscriptions.MessageForCode("WORKSPACE_SEARCH"),
		"search_description":          subscriptions.MessageForCode("WORKSPACE_SEARCH_DESC"),
		"rename_action_label":         subscriptions.ActionLabelForCode("WORKSPACE_RENAME"),
		"rename_pending_label":        subscriptions.PendingLabelForCode("WORKSPACE_RENAME"),
		"rename_save_action_label":    subscriptions.ActionLabelForCode("WORKSPACE_RENAME_SAVE"),
		"rename_save_pending_label":   subscriptions.PendingLabelForCode("WORKSPACE_RENAME_SAVE"),
		"rename_title":                subscriptions.MessageForCode("WORKSPACE_RENAME_TITLE"),
		"delete_action_label":         subscriptions.ActionLabelForCode("WORKSPACE_DELETE"),
		"delete_pending_label":        subscriptions.PendingLabelForCode("WORKSPACE_DELETE"),
		"delete_confirm_message":      subscriptions.MessageForCode("WORKSPACE_DELETE_CONFIRM"),
		"bulk_delete_confirm_message": subscriptions.MessageForCode("WORKSPACE_BULK_DELETE_CONFIRM"),
		"role_change_action_label":    subscriptions.ActionLabelForCode("WORKSPACE_ROLE_CHANGE"),
		"role_change_pending_label":   subscriptions.PendingLabelForCode("WORKSPACE_ROLE_CHANGE"),
		"role_change_title":           subscriptions.MessageForCode("WORKSPACE_ROLE_CHANGE_TITLE"),
		"role_label":                  subscriptions.MessageForCode("WORKSPACE_ROLE_LABEL"),
		"role_description":            subscriptions.MessageForCode("WORKSPACE_ROLE_DESCRIPTION"),
		"invite_roles":                roleOptions("member", "admin"),
		"member_roles":                roleOptions("member", "admin", "owner"),
		"q":                           q,
	})
}

type createReq struct {
	Name string `json:"name"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}
	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeJSONErr(w, http.StatusBadRequest, "WS_NAME_REQUIRED")
		return
	}

	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_TX_BEGIN_FAILED")
		return
	}
	defer tx.Rollback(r.Context())

	var planTierNS *string
	allocatedSeats := 0
	var credits int
	err = tx.QueryRow(r.Context(), `
		select (
			select s.plan_tier
			from public.subscriptions s
			inner join public.plan_catalog p on p.id = s.plan_tier and p.is_active = true
			where s.user_id = $1
			  and s.status in ('active', 'trialing', 'past_due')
			  and coalesce(s.expires_at, s.current_period_end) > now()
			order by p.plan_rank desc nulls last, s.updated_at desc nulls last
			limit 1
		)
	`, userID).Scan(&planTierNS)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_RESOLVE_PLAN_FAILED")
		return
	}
	planTier := ""
	if planTierNS != nil {
		planTier = strings.TrimSpace(*planTierNS)
	}
	// When no active subscription row yet, inherit plan_tier from user_quotas (webhook-provisioned).
	fromQuota := false
	var quotaLimit int
	if planTier == "" {
		var qt string
		qErr := tx.QueryRow(r.Context(), `
			select uq.plan_tier, coalesce(uq.monthly_credit_limit, 0)
			from public.user_quotas uq
			inner join public.plan_catalog p on p.id = uq.plan_tier and p.is_active = true
			where uq.user_id = $1::uuid
		`, userID).Scan(&qt, &quotaLimit)
		if qErr == nil {
			planTier = strings.TrimSpace(qt)
			fromQuota = planTier != ""
		}
	}
	if planTier == "" {
		planTier = subscriptions.MessageForCode("DEFAULT_PLAN_TIER")
		if planTier == "" {
			writeJSONErr(w, http.StatusInternalServerError, "WS_DEFAULT_PLAN_TIER_MISSING")
			return
		}
	}
	// Per-seat catalog plans: inherit seat_quantity from the active Paddle subscription (never invent plan ids).
	var perSeat bool
	var planKind string
	err = tx.QueryRow(r.Context(), `
		select coalesce(per_seat, false), plan_kind
		from public.plan_catalog
		where id = $1 and is_active = true
	`, planTier).Scan(&perSeat, &planKind)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_PLAN_CATALOG_MISSING")
		return
	}
	if perSeat {
		var seats int
		err = tx.QueryRow(r.Context(), `
			select s.seat_quantity
			from public.subscriptions s
			where s.user_id = $1
			  and s.plan_tier = $2
			  and s.status in ('active', 'trialing', 'past_due')
			  and coalesce(s.expires_at, s.current_period_end) > now()
			order by s.updated_at desc nulls last
			limit 1
		`, userID, planTier).Scan(&seats)
		if err != nil && fromQuota {
			// Prefer seats from a Paddle-fulfilled enterprise inquiry, else max owned workspace seats.
			_ = tx.QueryRow(r.Context(), `
				select coalesce(
					(select ei.offered_seat_quantity
					 from public.enterprise_inquiries ei
					 where ei.user_id = $1::uuid and ei.status = 'activated'
					   and ei.offered_seat_quantity is not null and ei.offered_seat_quantity >= 1
					   and (
					     coalesce(ei.paddle_subscription_id, '') <> ''
					     or coalesce(ei.paddle_transaction_id, '') <> ''
					   )
					 order by ei.updated_at desc nulls last
					 limit 1),
					(select max(w.allocated_seats)
					 from public.workspaces w
					 where w.owner_id = $1::uuid and w.allocated_seats >= 1),
					0
				)
			`, userID).Scan(&seats)
			err = nil
			if seats < 1 {
				err = pgx.ErrNoRows
			}
		}
		if err != nil {
			writeJSONErr(w, http.StatusInternalServerError, "WS_RESOLVE_SEATS_FAILED")
			return
		}
		if seats < 1 {
			writeJSONErr(w, http.StatusInternalServerError, "WS_SEAT_QUANTITY_INVALID")
			return
		}
		allocatedSeats = seats
	} else {
		// Free / non-per-seat: never allocate 0 seats (invite gates require >= 1).
		if n, seatErr := billingsettings.DefaultSeatQuantity(r.Context(), h.DB); seatErr == nil && n >= 1 {
			allocatedSeats = n
		} else {
			allocatedSeats = 1
		}
	}
	err = tx.QueryRow(r.Context(), `
		select credits_monthly
		from public.plan_catalog
		where id = $1 and is_active = true
	`, planTier).Scan(&credits)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_PLAN_CREDITS_MISSING")
		return
	}
	if perSeat {
		credits = credits * allocatedSeats
	}
	// Prefer paid-quota limit when it already encodes seat×credits (Paddle webhook / enterprise).
	if fromQuota && quotaLimit > 0 && (perSeat || planKind == "enterprise") {
		credits = quotaLimit
	}

	var workspaceID string
	err = tx.QueryRow(r.Context(), `
		insert into public.workspaces (name, owner_id, plan_tier, allocated_seats)
		values ($1, $2, $3, $4)
		returning id::text
	`, req.Name, userID, planTier, allocatedSeats).Scan(&workspaceID)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_CREATE_FAILED")
		return
	}
	if _, err := tx.Exec(r.Context(), `
		insert into public.workspace_members (workspace_id, user_id, role)
		values ($1::uuid, $2, 'owner')
	`, workspaceID, userID); err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_ADD_OWNER_FAILED")
		return
	}
	if _, err := tx.Exec(r.Context(), `
		insert into public.workspace_quotas (workspace_id, monthly_shared_credits, credits_consumed)
		values ($1::uuid, $2, 0)
	`, workspaceID, credits); err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_INIT_QUOTA_FAILED")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_COMMIT_FAILED")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":                     workspaceID,
		"name":                   req.Name,
		"plan_tier":              planTier,
		"allocated_seats":        allocatedSeats,
		"monthly_shared_credits": credits,
		"action_label":           subscriptions.ActionLabelForCode("WORKSPACE_CREATE"),
		"pending_label":          subscriptions.PendingLabelForCode("WORKSPACE_CREATE"),
		"message":                subscriptions.MessageForCode("WORKSPACE_CREATED"),
	})
}

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	workspaceID := chi.URLParam(r, "workspaceId")
	if userID == "" || workspaceID == "" {
		writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}
	if !h.isMember(r, userID, workspaceID) {
		writeJSONErr(w, http.StatusForbidden, "WS_FORBIDDEN")
		return
	}

	db := h.readPool()
	maxLimit, err := billingsettings.MaxPageSize(r.Context(), db)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	skipCap, err := billingsettings.SkipToMaxPages(r.Context(), db)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	params, err := pagination.Parse(r, maxLimit)
	if err != nil {
		writeJSONErr(w, http.StatusBadRequest, err.Error())
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 200 {
		q = q[:200]
	}

	where := `wm.workspace_id = $1::uuid`
	args := []interface{}{workspaceID}
	argN := 2
	if q != "" {
		like := "%" + escapeILikePattern(q) + "%"
		where += fmt.Sprintf(` and (
			coalesce(p.email, '') ilike $%d escape '\'
			or coalesce(p.full_name, '') ilike $%d escape '\'
			or coalesce(wm.role, '') ilike $%d escape '\'
			or wm.user_id::text ilike $%d escape '\'
		)`, argN, argN, argN, argN)
		args = append(args, like)
		argN++
	}

	var total int
	countSQL := `
		select count(*)
		from public.workspace_members wm
		join public.profiles p on p.id = wm.user_id
		where ` + where
	if err := db.QueryRow(r.Context(), countSQL, args...).Scan(&total); err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_MEMBERS_COUNT_FAILED")
		return
	}

	var ownerCount int
	if err := db.QueryRow(r.Context(), `
		select count(*) from public.workspace_members
		where workspace_id = $1::uuid and role = 'owner'
	`, workspaceID).Scan(&ownerCount); err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_OWNERS_COUNT_FAILED")
		return
	}

	listSQL := fmt.Sprintf(`
		select wm.id::text, wm.user_id::text, p.email, coalesce(p.full_name, ''), wm.role, wm.joined_at::text
		from public.workspace_members wm
		join public.profiles p on p.id = wm.user_id
		where %s
		order by wm.joined_at asc
		offset $%d limit $%d
	`, where, argN, argN+1)
	listArgs := append(append([]interface{}{}, args...), params.Skip, params.Limit)
	rows, err := db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_MEMBERS_LIST_FAILED")
		return
	}
	defer rows.Close()

	items := make([]memberRow, 0)
	for rows.Next() {
		var m memberRow
		if err := rows.Scan(&m.ID, &m.UserID, &m.Email, &m.FullName, &m.Role, &m.JoinedAt); err != nil {
			writeJSONErr(w, http.StatusInternalServerError, "WS_MEMBER_SCAN_FAILED")
			return
		}
		m.RoleLabel = subscriptions.RoleLabel(m.Role)
		items = append(items, m)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"items":                      items,
		"meta":                       pagination.BuildMeta(params, total, skipCap),
		"owner_count":                ownerCount,
		"invite_action_label":        subscriptions.ActionLabelForCode("WORKSPACE_INVITE"),
		"invite_pending_label":       subscriptions.PendingLabelForCode("WORKSPACE_INVITE"),
		"remove_action_label":        subscriptions.ActionLabelForCode("WORKSPACE_REMOVE"),
		"remove_pending_label":       subscriptions.PendingLabelForCode("WORKSPACE_REMOVE"),
		"remove_confirm_message":     subscriptions.MessageForCode("WORKSPACE_REMOVE_CONFIRM"),
		"leave_action_label":         subscriptions.ActionLabelForCode("WORKSPACE_LEAVE"),
		"leave_pending_label":        subscriptions.PendingLabelForCode("WORKSPACE_LEAVE"),
		"leave_confirm_message":      subscriptions.MessageForCode("WORKSPACE_LEAVE_CONFIRM"),
		"revoke_action_label":        subscriptions.ActionLabelForCode("WORKSPACE_INVITE_REVOKE"),
		"revoke_pending_label":       subscriptions.PendingLabelForCode("WORKSPACE_INVITE_REVOKE"),
		"copy_invite_action_label":   subscriptions.ActionLabelForCode("WORKSPACE_INVITE_COPY"),
		"copy_invite_pending_label":  subscriptions.PendingLabelForCode("WORKSPACE_INVITE_COPY"),
		"copied_invite_action_label": subscriptions.ActionLabelForCode("WORKSPACE_INVITE_COPIED"),
		"table_select_all":           subscriptions.MessageForCode("TABLE_SELECT_ALL"),
		"table_select_row":           subscriptions.MessageForCode("TABLE_SELECT_ROW"),
		"table_selected_fmt":         subscriptions.MessageForCode("TABLE_SELECTED_FMT"),
		"table_row_actions":          subscriptions.MessageForCode("TABLE_ROW_ACTIONS"),
		"table_bulk_revoke":          subscriptions.MessageForCode("TABLE_BULK_REVOKE"),
		"table_bulk_remove":          subscriptions.MessageForCode("TABLE_BULK_REMOVE"),
		"table_clear_selection":      subscriptions.MessageForCode("TABLE_CLEAR_SELECTION"),
		"empty_message":              subscriptions.MessageForCode("WORKSPACE_MEMBERS_EMPTY"),
		"search_placeholder":         subscriptions.MessageForCode("WORKSPACE_MEMBERS_SEARCH"),
		"search_description":         subscriptions.MessageForCode("WORKSPACE_MEMBERS_SEARCH_DESC"),
		"role_change_action_label":   subscriptions.ActionLabelForCode("WORKSPACE_ROLE_CHANGE"),
		"role_change_pending_label":  subscriptions.PendingLabelForCode("WORKSPACE_ROLE_CHANGE"),
		"role_change_title":          subscriptions.MessageForCode("WORKSPACE_ROLE_CHANGE_TITLE"),
		"role_label":                 subscriptions.MessageForCode("WORKSPACE_ROLE_LABEL"),
		"role_description":           subscriptions.MessageForCode("WORKSPACE_ROLE_DESCRIPTION"),
		"invite_roles":               roleOptions("member", "admin"),
		"member_roles":               roleOptions("member", "admin", "owner"),
		"q":                          q,
	})
}

type inviteReq struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// Invite creates a pending email invite with a one-time accept URL.
// Invitee does not need an existing Trim account yet.
func (h *Handler) Invite(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	workspaceID := chi.URLParam(r, "workspaceId")
	if userID == "" || workspaceID == "" {
		writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}
	role, ok := h.memberRole(r, userID, workspaceID)
	if !ok || (role != "owner" && role != "admin") {
		writeJSONErr(w, http.StatusForbidden, "WS_FORBIDDEN")
		return
	}
	if h.AppPublicURL == "" {
		writeJSONErr(w, http.StatusInternalServerError, "WS_APP_PUBLIC_URL_MISSING")
		return
	}

	var req inviteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Role = strings.TrimSpace(strings.ToLower(req.Role))
	inviteRoles := roleOptions("member", "admin")
	if len(inviteRoles) == 0 {
		writeJSONErr(w, http.StatusServiceUnavailable, "WS_INVALID_MEMBER_ROLE")
		return
	}
	roleAllowed := false
	for _, opt := range inviteRoles {
		if opt.ID == req.Role {
			roleAllowed = true
			break
		}
	}
	if !roleAllowed {
		writeJSONErr(w, http.StatusBadRequest, "WS_INVALID_MEMBER_ROLE")
		return
	}
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		writeJSONErr(w, http.StatusBadRequest, "WS_EMAIL_REQUIRED")
		return
	}
	if _, ok := emaildenylist.DomainOf(req.Email); !ok {
		writeJSONErr(w, http.StatusBadRequest, "WS_EMAIL_REQUIRED")
		return
	}
	if denied, err := emaildenylist.IsDenied(r.Context(), h.DB, req.Email); err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_EMAIL_DOMAIN_DENIED")
		return
	} else if denied {
		writeJSONErr(w, http.StatusBadRequest, "WS_EMAIL_DOMAIN_DENIED")
		return
	}

	var actorEmail string
	if err := h.DB.QueryRow(r.Context(), `
		select lower(email) from public.profiles where id = $1::uuid
	`, userID).Scan(&actorEmail); err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_INVITER_LOAD_FAILED")
		return
	}
	if actorEmail == req.Email {
		writeJSONErr(w, http.StatusBadRequest, "WS_CANNOT_INVITE_SELF")
		return
	}

	var alreadyMember bool
	_ = h.DB.QueryRow(r.Context(), `
		select exists(
			select 1
			from public.workspace_members wm
			join public.profiles p on p.id = wm.user_id
			where wm.workspace_id = $1::uuid and lower(p.email) = $2
		)
	`, workspaceID, req.Email).Scan(&alreadyMember)
	if alreadyMember {
		writeJSONErr(w, http.StatusConflict, "WS_ALREADY_MEMBER")
		return
	}

	var ttlHours int
	err := h.DB.QueryRow(r.Context(), `
		select workspace_invite_ttl_hours
		from public.billing_settings
		where id = 'default'
	`).Scan(&ttlHours)
	if err != nil || ttlHours <= 0 {
		writeJSONErr(w, http.StatusInternalServerError, "WS_INVITE_TTL_MISSING")
		return
	}

	rawToken, err := generateInviteToken()
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_INVITE_TOKEN_GEN_FAILED")
		return
	}
	tokenHash := hashToken(rawToken)
	expiresAt := time.Now().UTC().Add(time.Duration(ttlHours) * time.Hour)

	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_TX_BEGIN_FAILED")
		return
	}
	defer tx.Rollback(r.Context())

	// Lock workspace, revoke same-email pending invite, then capacity-check so a
	// re-invite does not false-fail (that invite already occupied a pending seat).
	var allocatedSeats int
	err = tx.QueryRow(r.Context(), `
		select w.allocated_seats
		from public.workspaces w
		where w.id = $1::uuid
		for update of w
	`, workspaceID).Scan(&allocatedSeats)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_SEAT_CAPACITY_LOAD_FAILED")
		return
	}
	if _, err := tx.Exec(r.Context(), `
		update public.workspace_invites
		set status = 'revoked', updated_at = now()
		where workspace_id = $1::uuid
		  and lower(email) = $2
		  and status = 'pending'
	`, workspaceID, req.Email); err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_REVOKE_PRIOR_INVITE_FAILED")
		return
	}
	var memberCount, pendingCount int
	err = tx.QueryRow(r.Context(), `
		select
		  (select count(*) from public.workspace_members m where m.workspace_id = $1::uuid),
		  (select count(*) from public.workspace_invites i
		    where i.workspace_id = $1::uuid and i.status = 'pending' and i.expires_at > now())
	`, workspaceID).Scan(&memberCount, &pendingCount)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_SEAT_CAPACITY_LOAD_FAILED")
		return
	}
	if memberCount+pendingCount >= allocatedSeats {
		writeJSONErr(w, http.StatusConflict, "WS_NO_SEATS_INVITE")
		return
	}

	var inviteID string
	err = tx.QueryRow(r.Context(), `
		insert into public.workspace_invites (
			workspace_id, email, role, token_hash, invited_by, status, expires_at
		) values (
			$1::uuid, $2, $3, $4, $5::uuid, 'pending', $6
		)
		returning id::text
	`, workspaceID, req.Email, req.Role, tokenHash, userID, expiresAt).Scan(&inviteID)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_CREATE_INVITE_FAILED")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_COMMIT_FAILED")
		return
	}

	invitePrefix := strings.TrimSpace(subscriptions.MessageForCode("APP_PATH_INVITE_PREFIX"))
	if invitePrefix == "" {
		writeJSONErr(w, http.StatusInternalServerError, "WS_INVITE_PATH_MISSING")
		return
	}
	if !strings.HasPrefix(invitePrefix, "/") || strings.HasPrefix(invitePrefix, "//") {
		writeJSONErr(w, http.StatusInternalServerError, "WS_INVITE_PATH_INVALID")
		return
	}
	inviteURL := strings.TrimRight(h.AppPublicURL, "/") + strings.TrimSuffix(invitePrefix, "/") + "/" + rawToken

	var workspaceName string
	_ = h.DB.QueryRow(r.Context(), `
		select name from public.workspaces where id = $1::uuid
	`, workspaceID).Scan(&workspaceName)

	emailSent := false
	emailError := ""
	roleLabel := subscriptions.RoleLabel(req.Role)
	if roleLabel == "" {
		roleLabel = req.Role
	}
	if h.Mailer.Enabled() {
		subject := subscriptions.MessageForCode("WORKSPACE_INVITE_EMAIL_SUBJECT_FALLBACK")
		if workspaceName != "" {
			subject = fmt.Sprintf(
				subscriptions.MessageForCode("WORKSPACE_INVITE_EMAIL_SUBJECT_NAMED_FMT"),
				workspaceName,
			)
		}
		body := fmt.Sprintf(
			subscriptions.MessageForCode("WORKSPACE_INVITE_EMAIL_BODY_FMT"),
			roleLabel,
			inviteURL,
			expiresAt.Format(time.RFC3339),
		)
		if err := h.Mailer.SendPlain(req.Email, subject, body); err != nil {
			log.Printf("workspaces: invite email to %s failed: %v", req.Email, err)
			emailError = subscriptions.MessageForCode("WORKSPACE_INVITE_MSG_SMTP_FAILED")
		} else {
			emailSent = true
		}
	}

	message := subscriptions.MessageForCode("WORKSPACE_INVITE_MSG_SHARE")
	if emailSent {
		message = subscriptions.MessageForCode("WORKSPACE_INVITE_MSG_SENT")
	} else if emailError != "" {
		message = emailError
	} else if !h.Mailer.Enabled() {
		message = subscriptions.MessageForCode("WORKSPACE_INVITE_MSG_SMTP_OFF")
	}

	var inviteeID string
	_ = h.DB.QueryRow(r.Context(), `
		select id::text from public.profiles where lower(email) = $1
	`, req.Email).Scan(&inviteeID)
	if inviteeID != "" {
		_ = notifications.Insert(r.Context(), h.DB, notifications.InsertOpts{
			RecipientID: inviteeID,
			Audience:    notifications.AudienceUser,
			KindCode:    notifications.KindUserWorkspaceInvite,
			BodyArgs:    []string{workspaceName, roleLabel},
			DedupeKey:   "ws_invite:" + inviteID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":              "pending",
		"invite_id":           inviteID,
		"email":               req.Email,
		"role":                req.Role,
		"expires_at":          expiresAt.Format(time.RFC3339),
		"invite_url":          inviteURL,
		"email_sent":          emailSent,
		"message":             message,
		"action_label":        subscriptions.ActionLabelForCode("WORKSPACE_INVITE"),
		"pending_label":       subscriptions.PendingLabelForCode("WORKSPACE_INVITE"),
		"copy_action_label":   subscriptions.ActionLabelForCode("WORKSPACE_INVITE_COPY"),
		"copy_pending_label":  subscriptions.PendingLabelForCode("WORKSPACE_INVITE_COPY"),
		"copied_action_label": subscriptions.ActionLabelForCode("WORKSPACE_INVITE_COPIED"),
	})
}

func (h *Handler) ListInvites(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	workspaceID := chi.URLParam(r, "workspaceId")
	if userID == "" || workspaceID == "" {
		writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}
	role, ok := h.memberRole(r, userID, workspaceID)
	if !ok || (role != "owner" && role != "admin") {
		writeJSONErr(w, http.StatusForbidden, "WS_FORBIDDEN")
		return
	}

	db := h.readPool()
	maxLimit, err := billingsettings.MaxPageSize(r.Context(), db)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	skipCap, err := billingsettings.SkipToMaxPages(r.Context(), db)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	params, err := pagination.Parse(r, maxLimit)
	if err != nil {
		writeJSONErr(w, http.StatusBadRequest, err.Error())
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 200 {
		q = q[:200]
	}

	// Expire stale pending rows opportunistically.
	_, _ = h.DB.Exec(r.Context(), `
		update public.workspace_invites
		set status = 'expired', updated_at = now()
		where workspace_id = $1::uuid
		  and status = 'pending'
		  and expires_at <= now()
	`, workspaceID)

	where := `workspace_id = $1::uuid and status = 'pending'`
	args := []interface{}{workspaceID}
	argN := 2
	if q != "" {
		like := "%" + escapeILikePattern(q) + "%"
		where += fmt.Sprintf(` and (
			coalesce(email, '') ilike $%d escape '\'
			or coalesce(role, '') ilike $%d escape '\'
			or coalesce(status, '') ilike $%d escape '\'
			or id::text ilike $%d escape '\'
		)`, argN, argN, argN, argN)
		args = append(args, like)
		argN++
	}

	var total int
	countSQL := `select count(*) from public.workspace_invites where ` + where
	if err := db.QueryRow(r.Context(), countSQL, args...).Scan(&total); err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_INVITES_COUNT_FAILED")
		return
	}

	listSQL := fmt.Sprintf(`
		select id::text, email, role, status, expires_at::text, created_at::text
		from public.workspace_invites
		where %s
		order by created_at desc
		offset $%d limit $%d
	`, where, argN, argN+1)
	listArgs := append(append([]interface{}{}, args...), params.Skip, params.Limit)
	rows, err := db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_INVITES_LIST_FAILED")
		return
	}
	defer rows.Close()

	items := make([]inviteRow, 0)
	for rows.Next() {
		var inv inviteRow
		if err := rows.Scan(&inv.ID, &inv.Email, &inv.Role, &inv.Status, &inv.ExpiresAt, &inv.CreatedAt); err != nil {
			writeJSONErr(w, http.StatusInternalServerError, "WS_INVITE_SCAN_FAILED")
			return
		}
		inv.RoleLabel = subscriptions.RoleLabel(inv.Role)
		inv.SummaryLine = fmt.Sprintf(
			"%s%s%s%s",
			inv.RoleLabel,
			subscriptions.MessageForCode("WORKSPACE_META_SEP"),
			subscriptions.MessageForCode("WORKSPACE_INVITE_EXPIRES_PREFIX"),
			subscriptions.FormatUTCDateTimeFromRFC3339(inv.ExpiresAt),
		)
		items = append(items, inv)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"items":                      items,
		"meta":                       pagination.BuildMeta(params, total, skipCap),
		"revoke_action_label":        subscriptions.ActionLabelForCode("WORKSPACE_INVITE_REVOKE"),
		"revoke_pending_label":       subscriptions.PendingLabelForCode("WORKSPACE_INVITE_REVOKE"),
		"invite_action_label":        subscriptions.ActionLabelForCode("WORKSPACE_INVITE"),
		"invite_pending_label":       subscriptions.PendingLabelForCode("WORKSPACE_INVITE"),
		"copy_invite_action_label":   subscriptions.ActionLabelForCode("WORKSPACE_INVITE_COPY"),
		"copy_invite_pending_label":  subscriptions.PendingLabelForCode("WORKSPACE_INVITE_COPY"),
		"copied_invite_action_label": subscriptions.ActionLabelForCode("WORKSPACE_INVITE_COPIED"),
		"table_select_all":           subscriptions.MessageForCode("TABLE_SELECT_ALL"),
		"table_select_row":           subscriptions.MessageForCode("TABLE_SELECT_ROW"),
		"table_selected_fmt":         subscriptions.MessageForCode("TABLE_SELECTED_FMT"),
		"table_row_actions":          subscriptions.MessageForCode("TABLE_ROW_ACTIONS"),
		"table_bulk_revoke":          subscriptions.MessageForCode("TABLE_BULK_REVOKE"),
		"table_bulk_remove":          subscriptions.MessageForCode("TABLE_BULK_REMOVE"),
		"table_clear_selection":      subscriptions.MessageForCode("TABLE_CLEAR_SELECTION"),
		"search_placeholder":         subscriptions.MessageForCode("WORKSPACE_INVITES_SEARCH"),
		"search_description":         subscriptions.MessageForCode("WORKSPACE_INVITES_SEARCH_DESC"),
		"q":                          q,
	})
}

func (h *Handler) RevokeInvite(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	workspaceID := chi.URLParam(r, "workspaceId")
	inviteID := chi.URLParam(r, "inviteId")
	if userID == "" || workspaceID == "" || inviteID == "" {
		writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}
	role, ok := h.memberRole(r, userID, workspaceID)
	if !ok || (role != "owner" && role != "admin") {
		writeJSONErr(w, http.StatusForbidden, "WS_FORBIDDEN")
		return
	}

	tag, err := h.DB.Exec(r.Context(), `
		update public.workspace_invites
		set status = 'revoked', updated_at = now()
		where id = $1::uuid
		  and workspace_id = $2::uuid
		  and status = 'pending'
	`, inviteID, workspaceID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": subscriptions.MessageForCode("INVITE_REVOKE_FAILED")})
		return
	}
	if tag.RowsAffected() == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": subscriptions.MessageForCode("INVITE_NOT_FOUND_OR_CLOSED")})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":        "revoked",
		"action_label":  subscriptions.ActionLabelForCode("WORKSPACE_INVITE_REVOKE"),
		"pending_label": subscriptions.PendingLabelForCode("WORKSPACE_INVITE_REVOKE"),
		"message":       subscriptions.MessageForCode("WORKSPACE_INVITE_REVOKED"),
	})
}

// PreviewInvite is public: shows workspace name and role for a raw token (no PII dump).
func (h *Handler) PreviewInvite(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(chi.URLParam(r, "token"))
	if token == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": subscriptions.MessageForCode("INVITE_TOKEN_REQUIRED")})
		return
	}
	tokenHash := hashToken(token)

	var (
		inviteID, workspaceID, workspaceName, email, role, status string
		expiresAt                                                 time.Time
	)
	err := h.readPool().QueryRow(r.Context(), `
		select i.id::text, i.workspace_id::text, w.name, i.email, i.role, i.status, i.expires_at
		from public.workspace_invites i
		join public.workspaces w on w.id = i.workspace_id
		where i.token_hash = $1
	`, tokenHash).Scan(&inviteID, &workspaceID, &workspaceName, &email, &role, &status, &expiresAt)
	if err == pgx.ErrNoRows {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": subscriptions.MessageForCode("INVITE_NOT_FOUND")})
		return
	}
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_INVITE_LOAD_FAILED")
		return
	}

	if status == "pending" && expiresAt.Before(time.Now().UTC()) {
		_, _ = h.DB.Exec(r.Context(), `
			update public.workspace_invites
			set status = 'expired', updated_at = now()
			where id = $1::uuid and status = 'pending'
		`, inviteID)
		status = "expired"
	}

	masked := maskEmail(email)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"workspace_id":           workspaceID,
		"workspace_name":         workspaceName,
		"email_masked":           masked,
		"role":                   role,
		"role_label":             subscriptions.RoleLabel(role),
		"status":                 status,
		"status_label":           subscriptions.InviteStatusLabel(status),
		"expires_at":             expiresAt.UTC().Format(time.RFC3339),
		"expires_at_label":       subscriptions.FormatUTCDateTime(expiresAt),
		"accept_action_label":    subscriptions.ActionLabelForCode("WORKSPACE_INVITE_ACCEPT"),
		"accept_pending_label":   subscriptions.PendingLabelForCode("WORKSPACE_INVITE_ACCEPT"),
		"sign_in_action_label":   subscriptions.ActionLabelForCode("WORKSPACE_INVITE_SIGN_IN"),
		"sign_in_pending_label":  subscriptions.PendingLabelForCode("WORKSPACE_INVITE_SIGN_IN"),
		"sign_in_href":           subscriptions.MessageForCode("APP_PATH_LOGIN"),
		"invite_path_prefix":     subscriptions.MessageForCode("APP_PATH_INVITE_PREFIX"),
		"open_team_action_label": subscriptions.MessageForCode("INVITE_OPEN_TEAM"),
		"open_team_href":         subscriptions.MessageForCode("APP_PATH_TEAM"),
		"eyebrow_label":          subscriptions.MessageForCode("INVITE_EYEBROW"),
		"status_title":           subscriptions.MessageForCode("INVITE_STATUS_TITLE"),
		"body_message":           subscriptions.MessageForCode("INVITE_BODY"),
		"role_prefix":            subscriptions.MessageForCode("INVITE_ROLE_PREFIX"),
		"for_prefix":             subscriptions.MessageForCode("INVITE_FOR_PREFIX"),
		"status_prefix":          subscriptions.MessageForCode("INVITE_STATUS_PREFIX"),
		"expires_prefix":         subscriptions.MessageForCode("INVITE_EXPIRES_PREFIX"),
	})
}

// AcceptInvite joins the authenticated user whose email matches the invite.
func (h *Handler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	token := strings.TrimSpace(chi.URLParam(r, "token"))
	if userID == "" {
		writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}
	if token == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": subscriptions.MessageForCode("INVITE_TOKEN_REQUIRED")})
		return
	}
	tokenHash := hashToken(token)

	var userEmail string
	if err := h.DB.QueryRow(r.Context(), `
		select lower(email) from public.profiles where id = $1::uuid
	`, userID).Scan(&userEmail); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": subscriptions.MessageForCode("PROFILE_NOT_FOUND")})
		return
	}

	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_TX_BEGIN_FAILED")
		return
	}
	defer tx.Rollback(r.Context())

	var (
		inviteID, workspaceID, inviteEmail, inviteRole, status string
		expiresAt                                              time.Time
		allocatedSeats                                         int
	)
	err = tx.QueryRow(r.Context(), `
		select i.id::text, i.workspace_id::text, lower(i.email), i.role, i.status, i.expires_at, w.allocated_seats
		from public.workspace_invites i
		join public.workspaces w on w.id = i.workspace_id
		where i.token_hash = $1
		for update of i, w
	`, tokenHash).Scan(&inviteID, &workspaceID, &inviteEmail, &inviteRole, &status, &expiresAt, &allocatedSeats)
	if err == pgx.ErrNoRows {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": subscriptions.MessageForCode("INVITE_NOT_FOUND")})
		return
	}
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_INVITE_LOAD_FAILED")
		return
	}

	if status != "pending" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": subscriptions.MessageForCode("INVITE_NO_LONGER_PENDING")})
		return
	}
	if expiresAt.Before(time.Now().UTC()) {
		_, _ = tx.Exec(r.Context(), `
			update public.workspace_invites
			set status = 'expired', updated_at = now()
			where id = $1::uuid
		`, inviteID)
		_ = tx.Commit(r.Context())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusGone)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": subscriptions.MessageForCode("INVITE_EXPIRED")})
		return
	}
	if userEmail != inviteEmail {
		writeJSONErr(w, http.StatusForbidden, "WS_EMAIL_MISMATCH")
		return
	}

	var memberCount int
	if err := tx.QueryRow(r.Context(), `
		select count(*) from public.workspace_members where workspace_id = $1::uuid
	`, workspaceID).Scan(&memberCount); err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_MEMBERS_COUNT_FAILED")
		return
	}
	var alreadyMember bool
	_ = tx.QueryRow(r.Context(), `
		select exists(
			select 1 from public.workspace_members
			where workspace_id = $1::uuid and user_id = $2::uuid
		)
	`, workspaceID, userID).Scan(&alreadyMember)
	if !alreadyMember && memberCount >= allocatedSeats {
		writeJSONErr(w, http.StatusConflict, "WS_NO_SEATS_JOIN")
		return
	}

	if !alreadyMember {
		if _, err := tx.Exec(r.Context(), `
			insert into public.workspace_members (workspace_id, user_id, role)
			values ($1::uuid, $2::uuid, $3)
			on conflict (workspace_id, user_id) do nothing
		`, workspaceID, userID, inviteRole); err != nil {
			writeJSONErr(w, http.StatusInternalServerError, "WS_JOIN_FAILED")
			return
		}
	}

	if _, err := tx.Exec(r.Context(), `
		update public.workspace_invites
		set status = 'accepted',
		    accepted_by = $2::uuid,
		    accepted_at = now(),
		    updated_at = now()
		where id = $1::uuid and status = 'pending'
	`, inviteID, userID); err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_MARK_ACCEPTED_FAILED")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_COMMIT_FAILED")
		return
	}

	var joinedName, workspaceName string
	_ = h.DB.QueryRow(r.Context(), `
		select coalesce(nullif(trim(full_name), ''), email) from public.profiles where id = $1::uuid
	`, userID).Scan(&joinedName)
	_ = h.DB.QueryRow(r.Context(), `
		select name from public.workspaces where id = $1::uuid
	`, workspaceID).Scan(&workspaceName)
	roleLabel := subscriptions.RoleLabel(inviteRole)
	if roleLabel == "" {
		roleLabel = inviteRole
	}
	mgrRows, err := h.DB.Query(r.Context(), `
		select user_id::text from public.workspace_members
		where workspace_id = $1::uuid and role in ('owner', 'admin') and user_id <> $2::uuid
	`, workspaceID, userID)
	if err == nil {
		defer mgrRows.Close()
		for mgrRows.Next() {
			var mgrID string
			if err := mgrRows.Scan(&mgrID); err != nil {
				continue
			}
			_ = notifications.Insert(r.Context(), h.DB, notifications.InsertOpts{
				RecipientID: mgrID,
				Audience:    notifications.AudienceUser,
				KindCode:    notifications.KindUserWorkspaceJoined,
				BodyArgs:    []string{joinedName, workspaceName, roleLabel},
				DedupeKey:   "ws_joined:" + inviteID + ":" + mgrID,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "accepted",
		"workspace_id":  workspaceID,
		"role":          inviteRole,
		"action_label":  subscriptions.ActionLabelForCode("WORKSPACE_INVITE_ACCEPT"),
		"pending_label": subscriptions.PendingLabelForCode("WORKSPACE_INVITE_ACCEPT"),
		"redirect_href": subscriptions.MessageForCode("APP_PATH_TEAM"),
		"message":       subscriptions.MessageForCode("WORKSPACE_INVITE_ACCEPTED"),
	})
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	actorID := middleware.UserIDFromContext(r.Context())
	workspaceID := chi.URLParam(r, "workspaceId")
	memberID := chi.URLParam(r, "memberId")
	if actorID == "" || workspaceID == "" || memberID == "" {
		writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}

	actorRole, ok := h.memberRole(r, actorID, workspaceID)
	if !ok {
		writeJSONErr(w, http.StatusForbidden, "WS_FORBIDDEN")
		return
	}

	var targetUserID, targetRole string
	err := h.DB.QueryRow(r.Context(), `
		select user_id::text, role
		from public.workspace_members
		where id = $1::uuid and workspace_id = $2::uuid
	`, memberID, workspaceID).Scan(&targetUserID, &targetRole)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": subscriptions.MessageForCode("MEMBER_NOT_FOUND")})
		return
	}

	selfLeave := targetUserID == actorID
	if !selfLeave && actorRole != "owner" && actorRole != "admin" {
		writeJSONErr(w, http.StatusForbidden, "WS_FORBIDDEN")
		return
	}
	if !selfLeave && actorRole == "admin" && targetRole == "owner" {
		writeJSONErr(w, http.StatusForbidden, "WS_ADMINS_CANNOT_REMOVE_OWNERS")
		return
	}

	if targetRole == "owner" {
		var ownerCount int
		if err := h.DB.QueryRow(r.Context(), `
			select count(*) from public.workspace_members
			where workspace_id = $1::uuid and role = 'owner'
		`, workspaceID).Scan(&ownerCount); err != nil {
			writeJSONErr(w, http.StatusInternalServerError, "WS_CHECK_OWNERS_FAILED")
			return
		}
		if ownerCount <= 1 {
			writeJSONErr(w, http.StatusConflict, "WS_CANNOT_REMOVE_LAST_OWNER")
			return
		}
	}

	tag, err := h.DB.Exec(r.Context(), `
		delete from public.workspace_members
		where id = $1::uuid and workspace_id = $2::uuid
	`, memberID, workspaceID)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_REMOVE_MEMBER_FAILED")
		return
	}
	if tag.RowsAffected() == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": subscriptions.MessageForCode("MEMBER_NOT_FOUND")})
		return
	}

	code := "WORKSPACE_REMOVE"
	if selfLeave {
		code = "WORKSPACE_LEAVE"
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":        "ok",
		"user_id":       targetUserID,
		"action_label":  subscriptions.ActionLabelForCode(code),
		"pending_label": subscriptions.PendingLabelForCode(code),
		"message":       subscriptions.MessageForCode("WORKSPACE_MEMBER_REMOVED"),
	})
}

type renameReq struct {
	Name string `json:"name"`
}

func (h *Handler) Rename(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	workspaceID := chi.URLParam(r, "workspaceId")
	if userID == "" || workspaceID == "" {
		writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}
	role, ok := h.memberRole(r, userID, workspaceID)
	if !ok || role != "owner" {
		writeJSONErr(w, http.StatusForbidden, "WS_RENAME_FORBIDDEN")
		return
	}
	var req renameReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeJSONErr(w, http.StatusBadRequest, "WS_NAME_REQUIRED")
		return
	}

	tag, err := h.DB.Exec(r.Context(), `
		update public.workspaces
		set name = $2, updated_at = now()
		where id = $1::uuid
	`, workspaceID, req.Name)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_RENAME_FAILED")
		return
	}
	if tag.RowsAffected() == 0 {
		writeJSONErr(w, http.StatusNotFound, "WS_WORKSPACE_NOT_FOUND")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":        "ok",
		"id":            workspaceID,
		"name":          req.Name,
		"action_label":  subscriptions.ActionLabelForCode("WORKSPACE_RENAME_SAVE"),
		"pending_label": subscriptions.PendingLabelForCode("WORKSPACE_RENAME_SAVE"),
		"message":       subscriptions.MessageForCode("WORKSPACE_RENAMED"),
	})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	workspaceID := chi.URLParam(r, "workspaceId")
	if userID == "" || workspaceID == "" {
		writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}
	role, ok := h.memberRole(r, userID, workspaceID)
	if !ok || role != "owner" {
		writeJSONErr(w, http.StatusForbidden, "WS_DELETE_FORBIDDEN")
		return
	}

	tag, err := h.DB.Exec(r.Context(), `
		delete from public.workspaces where id = $1::uuid
	`, workspaceID)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_DELETE_FAILED")
		return
	}
	if tag.RowsAffected() == 0 {
		writeJSONErr(w, http.StatusNotFound, "WS_WORKSPACE_NOT_FOUND")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":        "ok",
		"id":            workspaceID,
		"action_label":  subscriptions.ActionLabelForCode("WORKSPACE_DELETE"),
		"pending_label": subscriptions.PendingLabelForCode("WORKSPACE_DELETE"),
		"message":       subscriptions.MessageForCode("WORKSPACE_DELETED"),
	})
}

type updateMemberRoleReq struct {
	Role string `json:"role"`
}

func (h *Handler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	actorID := middleware.UserIDFromContext(r.Context())
	workspaceID := chi.URLParam(r, "workspaceId")
	memberID := chi.URLParam(r, "memberId")
	if actorID == "" || workspaceID == "" || memberID == "" {
		writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}

	actorRole, ok := h.memberRole(r, actorID, workspaceID)
	if !ok || (actorRole != "owner" && actorRole != "admin") {
		writeJSONErr(w, http.StatusForbidden, "WS_ROLE_CHANGE_FORBIDDEN")
		return
	}

	var req updateMemberRoleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	nextRole := strings.ToLower(strings.TrimSpace(req.Role))
	allowed := map[string]bool{}
	for _, opt := range roleOptions("member", "admin", "owner") {
		allowed[opt.ID] = true
	}
	if !allowed[nextRole] {
		writeJSONErr(w, http.StatusBadRequest, "WS_INVALID_MEMBER_ROLE")
		return
	}
	// Admins may only assign member/admin; owners may assign any allowed role.
	if actorRole == "admin" && nextRole == "owner" {
		writeJSONErr(w, http.StatusForbidden, "WS_ROLE_CHANGE_FORBIDDEN")
		return
	}

	var targetUserID, targetRole string
	err := h.DB.QueryRow(r.Context(), `
		select user_id::text, role
		from public.workspace_members
		where id = $1::uuid and workspace_id = $2::uuid
	`, memberID, workspaceID).Scan(&targetUserID, &targetRole)
	if err != nil {
		writeJSONErr(w, http.StatusNotFound, "WS_MEMBER_NOT_FOUND")
		return
	}
	if actorRole == "admin" && targetRole == "owner" {
		writeJSONErr(w, http.StatusForbidden, "WS_ROLE_CHANGE_FORBIDDEN")
		return
	}
	if targetRole == nextRole {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":        "ok",
			"id":            memberID,
			"user_id":       targetUserID,
			"role":          nextRole,
			"role_label":    subscriptions.RoleLabel(nextRole),
			"action_label":  subscriptions.ActionLabelForCode("WORKSPACE_ROLE_CHANGE"),
			"pending_label": subscriptions.PendingLabelForCode("WORKSPACE_ROLE_CHANGE"),
			"message":       subscriptions.MessageForCode("WORKSPACE_MEMBER_ROLE_UPDATED"),
		})
		return
	}

	if targetRole == "owner" && nextRole != "owner" {
		var ownerCount int
		if err := h.DB.QueryRow(r.Context(), `
			select count(*) from public.workspace_members
			where workspace_id = $1::uuid and role = 'owner'
		`, workspaceID).Scan(&ownerCount); err != nil {
			writeJSONErr(w, http.StatusInternalServerError, "WS_CHECK_OWNERS_FAILED")
			return
		}
		if ownerCount <= 1 {
			writeJSONErr(w, http.StatusConflict, "WS_CANNOT_DEMOTE_LAST_OWNER")
			return
		}
	}

	tag, err := h.DB.Exec(r.Context(), `
		update public.workspace_members
		set role = $3
		where id = $1::uuid and workspace_id = $2::uuid
	`, memberID, workspaceID, nextRole)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "WS_ROLE_CHANGE_FAILED")
		return
	}
	if tag.RowsAffected() == 0 {
		writeJSONErr(w, http.StatusNotFound, "WS_MEMBER_NOT_FOUND")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":        "ok",
		"id":            memberID,
		"user_id":       targetUserID,
		"role":          nextRole,
		"role_label":    subscriptions.RoleLabel(nextRole),
		"action_label":  subscriptions.ActionLabelForCode("WORKSPACE_ROLE_CHANGE"),
		"pending_label": subscriptions.PendingLabelForCode("WORKSPACE_ROLE_CHANGE"),
		"message":       subscriptions.MessageForCode("WORKSPACE_MEMBER_ROLE_UPDATED"),
	})
}

func (h *Handler) isMember(r *http.Request, userID, workspaceID string) bool {
	_, ok := h.memberRole(r, userID, workspaceID)
	return ok
}

func (h *Handler) memberRole(r *http.Request, userID, workspaceID string) (string, bool) {
	var role string
	err := h.DB.QueryRow(r.Context(), `
		select role from public.workspace_members
		where workspace_id = $1::uuid and user_id = $2::uuid
	`, workspaceID, userID).Scan(&role)
	if err != nil {
		return "", false
	}
	return role, true
}

type namedRole struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

func roleOptions(ids ...string) []namedRole {
	out := make([]namedRole, 0, len(ids))
	for _, id := range ids {
		id = strings.ToLower(strings.TrimSpace(id))
		label := strings.TrimSpace(subscriptions.RoleLabel(id))
		if id == "" || label == "" {
			continue
		}
		out = append(out, namedRole{ID: id, Label: label})
	}
	return out
}

func generateInviteToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func maskEmail(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return "***"
	}
	local := parts[0]
	if len(local) <= 2 {
		return local[:1] + "***@" + parts[1]
	}
	return local[:1] + "***" + local[len(local)-1:] + "@" + parts[1]
}
