package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/usetrim/trim/server/internal/billingsettings"
	"github.com/usetrim/trim/server/internal/pagination"
	"github.com/usetrim/trim/server/internal/subscriptions"
)

const (
	AudienceUser  = "user"
	AudienceAdmin = "admin"

	KindUserWorkspaceInvite      = "user.workspace_invite"
	KindUserWorkspaceJoined      = "user.workspace_joined"
	KindUserSubscriptionActive   = "user.subscription_active"
	KindUserSubscriptionCanceled = "user.subscription_canceled"
	KindUserReceiptReady         = "user.receipt_ready"
	KindUserAccountStatus        = "user.account_status"
	KindUserEnterpriseActivated  = "user.enterprise_activated"
	KindUserEnterpriseOfferReady = "user.enterprise_offer_ready"
	KindAdminEnterpriseInquiry   = "admin.enterprise_inquiry"
	KindAdminBreakGlassRequest   = "admin.break_glass_request"
	KindAdminReceiptReady        = "admin.receipt_ready"
)

type InsertOpts struct {
	RecipientID  string
	Audience     string
	KindCode     string
	BodyArgs     []string
	DedupeKey    string
	HrefEntityID string
}

// Insert one notification. Empty kind / recipient / audience fails closed.
// Duplicate dedupe_key for the same recipient is ignored (no error).
func Insert(ctx context.Context, db *pgxpool.Pool, opts InsertOpts) error {
	if db == nil {
		return fmt.Errorf("DATABASE_UNAVAILABLE")
	}
	opts.RecipientID = strings.TrimSpace(opts.RecipientID)
	opts.Audience = strings.TrimSpace(opts.Audience)
	opts.KindCode = strings.TrimSpace(opts.KindCode)
	if opts.RecipientID == "" || opts.Audience == "" || opts.KindCode == "" {
		return fmt.Errorf("NOTIF_INSERT_INVALID")
	}
	if opts.BodyArgs == nil {
		opts.BodyArgs = []string{}
	}
	argsJSON, err := json.Marshal(opts.BodyArgs)
	if err != nil {
		return fmt.Errorf("NOTIF_INSERT_INVALID")
	}
	dedupe := strings.TrimSpace(opts.DedupeKey)
	entityID := strings.TrimSpace(opts.HrefEntityID)
	_, err = db.Exec(ctx, `
		insert into public.app_notifications
			(recipient_id, audience, kind_code, body_args, dedupe_key, href_entity_id)
		values ($1::uuid, $2, $3, $4::jsonb, nullif($5, ''), coalesce(nullif($6, ''), ''))
		on conflict (recipient_id, dedupe_key) where dedupe_key is not null and dedupe_key <> ''
		do nothing
	`, opts.RecipientID, opts.Audience, opts.KindCode, string(argsJSON), dedupe, entityID)
	if err != nil {
		return fmt.Errorf("NOTIF_INSERT_FAILED")
	}
	return nil
}

// InsertForAdminsWithPermission notifies every platform admin who holds perm
// (owners always included). Skips skipUserID when non-empty.
func InsertForAdminsWithPermission(
	ctx context.Context,
	db *pgxpool.Pool,
	perm string,
	kindCode string,
	bodyArgs []string,
	dedupeKeyPrefix string,
	skipUserID string,
	hrefEntityID string,
) error {
	if db == nil {
		return fmt.Errorf("DATABASE_UNAVAILABLE")
	}
	perm = strings.TrimSpace(perm)
	kindCode = strings.TrimSpace(kindCode)
	if perm == "" || kindCode == "" {
		return fmt.Errorf("NOTIF_INSERT_INVALID")
	}
	rows, err := db.Query(ctx, `
		select pa.user_id::text, coalesce(r.is_owner, false)
		from public.platform_admins pa
		join public.platform_roles r on r.id = pa.role_id
		where pa.status = 'active'
	`)
	if err != nil {
		return fmt.Errorf("NOTIF_ADMINS_LOAD_FAILED")
	}
	defer rows.Close()

	type adminRow struct {
		id      string
		isOwner bool
	}
	admins := make([]adminRow, 0)
	for rows.Next() {
		var a adminRow
		if err := rows.Scan(&a.id, &a.isOwner); err != nil {
			continue
		}
		admins = append(admins, a)
	}

	for _, a := range admins {
		if skipUserID != "" && a.id == skipUserID {
			continue
		}
		ok := a.isOwner
		if !ok {
			_ = db.QueryRow(ctx, `
				select exists(
					select 1
					from public.platform_admins pa
					join public.platform_role_permissions rp on rp.role_id = pa.role_id
					where pa.user_id = $1::uuid and pa.status = 'active' and rp.permission_code = $2
				)
			`, a.id, perm).Scan(&ok)
		}
		if !ok {
			continue
		}
		dedupe := ""
		if strings.TrimSpace(dedupeKeyPrefix) != "" {
			dedupe = dedupeKeyPrefix + ":" + a.id
		}
		_ = Insert(ctx, db, InsertOpts{
			RecipientID:  a.id,
			Audience:     AudienceAdmin,
			KindCode:     kindCode,
			BodyArgs:     bodyArgs,
			DedupeKey:    dedupe,
			HrefEntityID: hrefEntityID,
		})
	}
	return nil
}

