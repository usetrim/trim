package platformadmin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/usetrim/trim/server/internal/billing/catalogsync"
	"github.com/usetrim/trim/server/internal/pagination"
	"github.com/usetrim/trim/server/internal/subscriptions"
)

var planIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)

func (h *Handler) listPlans(w http.ResponseWriter, r *http.Request) {
	db := h.readPool()
	params, skipCap, err := h.listPage(r)
	if err != nil {
		h.writePaginateErr(w, err)
		return
	}
	q := querySearchQ(r)
	where := ""
	args := []any{}
	argN := 1
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		where = fmt.Sprintf(` where (
			lower(id) like $%d
			or lower(display_name) like $%d
			or lower(coalesce(description, '')) like $%d
			or lower(plan_kind) like $%d
			or lower(currency_code) like $%d
		)`, argN, argN, argN, argN, argN)
		args = append(args, like)
		argN++
	}
	var total int
	if err := db.QueryRow(r.Context(), `select count(*) from public.plan_catalog`+where, args...).Scan(&total); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	listSQL := fmt.Sprintf(`
		select id, display_name, description, plan_kind, plan_rank, currency_code,
		       price_monthly_cents, price_yearly_cents, credits_monthly, per_seat, unlimited, popular,
		       is_public, is_active, sort_order, updated_at::text,
		       coalesce(paddle_product_id, ''), coalesce(paddle_price_id_monthly, ''),
		       coalesce(paddle_price_id_yearly, ''), coalesce(paddle_price_id_topup, ''),
		       coalesce(features::text, '[]')
		from public.plan_catalog
		%s
		order by sort_order asc
		offset $%d limit $%d
	`, where, argN, argN+1)
	listArgs := append(append([]any{}, args...), params.Skip, params.Limit)
	rows, err := db.Query(r.Context(), listSQL, listArgs...)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	for rows.Next() {
		var id, displayName, desc, kind, currency, updated string
		var productID, priMonthly, priYearly, priTopup, featuresJSON string
		var rank, credits, sort int
		var monthly, yearly *int
		var perSeat, unlimited, popular, isPublic, isActive bool
		if err := rows.Scan(&id, &displayName, &desc, &kind, &rank, &currency, &monthly, &yearly, &credits, &perSeat, &unlimited, &popular, &isPublic, &isActive, &sort, &updated, &productID, &priMonthly, &priYearly, &priTopup, &featuresJSON); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		var features any
		if err := json.Unmarshal([]byte(featuresJSON), &features); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		item := map[string]any{
			"id": id, "display_name": displayName, "description": desc,
			"plan_kind": kind, "plan_rank": rank, "currency_code": currency,
			"price_monthly_cents": monthly, "price_yearly_cents": yearly,
			"credits_monthly": credits, "per_seat": perSeat, "unlimited": unlimited, "popular": popular,
			"is_public": isPublic, "is_active": isActive, "sort_order": sort,
			"updated_at": updated, "features": features,
		}
		if productID != "" {
			item["paddle_product_id"] = productID
		}
		if priMonthly != "" {
			item["paddle_price_id_monthly"] = priMonthly
		}
		if priYearly != "" {
			item["paddle_price_id_yearly"] = priYearly
		}
		if priTopup != "" {
			item["paddle_price_id_topup"] = priTopup
		}
		items = append(items, item)
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"meta":  pagination.BuildMeta(params, total, skipCap),
	})
}

type patchPlanBody struct {
	DisplayName       *string          `json:"display_name"`
	Description       *string          `json:"description"`
	PlanKind          *string          `json:"plan_kind"`
	PlanRank          *int             `json:"plan_rank"`
	PriceMonthlyCents *int             `json:"price_monthly_cents"`
	PriceYearlyCents  *int             `json:"price_yearly_cents"`
	CreditsMonthly    *int             `json:"credits_monthly"`
	PerSeat           *bool            `json:"per_seat"`
	Unlimited         *bool            `json:"unlimited"`
	Popular           *bool            `json:"popular"`
	IsPublic          *bool            `json:"is_public"`
	IsActive          *bool            `json:"is_active"`
	SortOrder         *int             `json:"sort_order"`
	Features          *json.RawMessage `json:"features"`
	// SyncToPaddle when true (default) pushes admin amounts to Paddle and stores pri_*/pro_*.
	SyncToPaddle *bool `json:"sync_to_paddle"`
}

