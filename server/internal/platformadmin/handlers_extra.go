package platformadmin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/usetrim/trim/server/internal/billingsettings"
	"github.com/usetrim/trim/server/internal/middleware"
	"github.com/usetrim/trim/server/internal/pagination"
	"github.com/usetrim/trim/server/internal/receipts"
	"github.com/usetrim/trim/server/internal/subscriptions"
)

func (h *Handler) breakGlassGrants(ctx context.Context, userID, code string) bool {
	var perm string
	var endsAt time.Time
	err := h.DB.QueryRow(ctx, `
		select coalesce(elevates_permission, ''), ends_at
		from public.admin_break_glass
		where requester_id = $1::uuid and status = 'approved'
		  and ends_at is not null and ends_at > now()
		order by ends_at desc
		limit 1
	`, userID).Scan(&perm, &endsAt)
	if err != nil {
		return false
	}
	return perm != "" && perm == code
}

func (h *Handler) postUserForceLogout(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p := PrincipalFromContext(r.Context())
	var reasonBody struct {
		Reason string `json:"reason"`
	}
	_ = decodeJSON(r, &reasonBody)
	reason := strings.TrimSpace(reasonBody.Reason)
	if reason == "" {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_REASON_REQUIRED")
		return
	}
	tag, err := h.DB.Exec(r.Context(), `
		update public.api_keys set revoked = true where user_id = $1::uuid and revoked = false
	`, id)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if h.Redis != nil {
		var ttlSec int
		err := h.readPool().QueryRow(r.Context(), `
			select coalesce(force_logout_ttl_sec, 0)
			from public.admin_retention_settings where id = 'default'
		`).Scan(&ttlSec)
		if err != nil || ttlSec < 60 {
			h.writeErr(w, http.StatusServiceUnavailable, "ADMIN_FORCE_LOGOUT_TTL_MISSING")
			return
		}
		if err := h.Redis.Set(r.Context(), "admin:force_logout:"+id, fmt.Sprintf("%d", time.Now().Unix()), time.Duration(ttlSec)*time.Second).Err(); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "users.force_logout", "profile", id, nil,
		map[string]any{"keys_revoked": tag.RowsAffected()}, reason, step)
	h.writeJSON(w, http.StatusOK, map[string]any{
		"message":      h.msg("ADMIN_FORCE_LOGOUT_DONE"),
		"keys_revoked": tag.RowsAffected(),
	})
}

func (h *Handler) postUserGDPRExport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p := PrincipalFromContext(r.Context())
	var reasonBody struct {
		Reason string `json:"reason"`
	}
	_ = decodeJSON(r, &reasonBody)
	reason := strings.TrimSpace(reasonBody.Reason)
	if reason == "" {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_REASON_REQUIRED")
		return
	}
	payload := map[string]any{}
	var profileJSON []byte
	err := h.DB.QueryRow(r.Context(), `
		select row_to_json(p) from public.profiles p where id = $1::uuid
	`, id).Scan(&profileJSON)
	if err == pgx.ErrNoRows {
		h.writeErr(w, http.StatusNotFound, "ADMIN_USER_NOT_FOUND")
		return
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if err := json.Unmarshal(profileJSON, &payload); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	appendJSONArray := func(key, sql string, args ...any) error {
		var raw []byte
		if err := h.DB.QueryRow(r.Context(), sql, args...).Scan(&raw); err != nil {
			return err
		}
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			return err
		}
		payload[key] = v
		return nil
	}
	if err := appendJSONArray("quota", `
		select coalesce(
		  (select row_to_json(q) from public.user_quotas q where user_id = $1::uuid),
		  '{}'::json
		)
	`, id); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if err := appendJSONArray("identities", `
		select coalesce(json_agg(json_build_object(
		  'provider', provider, 'identity_id', id::text, 'created_at', created_at
		) order by created_at), '[]'::json)
		from auth.identities where user_id = $1::uuid
	`, id); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if err := appendJSONArray("workspaces", `
		select coalesce(json_agg(json_build_object(
		  'workspace_id', w.id::text, 'name', w.name, 'role', m.role, 'plan_tier', w.plan_tier
		) order by w.name), '[]'::json)
		from public.workspace_members m
		join public.workspaces w on w.id = m.workspace_id
		where m.user_id = $1::uuid
	`, id); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if err := appendJSONArray("api_keys", `
		select coalesce(json_agg(json_build_object(
		  'id', id::text, 'prefix', key_prefix, 'revoked', revoked,
		  'created_at', created_at, 'expires_at', expires_at
		) order by created_at), '[]'::json)
		from public.api_keys where user_id = $1::uuid
	`, id); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if err := appendJSONArray("devices", `
		select coalesce(json_agg(json_build_object(
		  'id', id::text, 'hardware_uuid', hardware_uuid, 'last_seen_at', last_seen_at
		) order by last_seen_at desc nulls last), '[]'::json)
		from public.device_fingerprints where user_id = $1::uuid
	`, id); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if err := appendJSONArray("receipts", `
		select coalesce(json_agg(json_build_object(
		  'id', id::text, 'paddle_transaction_id', paddle_transaction_id,
		  'total_cents', total_cents, 'currency_code', currency_code,
		  'status', status, 'created_at', created_at
		) order by created_at desc), '[]'::json)
		from public.billing_receipts where user_id = $1::uuid
	`, id); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if err := appendJSONArray("subscriptions", `
		select coalesce(json_agg(json_build_object(
		  'id', id::text, 'plan_tier', plan_tier, 'status', status,
		  'billing_interval', billing_interval, 'created_at', created_at
		) order by created_at desc), '[]'::json)
		from public.subscriptions where user_id = $1::uuid
	`, id); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	var eventsSummary []byte
	if err := h.DB.QueryRow(r.Context(), `
		select json_build_object(
		  'total_events', count(*),
		  'tokens_before', coalesce(sum(tokens_before), 0),
		  'tokens_after', coalesce(sum(tokens_after), 0),
		  'tokens_saved', coalesce(sum(greatest(tokens_before - tokens_after, 0)), 0)
		)
		from public.trim_events where user_id = $1::uuid
	`, id).Scan(&eventsSummary); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	var ev any
	_ = json.Unmarshal(eventsSummary, &ev)
	payload["events_summary"] = ev

	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "users.gdpr_export", "profile", id, nil, map[string]string{"ok": "1"}, reason, step)
	generatedAt := time.Now().UTC().Format(time.RFC3339)
	filenameFmt := strings.TrimSpace(h.msg("ADMIN_GDPR_EXPORT_FILENAME_FMT"))
	filename := ""
	if filenameFmt != "" && strings.Contains(filenameFmt, "{generated_at}") {
		safeStamp := strings.Map(func(r rune) rune {
			switch r {
			case ':', '.', '+':
				return '-'
			default:
				return r
			}
		}, generatedAt)
		filename = strings.ReplaceAll(filenameFmt, "{generated_at}", safeStamp)
		filename = strings.ReplaceAll(filename, "{user_id}", id)
	}
	out := map[string]any{
		"message":      h.msg("ADMIN_GDPR_EXPORT_READY"),
		"export":       payload,
		"generated_at": generatedAt,
	}
	if filename != "" {
		out["filename"] = filename
	}
	h.writeJSON(w, http.StatusOK, out)
}

func (h *Handler) postUserGDPRErase(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p := PrincipalFromContext(r.Context())
	var reasonBody struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &reasonBody); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	reason := strings.TrimSpace(reasonBody.Reason)
	if reason == "" {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_REASON_REQUIRED")
		return
	}
	_, err := h.DB.Exec(r.Context(), `select public.execute_gdpr_deletion($1::uuid)`, id)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "users.gdpr_erase", "profile", id, nil, map[string]string{"erased": "1"}, reason, step)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_GDPR_ERASE_DONE")})
}

