package events

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/usetrim/trim/server/internal/subscriptions"
)

// usageAxisOnlySeriesID marks day-axis placeholder rows (tokens=0, no stack series).
const usageAxisOnlySeriesID = "__axis__"

// UsageDaySeries is one day×series point for the stacked usage chart.
// Tokens are cumulative; DailyTokens is that day's delta.
type UsageDaySeries struct {
	Day              string `json:"day"`
	DayLabel         string `json:"day_label"`
	SeriesID         string `json:"series_id"`
	SeriesLabel      string `json:"series_label"`
	Tokens           int64  `json:"tokens"`
	DailyTokens      int64  `json:"daily_tokens"`
	TokensLabel      string `json:"tokens_label,omitempty"`
	DailyTokensLabel string `json:"daily_tokens_label,omitempty"`
}

// UsageAxisDay is one X-axis tick for the usage chart (always emitted for the window).
type UsageAxisDay struct {
	Day             string `json:"day"`
	DayLabel        string `json:"day_label"`
	DailyTotal      int64  `json:"daily_total,omitempty"`
	DailyTotalLabel string `json:"daily_total_label,omitempty"`
	CumTotal        int64  `json:"cum_total,omitempty"`
	CumTotalLabel   string `json:"cum_total_label,omitempty"`
}

// NamedOption is a fail-closed chrome option (id + non-empty label).
type NamedOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// LocHeatmapDay is one contribution-calendar cell.
type LocHeatmapDay struct {
	Day          string `json:"day"`
	DayLabelLong string `json:"day_label_long"`
	WeekdayMon0  int    `json:"weekday_mon0"`
	MonthLabel   string `json:"month_label,omitempty"`
	Value        int64  `json:"value"`
	Intensity    int    `json:"intensity"`
}

// LocHeatmapStat is one backend-computed activity summary row (label + value from chrome + data).
type LocHeatmapStat struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Value string `json:"value"`
}

// UsageChrome holds site_messages for the usage stacked chart (web or admin prefix).
type UsageChrome struct {
	Title                  string
	Subtitle               string
	YAxis                  string
	TodayLabel             string
	GroupByPrefix          string
	GroupModel             string
	GroupMode              string
	Empty                  string
	TooltipBreakdown       string
	TooltipDailyTotal      string
	TooltipCumulativeTotal string
	TooltipShareFmt        string
}

// HeatmapChrome holds site_messages for the LOC contribution heatmap.
type HeatmapChrome struct {
	Title               string
	ScopeAll            string
	ScopeTab            string
	EmptyFmtAll         string
	EmptyFmtTab         string
	ValueFmtAll         string
	ValueFmtTab         string
	WdMon               string
	WdWed               string
	WdFri               string
	StatMostActiveMonth string
	StatMostActiveDay   string
	StatLongestStreak   string
	StatCurrentStreak   string
	StreakFmt           string
}

