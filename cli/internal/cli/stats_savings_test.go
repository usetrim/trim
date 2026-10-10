package cli

import "testing"

func TestFormatTokenPair(t *testing.T) {
	if got := formatTokenPair("%d -> %d", 10, 4); got != "10 -> 4" {
		t.Fatalf("got %q", got)
	}
	if got := formatTokenPair("", 3, 1); got != "3 -> 1" {
		t.Fatalf("empty fmt got %q", got)
	}
}

func TestPrintLiveSavingsClarityParses(t *testing.T) {
	body := []byte(`{
		"last_before_tokens":100,
		"last_after_tokens":100,
		"last_door":"anthropic",
		"last_deep_status":"skipped_empty",
		"last_deep_stage_before_tokens":0,
		"last_deep_stage_after_tokens":0,
		"last_deep_stage_saved_percent":0,
		"chrome":{"deep_status_labels":{"skipped_empty":"Deep skipped (nothing compressible / chrome frozen)"},"last_request_value_fmt":"%d -> %d"}
	}`)
	chrome := cliChrome{
		StatsDeepStatusFmt:    "Deep: %s",
		StatsDeepStageFmt:     "Deep stage: %s (%.1f%%)",
		StatsLastRequestFmt:   "Last request: %s (%.1f%%)",
		StatsDoorFmt:          "Door: %s",
		StatsDashboardTipFmt:  "Meter: http://127.0.0.1:{port}/dashboard → Show savings detail",
	}
	// Smoke: must not panic on valid JSON.
	printLiveSavingsClarity(chrome, "8888", body)
}

func TestPrintLiveSavingsClarityFailClosedLabel(t *testing.T) {
	body := []byte(`{
		"last_before_tokens":200,
		"last_after_tokens":200,
		"last_door":"anthropic",
		"last_deep_status":"fail_closed_expand",
		"last_deep_stage_before_tokens":180,
		"last_deep_stage_after_tokens":220,
		"last_deep_stage_saved_percent":0,
		"chrome":{
			"deep_status_labels":{"fail_closed_expand":"Deep kept Fast (expansion fail-closed)"},
			"last_request_value_fmt":"%d -> %d"
		}
	}`)
	chrome := cliChrome{
		StatsDeepStatusFmt:   "Deep: %s",
		StatsDeepStageFmt:    "Deep stage: %s (%.1f%%)",
		StatsLastRequestFmt:  "Last request: %s (%.1f%%)",
		StatsDoorFmt:         "Door: %s",
		StatsDashboardTipFmt: "Meter: http://127.0.0.1:{port}/dashboard → Show savings detail (0%% tip)",
	}
	printLiveSavingsClarity(chrome, "8888", body)
}