func (h *Handler) listSegmentsIndividuals(w http.ResponseWriter, r *http.Request) {
	h.listSegmentUsers(w, r, `uq.plan_tier in ('free', 'pro')`, "ADMIN_SEGMENT_INDIVIDUALS")
}

func (h *Handler) listSegmentsEnterprises(w http.ResponseWriter, r *http.Request) {
	h.listSegmentUsers(w, r, `uq.plan_tier = 'enterprise'`, "ADMIN_SEGMENT_ENTERPRISE")
}

func (h *Handler) listSegmentsTeams(w http.ResponseWriter, r *http.Request) {
	params, skipCap, err := h.listPage(r)
	if err != nil {
		h.writePaginateErr(w, err)
		return
	}
	db := h.readPool()
	plan := strings.TrimSpace(r.URL.Query().Get("plan"))
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	createdFrom := strings.TrimSpace(r.URL.Query().Get("created_from"))
	createdTo := strings.TrimSpace(r.URL.Query().Get("created_to"))
	where := `where 1=1`
	args := []any{}
	n := 1
	if plan != "" {
		where += fmt.Sprintf(` and w.plan_tier = $%d`, n)
		args = append(args, plan)
		n++
	}
	if q != "" {
		where += fmt.Sprintf(` and lower(w.name) like $%d`, n)
		args = append(args, "%"+strings.ToLower(q)+"%")
		n++
	}
	if createdFrom != "" {
		where += fmt.Sprintf(` and w.created_at::date >= $%d::date`, n)
		args = append(args, createdFrom)
		n++
	}
	if createdTo != "" {
		where += fmt.Sprintf(` and w.created_at::date <= $%d::date`, n)
		args = append(args, createdTo)
		n++
	}
	var total int
	if err := db.QueryRow(r.Context(), `select count(*) from public.workspaces w `+where, args...).Scan(&total); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	listSQL := fmt.Sprintf(`
		select w.id::text, w.name, w.plan_tier, w.allocated_seats,
		       (select count(*) from public.workspace_members m where m.workspace_id = w.id) as members,
		       (select count(*) from public.workspace_invites i where i.workspace_id = w.id and i.status = 'pending') as pending
		from public.workspaces w
		%s
		order by w.created_at desc
		offset $%d limit $%d
	`, where, n, n+1)
	listArgs := append(append([]any{}, args...), params.Skip, params.Limit)
	rows, err := db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, name, tier string
		var seats, members, pending int
		if err := rows.Scan(&id, &name, &tier, &seats, &members, &pending); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		items = append(items, map[string]any{
			"id": id, "name": name, "plan_tier": tier,
			"allocated_seats": seats, "members": members, "pending_invites": pending,
		})
	}
	meta := pagination.BuildMeta(params, total, skipCap)
	h.writeJSON(w, http.StatusOK, map[string]any{"items": items, "meta": meta, "segment": h.msg("ADMIN_SEGMENT_TEAMS")})
}

