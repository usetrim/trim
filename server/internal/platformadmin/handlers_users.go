package platformadmin

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/usetrim/trim/server/internal/billingsettings"
	"github.com/usetrim/trim/server/internal/notifications"
	"github.com/usetrim/trim/server/internal/pagination"
)

// listUserCountries returns distinct last_login_country codes from profiles (no invent).
func (h *Handler) listUserCountries(w http.ResponseWriter, r *http.Request) {
	db := h.readPool()
	rows, err := db.Query(r.Context(), `
		select distinct upper(trim(last_login_country)) as code
		from public.profiles
		where coalesce(trim(last_login_country), '') <> ''
		order by code asc
	`)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	type opt struct {
		Value string `json:"value"`
		Label string `json:"label"`
	}
	items := make([]opt, 0)
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		items = append(items, opt{Value: code, Label: code})
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	db := h.readPool()
	params, skipCap, err := h.listPage(r)
	if err != nil {
		h.writePaginateErr(w, err)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	country := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("country")))
	plan := strings.TrimSpace(r.URL.Query().Get("plan"))
	provider := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("provider")))
	createdFrom := strings.TrimSpace(r.URL.Query().Get("created_from"))
	createdTo := strings.TrimSpace(r.URL.Query().Get("created_to"))

	where := `where 1=1`
	args := []any{}
	n := 1
	if q != "" {
		where += fmt.Sprintf(` and (
		  lower(p.email) like $%d
		  or lower(coalesce(p.full_name, '')) like $%d
		  or p.id::text = $%d
		  or exists (
		    select 1 from auth.identities i
		    where i.user_id = p.id and lower(i.provider) like $%d
		  )
		)`, n, n, n+1, n)
		args = append(args, "%"+strings.ToLower(q)+"%", q)
		n += 2
	}
	if status != "" {
		where += fmt.Sprintf(` and p.account_status = $%d`, n)
		args = append(args, status)
		n++
	}
	if country != "" {
		where += fmt.Sprintf(` and upper(coalesce(p.last_login_country, '')) = $%d`, n)
		args = append(args, country)
		n++
	}
	if plan != "" {
		where += fmt.Sprintf(` and coalesce(uq.plan_tier, '') = $%d`, n)
		args = append(args, plan)
		n++
	}
	if provider != "" {
		where += fmt.Sprintf(` and (
		  lower(coalesce(p.auth_provider, '')) = $%d
		  or exists (
		    select 1 from auth.identities i
		    where i.user_id = p.id and lower(i.provider) = $%d
		  )
		)`, n, n)
		args = append(args, provider)
		n++
	}
	if createdFrom != "" {
		where += fmt.Sprintf(` and p.created_at::date >= $%d::date`, n)
		args = append(args, createdFrom)
		n++
	}
	if createdTo != "" {
		where += fmt.Sprintf(` and p.created_at::date <= $%d::date`, n)
		args = append(args, createdTo)
		n++
	}

	countSQL := `
		select count(*)
		from public.profiles p
		left join public.user_quotas uq on uq.user_id = p.id
	` + where
	var total int
	if err := db.QueryRow(r.Context(), countSQL, args...).Scan(&total); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}

	listSQL := fmt.Sprintf(`
		select p.id::text, p.email, coalesce(p.full_name, ''), p.account_status,
		       coalesce(p.last_login_country, ''), coalesce(uq.plan_tier, ''),
		       coalesce(p.auth_provider, ''), p.created_at::text
		from public.profiles p
		left join public.user_quotas uq on uq.user_id = p.id
		%s
		order by p.created_at desc
		offset $%d limit $%d
	`, where, n, n+1)
	listArgs := append(append([]any{}, args...), params.Skip, params.Limit)
	rows, err := db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()

	type row struct {
		ID                 string `json:"id"`
		Email              string `json:"email"`
		FullName           string `json:"full_name"`
		AccountStatus      string `json:"account_status"`
		AccountStatusLabel string `json:"account_status_label"`
		Country            string `json:"last_login_country"`
		PlanTier           string `json:"plan_tier"`
		AuthProvider       string `json:"auth_provider"`
		CreatedAt          string `json:"created_at"`
	}
	items := make([]row, 0)
	for rows.Next() {
		var item row
		if err := rows.Scan(&item.ID, &item.Email, &item.FullName, &item.AccountStatus, &item.Country, &item.PlanTier, &item.AuthProvider, &item.CreatedAt); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		item.AccountStatusLabel = h.accountStatusLabel(item.AccountStatus)
		items = append(items, item)
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"meta":  pagination.BuildMeta(params, total, skipCap),
	})
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	db := h.readPool()
	id := chi.URLParam(r, "id")
	var email, fullName, status, reason, notes, country, createdAt, authProvider string
	var statusChangedAt *string
	err := db.QueryRow(r.Context(), `
		select email, coalesce(full_name, ''), account_status,
		       coalesce(account_status_reason, ''), coalesce(admin_notes, ''),
		       coalesce(last_login_country, ''), created_at::text, account_status_changed_at::text,
		       coalesce(auth_provider, '')
		from public.profiles where id = $1::uuid
	`, id).Scan(&email, &fullName, &status, &reason, &notes, &country, &createdAt, &statusChangedAt, &authProvider)
	if err == pgx.ErrNoRows {
		h.writeErr(w, http.StatusNotFound, "ADMIN_USER_NOT_FOUND")
		return
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	var planTier string
	var limit, used, topup int
	_ = db.QueryRow(r.Context(), `
		select coalesce(plan_tier, ''), coalesce(monthly_credit_limit, 0),
		       coalesce(monthly_credit_used, 0), coalesce(purchased_topup_credits, 0)
		from public.user_quotas where user_id = $1::uuid
	`, id).Scan(&planTier, &limit, &used, &topup)

	devices := []map[string]any{}
	topN, topErr := billingsettings.ChartTopN(r.Context(), h.readPool())
	if topErr != nil {
		h.writeErr(w, http.StatusInternalServerError, topErr.Error())
		return
	}
	drows, _ := db.Query(r.Context(), `
		select id::text, hardware_uuid, last_seen_at::text
		from public.device_fingerprints where user_id = $1::uuid
		order by last_seen_at desc limit $2
	`, id, topN)
	if drows != nil {
		defer drows.Close()
		for drows.Next() {
			var did, hw, seen string
			if drows.Scan(&did, &hw, &seen) == nil {
				devices = append(devices, map[string]any{"id": did, "hardware_uuid": hw, "last_seen_at": seen})
			}
		}
	}
	workspaces := []map[string]any{}
	wrows, _ := db.Query(r.Context(), `
		select w.id::text, w.name, wm.role
		from public.workspace_members wm
		join public.workspaces w on w.id = wm.workspace_id
		where wm.user_id = $1::uuid
		order by w.name asc
		limit $2
	`, id, topN)
	if wrows != nil {
		defer wrows.Close()
		for wrows.Next() {
			var wid, name, role string
			if wrows.Scan(&wid, &name, &role) == nil {
				workspaces = append(workspaces, map[string]any{"id": wid, "name": name, "role": role})
			}
		}
	}
	var billCountry string
	_ = db.QueryRow(r.Context(), `
		select coalesce(bill_to_country, '') from public.billing_receipts
		where user_id = $1::uuid order by created_at desc limit 1
	`, id).Scan(&billCountry)

	identities := []map[string]any{}
	irows, _ := db.Query(r.Context(), `
		select provider, identity_data->>'email', created_at::text, last_sign_in_at::text
		from auth.identities where user_id = $1::uuid
		order by created_at asc
		limit $2
	`, id, topN)
	if irows != nil {
		defer irows.Close()
		for irows.Next() {
			var provider, iemail, created, lastSign string
			if irows.Scan(&provider, &iemail, &created, &lastSign) == nil {
				identities = append(identities, map[string]any{
					"provider": provider, "email": iemail,
					"created_at": created, "last_sign_in_at": lastSign,
				})
			}
		}
	}

	ja4Hits := []map[string]any{}
	jrows, _ := db.Query(r.Context(), `
		select ja4_hash, created_at::text, last_seen_at::text
		from public.ja4_fingerprints where user_id = $1::uuid
		order by last_seen_at desc nulls last limit $2
	`, id, topN)
	if jrows != nil {
		defer jrows.Close()
		for jrows.Next() {
			var hash, first, last string
			if jrows.Scan(&hash, &first, &last) == nil {
				ja4Hits = append(ja4Hits, map[string]any{
					"ja4_hash": hash, "created_at": first, "last_seen_at": last,
				})
			}
		}
	}

	linkedByJA4 := []map[string]any{}
	lrows, _ := db.Query(r.Context(), `
		select distinct p.id::text, p.email, j.ja4_hash
		from public.ja4_fingerprints j
		join public.ja4_fingerprints j2 on j2.ja4_hash = j.ja4_hash and j2.user_id <> j.user_id
		join public.profiles p on p.id = j2.user_id
		where j.user_id = $1::uuid
		limit $2
	`, id, topN)
	if lrows != nil {
		defer lrows.Close()
		for lrows.Next() {
			var lid, lemail, hash string
			if lrows.Scan(&lid, &lemail, &hash) == nil {
				linkedByJA4 = append(linkedByJA4, map[string]any{
					"id": lid, "email": lemail, "ja4_hash": hash,
				})
			}
		}
	}

	linkedByHW := []map[string]any{}
	hwrows, _ := db.Query(r.Context(), `
		select distinct p.id::text, p.email, d.hardware_uuid
		from public.device_fingerprints d
		join public.device_fingerprints d2
		  on d2.hardware_uuid = d.hardware_uuid and d2.user_id <> d.user_id
		join public.profiles p on p.id = d2.user_id
		where d.user_id = $1::uuid
		limit $2
	`, id, topN)
	if hwrows != nil {
		defer hwrows.Close()
		for hwrows.Next() {
			var lid, lemail, hw string
			if hwrows.Scan(&lid, &lemail, &hw) == nil {
				linkedByHW = append(linkedByHW, map[string]any{
					"id": lid, "email": lemail, "hardware_uuid": hw,
				})
			}
		}
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"id": id, "email": email, "full_name": fullName,
		"account_status": status, "account_status_label": h.accountStatusLabel(status),
		"account_status_reason":     reason,
		"account_status_changed_at": statusChangedAt,
		"admin_notes":               notes, "last_login_country": country,
		"bill_to_country": billCountry,
		"auth_provider":   authProvider,
		"created_at":      createdAt,
		"quota": map[string]any{
			"plan_tier": planTier, "monthly_credit_limit": limit,
			"monthly_credit_used": used, "purchased_topup_credits": topup,
		},
		"devices":            devices,
		"workspaces":         workspaces,
		"identities":         identities,
		"ja4_fingerprints":   ja4Hits,
		"linked_by_ja4":      linkedByJA4,
		"linked_by_hardware": linkedByHW,
	})
}

