package events

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/usetrim/trim/server/internal/billingsettings"
	"github.com/usetrim/trim/server/internal/middleware"
	"github.com/usetrim/trim/server/internal/subscriptions"
)

// Canonical trim_events.status values written by the proxy hot path.
// Stats success_rate matches these codes (not display labels).
const (
	StatusSuccess = "success"
	StatusError   = "error"
)

type Handler struct {
	DB               *pgxpool.Pool
	ReadDB           *pgxpool.Pool
	Redis            *redis.Client
	InsertTimeoutSec int
	wake             chan struct{}
	workers          int
	outboxPollSec    int
	outboxBatch      int
}

func writeJSONErr(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": subscriptions.MessageForCode(code),
	})
}

type IngestRequest struct {
	RequestID              string `json:"request_id"`
	Model                  string `json:"model"`
	TokensBefore           int    `json:"tokens_before"`
	TokensAfter            int    `json:"tokens_after"`
	LatencyMs              int    `json:"latency_ms"`
	Mode                   string `json:"mode"`
	Status                 string `json:"status"`
	ErrorCode              string `json:"error_code"`
	TabSuggestionsShown    int    `json:"tab_suggestions_shown"`
	TabSuggestionsAccepted int    `json:"tab_suggestions_accepted"`
	AILinesAdded           int    `json:"ai_lines_added"`
	AILinesDeleted         int    `json:"ai_lines_deleted"`
}

// Ingest stores a proxy trim event for the authenticated user (CLI or cloud).
func (h *Handler) Ingest(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}
	if h.DB == nil {
		writeJSONErr(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
		return
	}

	var req IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	req.Mode = strings.TrimSpace(req.Mode)
	req.Status = strings.TrimSpace(req.Status)
	if req.Mode == "" || req.Status == "" {
		writeJSONErr(w, http.StatusBadRequest, "EVENTS_MODE_STATUS_REQUIRED")
		return
	}
	if req.TokensBefore < 0 || req.TokensAfter < 0 || req.LatencyMs < 0 {
		writeJSONErr(w, http.StatusBadRequest, "EVENTS_INVALID_METRICS")
		return
	}
	if req.TabSuggestionsShown < 0 || req.TabSuggestionsAccepted < 0 ||
		req.AILinesAdded < 0 || req.AILinesDeleted < 0 {
		writeJSONErr(w, http.StatusBadRequest, "EVENTS_INVALID_COUNTERS")
		return
	}
	if req.TabSuggestionsAccepted > req.TabSuggestionsShown {
		writeJSONErr(w, http.StatusBadRequest, "EVENTS_TAB_ACCEPTED_EXCEEDS")
		return
	}

	var id string
	err := h.DB.QueryRow(r.Context(), `
		insert into public.trim_events (
			user_id, request_id, model, tokens_before, tokens_after, latency_ms, mode, status, error_code,
			tab_suggestions_shown, tab_suggestions_accepted, ai_lines_added, ai_lines_deleted
		) values ($1, nullif($2, ''), nullif($3, ''), $4, $5, $6, $7, $8, nullif($9, ''), $10, $11, $12, $13)
		returning id::text
	`, userID, req.RequestID, req.Model, req.TokensBefore, req.TokensAfter, req.LatencyMs, req.Mode, req.Status,
		strings.TrimSpace(strings.ToLower(req.ErrorCode)),
		req.TabSuggestionsShown, req.TabSuggestionsAccepted, req.AILinesAdded, req.AILinesDeleted).Scan(&id)
	if err != nil && isNoPartitionErr(err) {
		if ensureErr := ensureEventPartitions(r.Context(), h.DB); ensureErr != nil {
			writeJSONErr(w, http.StatusServiceUnavailable, "ADMIN_EVENTS_PARTITION_ENSURE_FAILED")
			return
		}
		err = h.DB.QueryRow(r.Context(), `
			insert into public.trim_events (
				user_id, request_id, model, tokens_before, tokens_after, latency_ms, mode, status, error_code,
				tab_suggestions_shown, tab_suggestions_accepted, ai_lines_added, ai_lines_deleted
			) values ($1, nullif($2, ''), nullif($3, ''), $4, $5, $6, $7, $8, nullif($9, ''), $10, $11, $12, $13)
			returning id::text
		`, userID, req.RequestID, req.Model, req.TokensBefore, req.TokensAfter, req.LatencyMs, req.Mode, req.Status,
			strings.TrimSpace(strings.ToLower(req.ErrorCode)),
			req.TabSuggestionsShown, req.TabSuggestionsAccepted, req.AILinesAdded, req.AILinesDeleted).Scan(&id)
	}
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "EVENTS_STORE_FAILED")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"id": id, "status": "ok"})
}