func (h *Handler) listSegmentUsers(w http.ResponseWriter, r *http.Request, planFilter, segmentChromeCode string) {
	params, skipCap, err := h.listPage(r)
	if err != nil {
		h.writePaginateErr(w, err)
		return
	}
	db := h.readPool()
	plan := strings.TrimSpace(r.URL.Query().Get("plan"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	country := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("country")))
	interval := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("interval")))
	createdFrom := strings.TrimSpace(r.URL.Query().Get("created_from"))
	createdTo := strings.TrimSpace(r.URL.Query().Get("created_to"))
	creditsLeftMax := strings.TrimSpace(r.URL.Query().Get("credits_left_max"))
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	where := `where ` + planFilter
	args := []any{}
	n := 1
	if plan != "" {
		where += fmt.Sprintf(` and uq.plan_tier = $%d`, n)
		args = append(args, plan)
		n++
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
	if interval != "" {
		where += fmt.Sprintf(` and lower(coalesce(s.billing_interval, '')) = $%d`, n)
		args = append(args, interval)
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
	if creditsLeftMax != "" {
		where += fmt.Sprintf(` and greatest(coalesce(uq.monthly_credit_limit, 0) - coalesce(uq.monthly_credit_used, 0), 0) <= $%d`, n)
		args = append(args, creditsLeftMax)
		n++
	}
	if q != "" {
		where += fmt.Sprintf(` and lower(p.email) like $%d`, n)
		args = append(args, "%"+strings.ToLower(q)+"%")
		n++
	}

	fromSQL := `
		from public.profiles p
		left join public.user_quotas uq on uq.user_id = p.id
		left join lateral (
		  select billing_interval from public.subscriptions s0
		  where s0.user_id = p.id
		  order by s0.created_at desc
		  limit 1
		) s on true
	`

	var total int
	qTotal := `select count(*) ` + fromSQL + ` ` + where
	if err := db.QueryRow(r.Context(), qTotal, args...).Scan(&total); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	listSQL := fmt.Sprintf(`
		select p.id::text, p.email, coalesce(uq.plan_tier, ''), p.account_status,
		       coalesce(p.last_login_country, ''),
		       coalesce(uq.monthly_credit_used, 0), coalesce(uq.monthly_credit_limit, 0),
		       coalesce(s.billing_interval, ''),
		       case when p.last_login_at is null then null
		            else extract(epoch from (now() - p.last_login_at)) / 86400.0
		       end,
		       coalesce((
		         select sum(greatest(e.tokens_before - e.tokens_after, 0))
		         from public.trim_events e
		         where e.user_id = p.id
		       ), 0)
		%s
		%s
		order by p.created_at desc
		offset $%d limit $%d
	`, fromSQL, where, n, n+1)
	listArgs := append(append([]any{}, args...), params.Skip, params.Limit)
	rows, err := db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()

	var highUsage, medUsage float64
	var highIdle, lowIdle int
	err = db.QueryRow(r.Context(), `
		select churn_high_usage_ratio, churn_medium_usage_ratio,
		       churn_high_idle_days, churn_low_idle_days
		from public.admin_product_settings where id = 'default'
	`).Scan(&highUsage, &medUsage, &highIdle, &lowIdle)
	if err == pgx.ErrNoRows || highUsage <= 0 || medUsage <= 0 || highIdle <= 0 || lowIdle <= 0 {
		h.writeErr(w, http.StatusConflict, "ADMIN_CHURN_THRESHOLDS_MISSING")
		return
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}

	items := []map[string]any{}
	for rows.Next() {
		var id, email, tier, statusVal, countryVal, billingInterval string
		var used, limit int
		var daysSinceLogin *float64
		var tokensSaved int64
		if err := rows.Scan(&id, &email, &tier, &statusVal, &countryVal, &used, &limit, &billingInterval, &daysSinceLogin, &tokensSaved); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		left := limit - used
		if left < 0 {
			left = 0
		}
		churnCode := "ADMIN_CHURN_NONE"
		neverLogin := daysSinceLogin == nil
		days := 0.0
		if daysSinceLogin != nil {
			days = *daysSinceLogin
		}
		usageRatio := 0.0
		if limit > 0 {
			usageRatio = float64(used) / float64(limit)
		}
		if limit > 0 && usageRatio >= highUsage && (neverLogin || days >= float64(highIdle)) {
			churnCode = "ADMIN_CHURN_HIGH"
		} else if limit > 0 && usageRatio >= medUsage {
			churnCode = "ADMIN_CHURN_MEDIUM"
		} else if neverLogin || days >= float64(lowIdle) {
			churnCode = "ADMIN_CHURN_LOW"
		}
		churnLabel := h.msg(churnCode)
		row := map[string]any{
			"id": id, "email": email, "plan_tier": tier, "account_status": statusVal,
			"account_status_label": h.accountStatusLabel(statusVal),
			"last_login_country":   countryVal, "credits_used": used, "credits_limit": limit,
			"credits_left": left, "billing_interval": billingInterval,
			"tokens_saved": tokensSaved,
		}
		if strings.TrimSpace(churnLabel) != "" {
			row["churn_risk"] = churnLabel
			row["churn_risk_code"] = churnCode
		}
		items = append(items, row)
	}
	meta := pagination.BuildMeta(params, total, skipCap)
	h.writeJSON(w, http.StatusOK, map[string]any{"items": items, "meta": meta, "segment": h.msg(segmentChromeCode)})
}

func (h *Handler) getProductSettings(w http.ResponseWriter, r *http.Request) {
	db := h.readPool()
	out, err := scanJSONSettings(r.Context(), h, "public.admin_product_settings", "default")
	if err == pgx.ErrNoRows {
		h.writeErr(w, http.StatusServiceUnavailable, "ADMIN_PRODUCT_SETTINGS_MISSING")
		return
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	var cliNotice string
	_ = db.QueryRow(r.Context(), `
		select coalesce(body, '') from public.site_messages where code = 'CLI_FORCE_UPGRADE_NOTICE'
	`).Scan(&cliNotice)
	out["cli_force_upgrade_notice"] = cliNotice
	var dbMinCLI string
	_ = db.QueryRow(r.Context(), `
		select coalesce(min_cli_version, '') from public.admin_product_settings where id = 'default'
	`).Scan(&dbMinCLI)
	effMinCLI := strings.TrimSpace(dbMinCLI)
	out["min_cli_version"] = strings.TrimSpace(dbMinCLI)
	statusOK := h.msg("ADMIN_CHECKLIST_STATUS_OK")
	statusFail := h.msg("ADMIN_CHECKLIST_STATUS_FAIL")
	boolLabel := func(ok bool) string {
		if ok {
			return statusOK
		}
		return statusFail
	}
	geoASN := strings.TrimSpace(h.Config.GeoLiteASNMMDBPath) != ""
	geoAnon := strings.TrimSpace(h.Config.GeoLiteAnonymousMMDBPath) != ""
	runtimeDefs := []struct {
		id    string
		code  string
		value string
	}{
		{"deployment_mode", "ADMIN_RUNTIME_DEPLOYMENT_MODE", strings.TrimSpace(h.Config.DeploymentMode)},
		{"compression_mode_env", "ADMIN_RUNTIME_COMPRESSION_MODE_ENV", strings.TrimSpace(h.Config.CompressionMode)},
		{"min_cli_version", "ADMIN_RUNTIME_MIN_CLI_VERSION", effMinCLI},
		{"min_cli_version_env", "ADMIN_RUNTIME_MIN_CLI_VERSION_ENV", strings.TrimSpace(h.Config.MinCLIVersion)},
		{"pow_difficulty", "ADMIN_RUNTIME_POW_DIFFICULTY", fmt.Sprintf("%d", h.Config.FreeTierPoWDifficulty)},
		{"rate_limit_ip", "ADMIN_RUNTIME_RATE_LIMIT_IP", fmt.Sprintf("%d", h.Config.RateLimitPerIPPerMin)},
		{"rate_limit_user", "ADMIN_RUNTIME_RATE_LIMIT_USER", fmt.Sprintf("%d", h.Config.RateLimitPerUserPerMin)},
		{"max_accounts_hw", "ADMIN_RUNTIME_MAX_ACCOUNTS_HW", fmt.Sprintf("%d", h.Config.MaxAccountsPerHardware)},
		{"max_accounts_ja4", "ADMIN_RUNTIME_MAX_ACCOUNTS_JA4", fmt.Sprintf("%d", h.Config.MaxAccountsPerJA4)},
		{"cf_threat_score_min", "ADMIN_RUNTIME_CF_THREAT_SCORE_MIN", fmt.Sprintf("%d", h.Config.CFThreatScoreMin)},
		{"geolite_asn_path", "ADMIN_RUNTIME_GEOLITE_ASN_PATH", boolLabel(geoASN)},
		{"geolite_anon_path", "ADMIN_RUNTIME_GEOLITE_ANON_PATH", boolLabel(geoAnon)},
		{"cheap_model", "ADMIN_RUNTIME_CHEAP_MODEL", strings.TrimSpace(h.Config.CheapModel)},
		{"route_max_tokens", "ADMIN_RUNTIME_ROUTE_MAX_TOKENS", strings.TrimSpace(h.Config.RouteMaxTokens)},
		{"smtp_configured", "ADMIN_RUNTIME_SMTP_CONFIGURED", boolLabel(h.Mailer.Enabled())},
	}
	runtimeItems := make([]map[string]any, 0, len(runtimeDefs))
	runtimeMap := make(map[string]any, len(runtimeDefs))
	for _, d := range runtimeDefs {
		label := strings.TrimSpace(h.msg(d.code))
		val := strings.TrimSpace(d.value)
		if label == "" || val == "" {
			continue
		}
		runtimeItems = append(runtimeItems, map[string]any{
			"id": d.id, "label": label, "value": val,
		})
		runtimeMap[d.id] = val
	}
	out["runtime"] = runtimeMap
	out["runtime_items"] = runtimeItems
	h.writeJSON(w, http.StatusOK, out)
}

func (h *Handler) patchProductSettings(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	var body map[string]any
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	before, _ := scanJSONSettings(r.Context(), h, "public.admin_product_settings", "default")
	_, err := h.DB.Exec(r.Context(), `
		update public.admin_product_settings set
		  default_compression_mode = coalesce($2, default_compression_mode),
		  default_deep_engine = coalesce($3, default_deep_engine),
		  treesitter_required = coalesce($4, treesitter_required),
		  deep_attach_default = coalesce($5, deep_attach_default),
		  model_routing_enabled = coalesce($6, model_routing_enabled),
		  rate_limit_ip_per_min = coalesce($7, rate_limit_ip_per_min),
		  rate_limit_user_per_min = coalesce($8, rate_limit_user_per_min),
		  pow_difficulty = coalesce($9, pow_difficulty),
		  max_accounts_per_hardware = coalesce($10, max_accounts_per_hardware),
		  max_accounts_per_ja4 = coalesce($11, max_accounts_per_ja4),
		  cf_threat_score_min = coalesce($12, cf_threat_score_min),
		  fast_balanced_min_lines = coalesce($13, fast_balanced_min_lines),
		  fast_aggressive_min_lines = coalesce($14, fast_aggressive_min_lines),
		  fast_mild_min_lines = coalesce($15, fast_mild_min_lines),
		  min_cli_version = coalesce($16, min_cli_version),
		  churn_high_usage_ratio = coalesce($17, churn_high_usage_ratio),
		  churn_medium_usage_ratio = coalesce($18, churn_medium_usage_ratio),
		  churn_high_idle_days = coalesce($19, churn_high_idle_days),
		  churn_low_idle_days = coalesce($20, churn_low_idle_days),
		  paddle_webhook_queue_warn_depth = coalesce($21, paddle_webhook_queue_warn_depth),
		  updated_at = now(),
		  updated_by = $1::uuid
		where id = 'default'
	`, p.UserID,
		strPtr(body, "default_compression_mode"),
		strPtr(body, "default_deep_engine"),
		boolPtr(body, "treesitter_required"),
		boolPtr(body, "deep_attach_default"),
		boolPtr(body, "model_routing_enabled"),
		intPtr(body, "rate_limit_ip_per_min"),
		intPtr(body, "rate_limit_user_per_min"),
		intPtr(body, "pow_difficulty"),
		intPtr(body, "max_accounts_per_hardware"),
		intPtr(body, "max_accounts_per_ja4"),
		intPtr(body, "cf_threat_score_min"),
		intPtr(body, "fast_balanced_min_lines"),
		intPtr(body, "fast_aggressive_min_lines"),
		intPtr(body, "fast_mild_min_lines"),
		strPtr(body, "min_cli_version"),
		floatPtr(body, "churn_high_usage_ratio"),
		floatPtr(body, "churn_medium_usage_ratio"),
		intPtr(body, "churn_high_idle_days"),
		intPtr(body, "churn_low_idle_days"),
		intPtr(body, "paddle_webhook_queue_warn_depth"),
	)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	// Explicit empty string clears DB override (falls back to process env).
	if v, ok := body["min_cli_version"].(string); ok && strings.TrimSpace(v) == "" {
		_, _ = h.DB.Exec(r.Context(), `
			update public.admin_product_settings set min_cli_version = null, updated_at = now(), updated_by = $1::uuid
			where id = 'default'
		`, p.UserID)
	}
	if notice, ok := body["cli_force_upgrade_notice"].(string); ok {
		_, _ = h.DB.Exec(r.Context(), `
			insert into public.site_messages (code, body) values ('CLI_FORCE_UPGRADE_NOTICE', $1)
			on conflict (code) do update set body = excluded.body
		`, notice)
		if err := subscriptions.ReloadAndBroadcastSiteMessages(r.Context(), h.DB, h.Redis); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
	}
	if ovr, err := middleware.LoadRuntimeOverridesFromDB(r.Context(), h.DB); err == nil {
		_ = middleware.PublishRuntimeOverrides(r.Context(), h.Redis, ovr)
	}
	after, _ := scanJSONSettings(r.Context(), h, "public.admin_product_settings", "default")
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "product.settings", "admin_product_settings", "default", before, after, "", step)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_PRODUCT_SETTINGS_SAVED")})
}

func intPtr(m map[string]any, k string) *int {
	v, ok := m[k]
	if !ok || v == nil {
		return nil
	}
	switch n := v.(type) {
	case float64:
		i := int(n)
		return &i
	case int:
		return &n
	case json.Number:
		i64, err := n.Int64()
		if err != nil {
			return nil
		}
		i := int(i64)
		return &i
	default:
		return nil
	}
}

func floatPtr(m map[string]any, k string) *float64 {
	v, ok := m[k]
	if !ok || v == nil {
		return nil
	}
	switch n := v.(type) {
	case float64:
		return &n
	case int:
		f := float64(n)
		return &f
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return nil
		}
		return &f
	case string:
		s := strings.TrimSpace(n)
		if s == "" {
			return nil
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil
		}
		return &f
	default:
		return nil
	}
}