type userStatusBody struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

func (h *Handler) postUserStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body userStatusBody
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	status := strings.TrimSpace(body.Status)
	reason := strings.TrimSpace(body.Reason)
	valid := map[string]bool{
		"active": true, "suspended": true, "banned": true, "pending_delete": true, "shadowbanned": true,
	}
	if !valid[status] {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_STATUS_INVALID")
		return
	}
	if reason == "" && status != "active" {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_REASON_REQUIRED")
		return
	}
	p := PrincipalFromContext(r.Context())
	switch status {
	case "suspended", "active":
		if p != nil && !p.IsOwner && !p.Permissions[PermUsersSuspend] {
			h.writeErr(w, http.StatusForbidden, "ADMIN_PERMISSION_DENIED")
			return
		}
	case "banned", "shadowbanned":
		if p != nil && !p.IsOwner && !p.Permissions[PermUsersBan] {
			h.writeErr(w, http.StatusForbidden, "ADMIN_PERMISSION_DENIED")
			return
		}
	}
	var beforeStatus string
	err := h.DB.QueryRow(r.Context(), `select account_status from public.profiles where id = $1::uuid`, id).Scan(&beforeStatus)
	if err == pgx.ErrNoRows {
		h.writeErr(w, http.StatusNotFound, "ADMIN_USER_NOT_FOUND")
		return
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	actor := ""
	if p != nil {
		actor = p.UserID
	}
	_, err = h.DB.Exec(r.Context(), `
		update public.profiles set
		  account_status = $2,
		  account_status_reason = nullif($3, ''),
		  account_status_changed_at = now(),
		  account_status_changed_by = nullif($4, '')::uuid,
		  updated_at = now()
		where id = $1::uuid
	`, id, status, reason, actor)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "users.status", "profile", id,
		map[string]string{"status": beforeStatus},
		map[string]string{"status": status, "reason": reason}, reason, step)
	if status == "suspended" || status == "banned" {
		h.maybeSendSuspendEmail(r.Context(), id, reason)
	}
	_ = notifications.Insert(r.Context(), h.DB, notifications.InsertOpts{
		RecipientID: id,
		Audience:    notifications.AudienceUser,
		KindCode:    notifications.KindUserAccountStatus,
		BodyArgs:    []string{status},
		DedupeKey:   "account_status:" + id + ":" + status,
	})
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_USER_STATUS_UPDATED")})
}