func (h *Handler) readPool() *pgxpool.Pool {
	if h.ReadDB != nil {
		return h.ReadDB
	}
	return h.DB
}

type StatsResponse struct {
	ModelBreakdown    []NamedCount    `json:"model_breakdown"`
	StatusBreakdown   []NamedCount    `json:"status_breakdown"`
	ModeBreakdown     []NamedCount    `json:"mode_breakdown"`
	TokenSeries       []DayPoint      `json:"token_series"`
	Acceptance        AcceptanceStats `json:"acceptance"`
	LOC               LOCStats        `json:"loc"`
	TotalEvents       int             `json:"total_events"`
	EmptyModels       string          `json:"empty_models_message"`
	EmptyStatus       string          `json:"empty_status_message"`
	EmptyModes        string          `json:"empty_modes_message"`
	EmptySeries       string          `json:"empty_series_message"`
	SuccessRatePrefix string          `json:"success_rate_prefix"`
	RunsSeriesName    string          `json:"runs_series_name"`
	ScopeFull         string          `json:"scope_full_label"`
	ScopePage         string          `json:"scope_page_label"`
	TracesUnit        string          `json:"traces_unit"`
	StatusSuccess     string          `json:"status_success_label"`
	StatusError       string          `json:"status_error_label"`
	StatusSuccessCode string          `json:"status_success_code"`
	RunsFmt           string          `json:"runs_fmt"`
	ModelTooltipFmt   string          `json:"model_tooltip_fmt"`
	SuccessCount      int64           `json:"success_count"`
	SuccessRate       float64         `json:"success_rate"`

	UsageSeries                 []UsageDaySeries `json:"usage_series"`
	UsageDays                   []UsageAxisDay   `json:"usage_days"`
	UsageGroupByOptions         []NamedOption    `json:"usage_group_by_options"`
	UsageGroupBySelected        string           `json:"usage_group_by_selected,omitempty"`
	UsageTitle                  string           `json:"usage_title,omitempty"`
	UsageSubtitle               string           `json:"usage_subtitle,omitempty"`
	UsageYAxis                  string           `json:"usage_y_axis,omitempty"`
	UsageTodayLabel             string           `json:"usage_today_label,omitempty"`
	UsageGroupByPrefix          string           `json:"usage_group_by_prefix,omitempty"`
	UsageEmpty                  string           `json:"usage_empty,omitempty"`
	UsageTooltipBreakdown       string           `json:"usage_tooltip_breakdown,omitempty"`
	UsageTooltipDailyTotal      string           `json:"usage_tooltip_daily_total,omitempty"`
	UsageTooltipCumulativeTotal string           `json:"usage_tooltip_cumulative_total,omitempty"`
	UsageTooltipShareFmt        string           `json:"usage_tooltip_share_fmt,omitempty"`
	TodayDay                    string           `json:"today_day,omitempty"`
	LocHeatmap                  []LocHeatmapDay  `json:"loc_heatmap"`
	LocHeatmapTotal             int64            `json:"loc_heatmap_total"`
	LocHeatmapScopes            []NamedOption    `json:"loc_heatmap_scopes"`
	LocHeatmapScopeSelected     string           `json:"loc_heatmap_scope_selected,omitempty"`
	LocHeatmapTitle             string           `json:"loc_heatmap_title,omitempty"`
	LocHeatmapEmptyFmt          string           `json:"loc_heatmap_empty_fmt,omitempty"`
	LocHeatmapValueFmt          string           `json:"loc_heatmap_value_fmt,omitempty"`
	LocHeatmapWeekdayLabels     []NamedOption    `json:"loc_heatmap_weekday_labels,omitempty"`
	LocHeatmapStats             []LocHeatmapStat `json:"loc_heatmap_stats,omitempty"`
}

type NamedCount struct {
	Name   string `json:"name"`
	Code   string `json:"code,omitempty"`
	Count  int64  `json:"count"`
	Tokens int64  `json:"tokens,omitempty"`
	Saved  int64  `json:"saved,omitempty"`
}

