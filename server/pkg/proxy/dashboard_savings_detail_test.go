package proxy_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tidwall/sjson"
	"github.com/usetrim/trim/server/pkg/localchrome"
	"github.com/usetrim/trim/server/pkg/proxy"
)

func TestDashboardShowsSavingsDetailAndDeepStatus(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant","content":[{"type":"text","text":"ok"}]}`))
	}))
	defer up.Close()

	deep := func(optimized []byte, _ string) (proxy.DeepResult, error) {
		out, err := sjson.SetBytes(optimized, "messages.0.content", "tiny")
		if err != nil {
			return proxy.DeepResult{}, err
		}
		return proxy.DeepResult{Body: out, Origin: 500, Compressed: 120, Status: proxy.DeepStatusApplied}, nil
	}

	ch := localchrome.Chrome{
		DocumentTitle:       "Trim local dashboard",
		Brand:               "Trim",
		LeadFmt:             "Active mode: %s.",
		LabelRequests:       "Requests",
		LabelTokensIn:       "Tokens in",
		LabelTokensOut:      "Tokens out",
		LabelSaved:          "Saved",
		LabelEstUSD:         "Est. USD saved",
		LabelLastLatency:    "Last latency",
		LabelLastRequest:    "Last request",
		LastRequestValueFmt: "%d -> %d",
		LabelLastSaved:      "Last saved",
		LabelFallbacks:      "Fallbacks",
		LabelLastDoor:       "Last door",
		LabelDeepStatus:     "Deep status",
		LabelDeepStage:      "Deep stage",
		LabelDeepStageSaved: "Deep stage saved",
		DeepStatusLabels: map[string]string{
			proxy.DeepStatusApplied: "Deep applied",
		},
		SavingsDetailTitle:     "Savings detail",
		SavingsDetailLead:      "Whole-request Saved can look low.",
		SavingsDetailShow:      "Show savings detail",
		SavingsDetailHide:      "Hide savings detail",
		SavingsDetailWireFmt:   "Whole request: {before} → {after} ({pct}% saved)",
		SavingsDetailStageFmt:  "Deep stage: {before} → {after} ({pct}% saved)",
		SavingsDetailStatusFmt: "Status: {status}",
		SavingsDetailChromeTip: "0% on Claude Code chrome is often normal.",
		HeadingBefore:          "Before",
		HeadingAfter:           "After",
		PlaygroundTitle:        "Playground",
		PlaygroundLead:         "lead",
		LabelMode:              "Mode",
		ModeMild:               "mild",
		ModeBalanced:           "balanced",
		ModeAggressive:         "aggressive",
		ModeCustom:             "custom",
		LabelUserPrompt:        "prompt",
		PreviewAction:          "Preview",
		HeadingPreviewIn:       "in",
		HeadingPreviewOut:      "out",
		PreviewIdle:            "idle",
		NoRequestYet:           "none",
		SeriesTitle:            "Series",
		SeriesLead:             "lead",
		FootJSON:               "JSON",
		FootSeries:             "Series",
		FootPreview:            "Preview",
		FootMetrics:            "Metrics",
		FootHealth:             "Health",
		TUIEstSavedFmt:         "$%.4f",
		TUILatencyFmt:          "%.1f ms",
		HtmlLang:               "en",
	}

	srv := proxy.NewServer(proxy.Options{
		UpstreamAnthropic:   up.URL,
		DoorAnthropic:       "anthropic",
		CompressionMode:     "balanced",
		DeepOptimize:        deep,
		UpstreamHTTPTimeout: 5 * time.Second,
		PreviewMaxChars:     800,
		SavingsUsdPerMTok:   3,
		Chrome:              ch,
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	reqBody := `{"model":"claude-haiku","max_tokens":16,"messages":[{"role":"user","content":"hello there verbose filler text for tokens"}]}`
	res, err := http.Post(ts.URL+"/v1/messages", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()
	if res.StatusCode >= 400 {
		t.Fatalf("messages status=%d", res.StatusCode)
	}

	st := srv.Stats()
	if st.LastDeepStatus != proxy.DeepStatusApplied {
		t.Fatalf("last_deep_status=%q", st.LastDeepStatus)
	}
	if st.LastDeepStageBefore != 500 || st.LastDeepStageAfter != 120 {
		t.Fatalf("stage=%d->%d", st.LastDeepStageBefore, st.LastDeepStageAfter)
	}
	if st.LastDeepStageSavedPct < 75 || st.LastDeepStageSavedPct > 77 {
		t.Fatalf("stage saved pct=%v", st.LastDeepStageSavedPct)
	}

	dash, err := http.Get(ts.URL + "/dashboard")
	if err != nil {
		t.Fatal(err)
	}
	defer dash.Body.Close()
	htmlBody, _ := io.ReadAll(io.LimitReader(dash.Body, 200000))
	html := string(htmlBody)
	for _, need := range []string{
		`src="/brand/trim-mark-white.png"`,
		`class="brand-mark"`,
		`value-text`,
		`card-span-2`,
		"Show savings detail",
		"Savings detail",
		"Deep status",
		"Deep stage",
		"Deep applied",
		"0% on Claude Code chrome is often normal.",
		"savingsDetailBtn",
	} {
		if !strings.Contains(html, need) {
			t.Fatalf("dashboard missing %q", need)
		}
	}

	markRes, err := http.Get(ts.URL + "/brand/trim-mark-white.png")
	if err != nil {
		t.Fatal(err)
	}
	defer markRes.Body.Close()
	if markRes.StatusCode != http.StatusOK {
		t.Fatalf("brand mark status=%d", markRes.StatusCode)
	}
	if ct := markRes.Header.Get("Content-Type"); !strings.Contains(ct, "image/png") {
		t.Fatalf("brand mark content-type=%q", ct)
	}

	statsRes, err := http.Get(ts.URL + "/v1/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer statsRes.Body.Close()
	statsBody, _ := io.ReadAll(io.LimitReader(statsRes.Body, 50000))
	s := string(statsBody)
	if !strings.Contains(s, `"last_deep_status":"applied"`) {
		t.Fatalf("stats missing deep status: %s", s)
	}
	if !strings.Contains(s, `"last_deep_stage_before_tokens":500`) {
		t.Fatalf("stats missing stage before: %s", s)
	}
}

func TestDashboardShowsFailClosedDeepStatus(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant","content":[{"type":"text","text":"ok"}]}`))
	}))
	defer up.Close()

	deep := func(optimized []byte, _ string) (proxy.DeepResult, error) {
		// Simulate expansion fail-closed: keep Fast body, report expand status.
		n := (len(optimized) + 3) / 4
		return proxy.DeepResult{
			Body:       optimized,
			Origin:     n,
			Compressed: n + 50,
			Status:     proxy.DeepStatusFailClosedExpand,
		}, nil
	}

	ch := localchrome.Chrome{
		DocumentTitle: "Trim", Brand: "Trim", LeadFmt: "mode %s",
		LabelRequests: "Requests", LabelTokensIn: "Tokens in", LabelTokensOut: "Tokens out",
		LabelSaved: "Saved", LabelEstUSD: "USD", LabelLastLatency: "Latency",
		LabelLastRequest: "Last", LastRequestValueFmt: "%d -> %d", LabelLastSaved: "Last saved",
		LabelFallbacks: "Fallbacks", LabelLastDoor: "Door",
		LabelDeepStatus: "Deep status", LabelDeepStage: "Deep stage", LabelDeepStageSaved: "Deep stage saved",
		DeepStatusLabels: map[string]string{
			proxy.DeepStatusFailClosedExpand: "Deep kept Fast (expansion fail-closed)",
		},
		SavingsDetailTitle: "Savings detail", SavingsDetailLead: "lead",
		SavingsDetailShow: "Show savings detail", SavingsDetailHide: "Hide",
		SavingsDetailWireFmt: "Whole request: {before} → {after} ({pct}% saved)",
		SavingsDetailStageFmt: "Deep stage: {before} → {after} ({pct}% saved)",
		SavingsDetailStatusFmt: "Status: {status}",
		SavingsDetailChromeTip: "0% chrome normal",
		HeadingBefore: "Before", HeadingAfter: "After",
		PlaygroundTitle: "Playground", PlaygroundLead: "lead", LabelMode: "Mode",
		ModeMild: "mild", ModeBalanced: "balanced", ModeAggressive: "aggressive", ModeCustom: "custom",
		LabelUserPrompt: "prompt", PreviewAction: "Preview",
		HeadingPreviewIn: "in", HeadingPreviewOut: "out", PreviewIdle: "idle", NoRequestYet: "none",
		SeriesTitle: "Series", SeriesLead: "lead",
		FootJSON: "JSON", FootSeries: "Series", FootPreview: "Preview", FootMetrics: "Metrics", FootHealth: "Health",
		TUIEstSavedFmt: "$%.4f", TUILatencyFmt: "%.1f ms", HtmlLang: "en",
	}

	srv := proxy.NewServer(proxy.Options{
		UpstreamAnthropic:   up.URL,
		DoorAnthropic:       "anthropic",
		CompressionMode:     "balanced",
		DeepOptimize:        deep,
		UpstreamHTTPTimeout: 5 * time.Second,
		PreviewMaxChars:     800,
		SavingsUsdPerMTok:   3,
		Chrome:              ch,
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	reqBody := `{"model":"claude-haiku","max_tokens":16,"messages":[{"role":"user","content":"enough text for a normal request body"}]}`
	res, err := http.Post(ts.URL+"/v1/messages", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()
	if res.StatusCode >= 400 {
		t.Fatalf("messages status=%d", res.StatusCode)
	}

	st := srv.Stats()
	if st.LastDeepStatus != proxy.DeepStatusFailClosedExpand {
		t.Fatalf("last_deep_status=%q", st.LastDeepStatus)
	}

	dash, err := http.Get(ts.URL + "/dashboard")
	if err != nil {
		t.Fatal(err)
	}
	defer dash.Body.Close()
	htmlBody, _ := io.ReadAll(io.LimitReader(dash.Body, 200000))
	html := string(htmlBody)
	if !strings.Contains(html, "Deep kept Fast (expansion fail-closed)") {
		t.Fatalf("dashboard missing fail-closed label: %s", html)
	}
	if !strings.Contains(html, "Show savings detail") {
		t.Fatal("dashboard missing Show savings detail")
	}

	statsRes, err := http.Get(ts.URL + "/v1/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer statsRes.Body.Close()
	statsBody, _ := io.ReadAll(io.LimitReader(statsRes.Body, 50000))
	if !strings.Contains(string(statsBody), `"last_deep_status":"fail_closed_expand"`) {
		t.Fatalf("stats missing fail_closed_expand: %s", statsBody)
	}
}