func strPtr(m map[string]any, k string) *string {
	v, ok := m[k]
	if !ok || v == nil {
		return nil
	}
	s, ok := v.(string)
	if !ok {
		return nil
	}
	s = strings.TrimSpace(s)
	return &s
}

func boolPtr(m map[string]any, k string) *bool {
	v, ok := m[k]
	if !ok || v == nil {
		return nil
	}
	b, ok := v.(bool)
	if !ok {
		return nil
	}
	return &b
}

func (h *Handler) listAdminReceipts(w http.ResponseWriter, r *http.Request) {
	params, skipCap, err := h.listPage(r)
	if err != nil {
		h.writePaginateErr(w, err)
		return
	}
	db := h.readPool()
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	var total int
	if q == "" {
		_ = db.QueryRow(r.Context(), `select count(*) from public.billing_receipts`).Scan(&total)
	} else {
		_ = db.QueryRow(r.Context(), `
			select count(*) from public.billing_receipts
			where coalesce(paddle_transaction_id, '') ilike $1
			   or coalesce(bill_to_email, '') ilike $1
		`, "%"+q+"%").Scan(&total)
	}
	args := []any{params.Skip, params.Limit}
	sql := `
		select id::text, coalesce(user_id::text, ''), coalesce(paddle_transaction_id, ''),
		       coalesce(bill_to_email, ''), coalesce(status, ''),
		       coalesce(total_cents, 0), coalesce(currency_code, ''),
		       coalesce(paddle_invoice_number, ''),
		       coalesce(paid_at::text, ''), created_at::text
		from public.billing_receipts`
	if q != "" {
		sql += ` where coalesce(paddle_transaction_id, '') ilike $3
		            or coalesce(bill_to_email, '') ilike $3
		            or coalesce(paddle_invoice_number, '') ilike $3`
		args = append(args, "%"+q+"%")
	}
	sql += ` order by coalesce(paid_at, created_at) desc offset $1 limit $2`
	rows, err := db.Query(r.Context(), sql, args...)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, uid, tx, email, status, cur, inv, paidAt, created string
		var totalCents int
		if err := rows.Scan(&id, &uid, &tx, &email, &status, &totalCents, &cur, &inv, &paidAt, &created); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		displayID := strings.TrimSpace(inv)
		if displayID == "" {
			displayID = tx
		}
		item := map[string]any{
			"id":                    id,
			"user_id":               uid,
			"paddle_transaction_id": tx,
			"bill_to_email":         email,
			"status":                status,
			"status_label":          subscriptions.ReceiptStatusLabel(status),
			"total_cents":           totalCents,
			"currency_code":         cur,
			"display_id":            displayID,
			"created_at":            created,
			"paid_at":               paidAt,
		}
		if inv != "" {
			item["paddle_invoice_number"] = inv
		}
		items = append(items, item)
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"items": items, "meta": pagination.BuildMeta(params, total, skipCap)})
}

func (h *Handler) getAdminReceipt(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		h.writeErr(w, http.StatusBadRequest, "RECEIPT_ID_REQUIRED")
		return
	}
	db := h.readPool()
	seller := receipts.SellerLetterhead{
		LegalName:    h.Config.CompanyLegalName,
		SupportEmail: h.Config.CompanySupportEmail,
		LogoURL:      h.Config.CompanyLogoURL,
		AddressLine1: h.Config.CompanyAddressLine1,
		AddressLine2: h.Config.CompanyAddressLine2,
		City:         h.Config.CompanyCity,
		Region:       h.Config.CompanyRegion,
		PostalCode:   h.Config.CompanyPostalCode,
		Country:      h.Config.CompanyCountry,
		VATID:        h.Config.CompanyVATID,
		Registration: h.Config.CompanyRegistration,
	}
	d, err := receipts.LoadDetail(r.Context(), db, id, "", seller)
	if err != nil {
		code := err.Error()
		switch code {
		case "RECEIPT_NOT_FOUND":
			h.writeErr(w, http.StatusNotFound, "RECEIPT_NOT_FOUND_ERROR")
		case "RECEIPT_MONEY_LOCALE_REQUIRED":
			h.writeErr(w, http.StatusServiceUnavailable, code)
		case "RECEIPT_ID_REQUIRED":
			h.writeErr(w, http.StatusBadRequest, code)
		default:
			h.writeErr(w, http.StatusInternalServerError, "RECEIPT_OPERATION_FAILED")
		}
		return
	}
	back := strings.TrimSpace(h.adminNavHref(r.Context(), "receipts"))
	if back == "" {
		back = "/billing/receipts"
	}
	d.BackHref = back
	d.BackActionLabel = h.msg("RECEIPT_BACK")
	if strings.TrimSpace(d.BackActionLabel) == "" {
		d.BackActionLabel = h.msg("ADMIN_DIALOG_CLOSE")
	}
	d.FirstPartyPDFHref = "/api/v1/admin/billing/receipts/" + d.ID + "/pdf"
	h.writeJSON(w, http.StatusOK, d)
}