type DayPoint struct {
	Day      string `json:"day"`
	DayLabel string `json:"day_label"`
	Before   int64  `json:"tokens_before"`
	After    int64  `json:"tokens_after"`
	Saved    int64  `json:"tokens_saved"`
}

type AcceptanceStats struct {
	Shown    int64   `json:"tab_suggestions_shown"`
	Accepted int64   `json:"tab_suggestions_accepted"`
	Rate     float64 `json:"acceptance_rate"`
}

type LOCStats struct {
	Added   int64 `json:"ai_lines_added"`
	Deleted int64 `json:"ai_lines_deleted"`
}

// Stats returns full-history aggregates for the authenticated user (not page-scoped).
func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}
	if h.DB == nil && h.ReadDB == nil {
		writeJSONErr(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
		return
	}
	rdb := h.readPool()
	if rdb == nil {
		writeJSONErr(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
		return
	}
	ctx := r.Context()
	var chartTopN, chartSeriesDays int
	_ = rdb.QueryRow(ctx, `
		select coalesce(chart_top_n, 0), coalesce(chart_series_days, 0)
		from public.billing_settings where id = 'default'
	`).Scan(&chartTopN, &chartSeriesDays)
	if chartTopN < 1 {
		writeJSONErr(w, http.StatusInternalServerError, "CHART_TOP_N_MISSING")
		return
	}
	if chartSeriesDays < 1 {
		writeJSONErr(w, http.StatusInternalServerError, "CHART_SERIES_DAYS_MISSING")
		return
	}
	out := StatsResponse{
		ModelBreakdown:      make([]NamedCount, 0),
		StatusBreakdown:     make([]NamedCount, 0),
		ModeBreakdown:       make([]NamedCount, 0),
		TokenSeries:         make([]DayPoint, 0),
		UsageSeries:         make([]UsageDaySeries, 0),
		UsageDays:           make([]UsageAxisDay, 0),
		UsageGroupByOptions: make([]NamedOption, 0),
		LocHeatmap:          make([]LocHeatmapDay, 0),
		LocHeatmapScopes:    make([]NamedOption, 0),
		EmptyModels:         subscriptions.MessageForCode("CHART_EMPTY_MODELS"),
		EmptyStatus:         subscriptions.MessageForCode("CHART_EMPTY_STATUS"),
		EmptyModes:          subscriptions.MessageForCode("CHART_EMPTY_MODES"),
		EmptySeries:         subscriptions.MessageForCode("CHART_EMPTY_SERIES"),
		SuccessRatePrefix:   subscriptions.MessageForCode("CHART_SUCCESS_RATE_PREFIX"),
		RunsSeriesName:      subscriptions.MessageForCode("CHART_RUNS_SERIES"),
		ScopeFull:           subscriptions.MessageForCode("CHART_SCOPE_FULL"),
		ScopePage:           subscriptions.MessageForCode("CHART_SCOPE_PAGE"),
		TracesUnit:          subscriptions.MessageForCode("CHART_TRACES_UNIT"),
		StatusSuccess:       subscriptions.MessageForCode("CHART_STATUS_SUCCESS"),
		StatusError:         subscriptions.MessageForCode("CHART_STATUS_ERROR"),
		StatusSuccessCode:   StatusSuccess,
		RunsFmt:             subscriptions.MessageForCode("CHART_RUNS_FMT"),
		ModelTooltipFmt:     subscriptions.MessageForCode("CHART_MODEL_TOOLTIP_FMT"),
	}

	if err := rdb.QueryRow(ctx, `select count(*) from public.trim_events where user_id = $1`, userID).Scan(&out.TotalEvents); err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "EVENTS_COUNT_FAILED")
		return
	}

	rows, err := rdb.Query(ctx, `
		select coalesce(nullif(trim(model), ''), ''),
		       count(*), coalesce(sum(tokens_before), 0),
		       coalesce(sum(greatest(tokens_before - tokens_after, 0)), 0)
		from public.trim_events where user_id = $1
		group by 1 order by 3 desc limit $2
	`, userID, chartTopN)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "EVENTS_AGG_MODELS_FAILED")
		return
	}
	unknownLabel := subscriptions.MessageForCode("STATS_UNKNOWN_LABEL")
	for rows.Next() {
		var n NamedCount
		if err := rows.Scan(&n.Name, &n.Count, &n.Tokens, &n.Saved); err != nil {
			rows.Close()
			writeJSONErr(w, http.StatusInternalServerError, "EVENTS_SCAN_MODELS_FAILED")
			return
		}
		if n.Name == "" {
			n.Name = unknownLabel
		}
		out.ModelBreakdown = append(out.ModelBreakdown, n)
	}
	rows.Close()

	rows, err = rdb.Query(ctx, `
		select coalesce(nullif(trim(status), ''), ''), count(*)
		from public.trim_events where user_id = $1
		group by 1 order by 2 desc limit $2
	`, userID, chartTopN)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "EVENTS_AGG_STATUS_FAILED")
		return
	}
	for rows.Next() {
		var n NamedCount
		var raw string
		if err := rows.Scan(&raw, &n.Count); err != nil {
			rows.Close()
			writeJSONErr(w, http.StatusInternalServerError, "EVENTS_SCAN_STATUS_FAILED")
			return
		}
		n.Code = raw
		if raw == "" {
			n.Name = unknownLabel
		} else if label := subscriptions.EventStatusLabel(raw); label != "" {
			n.Name = label
		} else {
			n.Name = raw
		}
		if strings.EqualFold(raw, out.StatusSuccessCode) {
			out.SuccessCount += n.Count
		}
		out.StatusBreakdown = append(out.StatusBreakdown, n)
	}
	rows.Close()
	if out.TotalEvents > 0 {
		out.SuccessRate = float64(out.SuccessCount) / float64(out.TotalEvents) * 100
		out.SuccessRate = float64(int(out.SuccessRate*10+0.5)) / 10
	}
	rows, err = rdb.Query(ctx, `
		select coalesce(nullif(trim(mode), ''), ''), count(*)
		from public.trim_events where user_id = $1
		group by 1 order by 2 desc limit $2
	`, userID, chartTopN)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "EVENTS_AGG_MODES_FAILED")
		return
	}
	for rows.Next() {
		var n NamedCount
		var raw string
		if err := rows.Scan(&raw, &n.Count); err != nil {
			rows.Close()
			writeJSONErr(w, http.StatusInternalServerError, "EVENTS_SCAN_MODES_FAILED")
			return
		}
		n.Code = raw
		if raw == "" {
			n.Name = unknownLabel
		} else if label := subscriptions.EventModeLabel(raw); label != "" {
			n.Name = label
		} else {
			n.Name = raw
		}
		out.ModeBreakdown = append(out.ModeBreakdown, n)
	}
	rows.Close()

	rows, err = rdb.Query(ctx, `
		select to_char(created_at at time zone 'utc', 'YYYY-MM-DD'),
		       coalesce(sum(tokens_before), 0),
		       coalesce(sum(tokens_after), 0),
		       coalesce(sum(greatest(tokens_before - tokens_after, 0)), 0)
		from public.trim_events
		where user_id = $1 and created_at >= (now() - make_interval(days => $2))
		group by 1 order by 1 asc
	`, userID, chartSeriesDays)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "EVENTS_AGG_SERIES_FAILED")
		return
	}
	for rows.Next() {
		var p DayPoint
		if err := rows.Scan(&p.Day, &p.Before, &p.After, &p.Saved); err != nil {
			rows.Close()
			writeJSONErr(w, http.StatusInternalServerError, "EVENTS_SCAN_SERIES_FAILED")
			return
		}
		p.DayLabel = subscriptions.FormatUTCChartDay(p.Day)
		out.TokenSeries = append(out.TokenSeries, p)
	}
	rows.Close()

	_ = rdb.QueryRow(ctx, `
		select coalesce(sum(tab_suggestions_shown), 0),
		       coalesce(sum(tab_suggestions_accepted), 0),
		       coalesce(sum(ai_lines_added), 0),
		       coalesce(sum(ai_lines_deleted), 0)
		from public.trim_events where user_id = $1
	`, userID).Scan(
		&out.Acceptance.Shown, &out.Acceptance.Accepted,
		&out.LOC.Added, &out.LOC.Deleted,
	)
	if out.Acceptance.Shown > 0 {
		out.Acceptance.Rate = float64(out.Acceptance.Accepted) / float64(out.Acceptance.Shown)
	}

	groupBy := r.URL.Query().Get("group_by")
	heatmapScope := r.URL.Query().Get("heatmap_scope")
	cacheKey := h.chartCacheKey(userID, groupBy, heatmapScope, chartSeriesDays, chartTopN)
	chartTTL, ttlErr := billingsettings.ChartCacheTTLSec(ctx, rdb)
	if ttlErr != nil {
		writeJSONErr(w, http.StatusInternalServerError, ttlErr.Error())
		return
	}
	var charts DashboardCharts
	if cached, ok := h.getCachedCharts(ctx, cacheKey, chartTTL); ok {
		charts = *cached
	} else {
		built, err := BuildDashboardCharts(
			ctx, rdb, userID,
			groupBy,
			heatmapScope,
			chartSeriesDays, chartTopN,
			LoadUsageChrome("DASHBOARD"),
			LoadHeatmapChrome("DASHBOARD"),
		)
		if err != nil {
			writeJSONErr(w, http.StatusInternalServerError, "EVENTS_AGG_SERIES_FAILED")
			return
		}
		charts = built
		h.setCachedCharts(ctx, cacheKey, charts, chartTTL)
	}
	out.UsageSeries = charts.UsageSeries
	out.UsageDays = charts.UsageDays
	out.UsageGroupByOptions = charts.UsageGroupByOptions
	out.UsageGroupBySelected = charts.UsageGroupBySelected
	out.UsageTitle = charts.UsageTitle
	out.UsageSubtitle = charts.UsageSubtitle
	out.UsageYAxis = charts.UsageYAxis
	out.UsageTodayLabel = charts.UsageTodayLabel
	out.UsageGroupByPrefix = charts.UsageGroupByPrefix
	out.UsageEmpty = charts.UsageEmpty
	out.UsageTooltipBreakdown = charts.UsageTooltipBreakdown
	out.UsageTooltipDailyTotal = charts.UsageTooltipDailyTotal
	out.UsageTooltipCumulativeTotal = charts.UsageTooltipCumulativeTotal
	out.UsageTooltipShareFmt = charts.UsageTooltipShareFmt
	out.TodayDay = charts.TodayDay
	out.LocHeatmap = charts.LocHeatmap
	out.LocHeatmapTotal = charts.LocHeatmapTotal
	out.LocHeatmapScopes = charts.LocHeatmapScopes
	out.LocHeatmapScopeSelected = charts.LocHeatmapScopeSelected
	out.LocHeatmapTitle = charts.LocHeatmapTitle
	out.LocHeatmapEmptyFmt = charts.LocHeatmapEmptyFmt
	out.LocHeatmapValueFmt = charts.LocHeatmapValueFmt
	out.LocHeatmapWeekdayLabels = charts.LocHeatmapWeekdayLabels
	out.LocHeatmapStats = charts.LocHeatmapStats

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