// DashboardCharts is the shared usage + heatmap payload shape.
type DashboardCharts struct {
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

// LoadUsageChrome reads DASHBOARD_* or ADMIN_* usage chrome codes (empty = fail closed).
func LoadUsageChrome(prefix string) UsageChrome {
	p := strings.TrimSpace(prefix)
	if p == "" {
		p = "DASHBOARD"
	}
	return UsageChrome{
		Title:                  subscriptions.MessageForCode(p + "_USAGE_TITLE"),
		Subtitle:               subscriptions.MessageForCode(p + "_USAGE_SUBTITLE"),
		YAxis:                  subscriptions.MessageForCode(p + "_USAGE_Y_AXIS"),
		TodayLabel:             subscriptions.MessageForCode(p + "_USAGE_TODAY"),
		GroupByPrefix:          subscriptions.MessageForCode(p + "_USAGE_GROUP_BY_PREFIX"),
		GroupModel:             subscriptions.MessageForCode(p + "_USAGE_GROUP_MODEL"),
		GroupMode:              subscriptions.MessageForCode(p + "_USAGE_GROUP_MODE"),
		Empty:                  subscriptions.MessageForCode(p + "_USAGE_EMPTY"),
		TooltipBreakdown:       subscriptions.MessageForCode(p + "_USAGE_TOOLTIP_BREAKDOWN"),
		TooltipDailyTotal:      subscriptions.MessageForCode(p + "_USAGE_TOOLTIP_DAILY_TOTAL"),
		TooltipCumulativeTotal: subscriptions.MessageForCode(p + "_USAGE_TOOLTIP_CUMULATIVE_TOTAL"),
		TooltipShareFmt:        subscriptions.MessageForCode(p + "_USAGE_TOOLTIP_SHARE_FMT"),
	}
}

// LoadHeatmapChrome reads DASHBOARD_* or ADMIN_* heatmap chrome codes.
func LoadHeatmapChrome(prefix string) HeatmapChrome {
	p := strings.TrimSpace(prefix)
	if p == "" {
		p = "DASHBOARD"
	}
	return HeatmapChrome{
		Title:               subscriptions.MessageForCode(p + "_HEATMAP_TITLE"),
		ScopeAll:            subscriptions.MessageForCode(p + "_HEATMAP_SCOPE_ALL"),
		ScopeTab:            subscriptions.MessageForCode(p + "_HEATMAP_SCOPE_TAB"),
		EmptyFmtAll:         subscriptions.MessageForCode(p + "_HEATMAP_EMPTY_FMT_ALL"),
		EmptyFmtTab:         subscriptions.MessageForCode(p + "_HEATMAP_EMPTY_FMT_TAB"),
		ValueFmtAll:         subscriptions.MessageForCode(p + "_HEATMAP_VALUE_FMT_ALL"),
		ValueFmtTab:         subscriptions.MessageForCode(p + "_HEATMAP_VALUE_FMT_TAB"),
		WdMon:               subscriptions.MessageForCode(p + "_HEATMAP_WD_MON"),
		WdWed:               subscriptions.MessageForCode(p + "_HEATMAP_WD_WED"),
		WdFri:               subscriptions.MessageForCode(p + "_HEATMAP_WD_FRI"),
		StatMostActiveMonth: subscriptions.MessageForCode(p + "_HEATMAP_STAT_MOST_ACTIVE_MONTH"),
		StatMostActiveDay:   subscriptions.MessageForCode(p + "_HEATMAP_STAT_MOST_ACTIVE_DAY"),
		StatLongestStreak:   subscriptions.MessageForCode(p + "_HEATMAP_STAT_LONGEST_STREAK"),
		StatCurrentStreak:   subscriptions.MessageForCode(p + "_HEATMAP_STAT_CURRENT_STREAK"),
		StreakFmt:           subscriptions.MessageForCode(p + "_HEATMAP_STREAK_FMT"),
	}
}

// BuildDashboardCharts loads usage series + LOC heatmap for one user (userID set) or platform-wide (userID empty).
func BuildDashboardCharts(
	ctx context.Context,
	db *pgxpool.Pool,
	userID string,
	groupBy string,
	heatmapScope string,
	seriesDays int,
	topN int,
	usage UsageChrome,
	heat HeatmapChrome,
) (DashboardCharts, error) {
	out := DashboardCharts{
		UsageSeries:         make([]UsageDaySeries, 0),
		UsageDays:           make([]UsageAxisDay, 0),
		UsageGroupByOptions: make([]NamedOption, 0),
		LocHeatmap:          make([]LocHeatmapDay, 0),
		LocHeatmapScopes:    make([]NamedOption, 0),
		LocHeatmapStats:     make([]LocHeatmapStat, 0),
	}
	if db == nil || seriesDays < 1 || topN < 1 {
		return out, nil
	}

	today := time.Now().UTC()
	todayDay := today.Format("2006-01-02")
	out.TodayDay = todayDay

	dayKeys := make([]string, 0, seriesDays)
	start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC).
		AddDate(0, 0, -(seriesDays - 1))
	for d := start; !d.After(time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)); d = d.AddDate(0, 0, 1) {
		dayKeys = append(dayKeys, d.Format("2006-01-02"))
	}

	if err := buildUsageSeries(ctx, db, userID, groupBy, seriesDays, topN, usage, dayKeys, &out); err != nil {
		return out, err
	}
	if err := buildLocHeatmap(ctx, db, userID, heatmapScope, seriesDays, heat, dayKeys, &out); err != nil {
		return out, err
	}
	return out, nil
}

