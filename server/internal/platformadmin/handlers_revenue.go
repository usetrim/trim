package platformadmin

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// getBillingRevenue returns settled (completed) receipt analytics for the Sales revenue page.
// Revenue is ledger-backed only: billing_receipts.status = 'completed'. Catalog × seats is never used.
func (h *Handler) getBillingRevenue(w http.ResponseWriter, r *http.Request) {
	db := h.readPool()
	rangeID, from, to, errMsg := h.resolveRevenueRange(r)
	if errMsg != "" {
		h.writeErr(w, http.StatusBadRequest, errMsg)
		return
	}

	rangeOptions := h.loadRevenueRangeOptions(r.Context())
	if len(rangeOptions) == 0 {
		h.writeErr(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
		return
	}

	title := strings.TrimSpace(h.msg("ADMIN_REVENUE_TITLE"))
	intro := strings.TrimSpace(h.msg("ADMIN_REVENUE_INTRO"))
	chartTitle := strings.TrimSpace(h.msg("ADMIN_REVENUE_CHART_TITLE"))
	byPlanTitle := strings.TrimSpace(h.msg("ADMIN_REVENUE_BY_PLAN_TITLE"))
	drillTitle := strings.TrimSpace(h.msg("ADMIN_REVENUE_DRILL_TITLE"))
	drillLink := strings.TrimSpace(h.msg("ADMIN_REVENUE_DRILL_LINK"))
	emptyMsg := strings.TrimSpace(h.msg("ADMIN_REVENUE_EMPTY"))
	rangeLabel := strings.TrimSpace(h.msg("ADMIN_REVENUE_RANGE_LABEL"))
	rangeDesc := strings.TrimSpace(h.msg("ADMIN_REVENUE_RANGE_DESC"))
	unknownPlan := strings.TrimSpace(h.msg("ADMIN_REVENUE_UNKNOWN_PLAN"))
	colPlan := strings.TrimSpace(h.msg("ADMIN_REVENUE_COL_PLAN"))
	colKind := strings.TrimSpace(h.msg("ADMIN_REVENUE_COL_KIND"))
	colRevenue := strings.TrimSpace(h.msg("ADMIN_REVENUE_COL_REVENUE"))
	colReceipts := strings.TrimSpace(h.msg("ADMIN_REVENUE_COL_RECEIPTS"))

	kpiTotalLabel := strings.TrimSpace(h.msg("ADMIN_REVENUE_KPI_TOTAL"))
	kpiReceiptsLabel := strings.TrimSpace(h.msg("ADMIN_REVENUE_KPI_RECEIPTS"))
	kpiRefundedLabel := strings.TrimSpace(h.msg("ADMIN_REVENUE_KPI_REFUNDED"))
	kpiFreeLabel := strings.TrimSpace(h.msg("ADMIN_REVENUE_KPI_FREE_ACCOUNTS"))
	kpiPaidLabel := strings.TrimSpace(h.msg("ADMIN_REVENUE_KPI_PAID_ACCOUNTS"))

	var totalCents int64
	var receiptCount, refundedCount int
	var currencyCode string
	err := db.QueryRow(r.Context(), `
		select
		  coalesce(sum(case when status = 'completed' then total_cents else 0 end), 0),
		  coalesce(count(*) filter (where status = 'completed'), 0),
		  coalesce(count(*) filter (where status in ('refunded', 'partially_refunded')), 0),
		  coalesce(max(case when status = 'completed' then currency_code end), '')
		from public.billing_receipts
		where coalesce(paid_at, created_at) >= $1::timestamptz
		  and coalesce(paid_at, created_at) < $2::timestamptz
	`, from, to).Scan(&totalCents, &receiptCount, &refundedCount, &currencyCode)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}

	var freeAccounts, paidAccounts int
	_ = db.QueryRow(r.Context(), `
		select
		  coalesce((
		    select count(*) from public.user_quotas uq
		    join public.plan_catalog pc on pc.id = uq.plan_tier
		    where coalesce(pc.price_monthly_cents, 0) = 0
		      and coalesce(pc.price_yearly_cents, 0) = 0
		      and lower(coalesce(pc.plan_kind, '')) <> 'topup'
		  ), 0),
		  coalesce((
		    select count(distinct user_id) from public.subscriptions
		    where status in ('active', 'trialing')
		      and coalesce(user_id::text, '') <> ''
		  ), 0)
	`).Scan(&freeAccounts, &paidAccounts)

	rows, err := db.Query(r.Context(), `
		with completed as (
		  select r.id, r.total_cents, r.currency_code,
		         (coalesce(r.paid_at, r.created_at))::date as day
		  from public.billing_receipts r
		  where r.status = 'completed'
		    and coalesce(r.paid_at, r.created_at) >= $1::timestamptz
		    and coalesce(r.paid_at, r.created_at) < $2::timestamptz
		),
		priced as (
		  select c.day, c.total_cents, c.currency_code,
		         coalesce(nullif(btrim(li.price_id), ''), '') as price_id
		  from completed c
		  left join lateral (
		    select price_id
		    from public.billing_receipt_line_items
		    where receipt_id = c.id
		    order by position asc
		    limit 1
		  ) li on true
		),
		mapped as (
		  select p.day, p.total_cents, p.currency_code,
		         coalesce(pc.id, '') as plan_id,
		         coalesce(pc.display_name, '') as plan_name,
		         coalesce(pc.plan_kind, '') as plan_kind
		  from priced p
		  left join public.plan_catalog pc
		    on nullif(btrim(p.price_id), '') is not null
		   and (
		     pc.paddle_price_id_monthly = p.price_id
		     or pc.paddle_price_id_yearly = p.price_id
		     or pc.paddle_price_id_topup = p.price_id
		   )
		)
		select day::text,
		       coalesce(nullif(plan_id, ''), '__unmapped__') as plan_id,
		       case when plan_id = '' then $3 else plan_name end as plan_label,
		       coalesce(nullif(plan_kind, ''), '') as plan_kind,
		       sum(total_cents)::bigint as revenue_cents,
		       count(*)::int as receipt_count,
		       coalesce(max(currency_code), '') as currency_code
		from mapped
		group by day, coalesce(nullif(plan_id, ''), '__unmapped__'),
		         case when plan_id = '' then $3 else plan_name end,
		         coalesce(nullif(plan_kind, ''), '')
		order by day asc, plan_label asc
	`, from, to, unknownPlan)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()

	type dayPlan struct {
		day, planID, planLabel, planKind, currency string
		cents                                      int64
		count                                      int
	}
	points := make([]dayPlan, 0)
	planTotals := map[string]*struct {
		label, kind, currency string
		cents                 int64
		count                 int
	}{}
	planOrder := make([]string, 0)

	for rows.Next() {
		var d dayPlan
		if err := rows.Scan(&d.day, &d.planID, &d.planLabel, &d.planKind, &d.cents, &d.count, &d.currency); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		points = append(points, d)
		agg, ok := planTotals[d.planID]
		if !ok {
			agg = &struct {
				label, kind, currency string
				cents                 int64
				count                 int
			}{label: d.planLabel, kind: d.planKind, currency: d.currency}
			planTotals[d.planID] = agg
			planOrder = append(planOrder, d.planID)
		}
		agg.cents += d.cents
		agg.count += d.count
		if agg.currency == "" {
			agg.currency = d.currency
		}
	}

	seriesMeta := make([]map[string]any, 0, len(planOrder))
	for _, id := range planOrder {
		agg := planTotals[id]
		seriesMeta = append(seriesMeta, map[string]any{
			"id":    id,
			"label": agg.label,
		})
	}

	dayMap := map[string]map[string]any{}
	dayOrder := make([]string, 0)
	for _, p := range points {
		row, ok := dayMap[p.day]
		if !ok {
			row = map[string]any{
				"day":       p.day,
				"day_label": p.day,
			}
			dayMap[p.day] = row
			dayOrder = append(dayOrder, p.day)
		}
		row["d_"+p.planID] = p.cents
		row["d_"+p.planID+"_label"] = formatCentsMajor(p.cents, firstNonEmpty(p.currency, currencyCode))
	}
	series := make([]map[string]any, 0, len(dayOrder))
	for _, day := range dayOrder {
		series = append(series, dayMap[day])
	}

	byPlan := make([]map[string]any, 0, len(planOrder))
	for _, id := range planOrder {
		agg := planTotals[id]
		cur := firstNonEmpty(agg.currency, currencyCode)
		byPlan = append(byPlan, map[string]any{
			"plan_id":       id,
			"plan_label":    agg.label,
			"plan_kind":     agg.kind,
			"revenue_cents": agg.cents,
			"revenue_label": formatCentsMajor(agg.cents, cur),
			"receipt_count": agg.count,
			"currency_code": cur,
		})
	}

	kpis := make([]map[string]any, 0, 5)
	if kpiTotalLabel != "" {
		kpis = append(kpis, map[string]any{
			"id":    "settled_revenue",
			"label": kpiTotalLabel,
			"value": formatCentsMajor(totalCents, currencyCode),
		})
	}
	if kpiReceiptsLabel != "" {
		kpis = append(kpis, map[string]any{
			"id": "completed_receipts", "label": kpiReceiptsLabel, "value": receiptCount,
		})
	}
	if kpiRefundedLabel != "" {
		kpis = append(kpis, map[string]any{
			"id": "refunded_receipts", "label": kpiRefundedLabel, "value": refundedCount,
		})
	}
	if kpiFreeLabel != "" {
		kpis = append(kpis, map[string]any{
			"id": "free_accounts", "label": kpiFreeLabel, "value": freeAccounts,
		})
	}
	if kpiPaidLabel != "" {
		kpis = append(kpis, map[string]any{
			"id": "paid_accounts", "label": kpiPaidLabel, "value": paidAccounts,
		})
	}

	chrome := map[string]string{}
	put := func(k, v string) {
		if strings.TrimSpace(v) != "" {
			chrome[k] = strings.TrimSpace(v)
		}
	}
	put("title", title)
	put("intro", intro)
	put("chart_title", chartTitle)
	put("by_plan_title", byPlanTitle)
	put("drill_title", drillTitle)
	put("drill_link", drillLink)
	put("empty", emptyMsg)
	put("range_label", rangeLabel)
	put("range_desc", rangeDesc)
	put("col_plan", colPlan)
	put("col_kind", colKind)
	put("col_revenue", colRevenue)
	put("col_receipts", colReceipts)
	put("receipts_href", h.adminNavHref(r.Context(), "receipts"))
	put("date_range_placeholder", h.msg("ADMIN_DATE_RANGE_PLACEHOLDER"))
	put("date_range_clear", h.msg("ADMIN_DATE_RANGE_CLEAR"))
	put("date_range_apply", h.msg("ADMIN_DATE_RANGE_APPLY"))
	put("html_lang", h.msg("SITE_HTML_LANG"))

	var dateRangeMonths int
	_ = db.QueryRow(r.Context(), `
		select coalesce(date_range_months, 0) from public.billing_settings where id = 'default'
	`).Scan(&dateRangeMonths)

	h.writeJSON(w, http.StatusOK, map[string]any{
		"range":             rangeID,
		"from":              from.UTC().Format(time.RFC3339),
		"to":                to.UTC().Format(time.RFC3339),
		"range_options":     rangeOptions,
		"currency_code":     currencyCode,
		"kpi_items":         kpis,
		"series_meta":       seriesMeta,
		"series":            series,
		"by_plan":           byPlan,
		"date_range_months": dateRangeMonths,
		"chrome":            chrome,
	})
}