func (h *Handler) getAdminReceiptPDF(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		h.writeErr(w, http.StatusBadRequest, "RECEIPT_ID_REQUIRED")
		return
	}
	db := h.readPool()
	seller := receipts.SellerLetterhead{
		LegalName:    h.Config.CompanyLegalName,
		SupportEmail: h.Config.CompanySupportEmail,
		LogoURL:      h.Config.CompanyLogoURL,
		AddressLine1: h.Config.CompanyAddressLine1,
		AddressLine2: h.Config.CompanyAddressLine2,
		City:         h.Config.CompanyCity,
		Region:       h.Config.CompanyRegion,
		PostalCode:   h.Config.CompanyPostalCode,
		Country:      h.Config.CompanyCountry,
		VATID:        h.Config.CompanyVATID,
		Registration: h.Config.CompanyRegistration,
	}
	d, err := receipts.LoadDetail(r.Context(), db, id, "", seller)
	if err != nil {
		code := err.Error()
		switch code {
		case "RECEIPT_NOT_FOUND":
			h.writeErr(w, http.StatusNotFound, "RECEIPT_NOT_FOUND_ERROR")
		case "RECEIPT_MONEY_LOCALE_REQUIRED":
			h.writeErr(w, http.StatusServiceUnavailable, code)
		case "RECEIPT_ID_REQUIRED":
			h.writeErr(w, http.StatusBadRequest, code)
		default:
			h.writeErr(w, http.StatusInternalServerError, "RECEIPT_OPERATION_FAILED")
		}
		return
	}
	receipts.WritePDFResponse(w, d)
}

func (h *Handler) postReceiptResync(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if h.Billing == nil {
		h.writeErr(w, http.StatusServiceUnavailable, "ADMIN_PADDLE_UNAVAILABLE")
		return
	}
	var txID string
	err := h.DB.QueryRow(r.Context(), `
		select coalesce(paddle_transaction_id, '') from public.billing_receipts where id = $1::uuid
	`, id).Scan(&txID)
	if err == pgx.ErrNoRows || strings.TrimSpace(txID) == "" {
		h.writeErr(w, http.StatusNotFound, "ADMIN_RECEIPT_RESYNC_FAILED")
		return
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if err := h.Billing.ResyncTransactionByID(r.Context(), txID); err != nil {
		code := err.Error()
		if strings.HasPrefix(code, "ADMIN_") {
			h.writeErr(w, http.StatusBadGateway, code)
			return
		}
		h.writeErr(w, http.StatusBadGateway, "ADMIN_RECEIPT_RESYNC_FAILED")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "billing.receipt_resync", "billing_receipts", id, nil,
		map[string]string{"paddle_transaction_id": txID}, "", step)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_RECEIPT_RESYNC_DONE")})
}

func (h *Handler) postReceiptPDFReissue(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if h.Billing == nil {
		h.writeErr(w, http.StatusServiceUnavailable, "ADMIN_PADDLE_UNAVAILABLE")
		return
	}
	var txID string
	err := h.DB.QueryRow(r.Context(), `
		select coalesce(paddle_transaction_id, '') from public.billing_receipts where id = $1::uuid
	`, id).Scan(&txID)
	if err == pgx.ErrNoRows || strings.TrimSpace(txID) == "" {
		h.writeErr(w, http.StatusNotFound, "ADMIN_RECEIPT_PDF_FAILED")
		return
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	pdfURL, err := h.Billing.RefreshInvoicePDF(r.Context(), id, txID)
	if err != nil {
		code := err.Error()
		if strings.HasPrefix(code, "ADMIN_") {
			h.writeErr(w, http.StatusBadGateway, code)
			return
		}
		h.writeErr(w, http.StatusBadGateway, "ADMIN_RECEIPT_PDF_FAILED")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "billing.receipt_pdf_reissue", "billing_receipts", id, nil,
		map[string]string{"paddle_transaction_id": txID, "pdf_url": pdfURL}, "", step)
	h.writeJSON(w, http.StatusOK, map[string]string{
		"message":                h.msg("ADMIN_RECEIPT_PDF_DONE"),
		"paddle_invoice_pdf_url": pdfURL,
	})
}

func (h *Handler) listEmailTemplates(w http.ResponseWriter, r *http.Request) {
	params, skipCap, err := h.listPage(r)
	if err != nil {
		h.writePaginateErr(w, err)
		return
	}
	db := h.readPool()
	q := querySearchQ(r)
	where := ""
	args := []any{}
	argN := 1
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		where = fmt.Sprintf(` where (
			lower(c.code) like $%d
			or lower(c.kind) like $%d
			or lower(coalesce(m.body, '')) like $%d
		)`, argN, argN, argN)
		args = append(args, like)
		argN++
	}
	var total int
	countSQL := `
		select count(*)
		from public.admin_email_template_catalog c
		left join public.site_messages m on m.code = c.code
	` + where
	if err := db.QueryRow(r.Context(), countSQL, args...).Scan(&total); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	listSQL := fmt.Sprintf(`
		select c.code, c.kind, c.sort_order, coalesce(m.body, '')
		from public.admin_email_template_catalog c
		left join public.site_messages m on m.code = c.code
		%s
		order by c.kind, c.sort_order, c.code
		offset $%d limit $%d
	`, where, argN, argN+1)
	listArgs := append(append([]any{}, args...), params.Skip, params.Limit)
	rows, err := db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var code, kind, body string
		var sort int
		if err := rows.Scan(&code, &kind, &sort, &body); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		kindLabel := ""
		switch kind {
		case "invite":
			kindLabel = h.msg("ADMIN_EMAIL_KIND_INVITE")
		case "suspend":
			kindLabel = h.msg("ADMIN_EMAIL_KIND_SUSPEND")
		case "receipt":
			kindLabel = h.msg("ADMIN_EMAIL_KIND_RECEIPT")
		}
		items = append(items, map[string]any{
			"code": code, "kind": kind, "kind_label": kindLabel,
			"sort_order": sort, "body": body,
		})
	}
	smtpOn := h.Mailer.Enabled()
	smtpLabel := h.msg("ADMIN_SMTP_STATUS_OFF")
	if smtpOn {
		smtpLabel = h.msg("ADMIN_SMTP_STATUS_ON")
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"items":             items,
		"meta":              pagination.BuildMeta(params, total, skipCap),
		"smtp_configured":   smtpOn,
		"smtp_status_label": smtpLabel,
	})
}

func (h *Handler) patchEmailTemplate(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	code := chi.URLParam(r, "code")
	var exists bool
	_ = h.DB.QueryRow(r.Context(), `
		select exists(select 1 from public.admin_email_template_catalog where code = $1)
	`, code).Scan(&exists)
	if !exists {
		h.writeErr(w, http.StatusNotFound, "ADMIN_EMAIL_TEMPLATE_NOT_FOUND")
		return
	}
	var body struct {
		Body string `json:"body"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	text := strings.TrimSpace(body.Body)
	if text == "" {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	var before string
	_ = h.DB.QueryRow(r.Context(), `select coalesce(body, '') from public.site_messages where code = $1`, code).Scan(&before)
	_, err := h.DB.Exec(r.Context(), `
		insert into public.site_messages (code, body) values ($1, $2)
		on conflict (code) do update set body = excluded.body
	`, code, text)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if err := subscriptions.ReloadAndBroadcastSiteMessages(r.Context(), h.DB, h.Redis); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "email.template_update", "site_messages", code, before, text, "", step)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_EMAIL_TEMPLATE_SAVED")})
}

func (h *Handler) postCreditGrant(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	var body struct {
		UserID  string `json:"user_id"`
		Credits int    `json:"credits"`
		Reason  string `json:"reason"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	if strings.TrimSpace(body.UserID) == "" || body.Credits == 0 || strings.TrimSpace(body.Reason) == "" {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_REASON_REQUIRED")
		return
	}
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer tx.Rollback(r.Context())
	var id string
	err = tx.QueryRow(r.Context(), `
		insert into public.admin_credit_grants (user_id, credits, reason, created_by)
		values ($1::uuid, $2, $3, $4::uuid) returning id::text
	`, body.UserID, body.Credits, strings.TrimSpace(body.Reason), p.UserID).Scan(&id)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_USER_NOT_FOUND")
		return
	}
	_, err = tx.Exec(r.Context(), `
		update public.user_quotas
		set purchased_topup_credits = purchased_topup_credits + $2, updated_at = now()
		where user_id = $1::uuid
	`, body.UserID, body.Credits)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if h.Redis != nil {
		_ = h.Redis.Del(r.Context(), "user_quota:"+body.UserID).Err()
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "billing.credit_grant", "admin_credit_grants", id, nil, body, body.Reason, step)
	h.writeJSON(w, http.StatusCreated, map[string]string{
		"id":      id,
		"message": h.msg("ADMIN_CREDIT_GRANTED"),
	})
}