func buildUsageSeries(
	ctx context.Context,
	db *pgxpool.Pool,
	userID string,
	groupBy string,
	seriesDays int,
	topN int,
	usage UsageChrome,
	dayKeys []string,
	out *DashboardCharts,
) error {
	if usage.GroupModel != "" {
		out.UsageGroupByOptions = append(out.UsageGroupByOptions, NamedOption{ID: "model", Label: usage.GroupModel})
	}
	if usage.GroupMode != "" {
		out.UsageGroupByOptions = append(out.UsageGroupByOptions, NamedOption{ID: "mode", Label: usage.GroupMode})
	}
	if len(out.UsageGroupByOptions) == 0 {
		return nil
	}

	selected := strings.ToLower(strings.TrimSpace(groupBy))
	if selected == "" {
		if usage.GroupModel != "" {
			selected = "model"
		} else {
			return nil
		}
	}
	valid := false
	for _, opt := range out.UsageGroupByOptions {
		if opt.ID == selected {
			valid = true
			break
		}
	}
	if !valid {
		return nil
	}

	if usage.Title == "" || usage.YAxis == "" {
		return nil
	}

	var col string
	switch selected {
	case "model":
		col = "model"
	case "mode":
		col = "mode"
	default:
		return nil
	}

	userFilter := ""
	args := []any{seriesDays, topN}
	if userID != "" {
		userFilter = " and user_id = $3"
		args = []any{seriesDays, topN, userID}
	}

	q := `
		with daily as (
			select to_char(created_at at time zone 'utc', 'YYYY-MM-DD') as day,
			       coalesce(nullif(trim(` + col + `), ''), '') as series_id,
			       coalesce(sum(tokens_before), 0)::bigint as tokens
			from public.trim_events
			where created_at >= (now() - make_interval(days => $1))` + userFilter + `
			group by 1, 2
		),
		totals as (
			select series_id, sum(tokens)::bigint as total
			from daily
			group by 1
			order by total desc
			limit $2
		)
		select d.day, d.series_id, d.tokens
		from daily d
		inner join totals t on t.series_id = d.series_id
		order by d.day asc, d.series_id asc
	`

	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	type daySeriesKey struct {
		day, series string
	}
	daily := map[daySeriesKey]int64{}
	seriesTotals := map[string]int64{}
	unknownLabel := subscriptions.MessageForCode("STATS_UNKNOWN_LABEL")

	for rows.Next() {
		var day, seriesID string
		var tokens int64
		if err := rows.Scan(&day, &seriesID, &tokens); err != nil {
			return err
		}
		daily[daySeriesKey{day: day, series: seriesID}] = tokens
		seriesTotals[seriesID] += tokens
	}
	if err := rows.Err(); err != nil {
		return err
	}

	seriesOrder := make([]string, 0, len(seriesTotals))
	for sid := range seriesTotals {
		seriesOrder = append(seriesOrder, sid)
	}
	for i := 0; i < len(seriesOrder); i++ {
		for j := i + 1; j < len(seriesOrder); j++ {
			if seriesTotals[seriesOrder[j]] > seriesTotals[seriesOrder[i]] ||
				(seriesTotals[seriesOrder[j]] == seriesTotals[seriesOrder[i]] && seriesOrder[j] < seriesOrder[i]) {
				seriesOrder[i], seriesOrder[j] = seriesOrder[j], seriesOrder[i]
			}
		}
	}

	out.UsageGroupBySelected = selected
	out.UsageTitle = usage.Title
	out.UsageSubtitle = usage.Subtitle
	out.UsageYAxis = usage.YAxis
	out.UsageTodayLabel = usage.TodayLabel
	out.UsageGroupByPrefix = usage.GroupByPrefix
	out.UsageEmpty = usage.Empty
	out.UsageTooltipBreakdown = usage.TooltipBreakdown
	out.UsageTooltipDailyTotal = usage.TooltipDailyTotal
	out.UsageTooltipCumulativeTotal = usage.TooltipCumulativeTotal
	out.UsageTooltipShareFmt = usage.TooltipShareFmt
	out.UsageDays = make([]UsageAxisDay, 0, len(dayKeys))
	for _, day := range dayKeys {
		out.UsageDays = append(out.UsageDays, UsageAxisDay{
			Day:      day,
			DayLabel: subscriptions.FormatUTCChartDay(day),
		})
	}
	if len(seriesOrder) == 0 {
		out.UsageSeries = make([]UsageDaySeries, 0, len(dayKeys))
		for i, day := range dayKeys {
			out.UsageSeries = append(out.UsageSeries, UsageDaySeries{
				Day:         day,
				DayLabel:    subscriptions.FormatUTCChartDay(day),
				SeriesID:    usageAxisOnlySeriesID,
				SeriesLabel: "",
				Tokens:      0,
				DailyTokens: 0,
			})
			out.UsageDays[i].DailyTotal = 0
			out.UsageDays[i].DailyTotalLabel = FormatCompactCount(0)
			out.UsageDays[i].CumTotal = 0
			out.UsageDays[i].CumTotalLabel = FormatCompactCount(0)
		}
		return nil
	}

	cum := map[string]int64{}
	dayDailyTotal := map[string]int64{}
	dayCumTotal := map[string]int64{}
	out.UsageSeries = make([]UsageDaySeries, 0, len(dayKeys)*len(seriesOrder))
	for _, day := range dayKeys {
		var daySum int64
		for _, sid := range seriesOrder {
			dayTok := daily[daySeriesKey{day: day, series: sid}]
			cum[sid] += dayTok
			daySum += dayTok
			label := sid
			if sid == "" {
				label = unknownLabel
			} else if selected == "mode" {
				if ml := subscriptions.EventModeLabel(sid); ml != "" {
					label = ml
				}
			}
			out.UsageSeries = append(out.UsageSeries, UsageDaySeries{
				Day:              day,
				DayLabel:         subscriptions.FormatUTCChartDay(day),
				SeriesID:         sid,
				SeriesLabel:      label,
				Tokens:           cum[sid],
				DailyTokens:      dayTok,
				TokensLabel:      FormatCompactCount(cum[sid]),
				DailyTokensLabel: FormatCompactCount(dayTok),
			})
		}
		var cumSum int64
		for _, sid := range seriesOrder {
			cumSum += cum[sid]
		}
		dayDailyTotal[day] = daySum
		dayCumTotal[day] = cumSum
	}
	for i := range out.UsageDays {
		d := out.UsageDays[i].Day
		out.UsageDays[i].DailyTotal = dayDailyTotal[d]
		out.UsageDays[i].DailyTotalLabel = FormatCompactCount(dayDailyTotal[d])
		out.UsageDays[i].CumTotal = dayCumTotal[d]
		out.UsageDays[i].CumTotalLabel = FormatCompactCount(dayCumTotal[d])
	}

	return nil
}