func (h *Handler) patchPlan(w http.ResponseWriter, r *http.Request) {
	planID := chi.URLParam(r, "id")
	p := PrincipalFromContext(r.Context())
	var body patchPlanBody
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	doSync := body.SyncToPaddle == nil || *body.SyncToPaddle
	// Fail closed before DB write when sync is requested and catalog sync cannot succeed
	// (invalid currency / placeholder key). Soft-skip only when Paddle client is absent.
	if doSync && h.Catalog != nil {
		if err := h.Catalog.PreflightSync(r.Context()); err != nil {
			if !errors.Is(err, catalogsync.ErrPaddleUnavailable) {
				h.writeCatalogSyncErr(w, err)
				return
			}
		}
	}
	sets := []string{"updated_at = now()"}
	args := []any{planID}
	n := 2
	add := func(col string, val any) {
		sets = append(sets, fmt.Sprintf("%s = $%d", col, n))
		args = append(args, val)
		n++
	}
	if body.DisplayName != nil {
		add("display_name", strings.TrimSpace(*body.DisplayName))
	}
	if body.Description != nil {
		add("description", strings.TrimSpace(*body.Description))
	}
	if body.PlanKind != nil {
		kind := strings.TrimSpace(strings.ToLower(*body.PlanKind))
		switch kind {
		case "subscription", "topup", "enterprise":
			add("plan_kind", kind)
		default:
			h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
			return
		}
	}
	if body.PlanRank != nil {
		add("plan_rank", *body.PlanRank)
	}
	if body.PriceMonthlyCents != nil {
		if *body.PriceMonthlyCents < 0 {
			h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
			return
		}
		add("price_monthly_cents", *body.PriceMonthlyCents)
	}
	if body.PriceYearlyCents != nil {
		if *body.PriceYearlyCents < 0 {
			h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
			return
		}
		add("price_yearly_cents", *body.PriceYearlyCents)
	}
	if body.CreditsMonthly != nil {
		if *body.CreditsMonthly < 0 {
			h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
			return
		}
		add("credits_monthly", *body.CreditsMonthly)
	}
	if body.PerSeat != nil {
		add("per_seat", *body.PerSeat)
	}
	if body.Unlimited != nil {
		add("unlimited", *body.Unlimited)
	}
	if body.Popular != nil {
		add("popular", *body.Popular)
	}
	if body.IsPublic != nil {
		add("is_public", *body.IsPublic)
	}
	if body.IsActive != nil {
		add("is_active", *body.IsActive)
	}
	if body.SortOrder != nil {
		add("sort_order", *body.SortOrder)
	}
	if body.Features != nil {
		raw := strings.TrimSpace(string(*body.Features))
		if raw == "" {
			h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
			return
		}
		var probe any
		if err := json.Unmarshal([]byte(raw), &probe); err != nil {
			h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
			return
		}
		sets = append(sets, fmt.Sprintf("features = $%d::jsonb", n))
		args = append(args, raw)
		n++
	}
	if len(sets) == 1 {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	sql := fmt.Sprintf(`update public.plan_catalog set %s where id = $1`, strings.Join(sets, ", "))
	tag, err := h.DB.Exec(r.Context(), sql, args...)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if tag.RowsAffected() == 0 {
		h.writeErr(w, http.StatusNotFound, "PLAN_NOT_FOUND")
		return
	}
	// Metering flags (unlimited / credits) live in Redis quota hashes; drop them so the next
	// request reloads plan_catalog.unlimited and does not keep a stale growth-dial value.
	if body.Unlimited != nil || body.CreditsMonthly != nil {
		h.invalidateQuotaCachesForPlan(r.Context(), planID)
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "billing.plan_update", "plan_catalog", planID, nil, body, "", step)

	var syncResult any
	if doSync && h.Catalog != nil {
		res, serr := h.Catalog.SyncPlan(r.Context(), planID)
		if serr != nil {
			// Plan row is already saved. Soft-skip when Paddle is not configured (non-cloud).
			if errors.Is(serr, catalogsync.ErrPaddleUnavailable) {
				syncResult = map[string]any{"skipped": true, "reason": "paddle_unavailable"}
			} else {
				// Recompute bound even on sync failure so stale pricing_bound=true cannot linger.
				if _, berr := h.Catalog.RefreshPricingBound(r.Context()); berr != nil {
					log.Printf("admin pricing_bound refresh after sync fail: %v", berr)
				}
				h.writeCatalogSyncErr(w, serr)
				return
			}
		} else {
			syncResult = res
			h.audit(r.Context(), r, "billing.plan_paddle_sync", "plan_catalog", planID, nil, res, "", step)
		}
		if _, berr := h.Catalog.RefreshPricingBound(r.Context()); berr != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"sync":    syncResult,
		"message": h.msg("ADMIN_PLAN_SAVED"),
	})
}