func (h *Handler) maybeSendSuspendEmail(ctx context.Context, userID, reason string) {
	if !h.Mailer.Enabled() {
		return
	}
	var email string
	_ = h.DB.QueryRow(ctx, `select email from public.profiles where id = $1::uuid`, userID).Scan(&email)
	email = strings.TrimSpace(email)
	if email == "" {
		return
	}
	subject := strings.TrimSpace(h.msg("EMAIL_SUSPEND_SUBJECT"))
	bodyFmt := strings.TrimSpace(h.msg("EMAIL_SUSPEND_BODY_FMT"))
	if subject == "" || bodyFmt == "" {
		return
	}
	body := strings.ReplaceAll(bodyFmt, "{email}", email)
	body = strings.ReplaceAll(body, "{reason}", reason)
	_ = h.Mailer.SendPlain(email, subject, body)
}

type userNotesBody struct {
	Notes string `json:"notes"`
}

func (h *Handler) patchUserNotes(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body userNotesBody
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	var before *string
	_ = h.DB.QueryRow(r.Context(), `select admin_notes from public.profiles where id = $1::uuid`, id).Scan(&before)
	tag, err := h.DB.Exec(r.Context(), `
		update public.profiles set admin_notes = $2, updated_at = now() where id = $1::uuid
	`, id, strings.TrimSpace(body.Notes))
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if tag.RowsAffected() == 0 {
		h.writeErr(w, http.StatusNotFound, "ADMIN_USER_NOT_FOUND")
		return
	}
	h.audit(r.Context(), r, "users.notes", "profile", id, before, body.Notes, "", false)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_USER_NOTES_SAVED")})
}

