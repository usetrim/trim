package platformadmin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	if p == nil {
		h.writeErr(w, http.StatusForbidden, "ADMIN_FORBIDDEN")
		return
	}
	perms := make([]string, 0, len(p.Permissions))
	for code, ok := range p.Permissions {
		if ok {
			perms = append(perms, code)
		}
	}
	enrolled, _ := h.totpEnrolled(r.Context(), p.UserID)
	webauthnOK, _ := h.webauthnEnrolled(r.Context(), p.UserID)
	_, keyOK := h.totpKey()
	_, rpOK := h.webauthnRPID()
	nav := h.loadAdminNav(r.Context(), p)
	navSections := h.loadAdminNavSections(r.Context(), p)
	dialogClose := strings.TrimSpace(h.msg("ADMIN_DIALOG_CLOSE"))
	if dialogClose == "" {
		dialogClose = strings.TrimSpace(h.msg("ADMIN_CLOSE"))
	}
	shellChrome := map[string]string{}
	putShell := func(code, body string) {
		body = strings.TrimSpace(body)
		if body != "" {
			shellChrome[code] = body
		}
	}
	putShell("ADMIN_SIGN_OUT", h.msg("ADMIN_SIGN_OUT"))
	putShell("ADMIN_SIGN_OUT_PENDING", h.msg("ADMIN_SIGN_OUT_PENDING"))
	putShell("ADMIN_THEME_LIGHT", h.msg("ADMIN_THEME_LIGHT"))
	putShell("ADMIN_THEME_DARK", h.msg("ADMIN_THEME_DARK"))
	putShell("ADMIN_NAV_MENU", h.msg("ADMIN_NAV_MENU"))
	putShell("ADMIN_DIALOG_CLOSE", dialogClose)
	putShell("ADMIN_ROUTE_FORBIDDEN", h.msg("ADMIN_ROUTE_FORBIDDEN"))
	h.writeJSON(w, http.StatusOK, map[string]any{
		"user_id":           p.UserID,
		"role_id":           p.RoleID,
		"role_slug":         p.RoleSlug,
		"is_owner":          p.IsOwner,
		"permissions":       perms,
		"brand":             h.msg("ADMIN_BRAND"),
		"tagline":           h.msg("ADMIN_TAGLINE"),
		"totp_enrolled":     enrolled,
		"totp_key_ready":    keyOK,
		"webauthn_enrolled": webauthnOK,
		"webauthn_rp_ready": rpOK && len(h.Config.AdminAllowedOrigins) > 0,
		"nav":               nav,
		"nav_sections":      navSections,
		// Shell chrome on /me so logout/theme/menu never depend on chrome.read ui-map.
		// Only non-empty bodies are included (empty must not clobber ui-map).
		"chrome": shellChrome,
	})
}

// adminNavHref returns the active href for an admin_nav_items id (fail closed: empty when missing).
func (h *Handler) adminNavHref(ctx context.Context, id string) string {
	id = strings.TrimSpace(id)
	if id == "" || h.readPool() == nil {
		return ""
	}
	var href string
	err := h.readPool().QueryRow(ctx, `
		select coalesce(nullif(btrim(href), ''), '')
		from public.admin_nav_items
		where id = $1 and is_active = true
	`, id).Scan(&href)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(href)
}

func (h *Handler) loadAdminNav(ctx context.Context, p *AdminPrincipal) []map[string]string {
	rows, err := h.readPool().Query(ctx, `
		select id, href, chrome_code, permission_code
		from public.admin_nav_items
		where is_active = true
		order by sort_order asc, id asc
	`)
	if err != nil {
		return []map[string]string{}
	}
	defer rows.Close()
	out := make([]map[string]string, 0)
	for rows.Next() {
		var id, href, code, perm string
		if err := rows.Scan(&id, &href, &code, &perm); err != nil {
			continue
		}
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		label := strings.TrimSpace(h.msg(code))
		if label == "" {
			continue
		}
		if !p.IsOwner && !p.Permissions[perm] {
			continue
		}
		out = append(out, map[string]string{
			"id": id, "href": href, "code": code, "perm": perm, "label": label,
		})
	}
	return out
}

