package proxy_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tidwall/gjson"
	"github.com/usetrim/trim/server/pkg/provideradapt"
	"github.com/usetrim/trim/server/pkg/proxy"
)

// Offline E2E through the real proxy server: OpenAI chat → Anthropic adapter path
// (mock Anthropic upstream). Registry is explicit config; no invent routing.
func TestProxyOpenAIToAnthropicAdapterE2E(t *testing.T) {
	var sawPath string
	var sawAPIKey string
	var sawVersion string
	var sawWorkspace string
	var sawBody []byte

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawPath = r.URL.Path
		sawAPIKey = r.Header.Get("x-api-key")
		sawVersion = r.Header.Get("anthropic-version")
		sawWorkspace = r.Header.Get("anthropic-workspace-id")
		var err error
		sawBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read upstream body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"msg_e2e",
			"model":"claude-sonnet-5-5",
			"role":"assistant",
			"content":[{"type":"text","text":"pong"}],
			"stop_reason":"end_turn",
			"usage":{"input_tokens":3,"output_tokens":1}
		}`))
	}))
	defer upstream.Close()

	reg, err := provideradapt.NewRegistry(
		[]provideradapt.AdapterConfig{{
			ID:                 "anthropic",
			Enabled:            true,
			SortOrder:          10,
			Dialect:            provideradapt.DialectAnthropicMessages,
			DoorLabel:          "openai_to_anthropic",
			MatchModelPrefixes: []string{"claude-", "anthropic/", "trim-claude-"},
			ModelAliases: map[string]string{
				"trim-claude-sonnet": "claude-sonnet-5-5",
			},
			UpstreamKind:         provideradapt.UpstreamAnthropic,
			UpstreamPath:         "/v1/messages",
			AuthMode:             provideradapt.AuthBearerToXAPIKey,
			AnthropicVersion:     "2023-06-01",
			DefaultMaxTokens:     256,
			RequireAlias:         false,
		}},
		[]provideradapt.ModelAlias{},
		[]provideradapt.DiscoverableModel{},
		provideradapt.Chrome{},
	)
	if err != nil {
		t.Fatal(err)
	}

	srv := proxy.NewServer(proxy.Options{
		UpstreamOpenAI:         "http://127.0.0.1:9", // must not be hit for Claude model
		UpstreamAnthropic:      upstream.URL,
		ProviderAdapters:       reg,
		AnthropicWorkspaceID:   "wrkspc_01TestFromEnvAAAAAAAAAAAAAAA",
		DoorOpenAI:             "openai",
		DoorAnthropic:          "anthropic",
		CompressionMode:        "balanced",
		SavingsUsdPerMTok:      1,
		PreviewMaxChars:        120,
		UpstreamHTTPTimeout:    5 * time.Second,
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	reqBody := `{"model":"trim-claude-sonnet","messages":[{"role":"user","content":"ping"}],"max_tokens":64}`
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer sk-ant-test")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	out, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("status=%d body=%s", res.StatusCode, out)
	}
	if door := res.Header.Get("X-Trim-Door"); door != "openai_to_anthropic" {
		t.Fatalf("X-Trim-Door=%q", door)
	}
	if !strings.HasSuffix(sawPath, "/v1/messages") {
		t.Fatalf("upstream path=%q", sawPath)
	}
	if sawAPIKey != "sk-ant-test" {
		t.Fatalf("x-api-key=%q", sawAPIKey)
	}
	if sawVersion != "2023-06-01" {
		t.Fatalf("anthropic-version=%q", sawVersion)
	}
	if sawWorkspace != "wrkspc_01TestFromEnvAAAAAAAAAAAAAAA" {
		t.Fatalf("anthropic-workspace-id=%q", sawWorkspace)
	}
	if gjson.GetBytes(sawBody, "model").String() != "claude-sonnet-5-5" {
		t.Fatalf("upstream model=%s body=%s", gjson.GetBytes(sawBody, "model").String(), sawBody)
	}
	if gjson.GetBytes(out, "choices.0.message.content").String() != "pong" {
		t.Fatalf("client body=%s", out)
	}
	st := srv.Stats()
	if st.LastDoor != "openai_to_anthropic" {
		t.Fatalf("stats last_door=%q", st.LastDoor)
	}
	if st.AdapterRequests < 1 {
		t.Fatalf("adapter_requests=%d", st.AdapterRequests)
	}
}

// Stream + tool_use round-trip through the proxy adapter path (mock Anthropic SSE).
func TestProxyOpenAIToAnthropicAdapterStreamToolsE2E(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !gjson.GetBytes(body, "stream").Bool() {
			t.Errorf("expected stream=true body=%s", body)
		}
		if !gjson.GetBytes(body, "tools").IsArray() {
			t.Errorf("expected tools body=%s", body)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("no flusher")
		}
		events := []string{
			`data: {"type":"message_start","message":{"id":"msg_s","model":"claude-sonnet-5-5","role":"assistant","content":[]}}`,
			`data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_1","name":"lookup","input":{}}}`,
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"q\":\"x\"}"}}`,
			`data: {"type":"content_block_stop","index":0}`,
			`data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":4}}`,
			`data: {"type":"message_stop"}`,
		}
		for _, e := range events {
			_, _ = io.WriteString(w, e+"\n\n")
			flusher.Flush()
		}
	}))
	defer upstream.Close()

	reg, err := provideradapt.NewRegistry(
		[]provideradapt.AdapterConfig{{
			ID:                 "anthropic",
			Enabled:            true,
			SortOrder:          10,
			Dialect:            provideradapt.DialectAnthropicMessages,
			DoorLabel:          "openai_to_anthropic",
			MatchModelPrefixes: []string{"claude-"},
			ModelAliases:       map[string]string{},
			UpstreamKind:       provideradapt.UpstreamAnthropic,
			UpstreamPath:       "/v1/messages",
			AuthMode:           provideradapt.AuthBearerToXAPIKey,
			AnthropicVersion:   "2023-06-01",
			DefaultMaxTokens:   256,
			RequireAlias:       false,
		}},
		[]provideradapt.ModelAlias{},
		[]provideradapt.DiscoverableModel{},
		provideradapt.Chrome{},
	)
	if err != nil {
		t.Fatal(err)
	}

	srv := proxy.NewServer(proxy.Options{
		UpstreamAnthropic:   upstream.URL,
		ProviderAdapters:    reg,
		DoorOpenAI:          "openai",
		SavingsUsdPerMTok:   1,
		PreviewMaxChars:     80,
		UpstreamHTTPTimeout: 5 * time.Second,
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	reqBody := `{
		"model":"claude-sonnet-5-5",
		"stream":true,
		"messages":[{"role":"user","content":"use tool"}],
		"tools":[{"type":"function","function":{"name":"lookup","parameters":{"type":"object","properties":{"q":{"type":"string"}}}}}],
		"max_tokens":64
	}`
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer sk-ant-test")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	out, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("status=%d body=%s", res.StatusCode, out)
	}
	if door := res.Header.Get("X-Trim-Door"); door != "openai_to_anthropic" {
		t.Fatalf("X-Trim-Door=%q", door)
	}
	s := string(out)
	if !strings.Contains(s, `"tool_calls"`) {
		t.Fatalf("missing tool_calls in stream: %s", s)
	}
	if !strings.Contains(s, "lookup") {
		t.Fatalf("missing tool name in stream: %s", s)
	}
	if !strings.Contains(s, "data: [DONE]") {
		t.Fatalf("missing DONE: %s", s)
	}
}