func buildLocHeatmap(
	ctx context.Context,
	db *pgxpool.Pool,
	userID string,
	heatmapScope string,
	seriesDays int,
	heat HeatmapChrome,
	dayKeys []string,
	out *DashboardCharts,
) error {
	if heat.ScopeAll != "" {
		out.LocHeatmapScopes = append(out.LocHeatmapScopes, NamedOption{ID: "all", Label: heat.ScopeAll})
	}
	if heat.ScopeTab != "" {
		out.LocHeatmapScopes = append(out.LocHeatmapScopes, NamedOption{ID: "tab", Label: heat.ScopeTab})
	}
	if len(out.LocHeatmapScopes) == 0 || heat.Title == "" {
		return nil
	}

	selected := strings.ToLower(strings.TrimSpace(heatmapScope))
	if selected == "" {
		selected = out.LocHeatmapScopes[0].ID
	}
	valid := false
	for _, opt := range out.LocHeatmapScopes {
		if opt.ID == selected {
			valid = true
			break
		}
	}
	if !valid {
		return nil
	}

	var valueFmt, emptyFmt string
	switch selected {
	case "all":
		valueFmt = heat.ValueFmtAll
		emptyFmt = heat.EmptyFmtAll
	case "tab":
		valueFmt = heat.ValueFmtTab
		emptyFmt = heat.EmptyFmtTab
	default:
		return nil
	}
	if valueFmt == "" && emptyFmt == "" {
		return nil
	}

	var valueExpr string
	switch selected {
	case "all":
		valueExpr = "coalesce(sum(ai_lines_added), 0)"
	case "tab":
		valueExpr = "coalesce(sum(tab_suggestions_accepted), 0)"
	default:
		return nil
	}

	userFilter := ""
	args := []any{seriesDays}
	if userID != "" {
		userFilter = " and user_id = $2"
		args = []any{seriesDays, userID}
	}

	q := `
		select to_char(created_at at time zone 'utc', 'YYYY-MM-DD'),
		       ` + valueExpr + `::bigint
		from public.trim_events
		where created_at >= (now() - make_interval(days => $1))` + userFilter + `
		group by 1
		order by 1 asc
	`
	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	byDay := map[string]int64{}
	var total int64
	for rows.Next() {
		var day string
		var v int64
		if err := rows.Scan(&day, &v); err != nil {
			return err
		}
		byDay[day] = v
		total += v
	}
	if err := rows.Err(); err != nil {
		return err
	}

	var maxVal int64
	for _, day := range dayKeys {
		if v := byDay[day]; v > maxVal {
			maxVal = v
		}
	}

	out.LocHeatmap = make([]LocHeatmapDay, 0, len(dayKeys))
	var prevMonth string
	for _, day := range dayKeys {
		v := byDay[day]
		wd := subscriptions.WeekdayMon0(day)
		if wd < 0 {
			wd = 0
		}
		monthLetter := subscriptions.FormatUTCChartMonthLetter(day)
		monthLabel := ""
		if monthLetter != "" && monthLetter != prevMonth {
			monthLabel = monthLetter
			prevMonth = monthLetter
		}
		out.LocHeatmap = append(out.LocHeatmap, LocHeatmapDay{
			Day:          day,
			DayLabelLong: subscriptions.FormatUTCChartDayLong(day),
			WeekdayMon0:  wd,
			MonthLabel:   monthLabel,
			Value:        v,
			Intensity:    heatmapIntensity(v, maxVal),
		})
	}
	out.LocHeatmapTotal = total
	out.LocHeatmapScopeSelected = selected
	out.LocHeatmapTitle = heat.Title
	out.LocHeatmapEmptyFmt = emptyFmt
	out.LocHeatmapValueFmt = valueFmt
	out.LocHeatmapWeekdayLabels = make([]NamedOption, 0, 3)
	if heat.WdMon != "" {
		out.LocHeatmapWeekdayLabels = append(out.LocHeatmapWeekdayLabels, NamedOption{ID: "0", Label: heat.WdMon})
	}
	if heat.WdWed != "" {
		out.LocHeatmapWeekdayLabels = append(out.LocHeatmapWeekdayLabels, NamedOption{ID: "2", Label: heat.WdWed})
	}
	if heat.WdFri != "" {
		out.LocHeatmapWeekdayLabels = append(out.LocHeatmapWeekdayLabels, NamedOption{ID: "4", Label: heat.WdFri})
	}
	out.LocHeatmapStats = buildLocHeatmapStats(dayKeys, byDay, heat)
	return nil
}

