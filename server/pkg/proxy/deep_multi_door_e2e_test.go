package proxy_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/usetrim/trim/server/pkg/localchrome"
	"github.com/usetrim/trim/server/pkg/provideradapt"
	"github.com/usetrim/trim/server/pkg/proxy"
)

// Offline multi-door Deep E2E: proves Fast+Deep intercept preserves scaffolding for
// Anthropic /v1/messages and OpenAI-compat hosts (GPT/Gemini/DeepSeek/Mistral shape).
// Uses a mock DeepOptimize (no LLMLingua) so CI stays deterministic.
func TestProxyLiveDeepPreservesScaffoldingMultiDoor(t *testing.T) {
	var deepCalls atomic.Int32

	mockDeep := func(optimized []byte, path string) (proxy.DeepResult, error) {
		deepCalls.Add(1)
		msgs := gjson.GetBytes(optimized, "messages")
		if !msgs.IsArray() || len(msgs.Array()) == 0 {
			n := (len(optimized) + 3) / 4
			return proxy.DeepResult{Body: optimized, Origin: n, Compressed: n, Status: proxy.DeepStatusSkippedEmpty}, nil
		}
		arr := msgs.Array()
		last := arr[len(arr)-1]
		role := strings.ToLower(last.Get("role").String())
		if role != "user" {
			n := (len(optimized) + 3) / 4
			return proxy.DeepResult{Body: optimized, Origin: n, Compressed: n, Status: proxy.DeepStatusSkippedEmpty}, nil
		}
		// Refuse pure tool-only trailing turns (mirrors ShouldSkipDeepForChatTurn).
		// Official Anthropic: [tool_result, text] still Deep-rewrites text only.
		if last.Get("tool_call_id").Exists() {
			n := (len(optimized) + 3) / 4
			return proxy.DeepResult{Body: optimized, Origin: n, Compressed: n, Status: proxy.DeepStatusSkippedEmpty}, nil
		}
		if content := last.Get("content"); content.IsArray() {
			hasText := false
			for _, part := range content.Array() {
				if part.Get("type").String() == "text" &&
					!strings.Contains(part.Get("text").String(), "<system-reminder>") &&
					strings.TrimSpace(part.Get("text").String()) != "" {
					hasText = true
					break
				}
			}
			if !hasText {
				n := (len(optimized) + 3) / 4
				return proxy.DeepResult{Body: optimized, Origin: n, Compressed: n, Status: proxy.DeepStatusSkippedEmpty}, nil
			}
		}
		content := last.Get("content")
		idx := len(arr) - 1
		var out []byte
		var err error
		if content.IsArray() {
			// Keep non-text / reminder / tool_result / signed / cached parts; replace first compressible text.
			replaced := false
			for j, part := range content.Array() {
				if part.Get("type").String() != "text" {
					continue
				}
				txt := part.Get("text").String()
				if strings.Contains(txt, "<system-reminder>") {
					continue
				}
				// Parity with deepopt.contentPartFrozenForDeep / Fast freeze (all doors).
				if part.Get("cache_control").Exists() ||
					part.Get("prompt_cache_breakpoint").Exists() ||
					part.Get("citations").Exists() ||
					part.Get("annotations").Exists() ||
					strings.EqualFold(part.Get("type").String(), "refusal") ||
					strings.TrimSpace(part.Get("thought_signature").String()) != "" ||
					strings.TrimSpace(part.Get("thoughtSignature").String()) != "" ||
					strings.TrimSpace(part.Get("extra_content.google.thought_signature").String()) != "" {
					continue
				}
				p := "messages." + itoa(idx) + ".content." + itoa(j) + ".text"
				out, err = sjson.SetBytes(optimized, p, "DEEP_OK")
				replaced = true
				break
			}
			if !replaced {
				n := (len(optimized) + 3) / 4
				return proxy.DeepResult{Body: optimized, Origin: n, Compressed: n, Status: proxy.DeepStatusSkippedEmpty}, nil
			}
		} else {
			out, err = sjson.SetBytes(optimized, "messages."+itoa(idx)+".content", "DEEP_OK")
		}
		if err != nil {
			return proxy.DeepResult{}, err
		}
		return proxy.DeepResult{Body: out, Origin: 40, Compressed: 8, Status: proxy.DeepStatusApplied}, nil
	}

	t.Run("anthropic_count_tokens_skips_deep", func(t *testing.T) {
		var saw []byte
		up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			saw, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"input_tokens":123}`))
		}))
		defer up.Close()

		before := deepCalls.Load()
		srv := proxy.NewServer(proxy.Options{
			UpstreamAnthropic:   up.URL,
			DoorAnthropic:       "anthropic",
			CompressionMode:     "balanced",
			DeepOptimize:        mockDeep,
			UpstreamHTTPTimeout: 5 * time.Second,
			Chrome:              localchrome.Chrome{ErrBodyRead: "body"},
		})
		ts := httptest.NewServer(srv.Handler())
		defer ts.Close()

		body := `{
			"model":"claude-opus-4",
			"messages":[
				{"role":"user","content":"FILLER A\nFILLER B\nwhat is 2+2?"}
			]
		}`
		res, err := http.Post(ts.URL+"/v1/messages/count_tokens", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode >= 400 {
			b, _ := io.ReadAll(res.Body)
			t.Fatalf("status=%d body=%s", res.StatusCode, b)
		}
		if deepCalls.Load() != before {
			t.Fatal("DeepOptimize must not run on count_tokens (Fast-only companion RPC)")
		}
		if strings.Contains(string(saw), "DEEP_OK") {
			t.Fatalf("count_tokens body must not be Deep-rewritten: %s", string(saw))
		}
	})

	t.Run("anthropic_messages_door", func(t *testing.T) {
		var saw []byte
		up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			saw, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"msg_1","role":"assistant","content":[{"type":"text","text":"4"}],"stop_reason":"end_turn"}`))
		}))
		defer up.Close()

		srv := proxy.NewServer(proxy.Options{
			UpstreamAnthropic:   up.URL,
			DoorAnthropic:       "anthropic",
			CompressionMode:     "balanced",
			DeepOptimize:        mockDeep,
			UpstreamHTTPTimeout: 5 * time.Second,
			Chrome:              localchrome.Chrome{ErrBodyRead: "body"},
		})
		ts := httptest.NewServer(srv.Handler())
		defer ts.Close()

		body := `{
			"model":"claude-opus-4",
			"system":"# Environment\nhuge agent chrome KEEP",
			"messages":[
				{"role":"user","content":[
					{"type":"text","text":"<system-reminder>\nkeep me\n</system-reminder>\n"},
					{"type":"text","text":"FILLER line A\nFILLER line B\nwhat is 2+2?"}
				]}
			]
		}`
		res, err := http.Post(ts.URL+"/v1/messages", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode >= 400 {
			b, _ := io.ReadAll(res.Body)
			t.Fatalf("status=%d body=%s", res.StatusCode, b)
		}
		if !strings.Contains(gjson.GetBytes(saw, "system").Raw, "KEEP") &&
			!strings.Contains(gjson.GetBytes(saw, "system").String(), "KEEP") {
			t.Fatalf("system chrome lost: %s", string(saw))
		}
		parts := gjson.GetBytes(saw, "messages.0.content").Array()
		if len(parts) < 2 {
			t.Fatalf("user parts=%s", string(saw))
		}
		if !strings.Contains(parts[0].Get("text").String(), "<system-reminder>") {
			t.Fatalf("reminder lost: %s", parts[0].Raw)
		}
		if parts[1].Get("text").String() != "DEEP_OK" {
			t.Fatalf("expected Deep rewrite of compressible text only, got %s", parts[1].Raw)
		}
	})

	t.Run("anthropic_messages_preserves_input_schema_tools", func(t *testing.T) {
		// Official Anthropic: tools use input_schema. Chat function wrapping must never run on this door.
		var saw []byte
		up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			saw, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"msg_t","role":"assistant","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn"}`))
		}))
		defer up.Close()

		srv := proxy.NewServer(proxy.Options{
			UpstreamAnthropic:   up.URL,
			DoorAnthropic:       "anthropic",
			CompressionMode:     "balanced",
			DeepOptimize:        mockDeep,
			UpstreamHTTPTimeout: 5 * time.Second,
			Chrome:              localchrome.Chrome{ErrBodyRead: "body"},
		})
		ts := httptest.NewServer(srv.Handler())
		defer ts.Close()

		body := `{
			"model":"claude-opus-4",
			"max_tokens":256,
			"tools":[
				{"name":"Bash","description":"run","input_schema":{"type":"object","properties":{}}},
				{"type":"bash_20250124","name":"bash"}
			],
			"messages":[{"role":"user","content":"FILLER A\nFILLER B\nwhat is 2+2?"}]
		}`
		res, err := http.Post(ts.URL+"/v1/messages", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode >= 400 {
			b, _ := io.ReadAll(res.Body)
			t.Fatalf("status=%d body=%s", res.StatusCode, b)
		}
		if gjson.GetBytes(saw, "tools.0.function").Exists() {
			t.Fatalf("native Anthropic tools smashed into Chat function: %s", string(saw))
		}
		if gjson.GetBytes(saw, "tools.0.input_schema.type").String() != "object" {
			t.Fatalf("input_schema lost: %s", string(saw))
		}
		if gjson.GetBytes(saw, "tools.1.type").String() != "bash_20250124" {
			t.Fatalf("dated server tool smashed: %s", string(saw))
		}
	})

	t.Run("openai_compat_gemini_shape", func(t *testing.T) {
		var saw []byte
		up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			saw, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"chat_1","choices":[{"message":{"role":"assistant","content":"4"}}]}`))
		}))
		defer up.Close()

		reg, err := provideradapt.NewRegistry(
			[]provideradapt.AdapterConfig{{
				ID:                   "gemini",
				Enabled:              true,
				SortOrder:            20,
				Dialect:              provideradapt.DialectOpenAICompat,
				DoorLabel:            "gemini",
				MatchModelPrefixes:   []string{"gemini-", "trim-gemini-"},
				UpstreamKind:         provideradapt.UpstreamOpenAI,
				UpstreamBaseURL:      up.URL,
				UpstreamPath:         "/v1/chat/completions",
				AuthMode:             provideradapt.AuthPassthrough,
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
			ProviderAdapters:    reg,
			DoorOpenAI:          "openai",
			CompressionMode:     "balanced",
			DeepOptimize:        mockDeep,
			UpstreamHTTPTimeout: 5 * time.Second,
			Chrome:              localchrome.Chrome{ErrBodyRead: "body"},
		})
		ts := httptest.NewServer(srv.Handler())
		defer ts.Close()

		body := `{
			"model":"gemini-flash-latest",
			"tools":[{"type":"function","function":{"name":"lookup","parameters":{"type":"object"}}}],
			"messages":[
				{"role":"system","content":"SYS_KEEP"},
				{"role":"developer","content":"DEV_KEEP"},
				{"role":"user","content":"old"},
				{"role":"assistant","content":"ok"},
				{"role":"user","content":"FILLER A\nFILLER B\nwhat is 2+2?"}
			]
		}`
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/chat/completions", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer gemini-key")
		req.Header.Set("Content-Type", "application/json")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode >= 400 {
			b, _ := io.ReadAll(res.Body)
			t.Fatalf("status=%d body=%s", res.StatusCode, b)
		}
		if gjson.GetBytes(saw, "messages.0.content").String() != "SYS_KEEP" {
			t.Fatalf("system lost: %s", string(saw))
		}
		if gjson.GetBytes(saw, "messages.1.content").String() != "DEV_KEEP" {
			t.Fatalf("developer lost: %s", string(saw))
		}
		if gjson.GetBytes(saw, "tools.0.function.name").String() != "lookup" {
			t.Fatalf("tools lost: %s", string(saw))
		}
		last := gjson.GetBytes(saw, "messages").Array()
		if last[len(last)-1].Get("content").String() != "DEEP_OK" {
			t.Fatalf("last user not Deep-rewritten: %s", string(saw))
		}
	})

	t.Run("openai_tool_loop_skips_deep_rewrite", func(t *testing.T) {
		before := deepCalls.Load()
		var saw []byte
		up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			saw, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"chat_2","choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
		}))
		defer up.Close()

		reg, err := provideradapt.NewRegistry(
			[]provideradapt.AdapterConfig{{
				ID:                 "deepseek",
				Enabled:            true,
				SortOrder:          30,
				Dialect:            provideradapt.DialectOpenAICompat,
				DoorLabel:          "deepseek",
				MatchModelPrefixes: []string{"deepseek-"},
				UpstreamKind:       provideradapt.UpstreamOpenAI,
				UpstreamBaseURL:    up.URL,
				UpstreamPath:       "/v1/chat/completions",
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
			ProviderAdapters:    reg,
			DoorOpenAI:          "openai",
			CompressionMode:     "balanced",
			DeepOptimize:        mockDeep,
			UpstreamHTTPTimeout: 5 * time.Second,
			Chrome:              localchrome.Chrome{ErrBodyRead: "body"},
		})
		ts := httptest.NewServer(srv.Handler())
		defer ts.Close()

		body := `{
			"model":"deepseek-chat",
			"messages":[
				{"role":"user","content":"weather?"},
				{"role":"assistant","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"get_weather","arguments":"{}"}}]},
				{"role":"tool","tool_call_id":"c1","content":"18C"}
			]
		}`
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/chat/completions", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer ds-key")
		req.Header.Set("Content-Type", "application/json")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		_ = res
		if deepCalls.Load() <= before {
			t.Fatal("DeepOptimize should still be invoked by proxy (policy skip is inside callback)")
		}
		if !gjson.GetBytes(saw, "messages.1.tool_calls").Exists() {
			t.Fatalf("tool_calls lost: %s", string(saw))
		}
		if gjson.GetBytes(saw, "messages.2.role").String() != "tool" {
			t.Fatalf("tool result lost: %s", string(saw))
		}
		// Mid-tool-loop: mock Deep must not rewrite tool content to DEEP_OK.
		if strings.Contains(string(saw), "DEEP_OK") {
			t.Fatalf("Deep must not rewrite mid-tool-loop: %s", string(saw))
		}
	})

	t.Run("anthropic_tool_result_plus_text_deep", func(t *testing.T) {
		var saw []byte
		up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			saw, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"msg_2","role":"assistant","content":[{"type":"text","text":"4"}],"stop_reason":"end_turn"}`))
		}))
		defer up.Close()

		srv := proxy.NewServer(proxy.Options{
			UpstreamAnthropic:   up.URL,
			DoorAnthropic:       "anthropic",
			CompressionMode:     "balanced",
			DeepOptimize:        mockDeep,
			UpstreamHTTPTimeout: 5 * time.Second,
			Chrome:              localchrome.Chrome{ErrBodyRead: "body"},
		})
		ts := httptest.NewServer(srv.Handler())
		defer ts.Close()

		body := `{
			"model":"claude-opus-4",
			"messages":[
				{"role":"assistant","content":[{"type":"tool_use","id":"1","name":"Read","input":{}}]},
				{"role":"user","content":[
					{"type":"tool_result","tool_use_id":"1","content":"TOOL_KEEP"},
					{"type":"text","text":"FILLER A\nFILLER B\nwhat is 2+2?"}
				]}
			]
		}`
		res, err := http.Post(ts.URL+"/v1/messages", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode >= 400 {
			b, _ := io.ReadAll(res.Body)
			t.Fatalf("status=%d body=%s", res.StatusCode, b)
		}
		parts := gjson.GetBytes(saw, "messages.1.content").Array()
		if len(parts) < 2 {
			t.Fatalf("parts=%s", string(saw))
		}
		if parts[0].Get("type").String() != "tool_result" || !strings.Contains(parts[0].Raw, "TOOL_KEEP") {
			t.Fatalf("tool_result must stay: %s", parts[0].Raw)
		}
		if parts[1].Get("text").String() != "DEEP_OK" {
			t.Fatalf("text after tool_result must Deep-rewrite: %s", parts[1].Raw)
		}
	})

	t.Run("responses_byok_tool_name_and_reasoning", func(t *testing.T) {
		var saw []byte
		up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			saw, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"chat_3","choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
		}))
		defer up.Close()

		reg, err := provideradapt.NewRegistry(
			[]provideradapt.AdapterConfig{{
				ID:                 "mistral",
				Enabled:            true,
				SortOrder:          40,
				Dialect:            provideradapt.DialectOpenAICompat,
				DoorLabel:          "mistral",
				MatchModelPrefixes: []string{"mistral-"},
				UpstreamKind:       provideradapt.UpstreamOpenAI,
				UpstreamBaseURL:    up.URL,
				UpstreamPath:       "/v1/chat/completions",
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
			ProviderAdapters:    reg,
			DoorOpenAI:          "openai",
			CompressionMode:     "balanced",
			DeepOptimize:        mockDeep,
			UpstreamHTTPTimeout: 5 * time.Second,
			Chrome:              localchrome.Chrome{ErrBodyRead: "body"},
		})
		ts := httptest.NewServer(srv.Handler())
		defer ts.Close()

		// Cursor BYOK Responses shape → Chat Completions for Mistral/GPT/Gemini/DeepSeek.
		body := `{
			"model":"mistral-large-latest",
			"input":[
				{"type":"message","role":"user","content":"weather?"},
				{"type":"function_call","call_id":"c1","name":"get_weather","arguments":"{}"},
				{"type":"function_call_output","call_id":"c1","output":"18C"},
				{"type":"reasoning","summary":[{"type":"summary_text","text":"reason keep"}]},
				{"type":"message","role":"user","content":"FILLER\nwhat is 2+2?"}
			]
		}`
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/chat/completions", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer mist-key")
		req.Header.Set("Content-Type", "application/json")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode >= 400 {
			b, _ := io.ReadAll(res.Body)
			t.Fatalf("status=%d body=%s", res.StatusCode, b)
		}
		if gjson.GetBytes(saw, "input").Exists() {
			t.Fatalf("input must be normalized away: %s", string(saw))
		}
		if gjson.GetBytes(saw, "messages.2.role").String() != "tool" {
			t.Fatalf("tool msg missing: %s", string(saw))
		}
		if gjson.GetBytes(saw, "messages.2.name").String() != "get_weather" {
			t.Fatalf("Mistral tool name not enriched: %s", string(saw))
		}
		if gjson.GetBytes(saw, "messages.1.reasoning_content").String() != "reason keep" {
			t.Fatalf("DeepSeek-style reasoning_content lost: %s", string(saw))
		}
		last := gjson.GetBytes(saw, "messages").Array()
		if last[len(last)-1].Get("content").String() != "DEEP_OK" {
			t.Fatalf("trailing user not Deep-rewritten: %s", string(saw))
		}
	})

	t.Run("responses_parallel_tool_calls_one_assistant", func(t *testing.T) {
		var saw []byte
		up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			saw, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"chat_4","choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
		}))
		defer up.Close()

		reg, err := provideradapt.NewRegistry(
			[]provideradapt.AdapterConfig{{
				ID:                 "gemini",
				Enabled:            true,
				SortOrder:          20,
				Dialect:            provideradapt.DialectOpenAICompat,
				DoorLabel:          "gemini",
				MatchModelPrefixes: []string{"gemini-"},
				UpstreamKind:       provideradapt.UpstreamOpenAI,
				UpstreamBaseURL:    up.URL,
				UpstreamPath:       "/v1/chat/completions",
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
			ProviderAdapters:    reg,
			DoorOpenAI:          "openai",
			CompressionMode:     "balanced",
			DeepOptimize:        mockDeep,
			UpstreamHTTPTimeout: 5 * time.Second,
			Chrome:              localchrome.Chrome{ErrBodyRead: "body"},
		})
		ts := httptest.NewServer(srv.Handler())
		defer ts.Close()

		body := `{
			"model":"gemini-flash-latest",
			"input":[
				{"type":"message","role":"user","content":"temps?"},
				{"type":"function_call","call_id":"c1","name":"get_temperature","arguments":"{\"c\":\"Paris\"}",
				 "extra_content":{"google":{"thought_signature":"SIG_A"}}},
				{"type":"function_call","call_id":"c2","name":"get_temperature","arguments":"{\"c\":\"London\"}"},
				{"type":"function_call_output","call_id":"c1","output":"18"},
				{"type":"function_call_output","call_id":"c2","output":"14"},
				{"type":"message","role":"user","content":"FILLER\nwhat is 2+2?"}
			]
		}`
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/chat/completions", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer gem-key")
		req.Header.Set("Content-Type", "application/json")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode >= 400 {
			b, _ := io.ReadAll(res.Body)
			t.Fatalf("status=%d body=%s", res.StatusCode, b)
		}
		asstCalls := gjson.GetBytes(saw, "messages.1.tool_calls").Array()
		if len(asstCalls) != 2 {
			t.Fatalf("parallel calls not merged into one assistant: %s", string(saw))
		}
		if asstCalls[0].Get("extra_content.google.thought_signature").String() != "SIG_A" {
			t.Fatalf("Gemini first-call signature lost: %s", string(saw))
		}
		// Must not interleave assistant/tool/assistant/tool (Gemini 400).
		roles := []string{}
		for _, m := range gjson.GetBytes(saw, "messages").Array() {
			roles = append(roles, m.Get("role").String())
		}
		joined := strings.Join(roles, ",")
		if strings.Contains(joined, "assistant,tool,assistant,tool") {
			t.Fatalf("interleaved parallel tool loop: %s raw=%s", joined, string(saw))
		}
		last := gjson.GetBytes(saw, "messages").Array()
		if last[len(last)-1].Get("content").String() != "DEEP_OK" {
			t.Fatalf("trailing Deep rewrite lost: %s", string(saw))
		}
	})
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