// GPT openai_compat must hit DB upstream_base_url, not UPSTREAM_OPENAI_URL (Gemini).
func TestProxyOpenAICompatGPTHostRouteE2E(t *testing.T) {
	var hitHost string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitHost = r.Host
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path=%s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_t","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer upstream.Close()

	reg, err := provideradapt.NewRegistry(
		[]provideradapt.AdapterConfig{{
			ID:                 "openai",
			Enabled:            true,
			SortOrder:          20,
			Dialect:            provideradapt.DialectOpenAICompat,
			DoorLabel:          "openai_compat_openai",
			MatchModelPrefixes: []string{"gpt-"},
			ModelAliases:       map[string]string{},
			UpstreamKind:       provideradapt.UpstreamOpenAI,
			UpstreamPath:       "/v1/chat/completions",
			UpstreamBaseURL:    upstream.URL,
			AuthMode:           provideradapt.AuthPassthrough,
		}},
		[]provideradapt.ModelAlias{},
		[]provideradapt.DiscoverableModel{},
		provideradapt.Chrome{},
	)
	if err != nil {
		t.Fatal(err)
	}

	srv := proxy.NewServer(proxy.Options{
		UpstreamOpenAI:      "http://127.0.0.1:9", // must not be hit for gpt-*
		ProviderAdapters:    reg,
		DoorOpenAI:          "openai",
		SavingsUsdPerMTok:   1,
		PreviewMaxChars:     80,
		UpstreamHTTPTimeout: 5 * time.Second,
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	reqBody := `{"model":"gpt-4.1","messages":[{"role":"user","content":"x"}],"max_tokens":4}`
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer sk-openai-test")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 {
		t.Fatalf("status=%d body=%s", res.StatusCode, out)
	}
	if door := res.Header.Get("X-Trim-Door"); door != "openai_compat_openai" {
		t.Fatalf("X-Trim-Door=%q", door)
	}
	wantHost := strings.TrimPrefix(strings.TrimPrefix(upstream.URL, "http://"), "https://")
	if hitHost != wantHost {
		t.Fatalf("hitHost=%q want=%q", hitHost, wantHost)
	}
	st := srv.Stats()
	if st.LastDoor != "openai_compat_openai" {
		t.Fatalf("last_door=%q", st.LastDoor)
	}
}

// When adapters are synced, unmatched model ids must not silently invent a host.
func TestProxyUnknownModelNoAdapterMatch(t *testing.T) {
	reg, err := provideradapt.NewRegistry(
		[]provideradapt.AdapterConfig{{
			ID:                 "openai",
			Enabled:            true,
			SortOrder:          20,
			Dialect:            provideradapt.DialectOpenAICompat,
			DoorLabel:          "openai_compat_openai",
			MatchModelPrefixes: []string{"gpt-"},
			ModelAliases:       map[string]string{},
			UpstreamKind:       provideradapt.UpstreamOpenAI,
			UpstreamPath:       "/v1/chat/completions",
			UpstreamBaseURL:    "http://127.0.0.1:9",
			AuthMode:           provideradapt.AuthPassthrough,
		}},
		[]provideradapt.ModelAlias{},
		[]provideradapt.DiscoverableModel{},
		provideradapt.Chrome{},
	)
	if err != nil {
		t.Fatal(err)
	}
	srv := proxy.NewServer(proxy.Options{
		UpstreamOpenAI:       "http://127.0.0.1:9",
		ProviderAdapters:     reg,
		UnknownModelFmt:      "No provider adapter matched model %q.",
		DoorOpenAI:           "openai",
		SavingsUsdPerMTok:    1,
		PreviewMaxChars:      80,
		UpstreamHTTPTimeout:  2 * time.Second,
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	reqBody := `{"model":"totally-unknown-model-xyz","messages":[{"role":"user","content":"x"}],"max_tokens":4}`
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer sk-x")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", res.StatusCode, out)
	}
	if !strings.Contains(string(out), "totally-unknown-model-xyz") {
		t.Fatalf("body=%s", out)
	}
}

func TestProxyModelNotFoundChromeRemap(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"not_found_error","message":"model: claude-opus-5-5"}}`))
	}))
	defer upstream.Close()

	reg, err := provideradapt.NewRegistry(
		[]provideradapt.AdapterConfig{{
			ID:                 "anthropic",
			Enabled:            true,
			SortOrder:          10,
			Dialect:            provideradapt.DialectAnthropicMessages,
			DoorLabel:          "openai_to_anthropic",
			MatchModelPrefixes: []string{"claude-", "trim-claude-"},
			ModelAliases:       map[string]string{"trim-claude-opus": "claude-opus-5-5"},
			UpstreamKind:       provideradapt.UpstreamAnthropic,
			UpstreamPath:       "/v1/messages",
			AuthMode:           provideradapt.AuthBearerToXAPIKey,
			AnthropicVersion:   "2023-06-01",
			DefaultMaxTokens:   256,
		}},
		[]provideradapt.ModelAlias{},
		[]provideradapt.DiscoverableModel{},
		provideradapt.Chrome{},
	)
	if err != nil {
		t.Fatal(err)
	}
	srv := proxy.NewServer(proxy.Options{
		UpstreamOpenAI:       "http://127.0.0.1:9",
		UpstreamAnthropic:    upstream.URL,
		ProviderAdapters:     reg,
		ModelNotFoundFmt:     `Upstream rejected model %q (not found or not allowed for this API key).`,
		DoorOpenAI:           "openai",
		DoorAnthropic:        "anthropic",
		SavingsUsdPerMTok:    1,
		PreviewMaxChars:      80,
		UpstreamHTTPTimeout:  5 * time.Second,
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	reqBody := `{"model":"trim-claude-opus","messages":[{"role":"user","content":"x"}],"max_tokens":4}`
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer sk-ant-test")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", res.StatusCode, out)
	}
	if !strings.Contains(string(out), "Upstream rejected model") || !strings.Contains(string(out), "claude-opus-5-5") {
		t.Fatalf("expected remapped chrome, body=%s", out)
	}
}

func TestProxyUpstreamAuthChromeRemap(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`))
	}))
	defer upstream.Close()

	reg, err := provideradapt.NewRegistry(
		[]provideradapt.AdapterConfig{{
			ID:                 "anthropic",
			Enabled:            true,
			SortOrder:          10,
			Dialect:            provideradapt.DialectAnthropicMessages,
			DoorLabel:          "openai_to_anthropic",
			MatchModelPrefixes: []string{"claude-"},
			ModelAliases:       map[string]string{},
			UpstreamKind:       provideradapt.UpstreamAnthropic,
			UpstreamPath:       "/v1/messages",
			AuthMode:           provideradapt.AuthBearerToXAPIKey,
			AnthropicVersion:   "2023-06-01",
			DefaultMaxTokens:   256,
		}},
		[]provideradapt.ModelAlias{},
		[]provideradapt.DiscoverableModel{},
		provideradapt.Chrome{},
	)
	if err != nil {
		t.Fatal(err)
	}
	srv := proxy.NewServer(proxy.Options{
		UpstreamOpenAI:      "http://127.0.0.1:9",
		UpstreamAnthropic:   upstream.URL,
		ProviderAdapters:    reg,
		UpstreamAuthFmt:     `Upstream rejected the API key for model %q (authentication failed).`,
		DoorOpenAI:          "openai",
		DoorAnthropic:       "anthropic",
		SavingsUsdPerMTok:   1,
		PreviewMaxChars:     80,
		UpstreamHTTPTimeout: 5 * time.Second,
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	reqBody := `{"model":"claude-haiku-4-5-20251001","messages":[{"role":"user","content":"x"}],"max_tokens":4}`
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer sk-ant-bad")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", res.StatusCode, out)
	}
	if !strings.Contains(string(out), "authentication failed") || !strings.Contains(string(out), "claude-haiku-4-5-20251001") {
		t.Fatalf("expected auth chrome, body=%s", out)
	}
}