func (h *Handler) listCreditGrants(w http.ResponseWriter, r *http.Request) {
	params, skipCap, err := h.listPage(r)
	if err != nil {
		h.writePaginateErr(w, err)
		return
	}
	db := h.readPool()
	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	var total int
	var rows pgx.Rows
	if userID != "" {
		if err := db.QueryRow(r.Context(), `
			select count(*) from public.admin_credit_grants where user_id = $1::uuid
		`, userID).Scan(&total); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		rows, err = db.Query(r.Context(), `
			select id::text, user_id::text, credits, reason, coalesce(created_by::text, ''), created_at::text
			from public.admin_credit_grants
			where user_id = $1::uuid
			order by created_at desc
			offset $2 limit $3
		`, userID, params.Skip, params.Limit)
	} else {
		if err := db.QueryRow(r.Context(), `
			select count(*) from public.admin_credit_grants
		`).Scan(&total); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		rows, err = db.Query(r.Context(), `
			select id::text, user_id::text, credits, reason, coalesce(created_by::text, ''), created_at::text
			from public.admin_credit_grants
			order by created_at desc
			offset $1 limit $2
		`, params.Skip, params.Limit)
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var id, uid, reason, createdBy, created string
		var credits int
		if err := rows.Scan(&id, &uid, &credits, &reason, &createdBy, &created); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		items = append(items, map[string]any{
			"id": id, "user_id": uid, "credits": credits, "reason": reason,
			"created_by": createdBy, "created_at": created,
		})
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"meta":  pagination.BuildMeta(params, total, skipCap),
	})
}

func (h *Handler) listTopupLedger(w http.ResponseWriter, r *http.Request) {
	params, skipCap, err := h.listPage(r)
	if err != nil {
		h.writePaginateErr(w, err)
		return
	}
	db := h.readPool()
	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	var total int
	var rows pgx.Rows
	if userID != "" {
		if err := db.QueryRow(r.Context(), `
			select count(*) from public.billing_topup_ledger where user_id = $1::uuid
		`, userID).Scan(&total); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		rows, err = db.Query(r.Context(), `
			select id::text, user_id::text, paddle_transaction_id, price_id,
			       coalesce(plan_id_hint, ''), credits_granted, quantity, created_at::text
			from public.billing_topup_ledger
			where user_id = $1::uuid
			order by created_at desc
			offset $2 limit $3
		`, userID, params.Skip, params.Limit)
	} else {
		if err := db.QueryRow(r.Context(), `
			select count(*) from public.billing_topup_ledger
		`).Scan(&total); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		rows, err = db.Query(r.Context(), `
			select id::text, user_id::text, paddle_transaction_id, price_id,
			       coalesce(plan_id_hint, ''), credits_granted, quantity, created_at::text
			from public.billing_topup_ledger
			order by created_at desc
			offset $1 limit $2
		`, params.Skip, params.Limit)
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var id, uid, txn, priceID, planHint, created string
		var credits, qty int
		if err := rows.Scan(&id, &uid, &txn, &priceID, &planHint, &credits, &qty, &created); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		items = append(items, map[string]any{
			"id": id, "user_id": uid, "paddle_transaction_id": txn, "price_id": priceID,
			"plan_id_hint": planHint, "credits_granted": credits, "quantity": qty, "created_at": created,
		})
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"meta":  pagination.BuildMeta(params, total, skipCap),
	})
}

func (h *Handler) listDisputeNotes(w http.ResponseWriter, r *http.Request) {
	params, skipCap, err := h.listPage(r)
	if err != nil {
		h.writePaginateErr(w, err)
		return
	}
	db := h.readPool()
	q := querySearchQ(r)
	where := ""
	args := []any{}
	argN := 1
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		where = fmt.Sprintf(` where (
			lower(coalesce(paddle_transaction_id, '')) like $%d
			or coalesce(user_id::text, '') like $%d
			or lower(note) like $%d
			or lower(status) like $%d
			or id::text like $%d
		)`, argN, argN, argN, argN, argN)
		args = append(args, like)
		argN++
	}
	var total int
	if err := db.QueryRow(r.Context(), `
		select count(*) from public.admin_dispute_notes
	`+where, args...).Scan(&total); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	listSQL := fmt.Sprintf(`
		select id::text, coalesce(paddle_transaction_id, ''), coalesce(user_id::text, ''),
		       note, status, coalesce(created_by::text, ''), created_at::text
		from public.admin_dispute_notes
		%s
		order by created_at desc
		offset $%d limit $%d
	`, where, argN, argN+1)
	listArgs := append(append([]any{}, args...), params.Skip, params.Limit)
	rows, err := db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, tx, uid, note, status, createdBy, created string
		if err := rows.Scan(&id, &tx, &uid, &note, &status, &createdBy, &created); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		items = append(items, map[string]any{
			"id":                    id,
			"paddle_transaction_id": tx,
			"user_id":               uid,
			"note":                  note,
			"status":                status,
			"status_label":          h.disputeStatusLabel(status),
			"created_by":            createdBy,
			"created_at":            created,
		})
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"meta":  pagination.BuildMeta(params, total, skipCap),
	})
}