// loadAdminNavSections returns accordion groups (docs-style). Empty-label sections/items are omitted.
func (h *Handler) loadAdminNavSections(ctx context.Context, p *AdminPrincipal) []map[string]any {
	rows, err := h.readPool().Query(ctx, `
		select s.id, s.chrome_code,
		       coalesce(i.href, ''), coalesce(i.chrome_code, ''), coalesce(i.permission_code, ''),
		       coalesce(i.sort_order, 0), coalesce(i.id, '')
		from public.admin_nav_sections s
		left join public.admin_nav_items i
		  on i.section_id = s.id and i.is_active = true
		where s.is_active = true
		order by s.sort_order asc, s.id asc, i.sort_order asc nulls last, i.id asc nulls last
	`)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	type sectionAcc struct {
		id    string
		code  string
		label string
		items []map[string]string
	}
	ordered := make([]*sectionAcc, 0)
	byID := map[string]*sectionAcc{}

	for rows.Next() {
		var sectionID, sectionCode, href, itemCode, perm, itemID string
		var itemSort int
		if err := rows.Scan(&sectionID, &sectionCode, &href, &itemCode, &perm, &itemSort, &itemID); err != nil {
			continue
		}
		sec, ok := byID[sectionID]
		if !ok {
			label := strings.TrimSpace(h.msg(sectionCode))
			if label == "" {
				continue
			}
			sec = &sectionAcc{id: sectionID, code: sectionCode, label: label, items: make([]map[string]string, 0)}
			byID[sectionID] = sec
			ordered = append(ordered, sec)
		}
		itemID = strings.TrimSpace(itemID)
		if itemID == "" || href == "" || itemCode == "" || perm == "" {
			continue
		}
		itemLabel := strings.TrimSpace(h.msg(itemCode))
		if itemLabel == "" {
			continue
		}
		if !p.IsOwner && !p.Permissions[perm] {
			continue
		}
		sec.items = append(sec.items, map[string]string{
			"id": itemID, "href": href, "code": itemCode, "perm": perm, "label": itemLabel,
		})
	}

	out := make([]map[string]any, 0, len(ordered))
	for _, sec := range ordered {
		if len(sec.items) == 0 {
			continue
		}
		out = append(out, map[string]any{
			"id":    sec.id,
			"code":  sec.code,
			"label": sec.label,
			"items": sec.items,
		})
	}
	return out
}

func (h *Handler) postStepUp(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	if p == nil {
		h.writeErr(w, http.StatusForbidden, "ADMIN_FORBIDDEN")
		return
	}
	if h.Redis == nil {
		h.writeErr(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
		return
	}
	var body struct {
		Code string `json:"code"`
	}
	if r.Body != nil {
		dec := json.NewDecoder(r.Body)
		_ = dec.Decode(&body)
	}
	enrolled, err := h.totpEnrolled(r.Context(), p.UserID)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if !enrolled {
		h.writeErr(w, http.StatusForbidden, "ADMIN_TOTP_REQUIRED")
		return
	}
	if strings.TrimSpace(body.Code) == "" {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_TOTP_CODE_REQUIRED")
		return
	}
	secret, err := h.loadEnrolledSecret(r.Context(), p.UserID)
	if err == errTOTPKey {
		h.writeErr(w, http.StatusServiceUnavailable, "ADMIN_TOTP_KEY_MISSING")
		return
	}
	if err != nil {
		h.writeErr(w, http.StatusForbidden, "ADMIN_TOTP_REQUIRED")
		return
	}
	if !verifyTOTP(secret, body.Code, time.Now()) {
		h.writeErr(w, http.StatusForbidden, "ADMIN_TOTP_CODE_INVALID")
		return
	}
	tok, sec, err := h.issueStepUpToken(r.Context(), r, p.UserID, "totp")
	if err != nil {
		code := err.Error()
		if strings.HasPrefix(code, "ADMIN_") {
			h.writeErr(w, http.StatusServiceUnavailable, code)
			return
		}
		h.writeErr(w, http.StatusInternalServerError, "ADMIN_STEP_UP_INVALID")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"step_up_token":  tok,
		"expires_in_sec": sec,
	})
}