func (h *Handler) postSyncPaddleCatalog(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	if h.Catalog == nil {
		h.writeErr(w, http.StatusServiceUnavailable, "ADMIN_PADDLE_UNAVAILABLE")
		return
	}
	res, err := h.Catalog.SyncCatalog(r.Context())
	if err != nil {
		// Partial sync may have written some pri_*; recompute bound so stale true cannot linger.
		if _, berr := h.Catalog.RefreshPricingBound(r.Context()); berr != nil {
			log.Printf("admin pricing_bound refresh after catalog sync fail: %v", berr)
		}
		h.writeCatalogSyncErr(w, err)
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "billing.catalog_paddle_sync", "plan_catalog", "all", nil, res, "", step)
	h.writeJSON(w, http.StatusOK, struct {
		catalogsync.CatalogResult
		Message string `json:"message"`
	}{CatalogResult: res, Message: h.msg("ADMIN_PADDLE_CATALOG_SYNCED")})
}

func (h *Handler) writeCatalogSyncErr(w http.ResponseWriter, err error) {
	log.Printf("admin paddle catalog sync: %v", err)
	detail := sanitizeCatalogSyncDetail(err)
	switch {
	case errors.Is(err, catalogsync.ErrPaddleUnavailable):
		h.writeErr(w, http.StatusServiceUnavailable, "ADMIN_PADDLE_UNAVAILABLE")
	case errors.Is(err, catalogsync.ErrPaddleKeyInvalid):
		h.writeErrDetail(w, http.StatusBadGateway, "ADMIN_PADDLE_CATALOG_SYNC_FAILED", detail)
	case errors.Is(err, catalogsync.ErrCurrencyInvalid):
		h.writeErr(w, http.StatusBadRequest, "ADMIN_PADDLE_CURRENCY_INVALID")
	case errors.Is(err, catalogsync.ErrPlanNotFound):
		h.writeErr(w, http.StatusNotFound, "PLAN_NOT_FOUND")
	default:
		h.writeErrDetail(w, http.StatusBadGateway, "ADMIN_PADDLE_CATALOG_SYNC_FAILED", detail)
	}
}

// writeErrDetail is writeErr plus a redacted operator-facing detail (Paddle API message).
func (h *Handler) writeErrDetail(w http.ResponseWriter, status int, code, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	out := map[string]string{"error": h.msg(code), "code": code}
	if detail = strings.TrimSpace(detail); detail != "" {
		out["detail"] = detail
	}
	_ = json.NewEncoder(w).Encode(out)
}

// sanitizeCatalogSyncDetail keeps the Paddle/plan failure reason for operators
// without echoing API keys or other secrets that might appear in wrapped errors.
func sanitizeCatalogSyncDetail(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return ""
	}
	lower := strings.ToLower(msg)
	// Strip anything that looks like a Paddle secret prefix.
	for _, needle := range []string{"pdl_", "bearer "} {
		for {
			i := strings.Index(lower, needle)
			if i < 0 {
				break
			}
			end := i + len(needle)
			for end < len(msg) && msg[end] != ' ' && msg[end] != ',' && msg[end] != ';' && msg[end] != '"' {
				end++
			}
			msg = strings.TrimSpace(msg[:i] + "[redacted]" + msg[end:])
			lower = strings.ToLower(msg)
		}
	}
	if len(msg) > 400 {
		msg = msg[:400] + "…"
	}
	return msg
}

type createPlanBody struct {
	ID                string          `json:"id"`
	DisplayName       string          `json:"display_name"`
	Description       string          `json:"description"`
	PlanKind          string          `json:"plan_kind"`
	PlanRank          int             `json:"plan_rank"`
	PriceMonthlyCents *int            `json:"price_monthly_cents"`
	PriceYearlyCents  *int            `json:"price_yearly_cents"`
	CreditsMonthly    int             `json:"credits_monthly"`
	PerSeat           bool            `json:"per_seat"`
	Unlimited         bool            `json:"unlimited"`
	Popular           bool            `json:"popular"`
	IsPublic          bool            `json:"is_public"`
	IsActive          bool            `json:"is_active"`
	SortOrder         int             `json:"sort_order"`
	Features          json.RawMessage `json:"features"`
	SyncToPaddle      *bool           `json:"sync_to_paddle"`
}