func buildLocHeatmapStats(dayKeys []string, byDay map[string]int64, heat HeatmapChrome) []LocHeatmapStat {
	stats := make([]LocHeatmapStat, 0, 4)

	var peakDay string
	var peakVal int64
	monthTotals := map[string]int64{}
	for _, day := range dayKeys {
		v := byDay[day]
		if v > peakVal {
			peakVal = v
			peakDay = day
		}
		if len(day) >= 7 {
			monthTotals[day[:7]] += v
		}
	}

	var peakMonth string
	var peakMonthVal int64
	for ym, v := range monthTotals {
		if v > peakMonthVal || (v == peakMonthVal && (peakMonth == "" || ym < peakMonth)) {
			peakMonthVal = v
			peakMonth = ym
		}
	}

	if heat.StatMostActiveMonth != "" && peakMonthVal > 0 {
		if name := FormatUTCChartMonthName(peakMonth + "-01"); name != "" {
			stats = append(stats, LocHeatmapStat{
				ID:    "most_active_month",
				Label: heat.StatMostActiveMonth,
				Value: name,
			})
		}
	}
	if heat.StatMostActiveDay != "" && peakVal > 0 && peakDay != "" {
		if label := subscriptions.FormatUTCChartDayMedium(peakDay); label != "" {
			stats = append(stats, LocHeatmapStat{
				ID:    "most_active_day",
				Label: heat.StatMostActiveDay,
				Value: label,
			})
		}
	}

	longest, current := heatmapStreaks(dayKeys, byDay)
	if heat.StatLongestStreak != "" && heat.StreakFmt != "" {
		stats = append(stats, LocHeatmapStat{
			ID:    "longest_streak",
			Label: heat.StatLongestStreak,
			Value: applyCountFmt(heat.StreakFmt, longest),
		})
	}
	if heat.StatCurrentStreak != "" && heat.StreakFmt != "" {
		stats = append(stats, LocHeatmapStat{
			ID:    "current_streak",
			Label: heat.StatCurrentStreak,
			Value: applyCountFmt(heat.StreakFmt, current),
		})
	}
	return stats
}