type userQuotaBody struct {
	MonthlyCreditLimit    *int   `json:"monthly_credit_limit"`
	PurchasedTopupCredits *int   `json:"purchased_topup_credits"`
	Reason                string `json:"reason"`
}

func (h *Handler) postUserQuota(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p := PrincipalFromContext(r.Context())
	var body userQuotaBody
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	if strings.TrimSpace(body.Reason) == "" {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_REASON_REQUIRED")
		return
	}
	if body.MonthlyCreditLimit == nil && body.PurchasedTopupCredits == nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	var beforeLimit, beforeTopup int
	err := h.DB.QueryRow(r.Context(), `
		select coalesce(monthly_credit_limit, 0), coalesce(purchased_topup_credits, 0)
		from public.user_quotas where user_id = $1::uuid
	`, id).Scan(&beforeLimit, &beforeTopup)
	if err == pgx.ErrNoRows {
		h.writeErr(w, http.StatusNotFound, "ADMIN_USER_NOT_FOUND")
		return
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	limit := beforeLimit
	topup := beforeTopup
	if body.MonthlyCreditLimit != nil {
		if *body.MonthlyCreditLimit < 0 {
			h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
			return
		}
		limit = *body.MonthlyCreditLimit
	}
	if body.PurchasedTopupCredits != nil {
		if *body.PurchasedTopupCredits < 0 {
			h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
			return
		}
		topup = *body.PurchasedTopupCredits
	}
	_, err = h.DB.Exec(r.Context(), `
		update public.user_quotas set
		  monthly_credit_limit = $2,
		  purchased_topup_credits = $3,
		  updated_at = now()
		where user_id = $1::uuid
	`, id, limit, topup)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if h.Redis != nil {
		_ = h.Redis.Del(r.Context(), "user_quota:"+id).Err()
	}
	step := false
	if p != nil {
		step = h.stepUpUsed(r, p.UserID)
	}
	h.audit(r.Context(), r, "users.quota", "user_quota", id,
		map[string]int{"monthly_credit_limit": beforeLimit, "purchased_topup_credits": beforeTopup},
		map[string]int{"monthly_credit_limit": limit, "purchased_topup_credits": topup},
		body.Reason, step)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_USER_QUOTA_UPDATED")})
}

func (h *Handler) postUserRevokeKeys(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p := PrincipalFromContext(r.Context())
	var n int64
	tag, err := h.DB.Exec(r.Context(), `
		update public.api_keys set revoked = true where user_id = $1::uuid and revoked = false
	`, id)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	n = tag.RowsAffected()
	step := false
	if p != nil {
		step = h.stepUpUsed(r, p.UserID)
	}
	h.audit(r.Context(), r, "users.revoke_keys", "api_keys", id, nil,
		map[string]int64{"revoked_count": n}, "", step)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_USER_KEYS_REVOKED")})
}
