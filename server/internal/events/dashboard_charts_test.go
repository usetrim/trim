package events

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestHeatmapStreaks(t *testing.T) {
	days := []string{
		"2026-01-01", "2026-01-02", "2026-01-03", "2026-01-04", "2026-01-05",
		"2026-01-06", "2026-01-07",
	}
	byDay := map[string]int64{
		"2026-01-01": 2,
		"2026-01-02": 1,
		"2026-01-03": 0,
		"2026-01-04": 4,
		"2026-01-05": 3,
		"2026-01-06": 1,
		"2026-01-07": 5,
	}
	longest, current := heatmapStreaks(days, byDay)
	if longest != 4 {
		t.Fatalf("longest=%d want 4", longest)
	}
	if current != 4 {
		t.Fatalf("current=%d want 4", current)
	}
}

func TestHeatmapStreaksBrokenCurrent(t *testing.T) {
	days := []string{"2026-01-01", "2026-01-02", "2026-01-03"}
	byDay := map[string]int64{"2026-01-01": 1, "2026-01-02": 2, "2026-01-03": 0}
	longest, current := heatmapStreaks(days, byDay)
	if longest != 2 {
		t.Fatalf("longest=%d want 2", longest)
	}
	if current != 0 {
		t.Fatalf("current=%d want 0", current)
	}
}

func TestBuildLocHeatmapStats(t *testing.T) {
	days := []string{
		"2026-01-01", "2026-01-02", "2026-02-01", "2026-02-02", "2026-02-03",
	}
	byDay := map[string]int64{
		"2026-01-01": 1,
		"2026-01-02": 1,
		"2026-02-01": 10,
		"2026-02-02": 0,
		"2026-02-03": 2,
	}
	heat := HeatmapChrome{
		StatMostActiveMonth: "Most Active Month",
		StatMostActiveDay:   "Most Active Day",
		StatLongestStreak:   "Longest Streak",
		StatCurrentStreak:   "Current Streak",
		StreakFmt:           "{count}d",
	}
	stats := buildLocHeatmapStats(days, byDay, heat)
	if len(stats) != 4 {
		t.Fatalf("stats len=%d want 4: %+v", len(stats), stats)
	}
	byID := map[string]LocHeatmapStat{}
	for _, s := range stats {
		byID[s.ID] = s
	}
	if byID["most_active_month"].Value != "February" {
		t.Fatalf("month=%q want February", byID["most_active_month"].Value)
	}
	if byID["most_active_day"].Value != "Feb 1, 2026" {
		t.Fatalf("day=%q want Feb 1, 2026", byID["most_active_day"].Value)
	}
	if byID["longest_streak"].Value != "3d" {
		t.Fatalf("longest=%q want 3d", byID["longest_streak"].Value)
	}
	if byID["current_streak"].Value != "1d" {
		t.Fatalf("current=%q want 1d", byID["current_streak"].Value)
	}
}

func TestBuildLocHeatmapStatsFailClosedWithoutChrome(t *testing.T) {
	days := []string{"2026-01-01", "2026-01-02"}
	byDay := map[string]int64{"2026-01-01": 3, "2026-01-02": 1}
	stats := buildLocHeatmapStats(days, byDay, HeatmapChrome{})
	if len(stats) != 0 {
		t.Fatalf("expected empty stats without chrome, got %+v", stats)
	}
}

func TestFormatCompactCount(t *testing.T) {
	cases := map[int64]string{
		0:             "0",
		999:           "999",
		1000:          "1K",
		1500:          "1.5K",
		1_000_000:     "1M",
		2_500_000:     "2.5M",
		1_000_000_000: "1B",
	}
	for n, want := range cases {
		if got := FormatCompactCount(n); got != want {
			t.Fatalf("FormatCompactCount(%d)=%q want %q", n, got, want)
		}
	}
}

func TestStatsResponseJSONIncludesTooltipAndStats(t *testing.T) {
	out := StatsResponse{
		UsageTooltipBreakdown:       "Daily breakdown",
		UsageTooltipDailyTotal:      "Daily total",
		UsageTooltipCumulativeTotal: "Cumulative total",
		UsageTooltipShareFmt:        "{pct}%",
		LocHeatmapStats: []LocHeatmapStat{
			{ID: "longest_streak", Label: "Longest Streak", Value: "3d"},
		},
	}
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"usage_tooltip_breakdown",
		"usage_tooltip_daily_total",
		"usage_tooltip_cumulative_total",
		"usage_tooltip_share_fmt",
		"loc_heatmap_stats",
	} {
		if _, ok := raw[key]; !ok {
			t.Fatalf("missing json key %s in %s", key, string(b))
		}
	}
}

func TestTabScopeUsesAcceptsNotLines(t *testing.T) {
	src, err := os.ReadFile("dashboard_charts.go")
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	marker := "var valueExpr string"
	idx := strings.Index(body, marker)
	if idx < 0 {
		t.Fatal("missing valueExpr switch")
	}
	snippet := body[idx:]
	if end := strings.Index(snippet, "userFilter :="); end > 0 {
		snippet = snippet[:end]
	}
	tabIdx := strings.Index(snippet, `case "tab":`)
	if tabIdx < 0 {
		t.Fatal(`missing case "tab" in valueExpr switch`)
	}
	tabSnippet := snippet[tabIdx:]
	if end := strings.Index(tabSnippet, "default:"); end > 0 {
		tabSnippet = tabSnippet[:end]
	}
	if !strings.Contains(tabSnippet, "tab_suggestions_accepted") {
		t.Fatalf("tab scope must use tab_suggestions_accepted; got %q", tabSnippet)
	}
	if strings.Contains(tabSnippet, "ai_lines_added") || strings.Contains(tabSnippet, "ai_lines_deleted") {
		t.Fatalf("tab scope must not use ai_lines_*; got %q", tabSnippet)
	}
}