func TestProxyUpstreamQuotaChromeRemap(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"Rate limit reached","type":"rate_limit_error","code":"rate_limit_exceeded"}}`))
	}))
	defer upstream.Close()

	reg, err := provideradapt.NewRegistry(
		[]provideradapt.AdapterConfig{{
			ID:                 "openai",
			Enabled:            true,
			SortOrder:          20,
			Dialect:            provideradapt.DialectOpenAICompat,
			DoorLabel:          "openai_compat",
			MatchModelPrefixes: []string{"gpt-"},
			ModelAliases:       map[string]string{},
			UpstreamKind:       provideradapt.UpstreamOpenAI,
			UpstreamPath:       "/v1/chat/completions",
			UpstreamBaseURL:    upstream.URL,
			AuthMode:           provideradapt.AuthPassthrough,
		}},
		[]provideradapt.ModelAlias{},
		[]provideradapt.DiscoverableModel{},
		provideradapt.Chrome{},
	)
	if err != nil {
		t.Fatal(err)
	}
	srv := proxy.NewServer(proxy.Options{
		UpstreamOpenAI:      "http://127.0.0.1:9",
		UpstreamAnthropic:   "http://127.0.0.1:9",
		ProviderAdapters:    reg,
		UpstreamQuotaFmt:    `Upstream rate limit or quota exhausted for model %q.`,
		DoorOpenAI:          "openai",
		DoorAnthropic:       "anthropic",
		SavingsUsdPerMTok:   1,
		PreviewMaxChars:     80,
		UpstreamHTTPTimeout: 5 * time.Second,
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	reqBody := `{"model":"gpt-4o","messages":[{"role":"user","content":"x"}]}`
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer sk-test")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status=%d body=%s", res.StatusCode, out)
	}
	if !strings.Contains(string(out), "quota exhausted") || !strings.Contains(string(out), "gpt-4o") {
		t.Fatalf("expected quota chrome, body=%s", out)
	}
}

func TestProxyUpstreamUnavailableHTMLChromeRemap(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(522)
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><title>mistral.ai | 522: Connection timed out</title><body>cloudflare error</body></html>`))
	}))
	defer upstream.Close()

	reg, err := provideradapt.NewRegistry(
		[]provideradapt.AdapterConfig{{
			ID:                 "mistral",
			Enabled:            true,
			SortOrder:          50,
			Dialect:            provideradapt.DialectOpenAICompat,
			DoorLabel:          "openai_compat",
			MatchModelPrefixes: []string{"mistral-"},
			ModelAliases:       map[string]string{},
			UpstreamKind:       provideradapt.UpstreamOpenAI,
			UpstreamPath:       "/v1/chat/completions",
			UpstreamBaseURL:    upstream.URL,
			AuthMode:           provideradapt.AuthPassthrough,
		}},
		[]provideradapt.ModelAlias{},
		[]provideradapt.DiscoverableModel{},
		provideradapt.Chrome{},
	)
	if err != nil {
		t.Fatal(err)
	}
	srv := proxy.NewServer(proxy.Options{
		UpstreamOpenAI:           "http://127.0.0.1:9",
		UpstreamAnthropic:        "http://127.0.0.1:9",
		ProviderAdapters:         reg,
		UpstreamUnavailableFmt:   `Upstream provider is temporarily unavailable for model %q (gateway/host error).`,
		DoorOpenAI:               "openai",
		DoorAnthropic:            "anthropic",
		SavingsUsdPerMTok:        1,
		PreviewMaxChars:          80,
		UpstreamHTTPTimeout:      5 * time.Second,
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	reqBody := `{"model":"mistral-small-latest","messages":[{"role":"user","content":"x"}]}`
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer sk-test")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	if res.StatusCode != 522 {
		t.Fatalf("status=%d body=%s", res.StatusCode, out)
	}
	if !strings.Contains(string(out), "temporarily unavailable") || !strings.Contains(string(out), "mistral-small-latest") {
		t.Fatalf("expected unavailable chrome, body=%s", out)
	}
	if strings.Contains(string(out), "<html") {
		t.Fatalf("raw HTML should be remapped, body=%s", out)
	}
}