func (h *Handler) postDisputeNote(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	var body struct {
		PaddleTransactionID string `json:"paddle_transaction_id"`
		UserID              string `json:"user_id"`
		Note                string `json:"note"`
		Status              string `json:"status"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	if strings.TrimSpace(body.Note) == "" {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_REASON_REQUIRED")
		return
	}
	status := strings.TrimSpace(body.Status)
	if status != "open" && status != "watching" && status != "closed" {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_STATUS_INVALID")
		return
	}
	var id string
	err := h.DB.QueryRow(r.Context(), `
		insert into public.admin_dispute_notes (paddle_transaction_id, user_id, note, status, created_by)
		values (nullif($1, ''), nullif($2, '')::uuid, $3, $4, $5::uuid)
		returning id::text
	`, strings.TrimSpace(body.PaddleTransactionID), strings.TrimSpace(body.UserID),
		strings.TrimSpace(body.Note), status, p.UserID).Scan(&id)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "billing.dispute_note", "admin_dispute_notes", id, nil, body, body.Note, step)
	h.writeJSON(w, http.StatusCreated, map[string]string{
		"id":      id,
		"message": h.msg("ADMIN_DISPUTE_NOTE_SAVED"),
	})
}

func (h *Handler) postWebhookReplay(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	eventID := chi.URLParam(r, "eventId")
	var exists bool
	err := h.DB.QueryRow(r.Context(), `
		select exists(select 1 from public.paddle_webhook_events where event_id = $1)
	`, eventID).Scan(&exists)
	if err != nil || !exists {
		h.writeErr(w, http.StatusNotFound, "ADMIN_WEBHOOK_NOT_FOUND")
		return
	}
	// Idempotency table stores processed events; delete so a future delivery can re-enter.
	_, err = h.DB.Exec(r.Context(), `delete from public.paddle_webhook_events where event_id = $1`, eventID)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "webhooks.replay", "paddle_webhook_events", eventID, nil, map[string]string{"cleared": "1"}, "", step)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_WEBHOOK_REPLAY_DONE")})
}

func (h *Handler) exportAudit(w http.ResponseWriter, r *http.Request) {
	db := h.readPool()
	var exportCap int
	err := db.QueryRow(r.Context(), `
		select coalesce(audit_export_max_rows, 0)
		from public.admin_retention_settings where id = 'default'
	`).Scan(&exportCap)
	if err == pgx.ErrNoRows || exportCap <= 0 {
		h.writeErr(w, http.StatusConflict, "ADMIN_AUDIT_EXPORT_LIMIT_MISSING")
		return
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	rows, err := db.Query(r.Context(), `
		select id, coalesce(actor_user_id::text, ''), action, resource_type,
		       coalesce(resource_id, ''), coalesce(reason, ''), coalesce(ip, ''),
		       coalesce(country, ''), created_at::text
		from public.admin_audit_log
		order by id desc
		limit $1
	`, exportCap)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id int64
		var actor, action, rtype, rid, reason, ip, country, created string
		if err := rows.Scan(&id, &actor, &action, &rtype, &rid, &reason, &ip, &country, &created); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		items = append(items, map[string]any{
			"id": id, "actor_user_id": actor, "action": action, "resource_type": rtype,
			"resource_id": rid, "reason": reason, "ip": ip, "country": country, "created_at": created,
		})
	}
	generatedAt := time.Now().UTC().Format(time.RFC3339)
	filenameFmt := strings.TrimSpace(h.msg("ADMIN_AUDIT_EXPORT_FILENAME_FMT"))
	filename := ""
	if filenameFmt != "" && strings.Contains(filenameFmt, "{generated_at}") {
		safeStamp := strings.Map(func(r rune) rune {
			switch r {
			case ':', '.', '+':
				return '-'
			default:
				return r
			}
		}, generatedAt)
		filename = strings.ReplaceAll(filenameFmt, "{generated_at}", safeStamp)
	}
	h.audit(r.Context(), r, "audit.export", "admin_audit_log", "", nil, map[string]int{"rows": len(items)}, "", false)
	out := map[string]any{
		"items":        items,
		"generated_at": generatedAt,
		"message":      h.msg("ADMIN_AUDIT_EXPORT_READY"),
	}
	if filename != "" {
		out["filename"] = filename
	}
	h.writeJSON(w, http.StatusOK, out)
}

func (h *Handler) getAccessReview(w http.ResponseWriter, r *http.Request) {
	params, skipCap, err := h.listPage(r)
	if err != nil {
		h.writePaginateErr(w, err)
		return
	}
	db := h.readPool()
	q := querySearchQ(r)
	where := ""
	args := []any{}
	argN := 1
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		where = fmt.Sprintf(` where (
			lower(p.email) like $%d
			or lower(r.slug) like $%d
			or lower(r.name) like $%d
			or lower(a.status) like $%d
			or a.user_id::text like $%d
		)`, argN, argN, argN, argN, argN)
		args = append(args, like)
		argN++
	}
	var total int
	countSQL := `
		select count(*) from public.platform_admins a
		join public.profiles p on p.id = a.user_id
		join public.platform_roles r on r.id = a.role_id
	` + where
	if err := db.QueryRow(r.Context(), countSQL, args...).Scan(&total); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	listSQL := fmt.Sprintf(`
		select a.user_id::text, p.email, r.slug, r.name, a.status, a.created_at::text
		from public.platform_admins a
		join public.profiles p on p.id = a.user_id
		join public.platform_roles r on r.id = a.role_id
		%s
		order by r.is_owner desc, p.email
		offset $%d limit $%d
	`, where, argN, argN+1)
	listArgs := append(append([]any{}, args...), params.Skip, params.Limit)
	rows, err := db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var uid, email, slug, name, status, created string
		if err := rows.Scan(&uid, &email, &slug, &name, &status, &created); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		items = append(items, map[string]any{
			"user_id": uid, "email": email, "role_slug": slug, "role_name": name,
			"status": status, "status_label": h.platformAdminStatusLabel(status),
			"created_at": created,
		})
	}
	var attestLimit int
	err = db.QueryRow(r.Context(), `
		select coalesce(access_review_attestations_limit, 0)
		from public.admin_retention_settings where id = 'default'
	`).Scan(&attestLimit)
	if err == pgx.ErrNoRows || attestLimit <= 0 {
		h.writeErr(w, http.StatusConflict, "ADMIN_ACCESS_REVIEW_ATTEST_LIMIT_MISSING")
		return
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	attestRows, err := db.Query(r.Context(), `
		select id::text, attested_by::text, period_label, notes, created_at::text
		from public.admin_access_review_attestations
		order by created_at desc
		limit $1
	`, attestLimit)
	attestations := []map[string]any{}
	if err == nil {
		defer attestRows.Close()
		for attestRows.Next() {
			var id, by, period, notes, created string
			if attestRows.Scan(&id, &by, &period, &notes, &created) == nil {
				attestations = append(attestations, map[string]any{
					"id": id, "attested_by": by, "period_label": period,
					"notes": notes, "created_at": created,
				})
			}
		}
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"items":        items,
		"meta":         pagination.BuildMeta(params, total, skipCap),
		"attestations": attestations,
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handler) exportAccessReview(w http.ResponseWriter, r *http.Request) {
	items, err := h.loadAccessReviewRoster(r.Context())
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	generatedAt := time.Now().UTC().Format(time.RFC3339)
	filenameFmt := strings.TrimSpace(h.msg("ADMIN_ACCESS_REVIEW_FILENAME_FMT"))
	filename := ""
	if filenameFmt != "" && strings.Contains(filenameFmt, "{generated_at}") {
		safeStamp := strings.Map(func(r rune) rune {
			switch r {
			case ':', '.', '+':
				return '-'
			default:
				return r
			}
		}, generatedAt)
		filename = strings.ReplaceAll(filenameFmt, "{generated_at}", safeStamp)
	}
	h.audit(r.Context(), r, "compliance.access_review_export", "platform_admins", "", nil,
		map[string]int{"rows": len(items)}, "", false)
	out := map[string]any{
		"items":        items,
		"generated_at": generatedAt,
		"message":      h.msg("ADMIN_ACCESS_REVIEW_EXPORT_READY"),
	}
	if filename != "" {
		out["filename"] = filename
	}
	h.writeJSON(w, http.StatusOK, out)
}

func (h *Handler) postAccessReviewAttest(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	var body struct {
		PeriodLabel string `json:"period_label"`
		Notes       string `json:"notes"`
		Reason      string `json:"reason"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	period := strings.TrimSpace(body.PeriodLabel)
	reason := strings.TrimSpace(body.Reason)
	if period == "" || reason == "" {
		h.writeErr(w, http.StatusBadRequest, "ADMIN_REASON_REQUIRED")
		return
	}
	items, err := h.loadAccessReviewRoster(r.Context())
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	snap, err := json.Marshal(items)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	var id string
	err = h.DB.QueryRow(r.Context(), `
		insert into public.admin_access_review_attestations (attested_by, period_label, notes, roster_snapshot)
		values ($1::uuid, $2, $3, $4::jsonb)
		returning id::text
	`, p.UserID, period, strings.TrimSpace(body.Notes), string(snap)).Scan(&id)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "compliance.access_review_attest", "admin_access_review_attestations", id, nil, body, reason, step)
	h.writeJSON(w, http.StatusCreated, map[string]any{
		"id": id, "message": h.msg("ADMIN_ACCESS_REVIEW_DONE"),
	})
}