func (h *Handler) getDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	db := h.readPool()
	if err := db.Ping(ctx); err != nil {
		h.writeErr(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
		return
	}

	// One round-trip for core KPIs (remote DBs time out when these are sequential).
	var (
		usersTotal, subsActive, events24h, install7d int
		activeToday, active7d, active30d, paidSeats  int
		churn30d, newPaid30d, freeToPaid30d          int
		mrrCents                                     int64
	)
	if err := db.QueryRow(ctx, `
		select
			(select count(*) from public.profiles),
			(select count(*) from public.subscriptions where status in ('active', 'trialing')),
			(select count(*) from public.trim_events where created_at >= now() - interval '24 hours'),
			(select count(*) from public.install_hits where hit_at >= now() - interval '7 days'),
			(select count(*) from public.profiles where last_login_at >= now() - interval '1 day'),
			(select count(*) from public.profiles where last_login_at >= now() - interval '7 days'),
			(select count(*) from public.profiles where last_login_at >= now() - interval '30 days'),
			(select coalesce(sum(seat_quantity), 0) from public.subscriptions
			 where status in ('active', 'trialing')),
			(select coalesce(sum(
				case
					when lower(coalesce(s.billing_interval, '')) in ('year', 'yearly', 'annual')
						then (coalesce(pc.price_yearly_cents, 0) / 12) * coalesce(s.seat_quantity, 1)
					else coalesce(pc.price_monthly_cents, 0) * coalesce(s.seat_quantity, 1)
				end
			), 0)
			 from public.subscriptions s
			 join public.plan_catalog pc on pc.id = s.plan_tier
			 where s.status in ('active', 'trialing')),
			(select count(*) from public.subscriptions
			 where status = 'canceled' and coalesce(canceled_at, updated_at) >= now() - interval '30 days'),
			(select count(*) from public.subscriptions
			 where status in ('active', 'trialing') and created_at >= now() - interval '30 days'),
			(select count(distinct s.user_id)
			 from public.subscriptions s
			 where s.status in ('active', 'trialing')
			   and s.plan_tier <> 'free'
			   and s.created_at >= now() - interval '30 days'
			   and exists (
			     select 1 from public.subscriptions s0
			     where s0.user_id = s.user_id
			       and s0.plan_tier = 'free'
			       and s0.created_at < s.created_at
			   ))
	`).Scan(
		&usersTotal, &subsActive, &events24h, &install7d,
		&activeToday, &active7d, &active30d, &paidSeats,
		&mrrCents, &churn30d, &newPaid30d, &freeToPaid30d,
	); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	arrCents := mrrCents * 12

	dbOK := true
	redisOK := false
	if h.Redis != nil {
		redisOK = h.Redis.Ping(ctx).Err() == nil
	}
	owners, _ := h.countActiveOwners(ctx)
	checklist := []map[string]any{
		{"id": "postgres", "ok": dbOK, "label": h.msg("ADMIN_CHECKLIST_POSTGRES")},
		{"id": "redis", "ok": redisOK, "label": h.msg("ADMIN_CHECKLIST_REDIS")},
		{"id": "owners", "ok": owners > 0, "label": h.msg("ADMIN_CHECKLIST_OWNERS")},
		{"id": "github", "ok": strings.TrimSpace(h.Config.GitHubToken) != "" && strings.TrimSpace(h.Config.GitHubRepo) != "", "label": h.msg("ADMIN_CHECKLIST_GITHUB")},
	}
	var (
		pricingBound, auditExportCapOK, churnOK, totpTableOK, notifTableOK, webauthnTableOK bool
		creditsNavOK, minCLIColOK, queueWarnColOK, linkedHwChromeOK                         bool
		idpCount                                                                            int
	)
	_ = db.QueryRow(ctx, `
		select
			coalesce((select pricing_bound from public.billing_settings where id = 'default'), false),
			coalesce((select cardinality(allowed_providers) from public.auth_settings where id = 'default'), 0),
			exists (
			  select 1 from information_schema.tables
			  where table_schema = 'public' and table_name = 'platform_admin_totp'
			),
			exists (
			  select 1 from information_schema.tables
			  where table_schema = 'public' and table_name = 'app_notifications'
			),
			exists (
			  select 1 from information_schema.tables
			  where table_schema = 'public' and table_name = 'platform_admin_webauthn_credentials'
			),
			exists (
			  select 1 from public.admin_nav_items where id = 'credits' and is_active = true
			),
			coalesce((
			  select audit_export_max_rows > 0 from public.admin_retention_settings where id = 'default'
			), false),
			exists (
			  select 1 from information_schema.columns
			  where table_schema = 'public' and table_name = 'admin_product_settings'
			    and column_name = 'min_cli_version'
			),
			coalesce((
			  select churn_high_usage_ratio > 0
			     and churn_medium_usage_ratio > 0
			     and churn_high_idle_days > 0
			     and churn_low_idle_days > 0
			  from public.admin_product_settings where id = 'default'
			), false),
			exists (
			  select 1 from information_schema.columns
			  where table_schema = 'public' and table_name = 'admin_product_settings'
			    and column_name = 'paddle_webhook_queue_warn_depth'
			),
			exists (
			  select 1 from public.site_messages where code = 'ADMIN_USER_SECTION_LINKED_HW'
			)
	`).Scan(
		&pricingBound, &idpCount, &totpTableOK, &notifTableOK, &webauthnTableOK,
		&creditsNavOK, &auditExportCapOK, &minCLIColOK, &churnOK, &queueWarnColOK, &linkedHwChromeOK,
	)
	checklist = append(checklist,
		map[string]any{"id": "pricing", "ok": pricingBound, "label": h.msg("ADMIN_CHECKLIST_PRICING")},
		map[string]any{"id": "company", "ok": strings.TrimSpace(h.Config.CompanyLegalName) != "", "label": h.msg("ADMIN_CHECKLIST_COMPANY")},
		map[string]any{"id": "geolite", "ok": strings.TrimSpace(h.Config.GeoLiteASNMMDBPath) != "" || strings.TrimSpace(h.Config.GeoLiteAnonymousMMDBPath) != "", "label": h.msg("ADMIN_CHECKLIST_GEOLITE")},
		map[string]any{"id": "idp", "ok": idpCount > 0, "label": h.msg("ADMIN_CHECKLIST_IDP")},
		map[string]any{"id": "migrations_totp", "ok": totpTableOK, "label": h.msg("ADMIN_CHECKLIST_MIGRATIONS")},
		map[string]any{"id": "migrations_notifications", "ok": notifTableOK, "label": h.msg("ADMIN_CHECKLIST_MIGRATIONS_NOTIF")},
		map[string]any{"id": "migrations_webauthn", "ok": webauthnTableOK, "label": h.msg("ADMIN_CHECKLIST_MIGRATIONS_WEBAUTHN")},
		map[string]any{"id": "migrations_credits_nav", "ok": creditsNavOK, "label": h.msg("ADMIN_CHECKLIST_MIGRATIONS_CREDITS_NAV")},
		map[string]any{"id": "migrations_audit_export", "ok": auditExportCapOK, "label": h.msg("ADMIN_CHECKLIST_MIGRATIONS_AUDIT_EXPORT")},
		map[string]any{"id": "migrations_min_cli", "ok": minCLIColOK, "label": h.msg("ADMIN_CHECKLIST_MIGRATIONS_MIN_CLI")},
		map[string]any{"id": "migrations_churn", "ok": churnOK, "label": h.msg("ADMIN_CHECKLIST_MIGRATIONS_CHURN")},
		map[string]any{"id": "migrations_queue_warn", "ok": queueWarnColOK, "label": h.msg("ADMIN_CHECKLIST_MIGRATIONS_QUEUE_WARN")},
		map[string]any{"id": "migrations_linked_hw", "ok": linkedHwChromeOK, "label": h.msg("ADMIN_CHECKLIST_MIGRATIONS_LINKED_HW")},
	)
	_, totpKeyOK := h.totpKey()
	checklist = append(checklist, map[string]any{
		"id": "totp_key", "ok": totpKeyOK, "label": h.msg("ADMIN_CHECKLIST_TOTP_KEY"),
	})
	_, webauthnRPOK := h.webauthnRPID()
	checklist = append(checklist, map[string]any{
		"id": "webauthn_rp", "ok": webauthnRPOK && len(h.Config.AdminAllowedOrigins) > 0,
		"label": h.msg("ADMIN_CHECKLIST_WEBAUTHN_RP"),
	})
	checklist = append(checklist, map[string]any{
		"id": "admin_ip", "ok": len(h.Config.AdminAllowedCIDRs) > 0 || h.Config.DeploymentMode != "cloud",
		"label": h.msg("ADMIN_CHECKLIST_ADMIN_IP"),
	})
	checklist = append(checklist, map[string]any{
		"id": "cf_access",
		"ok": (strings.TrimSpace(h.Config.CFAccessTeamDomain) != "" && strings.TrimSpace(h.Config.CFAccessAUD) != "") ||
			h.Config.DeploymentMode != "cloud",
		"label": h.msg("ADMIN_CHECKLIST_CF_ACCESS"),
	})

	kpiDefs := []struct {
		id    string
		code  string
		value int
	}{
		{"users_total", "ADMIN_KPI_USERS_TOTAL", usersTotal},
		{"subscriptions_active", "ADMIN_KPI_SUBSCRIPTIONS_ACTIVE", subsActive},
		{"events_24h", "ADMIN_KPI_EVENTS_24H", events24h},
		{"install_hits_7d", "ADMIN_KPI_INSTALL_HITS_7D", install7d},
		{"active_today", "ADMIN_KPI_ACTIVE_TODAY", activeToday},
		{"active_7d", "ADMIN_KPI_ACTIVE_7D", active7d},
		{"active_30d", "ADMIN_KPI_ACTIVE_30D", active30d},
		{"paid_seats", "ADMIN_KPI_PAID_SEATS", paidSeats},
		{"mrr_cents", "ADMIN_KPI_MRR_CENTS", int(mrrCents)},
		{"arr_cents", "ADMIN_KPI_ARR_CENTS", int(arrCents)},
		{"churn_30d", "ADMIN_KPI_CHURN_30D", churn30d},
		{"new_paid_30d", "ADMIN_KPI_NEW_PAID_30D", newPaid30d},
		{"free_to_paid_30d", "ADMIN_KPI_FREE_TO_PAID_30D", freeToPaid30d},
	}
	kpis := make(map[string]int, len(kpiDefs))
	kpiItems := make([]map[string]any, 0, len(kpiDefs))
	for _, d := range kpiDefs {
		label := h.msg(d.code)
		if strings.TrimSpace(label) == "" {
			continue
		}
		kpis[d.id] = d.value
		kpiItems = append(kpiItems, map[string]any{
			"id": d.id, "label": label, "value": d.value,
		})
	}

	statusOK := h.msg("ADMIN_CHECKLIST_STATUS_OK")
	statusFail := h.msg("ADMIN_CHECKLIST_STATUS_FAIL")
	filteredChecklist := make([]map[string]any, 0, len(checklist))
	for i := range checklist {
		label, _ := checklist[i]["label"].(string)
		if strings.TrimSpace(label) == "" {
			continue
		}
		if ok, _ := checklist[i]["ok"].(bool); ok {
			checklist[i]["status_label"] = statusOK
		} else {
			checklist[i]["status_label"] = statusFail
		}
		filteredChecklist = append(filteredChecklist, checklist[i])
	}
	checklist = filteredChecklist

	healthItems := []map[string]any{}
	if lbl := h.msg("ADMIN_HEALTH_POSTGRES"); strings.TrimSpace(lbl) != "" {
		healthItems = append(healthItems, map[string]any{
			"id": "postgres", "ok": dbOK, "label": lbl,
			"status_label": map[bool]string{true: statusOK, false: statusFail}[dbOK],
		})
	}
	if lbl := h.msg("ADMIN_HEALTH_REDIS"); strings.TrimSpace(lbl) != "" {
		healthItems = append(healthItems, map[string]any{
			"id": "redis", "ok": redisOK, "label": lbl,
			"status_label": map[bool]string{true: statusOK, false: statusFail}[redisOK],
		})
	}

	var lastWebhookAgeSec *int64
	var lastWebhookAt *string
	var whAt *time.Time
	_ = db.QueryRow(ctx, `
		select processed_at from public.paddle_webhook_events
		order by processed_at desc nulls last limit 1
	`).Scan(&whAt)
	if whAt != nil {
		age := int64(time.Since(*whAt).Seconds())
		if age < 0 {
			age = 0
		}
		lastWebhookAgeSec = &age
		s := whAt.UTC().Format(time.RFC3339)
		lastWebhookAt = &s
	}
	if lbl := h.msg("ADMIN_HEALTH_PADDLE_WEBHOOKS"); strings.TrimSpace(lbl) != "" {
		ok := lastWebhookAgeSec != nil
		detail := statusFail
		if ok {
			detail = statusOK
		}
		if lastWebhookAgeSec != nil {
			ageFmt := strings.TrimSpace(h.msg("ADMIN_HEALTH_AGE_SUFFIX_FMT"))
			if ageFmt != "" && strings.Contains(ageFmt, "{age}") {
				detail = detail + strings.ReplaceAll(
					ageFmt, "{age}", fmt.Sprintf("%d", *lastWebhookAgeSec),
				)
			}
		}
		item := map[string]any{
			"id": "paddle_webhooks", "ok": ok, "label": lbl, "status_label": detail,
		}
		if lastWebhookAgeSec != nil {
			item["age_seconds"] = *lastWebhookAgeSec
		}
		if lastWebhookAt != nil {
			item["last_at"] = *lastWebhookAt
		}
		healthItems = append(healthItems, item)
	}

	var err24h, ok24h int
	_ = db.QueryRow(ctx, `
		select
			count(*) filter (where status = 'error'),
			count(*) filter (where status = 'success')
		from public.trim_events where created_at >= now() - interval '24 hours'
	`).Scan(&err24h, &ok24h)
	total24 := err24h + ok24h
	errorRatePct := 0
	if total24 > 0 {
		errorRatePct = (err24h * 100) / total24
	}
	if lbl := h.msg("ADMIN_HEALTH_ERROR_RATE"); strings.TrimSpace(lbl) != "" {
		statusLbl := ""
		fmtTpl := strings.TrimSpace(h.msg("ADMIN_HEALTH_ERROR_RATE_FMT"))
		if fmtTpl != "" && strings.Contains(fmtTpl, "{pct}") {
			statusLbl = strings.ReplaceAll(fmtTpl, "{pct}", fmt.Sprintf("%d", errorRatePct))
		}
		if statusLbl != "" {
			healthItems = append(healthItems, map[string]any{
				"id": "error_rate_24h", "ok": err24h == 0, "label": lbl,
				"status_label":  statusLbl,
				"error_count":   err24h,
				"success_count": ok24h,
			})
		}
	}

	if lbl := h.msg("ADMIN_HEALTH_GEOLITE"); strings.TrimSpace(lbl) != "" {
		geoOK := strings.TrimSpace(h.Config.GeoLiteASNMMDBPath) != "" || strings.TrimSpace(h.Config.GeoLiteAnonymousMMDBPath) != ""
		healthItems = append(healthItems, map[string]any{
			"id": "geolite", "ok": geoOK, "label": lbl,
			"status_label": map[bool]string{true: statusOK, false: statusFail}[geoOK],
		})
	}
	if lbl := h.msg("ADMIN_HEALTH_QUEUE_LAG"); strings.TrimSpace(lbl) != "" {
		depth := 0
		if h.Billing != nil {
			depth = h.Billing.WebhookQueueDepth()
		}
		var queueWarn int
		_ = db.QueryRow(ctx, `
			select coalesce(paddle_webhook_queue_warn_depth, 0)
			from public.admin_product_settings where id = 'default'
		`).Scan(&queueWarn)
		// Fail closed: without a positive DB threshold, queue is not marked healthy.
		ok := queueWarn > 0 && depth < queueWarn
		statusLbl := ""
		fmtTpl := strings.TrimSpace(h.msg("ADMIN_HEALTH_QUEUE_DEPTH_FMT"))
		if fmtTpl != "" && strings.Contains(fmtTpl, "{depth}") {
			statusLbl = strings.ReplaceAll(fmtTpl, "{depth}", fmt.Sprintf("%d", depth))
		}
		if statusLbl != "" {
			healthItems = append(healthItems, map[string]any{
				"id": "paddle_queue", "ok": ok, "label": lbl,
				"status_label": statusLbl,
				"depth":        depth,
				"warn_depth":   queueWarn,
			})
		}
	}

	var quotaExhausted, ja4CapBreach, hwCapBreach, suspended7d, webhookFailed, deepErr24h, oom24h int
	var maxJA4, maxHW int
	_ = db.QueryRow(ctx, `
		select
			(select count(*) from public.user_quotas
			 where monthly_credit_limit > 0 and monthly_credit_used >= monthly_credit_limit),
			coalesce((select max_accounts_per_ja4 from public.admin_product_settings where id = 'default'), 0),
			coalesce((select max_accounts_per_hardware from public.admin_product_settings where id = 'default'), 0),
			(select count(*) from public.profiles
			 where account_status in ('suspended', 'banned', 'shadowbanned')
			   and coalesce(account_status_changed_at, created_at) >= now() - interval '7 days'),
			(select count(*) from public.paddle_webhook_events where process_status = 'failed'),
			(select count(*) from public.trim_events
			 where status = 'error'
			   and created_at >= now() - interval '24 hours'
			   and (
			     lower(mode) like '%deep%'
			     or lower(mode) in ('v1', 'v2', 'long')
			   )),
			(select count(*) from public.trim_events
			 where created_at >= now() - interval '24 hours'
			   and lower(coalesce(error_code, '')) = 'oom')
	`).Scan(&quotaExhausted, &maxJA4, &maxHW, &suspended7d, &webhookFailed, &deepErr24h, &oom24h)
	if maxJA4 > 0 {
		_ = db.QueryRow(ctx, `
			select count(*) from (
				select ja4_hash from public.ja4_fingerprints
				group by ja4_hash
				having count(distinct user_id) > $1
			) t
		`, maxJA4).Scan(&ja4CapBreach)
	}
	if maxHW > 0 {
		_ = db.QueryRow(ctx, `
			select count(*) from (
				select hardware_uuid from public.device_fingerprints
				group by hardware_uuid
				having count(distinct user_id) > $1
			) t
		`, maxHW).Scan(&hwCapBreach)
	}

	alertDefs := []struct {
		id    string
		code  string
		count int
	}{
		{"event_errors_24h", "ADMIN_ALERT_EVENT_ERRORS_24H", err24h},
		{"deep_errors_24h", "ADMIN_ALERT_DEEP_ERRORS_24H", deepErr24h},
		{"oom_24h", "ADMIN_ALERT_OOM_24H", oom24h},
		{"quota_exhausted", "ADMIN_ALERT_QUOTA_EXHAUSTED", quotaExhausted},
		{"ja4_cap_breach", "ADMIN_ALERT_JA4_CAP_BREACH", ja4CapBreach},
		{"hardware_cap_breach", "ADMIN_ALERT_HARDWARE_CAP_BREACH", hwCapBreach},
		{"suspended_7d", "ADMIN_ALERT_SUSPENDED_7D", suspended7d},
		{"webhook_failed", "ADMIN_ALERT_WEBHOOK_FAILED", webhookFailed},
	}
	alerts := make([]map[string]any, 0)
	for _, d := range alertDefs {
		label := h.msg(d.code)
		if strings.TrimSpace(label) == "" {
			continue
		}
		alerts = append(alerts, map[string]any{
			"id": d.id, "label": label, "count": d.count, "active": d.count > 0,
		})
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"kpis":      kpis,
		"kpi_items": kpiItems,
		"health": map[string]bool{
			"postgres": dbOK,
			"redis":    redisOK,
		},
		"health_items":  healthItems,
		"ops_checklist": checklist,
		"alerts":        alerts,
		"generated_at":  time.Now().UTC().Format(time.RFC3339),
		"chrome": map[string]string{
			"brand":               h.msg("ADMIN_BRAND"),
			"tagline":             h.msg("ADMIN_TAGLINE"),
			"ops_checklist_title": h.msg("ADMIN_OPS_CHECKLIST_TITLE"),
			"health_title":        h.msg("ADMIN_HEALTH_TITLE"),
			"alerts_title":        h.msg("ADMIN_ALERTS_TITLE"),
			"revenue_hint":        h.msg("ADMIN_DASHBOARD_REVENUE_HINT"),
			"revenue_link":        h.msg("ADMIN_DASHBOARD_REVENUE_LINK"),
			"revenue_href":        h.adminNavHref(r.Context(), "revenue"),
		},
	})
}

// InstallHit records a public install.sh beacon (country from CF-IPCountry only).
func (h *Handler) InstallHit(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		h.writeErr(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
		return
	}
	country := strings.ToUpper(strings.TrimSpace(r.Header.Get("CF-IPCountry")))
	if len(country) > 8 {
		country = country[:8]
	}
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_INSTALL_PATH_REQUIRED")
		return
	}
	if len(path) > 256 {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_INSTALL_PATH_INVALID")
		return
	}
	_, err := h.DB.Exec(r.Context(), `
		insert into public.install_hits (country, path) values ($1, $2)
	`, country, path)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "ADMIN_DISTRIBUTION_SYNC_FAILED")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func scanJSONSettings(ctx context.Context, h *Handler, table string, id string) (map[string]any, error) {
	var raw []byte
	err := h.readPool().QueryRow(ctx, `
		select row_to_json(t) from (select * from `+table+` where id = $1) t
	`, id).Scan(&raw)
	if err == pgx.ErrNoRows {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, err
		}
	}
	return out, nil
}