func (h *Handler) loadRevenueRangeOptions(ctx context.Context) []map[string]string {
	rows, err := h.readPool().Query(ctx, `
		select id, chrome_code
		from public.admin_revenue_range_presets
		where is_active = true
		order by sort_order asc, id asc
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]map[string]string, 0)
	for rows.Next() {
		var id, code string
		if err := rows.Scan(&id, &code); err != nil {
			continue
		}
		label := strings.TrimSpace(h.msg(code))
		if label == "" {
			continue
		}
		out = append(out, map[string]string{"id": id, "label": label})
	}
	return out
}

// resolveRevenueRange reads range=7d|30d|mtd|ytd|custom (+ created_from/created_to for custom).
// Preset ids must exist and be active in admin_revenue_range_presets (fail closed).
func (h *Handler) resolveRevenueRange(r *http.Request) (rangeID string, from, to time.Time, errCode string) {
	now := time.Now().UTC()
	to = now
	rangeID = strings.TrimSpace(r.URL.Query().Get("range"))
	if rangeID == "" {
		_ = h.readPool().QueryRow(r.Context(), `
			select id from public.admin_revenue_range_presets
			where is_active = true
			order by sort_order asc, id asc
			limit 1
		`).Scan(&rangeID)
		if strings.TrimSpace(rangeID) == "" {
			return "", time.Time{}, time.Time{}, "DATABASE_UNAVAILABLE"
		}
	}

	var exists bool
	err := h.readPool().QueryRow(r.Context(), `
		select true from public.admin_revenue_range_presets
		where id = $1 and is_active = true
	`, rangeID).Scan(&exists)
	if err != nil || !exists {
		return "", time.Time{}, time.Time{}, "ADMIN_STATUS_INVALID"
	}

	switch rangeID {
	case "7d":
		from = now.Add(-7 * 24 * time.Hour)
	case "30d":
		from = now.Add(-30 * 24 * time.Hour)
	case "mtd":
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	case "ytd":
		from = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	case "custom":
		fromRaw := strings.TrimSpace(r.URL.Query().Get("created_from"))
		toRaw := strings.TrimSpace(r.URL.Query().Get("created_to"))
		if fromRaw == "" || toRaw == "" {
			return "", time.Time{}, time.Time{}, "ADMIN_STATUS_INVALID"
		}
		fromDay, err1 := time.ParseInLocation("2006-01-02", fromRaw, time.UTC)
		toDay, err2 := time.ParseInLocation("2006-01-02", toRaw, time.UTC)
		if err1 != nil || err2 != nil {
			return "", time.Time{}, time.Time{}, "ADMIN_STATUS_INVALID"
		}
		from = fromDay
		to = toDay.Add(24 * time.Hour)
		if !to.After(from) {
			return "", time.Time{}, time.Time{}, "ADMIN_STATUS_INVALID"
		}
	default:
		return "", time.Time{}, time.Time{}, "ADMIN_STATUS_INVALID"
	}
	return rangeID, from, to, ""
}

func formatCentsMajor(cents int64, currency string) string {
	code := strings.ToUpper(strings.TrimSpace(currency))
	if code == "" {
		return ""
	}
	sign := ""
	v := cents
	if v < 0 {
		sign = "-"
		v = -v
	}
	major := v / 100
	minor := v % 100
	return fmt.Sprintf("%s%s %d.%02d", sign, code, major, minor)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