func (h *Handler) loadAccessReviewRoster(ctx context.Context) ([]map[string]any, error) {
	rows, err := h.readPool().Query(ctx, `
		select a.user_id::text, p.email, r.slug, r.name, a.status, a.created_at::text
		from public.platform_admins a
		join public.profiles p on p.id = a.user_id
		join public.platform_roles r on r.id = a.role_id
		order by r.is_owner desc, p.email
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var uid, email, slug, name, status, created string
		if err := rows.Scan(&uid, &email, &slug, &name, &status, &created); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"user_id": uid, "email": email, "role_slug": slug, "role_name": name,
			"status": status, "status_label": h.platformAdminStatusLabel(status),
			"created_at": created,
		})
	}
	return items, nil
}

func (h *Handler) getCountryHeatmap(w http.ResponseWriter, r *http.Request) {
	db := h.readPool()
	topN, err := billingsettings.ChartTopN(r.Context(), db)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	rows, err := db.Query(r.Context(), `
		with logins as (
		  select coalesce(nullif(btrim(last_login_country), ''), 'ZZ') as country,
		         count(*)::bigint as login_users
		  from public.profiles
		  group by 1
		),
		paid as (
		  select coalesce(nullif(btrim(p.last_login_country), ''), 'ZZ') as country,
		         count(distinct s.user_id)::bigint as paid_users
		  from public.subscriptions s
		  join public.profiles p on p.id = s.user_id
		  where s.status in ('active', 'trialing')
		  group by 1
		)
		select coalesce(l.country, p.country) as country,
		       coalesce(l.login_users, 0) as login_users,
		       coalesce(p.paid_users, 0) as paid_users
		from logins l
		full outer join paid p on p.country = l.country
		order by (coalesce(l.login_users, 0) + coalesce(p.paid_users, 0)) desc,
		         coalesce(l.country, p.country) asc
		limit $1
	`, topN)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var c string
		var logins, paid int64
		if err := rows.Scan(&c, &logins, &paid); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		items = append(items, map[string]any{
			"country":     c,
			"login_users": logins,
			"paid_users":  paid,
		})
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"title": h.msg("ADMIN_OBS_HEATMAP_PAID_TITLE"),
	})
}

func (h *Handler) getOAuthScopeChrome(w http.ResponseWriter, r *http.Request) {
	rows, err := h.readPool().Query(r.Context(), `
		select code, body from public.site_messages
		where code ~ '^AUTH_[A-Z0-9]+_OAUTH_'
		  and body is not null
		  and length(trim(body)) > 0
		order by code
	`)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var code, body string
		if err := rows.Scan(&code, &body); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		out[code] = body
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"scopes": out})
}

func (h *Handler) syncGitHubDistribution(ctx context.Context) error {
	token := strings.TrimSpace(h.Config.GitHubToken)
	repo := strings.TrimSpace(h.Config.GitHubRepo)
	if token == "" || repo == "" {
		return fmt.Errorf("ADMIN_GITHUB_NOT_CONFIGURED")
	}
	if err := h.syncGitHubRepoTraffic(ctx, "github", repo); err != nil {
		return err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	// Release download aggregates (latest 30 releases assets).
	req2, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.github.com/repos/"+repo+"/releases?per_page=30", nil)
	if err != nil {
		return err
	}
	req2.Header.Set("Authorization", "Bearer "+token)
	req2.Header.Set("Accept", "application/vnd.github+json")
	res2, err := client.Do(req2)
	if err != nil {
		return err
	}
	defer res2.Body.Close()
	raw2, _ := io.ReadAll(res2.Body)
	if res2.StatusCode >= 300 {
		return fmt.Errorf("ADMIN_DISTRIBUTION_SYNC_FAILED")
	}
	var releases []struct {
		PublishedAt string `json:"published_at"`
		Assets      []struct {
			DownloadCount int64 `json:"download_count"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(raw2, &releases); err != nil {
		return err
	}
	for _, rel := range releases {
		day := strings.TrimSpace(rel.PublishedAt)
		if len(day) >= 10 {
			day = day[:10]
		}
		var sum int64
		for _, a := range rel.Assets {
			sum += a.DownloadCount
		}
		if day == "" {
			continue
		}
		_, _ = h.DB.Exec(ctx, `
			insert into public.distribution_daily_stats (day, source, metric, country, path, value)
			values ($1::date, 'github', 'release_downloads', '', '', $2)
			on conflict (day, source, metric, country, path) do update set value = excluded.value
		`, day, sum)
	}
	_, _ = h.DB.Exec(ctx, `
		insert into public.distribution_sync_state (source, last_synced_at, last_error, meta)
		values ('github', now(), null, jsonb_build_object('repo', $1::text))
		on conflict (source) do update set last_synced_at = now(), last_error = null, meta = excluded.meta
	`, repo)
	return nil
}

func (h *Handler) listLegalSections(w http.ResponseWriter, r *http.Request) {
	params, skipCap, err := h.listPage(r)
	if err != nil {
		h.writePaginateErr(w, err)
		return
	}
	db := h.readPool()
	q := querySearchQ(r)
	where := ""
	args := []any{}
	argN := 1
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		where = fmt.Sprintf(` where (
			lower(doc_kind) like $%d
			or lower(coalesce(heading, '')) like $%d
			or lower(coalesce(body, '')) like $%d
			or id::text like $%d
		)`, argN, argN, argN, argN)
		args = append(args, like)
		argN++
	}
	var total int
	if err := db.QueryRow(r.Context(), `select count(*) from public.site_legal_sections`+where, args...).Scan(&total); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	listSQL := fmt.Sprintf(`
		select id::text, doc_kind, sort_order, coalesce(heading, ''), coalesce(body, ''),
		       updated_at::text, coalesce(published_at::text, '')
		from public.site_legal_sections
		%s
		order by doc_kind, sort_order
		offset $%d limit $%d
	`, where, argN, argN+1)
	listArgs := append(append([]any{}, args...), params.Skip, params.Limit)
	rows, err := db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, docKind, heading, body, updated, published string
		var sort int
		if err := rows.Scan(&id, &docKind, &sort, &heading, &body, &updated, &published); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		pubLabel := h.msg("ADMIN_LEGAL_DRAFT")
		if strings.TrimSpace(published) != "" {
			pubLabel = h.msg("ADMIN_LEGAL_PUBLISHED")
		}
		row := map[string]any{
			"id": id, "page": docKind, "doc_kind": docKind, "sort_order": sort,
			"heading": heading, "body": body, "updated_at": updated,
			"published_at": published, "publish_status_label": pubLabel,
		}
		items = append(items, row)
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"meta":  pagination.BuildMeta(params, total, skipCap),
	})
}

func (h *Handler) patchLegalSection(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	id := chi.URLParam(r, "id")
	var body struct {
		Heading *string `json:"heading"`
		Body    *string `json:"body"`
		Publish *bool   `json:"publish"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	if body.Heading == nil && body.Body == nil && body.Publish == nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	tag, err := h.DB.Exec(r.Context(), `
		update public.site_legal_sections set
		  heading = coalesce($2, heading),
		  body = coalesce($3, body),
		  updated_at = now()
		where id = $1::uuid
	`, id, body.Heading, body.Body)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if tag.RowsAffected() == 0 {
		h.writeErr(w, http.StatusNotFound, "ADMIN_ROLE_NOT_FOUND")
		return
	}
	if body.Publish != nil {
		if *body.Publish {
			_, err = h.DB.Exec(r.Context(), `
				update public.site_legal_sections set published_at = now(), updated_at = now()
				where id = $1::uuid
			`, id)
		} else {
			_, err = h.DB.Exec(r.Context(), `
				update public.site_legal_sections set published_at = null, updated_at = now()
				where id = $1::uuid
			`, id)
		}
		if err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "chrome.legal", "site_legal_sections", id, nil, body, "", step)
	h.writeJSON(w, http.StatusOK, map[string]string{"message": h.msg("ADMIN_LEGAL_SECTION_SAVED")})
}