type bulkIDsRequest struct {
	IDs []string `json:"ids"`
}

// BulkDelete removes the signed-in user's own trim_events by id (DataTable selection).
func (h *Handler) BulkDelete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}
	if h.DB == nil {
		writeJSONErr(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE")
		return
	}
	var req bulkIDsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	ids := make([]string, 0, len(req.IDs))
	seen := map[string]struct{}{}
	for _, raw := range req.IDs {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		writeJSONErr(w, http.StatusBadRequest, "EVENTS_IDS_REQUIRED")
		return
	}
	maxLimit, err := billingsettings.MaxPageSize(r.Context(), h.DB)
	if err != nil || maxLimit < 1 {
		writeJSONErr(w, http.StatusInternalServerError, "BILLING_SETTINGS_UNAVAILABLE")
		return
	}
	if len(ids) > maxLimit {
		ids = ids[:maxLimit]
	}
	tag, err := h.DB.Exec(r.Context(), `
		delete from public.trim_events
		where user_id = $1::uuid and id = any($2::uuid[])
	`, userID, ids)
	if err != nil {
		writeJSONErr(w, http.StatusBadRequest, "EVENTS_IDS_INVALID")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":        "ok",
		"deleted":       tag.RowsAffected(),
		"action_label":  subscriptions.MessageForCode("EVENTS_DELETE"),
		"pending_label": subscriptions.MessageForCode("EVENTS_DELETE_PENDING"),
		"message":       subscriptions.MessageForCode("EVENTS_BULK_DELETED"),
	})
}