type Item struct {
	ID        string `json:"id"`
	KindCode  string `json:"kind_code"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Href      string `json:"href,omitempty"`
	Read      bool   `json:"read"`
	CreatedAt string `json:"created_at"`
}

type Chrome struct {
	BellAria         string `json:"bell_aria"`
	BellAriaCountFmt string `json:"bell_aria_count_fmt"`
	PanelTitle       string `json:"panel_title"`
	EmptyMessage     string `json:"empty_message"`
	MarkAllRead      string `json:"mark_all_read"`
	MarkAllPending   string `json:"mark_all_pending"`
	MarkRead         string `json:"mark_read"`
	MarkReadPending  string `json:"mark_read_pending"`
	UnreadLabel      string `json:"unread_label"`
	LoadingMore      string `json:"loading_more"`
	PollIntervalMs   int    `json:"poll_interval_ms"`
	ListFailed       string `json:"list_failed"`
	MarkFailed       string `json:"mark_failed"`
}

type ListResult struct {
	Items  []Item          `json:"items"`
	Meta   pagination.Meta `json:"meta"`
	Chrome Chrome          `json:"chrome"`
}

func LoadChrome() Chrome {
	poll := 0
	raw := strings.TrimSpace(subscriptions.MessageForCode("NOTIF_POLL_INTERVAL_MS"))
	if raw != "" {
		var n int
		if _, err := fmt.Sscanf(raw, "%d", &n); err == nil && n > 0 {
			poll = n
		}
	}
	return Chrome{
		BellAria:         strings.TrimSpace(subscriptions.MessageForCode("NOTIF_BELL_ARIA")),
		BellAriaCountFmt: strings.TrimSpace(subscriptions.MessageForCode("NOTIF_BELL_ARIA_COUNT_FMT")),
		PanelTitle:       strings.TrimSpace(subscriptions.MessageForCode("NOTIF_PANEL_TITLE")),
		EmptyMessage:     strings.TrimSpace(subscriptions.MessageForCode("NOTIF_EMPTY")),
		MarkAllRead:      strings.TrimSpace(subscriptions.MessageForCode("NOTIF_MARK_ALL_READ")),
		MarkAllPending:   strings.TrimSpace(subscriptions.MessageForCode("NOTIF_MARK_ALL_PENDING")),
		MarkRead:         strings.TrimSpace(subscriptions.MessageForCode("NOTIF_MARK_READ")),
		MarkReadPending:  strings.TrimSpace(subscriptions.MessageForCode("NOTIF_MARK_READ_PENDING")),
		UnreadLabel:      strings.TrimSpace(subscriptions.MessageForCode("NOTIF_UNREAD_LABEL")),
		LoadingMore:      strings.TrimSpace(subscriptions.MessageForCode("NOTIF_LOADING_MORE")),
		PollIntervalMs:   poll,
		ListFailed:       strings.TrimSpace(subscriptions.MessageForCode("NOTIF_LIST_FAILED")),
		MarkFailed:       strings.TrimSpace(subscriptions.MessageForCode("NOTIF_MARK_FAILED")),
	}
}

func List(
	ctx context.Context,
	db *pgxpool.Pool,
	r *http.Request,
	recipientID string,
	audience string,
) (ListResult, error) {
	out := ListResult{Items: []Item{}, Chrome: LoadChrome()}
	if db == nil {
		return out, fmt.Errorf("DATABASE_UNAVAILABLE")
	}
	recipientID = strings.TrimSpace(recipientID)
	audience = strings.TrimSpace(audience)
	if recipientID == "" || audience == "" {
		return out, fmt.Errorf("NOTIF_LIST_INVALID")
	}

	maxLimit, err := billingsettings.MaxPageSize(ctx, db)
	if err != nil {
		return out, err
	}
	skipCap, err := billingsettings.SkipToMaxPages(ctx, db)
	if err != nil {
		return out, err
	}
	params, err := pagination.Parse(r, maxLimit)
	if err != nil {
		return out, err
	}

	var total int
	if err := db.QueryRow(ctx, `
		select count(*) from public.app_notifications
		where recipient_id = $1::uuid and audience = $2
	`, recipientID, audience).Scan(&total); err != nil {
		return out, fmt.Errorf("NOTIF_COUNT_FAILED")
	}

	rows, err := db.Query(ctx, `
		select n.id::text, n.kind_code, n.body_args, n.read_at is not null, n.created_at::text,
		       coalesce(n.href_entity_id, ''),
		       k.title_message_code, k.body_message_code, k.href_ref_kind, k.href_ref
		from public.app_notifications n
		join public.notification_kinds k on k.code = n.kind_code
		where n.recipient_id = $1::uuid and n.audience = $2
		order by n.created_at desc
		offset $3 limit $4
	`, recipientID, audience, params.Skip, params.Limit)
	if err != nil {
		return out, fmt.Errorf("NOTIF_LIST_FAILED")
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id, kind, createdAt, entityID          string
			titleCode, bodyCode, hrefKind, hrefRef string
			read                                   bool
			argsRaw                                []byte
		)
		if err := rows.Scan(&id, &kind, &argsRaw, &read, &createdAt, &entityID, &titleCode, &bodyCode, &hrefKind, &hrefRef); err != nil {
			continue
		}
		title := strings.TrimSpace(subscriptions.MessageForCode(titleCode))
		bodyTpl := strings.TrimSpace(subscriptions.MessageForCode(bodyCode))
		if title == "" || bodyTpl == "" {
			continue
		}
		var args []string
		_ = json.Unmarshal(argsRaw, &args)
		body := formatBody(bodyTpl, args)
		href := resolveHref(ctx, db, hrefKind, hrefRef, entityID)
		out.Items = append(out.Items, Item{
			ID:        id,
			KindCode:  kind,
			Title:     title,
			Body:      body,
			Href:      href,
			Read:      read,
			CreatedAt: createdAt,
		})
	}

	out.Meta = pagination.BuildMeta(params, total, skipCap)
	return out, nil
}

func UnreadCount(ctx context.Context, db *pgxpool.Pool, recipientID, audience string) (count int, badge string, chrome Chrome, err error) {
	chrome = LoadChrome()
	if db == nil {
		return 0, "", chrome, fmt.Errorf("DATABASE_UNAVAILABLE")
	}
	recipientID = strings.TrimSpace(recipientID)
	audience = strings.TrimSpace(audience)
	if recipientID == "" || audience == "" {
		return 0, "", chrome, fmt.Errorf("NOTIF_LIST_INVALID")
	}
	err = db.QueryRow(ctx, `
		select count(*) from public.app_notifications
		where recipient_id = $1::uuid and audience = $2 and read_at is null
	`, recipientID, audience).Scan(&count)
	if err != nil {
		return 0, "", chrome, fmt.Errorf("NOTIF_COUNT_FAILED")
	}
	if count <= 0 {
		return 0, "", chrome, nil
	}
	badge = formatBadgeLabel(count)
	return count, badge, chrome, nil
}

// formatBadgeLabel builds the bell count from site_messages only.
// Exact count when NOTIF_BADGE_MAX is missing/invalid; overflow fmt when count exceeds max.
func formatBadgeLabel(count int) string {
	if count <= 0 {
		return ""
	}
	maxRaw := strings.TrimSpace(subscriptions.MessageForCode("NOTIF_BADGE_MAX"))
	max := 0
	if maxRaw != "" {
		var n int
		if _, err := fmt.Sscanf(maxRaw, "%d", &n); err == nil && n > 0 {
			max = n
		}
	}
	if max > 0 && count > max {
		overflowFmt := strings.TrimSpace(subscriptions.MessageForCode("NOTIF_BADGE_OVERFLOW_FMT"))
		if overflowFmt != "" && strings.Contains(overflowFmt, "{max}") {
			return strings.ReplaceAll(overflowFmt, "{max}", fmt.Sprintf("%d", max))
		}
		// Fail closed: no invent overflow string; show exact count.
	}
	return fmt.Sprintf("%d", count)
}

func MarkRead(ctx context.Context, db *pgxpool.Pool, recipientID, audience, id string) error {
	if db == nil {
		return fmt.Errorf("DATABASE_UNAVAILABLE")
	}
	tag, err := db.Exec(ctx, `
		update public.app_notifications
		set read_at = coalesce(read_at, now())
		where id = $1::uuid and recipient_id = $2::uuid and audience = $3
	`, strings.TrimSpace(id), strings.TrimSpace(recipientID), strings.TrimSpace(audience))
	if err != nil {
		return fmt.Errorf("NOTIF_MARK_FAILED")
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func MarkAllRead(ctx context.Context, db *pgxpool.Pool, recipientID, audience string) error {
	if db == nil {
		return fmt.Errorf("DATABASE_UNAVAILABLE")
	}
	_, err := db.Exec(ctx, `
		update public.app_notifications
		set read_at = now()
		where recipient_id = $1::uuid and audience = $2 and read_at is null
	`, strings.TrimSpace(recipientID), strings.TrimSpace(audience))
	if err != nil {
		return fmt.Errorf("NOTIF_MARK_FAILED")
	}
	return nil
}

func formatBody(tpl string, args []string) string {
	if len(args) == 0 {
		return tpl
	}
	anyArgs := make([]any, len(args))
	for i, a := range args {
		anyArgs[i] = a
	}
	defer func() { _ = recover() }()
	return fmt.Sprintf(tpl, anyArgs...)
}

func resolveHref(ctx context.Context, db *pgxpool.Pool, kind, ref, entityID string) string {
	kind = strings.TrimSpace(kind)
	ref = strings.TrimSpace(ref)
	entityID = strings.TrimSpace(entityID)
	switch kind {
	case "app_path":
		if ref == "" {
			return ""
		}
		return strings.TrimSpace(subscriptions.MessageForCode(ref))
	case "app_path_prefix":
		if ref == "" || entityID == "" {
			return ""
		}
		prefix := strings.TrimSpace(subscriptions.MessageForCode(ref))
		if prefix == "" {
			return ""
		}
		// Fail closed on open redirects / path traversal in entity ids.
		if strings.ContainsAny(entityID, "/\\?#") || strings.Contains(entityID, "..") {
			return ""
		}
		return prefix + entityID
	case "admin_nav":
		if ref == "" || db == nil {
			return ""
		}
		var href string
		_ = db.QueryRow(ctx, `
			select href from public.admin_nav_items
			where id = $1 and is_active = true
		`, ref).Scan(&href)
		return strings.TrimSpace(href)
	case "admin_nav_query":
		if ref == "" || db == nil || entityID == "" {
			return ""
		}
		if strings.ContainsAny(entityID, "/\\?#&=") || strings.Contains(entityID, "..") {
			return ""
		}
		var href string
		_ = db.QueryRow(ctx, `
			select href from public.admin_nav_items
			where id = $1 and is_active = true
		`, ref).Scan(&href)
		href = strings.TrimSpace(href)
		if href == "" {
			return ""
		}
		queryKey := strings.TrimSpace(subscriptions.MessageForCode("NOTIF_HREF_QUERY_ID"))
		if queryKey == "" {
			return href
		}
		sep := "?"
		if strings.Contains(href, "?") {
			sep = "&"
		}
		return href + sep + queryKey + "=" + entityID
	default:
		return ""
	}
}