func (h *Handler) postPlan(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	var body createPlanBody
	if err := decodeJSON(r, &body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	id := strings.TrimSpace(strings.ToLower(body.ID))
	name := strings.TrimSpace(body.DisplayName)
	kind := strings.TrimSpace(strings.ToLower(body.PlanKind))
	if !planIDPattern.MatchString(id) || name == "" {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	switch kind {
	case "subscription", "topup", "enterprise":
	default:
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	if body.CreditsMonthly < 0 ||
		(body.PriceMonthlyCents != nil && *body.PriceMonthlyCents < 0) ||
		(body.PriceYearlyCents != nil && *body.PriceYearlyCents < 0) {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	doSync := body.SyncToPaddle == nil || *body.SyncToPaddle
	if doSync && h.Catalog != nil {
		if err := h.Catalog.PreflightSync(r.Context()); err != nil {
			if !errors.Is(err, catalogsync.ErrPaddleUnavailable) {
				h.writeCatalogSyncErr(w, err)
				return
			}
		}
	}
	features := strings.TrimSpace(string(body.Features))
	if features == "" {
		features = "[]"
	}
	var probe any
	if err := json.Unmarshal([]byte(features), &probe); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	desc := strings.TrimSpace(body.Description)
	tag, err := h.DB.Exec(r.Context(), `
		insert into public.plan_catalog (
			id, display_name, description, plan_kind, plan_rank,
			price_monthly_cents, price_yearly_cents, credits_monthly, per_seat, unlimited, popular,
			is_public, is_active, sort_order, features, currency_code, updated_at
		)
		select $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15::jsonb,
		       coalesce((select default_currency from public.billing_settings where id = 'default'), 'USD'),
		       now()
	`, id, name, desc, kind, body.PlanRank,
		body.PriceMonthlyCents, body.PriceYearlyCents, body.CreditsMonthly, body.PerSeat, body.Unlimited, body.Popular,
		body.IsPublic, body.IsActive, body.SortOrder, features)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(err.Error(), "unique") {
			h.writeErr(w, http.StatusConflict, "PLAN_CONFLICT")
			return
		}
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	if tag.RowsAffected() == 0 {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "billing.plan_create", "plan_catalog", id, nil, body, "", step)

	var syncResult any
	if doSync && h.Catalog != nil {
		res, serr := h.Catalog.SyncPlan(r.Context(), id)
		if serr != nil {
			if errors.Is(serr, catalogsync.ErrPaddleUnavailable) {
				syncResult = map[string]any{"skipped": true, "reason": "paddle_unavailable"}
			} else {
				if _, berr := h.Catalog.RefreshPricingBound(r.Context()); berr != nil {
					log.Printf("admin pricing_bound refresh after create sync fail: %v", berr)
				}
				h.writeCatalogSyncErr(w, serr)
				return
			}
		} else {
			syncResult = res
			h.audit(r.Context(), r, "billing.plan_paddle_sync", "plan_catalog", id, nil, res, "", step)
		}
		if _, berr := h.Catalog.RefreshPricingBound(r.Context()); berr != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
	}
	h.writeJSON(w, http.StatusCreated, map[string]any{
		"id":      id,
		"sync":    syncResult,
		"message": h.msg("ADMIN_PLAN_CREATED"),
	})
}

func (h *Handler) getBillingSettings(w http.ResponseWriter, r *http.Request) {
	row, err := scanJSONSettings(r.Context(), h, "public.billing_settings", "default")
	if err == pgx.ErrNoRows {
		h.writeErr(w, http.StatusNotFound, "BILLING_SETTINGS_UNAVAILABLE")
		return
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	h.writeJSON(w, http.StatusOK, row)
}

func (h *Handler) patchBillingSettings(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	var patch map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	allowed := map[string]bool{
		"annual_discount_percent": true, "default_currency": true, "default_page_size": true,
		"max_page_size": true, "pagination_skip_to_max_pages": true, "chart_top_n": true,
		"chart_series_days": true, "date_range_months": true, "default_plan_interval": true,
		// pricing_bound is computed only by catalog sync / RefreshPricingBound - never operator-writable.
		"allow_cancel_at_period_end": true, "default_seat_quantity": true,
		"min_seat_quantity": true, "default_deep_target_token": true,
		"deep_target_token_min": true, "deep_target_token_max": true,
		"live_deep_min_input_tokens": true, "live_deep_oom_policy": true,
		"live_deep_warmup_on_start": true, "live_deep_skip_on_stream": true,
		"deep_v1_model": true, "deep_v2_model": true, "deep_long_model": true,
		"deep_v2_force_tokens": true,
		"allow_downgrades":     true, "upgrade_proration_mode": true,
		"allow_monthly_to_annual_as_upgrade": true,
		"apply_paddle_discount_on_annual":    true,
		"chart_cache_ttl_sec":                true,
	}
	sets := []string{"updated_at = now()"}
	args := []any{}
	n := 1
	for k, raw := range patch {
		if k == "id" || k == "pricing_bound" || !allowed[k] {
			continue
		}
		var val any
		if err := json.Unmarshal(raw, &val); err != nil {
			h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
			return
		}
		if k == "chart_cache_ttl_sec" {
			nVal, ok := asPositiveInt(val)
			if !ok || nVal > 86400 {
				h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
				return
			}
			val = nVal
		}
		if k == "default_currency" {
			cur, ok := val.(string)
			if !ok {
				h.writeErr(w, http.StatusBadRequest, "ADMIN_PADDLE_CURRENCY_INVALID")
				return
			}
			cur = strings.ToUpper(strings.TrimSpace(cur))
			if !catalogsync.ValidBillingCurrency(cur) {
				h.writeErr(w, http.StatusBadRequest, "ADMIN_PADDLE_CURRENCY_INVALID")
				return
			}
			val = cur
		}
		if k == "annual_discount_percent" {
			pct, ok := asNonNegInt(val)
			if !ok || pct > 90 {
				h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
				return
			}
			val = pct
		}
		// Self-serve annual charge uses yearly pri_* only. Never allow enabling a Paddle %
		// discount overlay (would double-discount vs baked price_yearly_cents).
		if k == "apply_paddle_discount_on_annual" {
			val = false
		}
		sets = append(sets, fmt.Sprintf("%s = $%d", k, n))
		args = append(args, val)
		n++
	}
	if len(sets) == 1 {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	args = append(args, "default")
	sql := fmt.Sprintf(`update public.billing_settings set %s where id = $%d`, strings.Join(sets, ", "), n)
	if _, err := h.DB.Exec(r.Context(), sql, args...); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	step := h.stepUpUsed(r, p.UserID)
	h.audit(r.Context(), r, "billing.settings_update", "billing_settings", "default", nil, patch, "", step)

	// When admin changes annual_discount_percent, bake it into price_yearly_cents then sync Paddle.
	// Web savings badges come from monthly vs yearly cents; yearly pri_* must match.
	if raw, ok := patch["annual_discount_percent"]; ok && h.Catalog != nil {
		var pct float64
		if err := json.Unmarshal(raw, &pct); err == nil {
			if _, err := h.Catalog.ApplyAnnualDiscountToYearlyCents(r.Context(), int(pct)); err != nil {
				h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
				return
			}
		}
	}

	needCatalogSync := false
	if _, ok := patch["default_currency"]; ok {
		needCatalogSync = true
	}
	if _, ok := patch["annual_discount_percent"]; ok {
		needCatalogSync = true
	}
	if needCatalogSync && h.Catalog != nil && h.Paddle != nil {
		if _, err := h.Catalog.SyncCatalog(r.Context()); err != nil {
			// Sync failed (e.g. currency flip to invalid mid-flight, or Paddle write error).
			// Always recompute bound so a previous true cannot stay fail-open.
			if _, berr := h.Catalog.RefreshPricingBound(r.Context()); berr != nil {
				log.Printf("admin pricing_bound refresh after settings sync fail: %v", berr)
			}
			h.writeCatalogSyncErr(w, err)
			return
		}
		h.audit(r.Context(), r, "billing.catalog_paddle_sync", "billing_settings", "default", nil, map[string]any{"trigger": "settings"}, "", step)
	} else if h.Catalog != nil {
		if _, err := h.Catalog.RefreshPricingBound(r.Context()); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
	}
	row, err := scanJSONSettings(r.Context(), h, "public.billing_settings", "default")
	if err == pgx.ErrNoRows {
		h.writeErr(w, http.StatusNotFound, "BILLING_SETTINGS_UNAVAILABLE")
		return
	}
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	row["message"] = h.msg("ADMIN_BILLING_SETTINGS_SAVED")
	h.writeJSON(w, http.StatusOK, row)
}

func (h *Handler) listSubscriptions(w http.ResponseWriter, r *http.Request) {
	db := h.readPool()
	params, skipCap, err := h.listPage(r)
	if err != nil {
		h.writePaginateErr(w, err)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	plan := strings.TrimSpace(r.URL.Query().Get("plan"))

	where := `where 1=1`
	args := []any{}
	n := 1
	if q != "" {
		where += fmt.Sprintf(` and p.email ilike $%d`, n)
		args = append(args, "%"+q+"%")
		n++
	}
	if status != "" {
		where += fmt.Sprintf(` and s.status = $%d`, n)
		args = append(args, status)
		n++
	}
	if plan != "" {
		where += fmt.Sprintf(` and s.plan_tier = $%d`, n)
		args = append(args, plan)
		n++
	}

	countSQL := `
		select count(*)
		from public.subscriptions s
		join public.profiles p on p.id = s.user_id
	` + where
	var total int
	if err := db.QueryRow(r.Context(), countSQL, args...).Scan(&total); err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	listSQL := fmt.Sprintf(`
		select s.id::text, s.user_id::text, p.email, s.plan_tier, s.status,
		       s.billing_interval, s.current_period_end::text, s.created_at::text,
		       coalesce(s.paddle_subscription_id, ''), coalesce(s.expires_at::text, ''),
		       coalesce((
		         select e.processed_at::text
		         from public.paddle_webhook_events e
		         where s.paddle_subscription_id <> ''
		           and e.payload is not null
		           and (
		             e.payload #>> '{data,id}' = s.paddle_subscription_id
		             or e.payload #>> '{data,subscription_id}' = s.paddle_subscription_id
		             or e.payload #>> '{data,subscription,id}' = s.paddle_subscription_id
		           )
		         order by e.processed_at desc
		         limit 1
		       ), '')
		from public.subscriptions s
		join public.profiles p on p.id = s.user_id
		%s
		order by s.created_at desc
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
		ID                   string `json:"id"`
		UserID               string `json:"user_id"`
		Email                string `json:"email"`
		PlanTier             string `json:"plan_tier"`
		Status               string `json:"status"`
		StatusLabel          string `json:"status_label"`
		BillingInterval      string `json:"billing_interval"`
		CurrentPeriodEnd     string `json:"current_period_end"`
		CreatedAt            string `json:"created_at"`
		PaddleSubscriptionID string `json:"paddle_subscription_id"`
		ExpiresAt            string `json:"expires_at"`
		LastWebhookAt        string `json:"last_webhook_at"`
	}
	items := make([]row, 0)
	for rows.Next() {
		var it row
		if err := rows.Scan(&it.ID, &it.UserID, &it.Email, &it.PlanTier, &it.Status, &it.BillingInterval, &it.CurrentPeriodEnd, &it.CreatedAt, &it.PaddleSubscriptionID, &it.ExpiresAt, &it.LastWebhookAt); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		it.StatusLabel = subscriptions.SubscriptionStatusLabel(it.Status)
		items = append(items, it)
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"meta":  pagination.BuildMeta(params, total, skipCap),
	})
}

// invalidateQuotaCachesForPlan drops Redis user/workspace quota hashes for everyone on planID
// so AuthQuota reloads plan_catalog.unlimited (and limits) on the next metered request.
func (h *Handler) invalidateQuotaCachesForPlan(ctx context.Context, planID string) {
	planID = strings.ToLower(strings.TrimSpace(planID))
	if h.Redis == nil || h.DB == nil || planID == "" {
		return
	}
	userRows, err := h.DB.Query(ctx, `select user_id::text from public.user_quotas where plan_tier = $1`, planID)
	if err == nil {
		defer userRows.Close()
		for userRows.Next() {
			var uid string
			if userRows.Scan(&uid) != nil || uid == "" {
				continue
			}
			_ = h.Redis.Del(ctx, "user_quota:"+uid).Err()
		}
	}
	wsRows, err := h.DB.Query(ctx, `select id::text from public.workspaces where plan_tier = $1`, planID)
	if err == nil {
		defer wsRows.Close()
		for wsRows.Next() {
			var wid string
			if wsRows.Scan(&wid) != nil || wid == "" {
				continue
			}
			_ = h.Redis.Del(ctx, "workspace_quota:"+wid).Err()
		}
	}
}