func TestProxyUpstreamModelsAdapterProxy(t *testing.T) {
	var sawAuth string
	var sawOrg string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" && !strings.HasSuffix(r.URL.Path, "/models") {
			t.Errorf("path=%s", r.URL.Path)
		}
		sawAuth = r.Header.Get("Authorization")
		sawOrg = r.Header.Get("OpenAI-Organization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"gpt-4o","object":"model"}]}`))
	}))
	defer upstream.Close()

	reg, err := provideradapt.NewRegistry(
		[]provideradapt.AdapterConfig{{
			ID:                 "openai",
			Enabled:            true,
			SortOrder:          20,
			Dialect:            provideradapt.DialectOpenAICompat,
			DoorLabel:          "openai_compat_openai",
			MatchModelPrefixes: []string{"gpt-"},
			ModelAliases:       map[string]string{},
			UpstreamKind:       provideradapt.UpstreamOpenAI,
			UpstreamPath:       "/v1/chat/completions",
			UpstreamBaseURL:    upstream.URL + "/v1",
			AuthMode:           provideradapt.AuthPassthrough,
		}},
		[]provideradapt.ModelAlias{},
		[]provideradapt.DiscoverableModel{{
			ID: "gpt-4o", Enabled: true, SortOrder: 1, OwnedBy: "openai", Created: 1,
		}},
		provideradapt.Chrome{},
	)
	if err != nil {
		t.Fatal(err)
	}
	srv := proxy.NewServer(proxy.Options{
		UpstreamOpenAI:     "http://127.0.0.1:9",
		ProviderAdapters:   reg,
		OpenAIOrganization: "org-from-env",
		DoorOpenAI:         "openai",
		SavingsUsdPerMTok:  1,
		PreviewMaxChars:    80,
		UpstreamHTTPTimeout: 5 * time.Second,
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/v1/models?adapter=openai", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer sk-live")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 {
		t.Fatalf("status=%d body=%s", res.StatusCode, out)
	}
	if sawAuth != "Bearer sk-live" {
		t.Fatalf("auth=%q", sawAuth)
	}
	if sawOrg != "org-from-env" {
		t.Fatalf("org=%q", sawOrg)
	}
	if !strings.Contains(string(out), "gpt-4o") {
		t.Fatalf("body=%s", out)
	}
}