func heatmapStreaks(dayKeys []string, byDay map[string]int64) (longest, current int) {
	run := 0
	for _, day := range dayKeys {
		if byDay[day] > 0 {
			run++
			if run > longest {
				longest = run
			}
		} else {
			run = 0
		}
	}
	for i := len(dayKeys) - 1; i >= 0; i-- {
		if byDay[dayKeys[i]] > 0 {
			current++
		} else {
			break
		}
	}
	return longest, current
}

func applyCountFmt(fmtStr string, count int) string {
	f := strings.TrimSpace(fmtStr)
	if f == "" || !strings.Contains(f, "{count}") {
		return ""
	}
	return strings.ReplaceAll(f, "{count}", strconv.Itoa(count))
}

func heatmapIntensity(value, max int64) int {
	if value <= 0 || max <= 0 {
		return 0
	}
	n := int((value*4 + max - 1) / max)
	if n < 1 {
		return 1
	}
	if n > 4 {
		return 4
	}
	return n
}

// FormatCompactCount formats a non-negative count for chart tooltips (K/M/B).
func FormatCompactCount(n int64) string {
	if n < 0 {
		return "-" + FormatCompactCount(-n)
	}
	switch {
	case n >= 1_000_000_000:
		return trimCompactFloat(float64(n)/1_000_000_000) + "B"
	case n >= 1_000_000:
		return trimCompactFloat(float64(n)/1_000_000) + "M"
	case n >= 1_000:
		return trimCompactFloat(float64(n)/1_000) + "K"
	default:
		return strconv.FormatInt(n, 10)
	}
}

func trimCompactFloat(v float64) string {
	s := fmt.Sprintf("%.1f", v)
	s = strings.TrimSuffix(s, ".0")
	return s
}

// FormatUTCChartMonthName labels YYYY-MM-DD or YYYY-MM as a full English month name.
func FormatUTCChartMonthName(dayOrMonth string) string {
	raw := strings.TrimSpace(dayOrMonth)
	if raw == "" {
		return ""
	}
	if len(raw) == 7 {
		raw = raw + "-01"
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return ""
	}
	return t.UTC().Format("January")
}
