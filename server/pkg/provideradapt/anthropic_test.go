package provideradapt_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
	"github.com/usetrim/trim/server/pkg/provideradapt"
)

func TestAnthropicTranslateRoundTrip(t *testing.T) {
	cfg := provideradapt.AdapterConfig{
		ID:                 "anthropic",
		Enabled:            true,
		SortOrder:          10,
		Dialect:            provideradapt.DialectAnthropicMessages,
		DoorLabel:          "openai_to_anthropic",
		MatchModelPrefixes: []string{"claude-", "anthropic/"},
		ModelAliases:       map[string]string{"anthropic/claude-sonnet-4": "claude-sonnet-4-20250514"},
		UpstreamKind:       provideradapt.UpstreamAnthropic,
		UpstreamPath:       "/v1/messages",
		AuthMode:           provideradapt.AuthBearerToXAPIKey,
		AnthropicVersion:   "2023-06-01",
		DefaultMaxTokens:   4096,
		RequireAlias:       false,
	}
	reg, err := provideradapt.NewRegistry([]provideradapt.AdapterConfig{cfg}, []provideradapt.ModelAlias{}, []provideradapt.DiscoverableModel{}, provideradapt.Chrome{})
	if err != nil {
		t.Fatal(err)
	}
	matched, ok := reg.Match("anthropic/claude-sonnet-4")
	if !ok {
		t.Fatal("expected match")
	}
	resolved, err := reg.ResolveUpstreamModel(matched, "anthropic/claude-sonnet-4")
	if err != nil {
		t.Fatal(err)
	}
	if resolved != "claude-sonnet-4-20250514" {
		t.Fatalf("resolved=%s", resolved)
	}
	in := []byte(`{
		"model":"anthropic/claude-sonnet-4",
		"messages":[
			{"role":"system","content":"be brief"},
			{"role":"user","content":"hi"}
		],
		"max_tokens":128
	}`)
	path, out, err := provideradapt.TranslateRequest(cfg, in, resolved)
	if err != nil {
		t.Fatal(err)
	}
	if path != "/v1/messages" {
		t.Fatalf("path=%s", path)
	}
	if gjson.GetBytes(out, "model").String() != resolved {
		t.Fatalf("model=%s", gjson.GetBytes(out, "model").String())
	}
	if gjson.GetBytes(out, "system").String() != "be brief" {
		t.Fatalf("system=%s", gjson.GetBytes(out, "system").String())
	}
	if gjson.GetBytes(out, "max_tokens").Int() != 128 {
		t.Fatalf("max_tokens=%d", gjson.GetBytes(out, "max_tokens").Int())
	}
	if len(gjson.GetBytes(out, "messages").Array()) != 1 {
		t.Fatalf("messages=%s", out)
	}

	anthResp := []byte(`{
		"id":"msg_1",
		"model":"claude-sonnet-4-20250514",
		"role":"assistant",
		"content":[{"type":"text","text":"hello"}],
		"stop_reason":"end_turn",
		"usage":{"input_tokens":10,"output_tokens":2}
	}`)
	oa, err := provideradapt.TranslateResponseBody(cfg, anthResp, 200)
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(oa, "choices.0.message.content").String() != "hello" {
		t.Fatalf("content=%s", oa)
	}
	if gjson.GetBytes(oa, "choices.0.finish_reason").String() != "stop" {
		t.Fatalf("finish=%s", oa)
	}
}

func TestTranslateRequestForwardsAnthropicPassthroughFields(t *testing.T) {
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 4096,
	}
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"metadata":{"user_id":"user-abc"},
		"container":"cont_123",
		"mcp_servers":[{"type":"url","name":"box","url":"https://example.com"}],
		"service_tier":"auto",
		"thinking":{"type":"enabled","budget_tokens":1024},
		"diagnostics":{"previous_message_id":"msg_prev_1"},
		"fallbacks":"default",
		"fallback_credit_token":"credit_tok_abc",
		"speed":"fast",
		"inference_geo":"us",
		"response_format":{"type":"json_schema","json_schema":{"name":"ans","schema":{"type":"object","properties":{"n":{"type":"number"}}}}},
		"messages":[{"role":"user","content":"hi"}]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out, "metadata.user_id").String() != "user-abc" {
		t.Fatalf("metadata lost: %s", string(out))
	}
	if gjson.GetBytes(out, "container").String() != "cont_123" {
		t.Fatalf("container lost: %s", string(out))
	}
	if gjson.GetBytes(out, "mcp_servers.0.name").String() != "box" {
		t.Fatalf("mcp_servers lost: %s", string(out))
	}
	if gjson.GetBytes(out, "service_tier").String() != "auto" {
		t.Fatalf("service_tier lost: %s", string(out))
	}
	if gjson.GetBytes(out, "thinking.type").String() != "enabled" {
		t.Fatalf("thinking lost: %s", string(out))
	}
	if gjson.GetBytes(out, "diagnostics.previous_message_id").String() != "msg_prev_1" {
		t.Fatalf("diagnostics lost: %s", string(out))
	}
	if gjson.GetBytes(out, "fallbacks").String() != "default" {
		t.Fatalf("fallbacks lost: %s", string(out))
	}
	if gjson.GetBytes(out, "fallback_credit_token").String() != "credit_tok_abc" {
		t.Fatalf("fallback_credit_token lost: %s", string(out))
	}
	if gjson.GetBytes(out, "speed").String() != "fast" {
		t.Fatalf("speed lost: %s", string(out))
	}
	if gjson.GetBytes(out, "inference_geo").String() != "us" {
		t.Fatalf("inference_geo lost: %s", string(out))
	}
	if gjson.GetBytes(out, "output_config.format.type").String() != "json_schema" {
		t.Fatalf("response_format not mapped to output_config: %s", string(out))
	}
	if gjson.GetBytes(out, "output_config.format.schema.properties.n.type").String() != "number" {
		t.Fatalf("json_schema lost: %s", string(out))
	}
}

func TestTranslateRequestPreservesToolResultCacheControl(t *testing.T) {
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 4096,
	}
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"messages":[
			{"role":"assistant","content":null,"tool_calls":[
				{"id":"c1","type":"function","function":{"name":"lookup","arguments":"{}"},"cache_control":{"type":"ephemeral"}}
			]},
			{"role":"tool","tool_call_id":"c1","content":"TOOL_OK","cache_control":{"type":"ephemeral","ttl":"1h"},"is_error":false}
		]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) < 2 {
		t.Fatalf("messages=%s", string(out))
	}
	asst := msgs[0].Get("content").Array()
	foundToolUse := false
	for _, p := range asst {
		if p.Get("type").String() == "tool_use" {
			foundToolUse = true
			if p.Get("cache_control.type").String() != "ephemeral" {
				t.Fatalf("tool_use cache_control lost: %s", p.Raw)
			}
		}
	}
	if !foundToolUse {
		t.Fatalf("tool_use missing: %s", msgs[0].Raw)
	}
	userParts := msgs[1].Get("content").Array()
	if len(userParts) < 1 || userParts[0].Get("type").String() != "tool_result" {
		t.Fatalf("tool_result missing: %s", msgs[1].Raw)
	}
	if userParts[0].Get("cache_control.ttl").String() != "1h" {
		t.Fatalf("tool_result cache_control lost: %s", userParts[0].Raw)
	}
	if userParts[0].Get("is_error").Bool() {
		t.Fatalf("is_error should be false: %s", userParts[0].Raw)
	}
	if userParts[0].Get("content").String() != "TOOL_OK" && !strings.Contains(userParts[0].Raw, "TOOL_OK") {
		t.Fatalf("tool content lost: %s", userParts[0].Raw)
	}
}

func TestTranslateRequestPreservesToolAndTopLevelCacheControl(t *testing.T) {
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 4096,
	}
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"cache_control":{"type":"ephemeral"},
		"tools":[{
			"type":"function",
			"function":{"name":"lookup","description":"d","parameters":{"type":"object"}},
			"cache_control":{"type":"ephemeral","ttl":"1h"}
		}],
		"messages":[
			{"role":"user","content":[
				{"type":"text","text":"PREFIX keep","prompt_cache_breakpoint":{"mode":"explicit"}},
				{"type":"text","text":"what is 2+2?"}
			]}
		]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out, "cache_control.type").String() != "ephemeral" {
		t.Fatalf("top-level cache_control lost: %s", string(out))
	}
	if gjson.GetBytes(out, "tools.0.cache_control.ttl").String() != "1h" {
		t.Fatalf("tool cache_control lost: %s", string(out))
	}
	parts := gjson.GetBytes(out, "messages.0.content").Array()
	if len(parts) < 2 {
		t.Fatalf("user parts=%d raw=%s", len(parts), string(out))
	}
	if parts[0].Get("cache_control.type").String() != "ephemeral" {
		t.Fatalf("user text prompt_cache_breakpoint not mapped: %s", parts[0].Raw)
	}
	if parts[0].Get("text").String() != "PREFIX keep" {
		t.Fatalf("user text mutated: %s", parts[0].Raw)
	}
}

func TestTranslateRequestMapsFlatInputSchemaTools(t *testing.T) {
	// Defense-in-depth: Chat door may still carry Anthropic-shaped {name,input_schema}.
	// Adapter must map input_schema → Anthropic input_schema (not empty object).
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 4096,
	}
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"tools":[{
			"name":"lookup",
			"description":"d",
			"input_schema":{"type":"object","properties":{"q":{"type":"string"}},"required":["q"]}
		}],
		"messages":[{"role":"user","content":"hi"}]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out, "tools.0.name").String() != "lookup" {
		t.Fatalf("tool name lost: %s", string(out))
	}
	if gjson.GetBytes(out, "tools.0.input_schema.properties.q.type").String() != "string" {
		t.Fatalf("flat input_schema lost in adapter: %s", string(out))
	}
}

func TestTranslateRequestToolChoiceAllowedToolsAndAnthropicPassthrough(t *testing.T) {
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 4096,
	}
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"tools":[{"type":"function","function":{"name":"add","parameters":{"type":"object"}}}],
		"tool_choice":{"type":"allowed_tools","mode":"auto","tools":[{"type":"function","name":"add"}]},
		"messages":[{"role":"user","content":"hi"}]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out, "tool_choice.type").String() != "auto" {
		t.Fatalf("allowed_tools must map to auto: %s", string(out))
	}
	in2 := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"tools":[{"type":"function","function":{"name":"add","parameters":{"type":"object"}}}],
		"tool_choice":{"type":"any","disable_parallel_tool_use":true},
		"messages":[{"role":"user","content":"hi"}]
	}`)
	_, out2, err := provideradapt.TranslateRequest(cfg, in2, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out2, "tool_choice.type").String() != "any" {
		t.Fatalf("Anthropic any must passthrough: %s", string(out2))
	}
	if !gjson.GetBytes(out2, "tool_choice.disable_parallel_tool_use").Bool() {
		t.Fatalf("disable_parallel_tool_use lost: %s", string(out2))
	}
}

func TestTranslateRequestSkipsUnknownHostedToolsKeepsFunctions(t *testing.T) {
	// Mixed Chat body: Responses hosted tool + valid function. Must not 400 the whole
	// openai_to_anthropic request (flexibility for GPT/Cursor → Claude).
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 4096,
	}
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"tools":[
			{"type":"web_search_preview"},
			{"type":"function","function":{"name":"add","description":"d","parameters":{"type":"object"}}},
			{"type":"file_search","vector_store_ids":["vs_1"]}
		],
		"messages":[{"role":"user","content":"hi"}]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	tools := gjson.GetBytes(out, "tools").Array()
	if len(tools) != 1 {
		t.Fatalf("want only function tool, got %d: %s", len(tools), string(out))
	}
	if gjson.GetBytes(out, "tools.0.name").String() != "add" {
		t.Fatalf("function tool lost: %s", string(out))
	}
}

func TestTranslateRequestSkipsCustomToolsKeepsFunctions(t *testing.T) {
	// Custom Chat tools must not 400 the whole openai_to_anthropic request.
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 4096,
	}
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"tools":[
			{"type":"custom","custom":{"name":"code_exec","description":"run"}},
			{"type":"function","function":{"name":"add","description":"d","parameters":{"type":"object"}}},
			{"type":"bash_20250124","name":"bash"}
		],
		"tool_choice":{"type":"custom","custom":{"name":"code_exec"}},
		"messages":[
			{"role":"user","content":"run"},
			{"role":"assistant","content":null,"tool_calls":[
				{"id":"call_c1","type":"custom","custom":{"name":"code_exec","input":"print(1)"}},
				{"id":"call_f1","type":"function","function":{"name":"add","arguments":"{\"a\":1}"}}
			]}
		]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	tools := gjson.GetBytes(out, "tools").Array()
	if len(tools) != 2 {
		t.Fatalf("want function+dated server tool, got %d: %s", len(tools), string(out))
	}
	if gjson.GetBytes(out, "tools.0.name").String() != "add" {
		t.Fatalf("function tool lost: %s", string(out))
	}
	if gjson.GetBytes(out, "tools.1.type").String() != "bash_20250124" {
		t.Fatalf("dated server tool lost: %s", string(out))
	}
	if gjson.GetBytes(out, "tool_choice.type").String() != "auto" {
		t.Fatalf("custom tool_choice must fall back to auto: %s", string(out))
	}
	// Custom tool_call skipped; function tool_call becomes tool_use.
	asst := gjson.GetBytes(out, "messages.1.content").Array()
	foundAdd, foundEmptyCustom := false, false
	for _, b := range asst {
		if b.Get("type").String() == "tool_use" {
			if b.Get("name").String() == "add" {
				foundAdd = true
			}
			if b.Get("name").String() == "" || b.Get("name").String() == "code_exec" {
				foundEmptyCustom = true
			}
		}
	}
	if !foundAdd {
		t.Fatalf("function tool_use missing: %s", string(out))
	}
	if foundEmptyCustom {
		t.Fatalf("custom tool_call must not become tool_use: %s", string(out))
	}
}

func TestTranslateRequestSkipsEmptyNameToolCallAndResult(t *testing.T) {
	// Empty-name function tool_calls are skipped (no valid tool_use). Matching tool
	// results must also be skipped or Anthropic returns unexpected tool_use_id.
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 4096,
	}
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"tools":[{"type":"function","function":{"name":"add","parameters":{"type":"object"}}}],
		"messages":[
			{"role":"user","content":"x"},
			{"role":"assistant","content":null,"tool_calls":[
				{"id":"bad","type":"function","function":{"name":"","arguments":"{}"}},
				{"id":"good","type":"function","function":{"name":"add","arguments":"{}"}}
			]},
			{"role":"tool","tool_call_id":"bad","content":"nope"},
			{"role":"tool","tool_call_id":"good","content":"ok"},
			{"role":"user","content":"next"}
		]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	wire := string(out)
	if strings.Contains(wire, `"tool_use_id":"bad"`) || strings.Contains(wire, "nope") {
		t.Fatalf("empty-name tool_result must be skipped: %s", wire)
	}
	if !strings.Contains(wire, "good") || !strings.Contains(wire, "ok") {
		t.Fatalf("valid function tool_result must stay: %s", wire)
	}
}

func TestTranslateRequestSkipsOrphanCustomToolResultsAndEmptyTools(t *testing.T) {
	// Official Anthropic: every tool_result must match a prior tool_use id.
	// Skipping custom tool_use while keeping role=tool → HTTP 400 unexpected tool_use_id.
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 4096,
	}
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"tools":[
			{"type":"custom","custom":{"name":"code_exec","description":"run"}},
			{"type":"function","function":{"name":"add","description":"d","parameters":{"type":"object"}}}
		],
		"messages":[
			{"role":"user","content":"run"},
			{"role":"assistant","content":null,"tool_calls":[
				{"id":"call_c1","type":"custom","custom":{"name":"code_exec","input":"print(1)"}},
				{"id":"call_f1","type":"function","function":{"name":"add","arguments":"{\"a\":1}"}}
			]},
			{"role":"tool","tool_call_id":"call_c1","content":"1"},
			{"role":"tool","tool_call_id":"call_f1","content":"2"},
			{"role":"user","content":"thanks"}
		]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	wire := string(out)
	if strings.Contains(wire, "call_c1") {
		t.Fatalf("custom tool_result must be skipped (orphan tool_use_id): %s", wire)
	}
	if !strings.Contains(wire, "call_f1") {
		t.Fatalf("function tool_result must stay: %s", wire)
	}
	// Pure-custom tools body must omit tools[] entirely (not emit empty array).
	in2 := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"tools":[{"type":"custom","custom":{"name":"code_exec"}}],
		"tool_choice":{"type":"custom","custom":{"name":"code_exec"}},
		"messages":[{"role":"user","content":"hi"}]
	}`)
	_, out2, err := provideradapt.TranslateRequest(cfg, in2, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out2, "tools").Exists() {
		t.Fatalf("all-custom tools must omit tools key, got: %s", string(out2))
	}
	if gjson.GetBytes(out2, "tool_choice").Exists() {
		t.Fatalf("tool_choice without tools must be omitted: %s", string(out2))
	}
}

func TestTranslateRequestPreservesSystemCacheBreakpoints(t *testing.T) {
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 4096,
	}
	// OpenAI developer parts with prompt_cache_breakpoint must become Anthropic system
	// TextBlockParam[] with cache_control (official Messages prompt caching).
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"messages":[
			{"role":"developer","content":[
				{"type":"text","text":"STABLE_SYS keep","prompt_cache_breakpoint":{"mode":"explicit"}},
				{"type":"text","text":"more sys"}
			]},
			{"role":"user","content":"hi"}
		]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	sys := gjson.GetBytes(out, "system")
	if !sys.IsArray() {
		t.Fatalf("system must be TextBlockParam array when cache chrome present: %s", string(out))
	}
	blocks := sys.Array()
	if len(blocks) < 2 {
		t.Fatalf("system blocks=%d raw=%s", len(blocks), string(out))
	}
	if blocks[0].Get("text").String() != "STABLE_SYS keep" {
		t.Fatalf("block0 text=%s", blocks[0].Raw)
	}
	if blocks[0].Get("cache_control.type").String() != "ephemeral" {
		t.Fatalf("prompt_cache_breakpoint not mapped to cache_control: %s", blocks[0].Raw)
	}
	if blocks[1].Get("text").String() != "more sys" {
		t.Fatalf("block1 text=%s", blocks[1].Raw)
	}
	// Native Anthropic cache_control on system-shaped content must also survive.
	in2 := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"messages":[
			{"role":"system","content":[
				{"type":"text","text":"CACHED","cache_control":{"type":"ephemeral","ttl":"1h"}}
			]},
			{"role":"user","content":"hi"}
		]
	}`)
	_, out2, err := provideradapt.TranslateRequest(cfg, in2, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out2, "system.0.cache_control.ttl").String() != "1h" {
		t.Fatalf("native cache_control lost: %s", string(out2))
	}
}

func TestTranslateRequestMidConversationSystemStaysInMessages(t *testing.T) {
	// Official Anthropic (2026): mid-conversation role:system must remain in messages[]
	// so prompt-cache prefixes stay valid. Folding into top-level system smashes the cache.
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 4096,
	}
	in := []byte(`{
		"model":"claude-opus-4-20250514",
		"messages":[
			{"role":"system","content":"STABLE_PREFIX keep"},
			{"role":"user","content":"first turn"},
			{"role":"assistant","content":"ok"},
			{"role":"system","content":"MID_INSTRUCTION apply now"},
			{"role":"user","content":"what is 2+2?"}
		]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-opus-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out, "system").String() != "STABLE_PREFIX keep" {
		t.Fatalf("leading system must stay top-level, got: %s", gjson.GetBytes(out, "system").Raw)
	}
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) < 4 {
		t.Fatalf("want mid-conversation system in messages, got %d: %s", len(msgs), string(out))
	}
	foundMid := false
	for _, m := range msgs {
		if m.Get("role").String() == "system" && strings.Contains(m.Get("content").String(), "MID_INSTRUCTION") {
			foundMid = true
			break
		}
	}
	if !foundMid {
		t.Fatalf("mid-conversation system folded into top-level (cache smash): %s", string(out))
	}
	if strings.Contains(gjson.GetBytes(out, "system").String(), "MID_INSTRUCTION") {
		t.Fatalf("mid system must not be merged into top-level system: %s", string(out))
	}
	// developer mid-turn also maps to role:system in messages (not top-level).
	in2 := []byte(`{
		"model":"claude-opus-4-20250514",
		"messages":[
			{"role":"user","content":"hi"},
			{"role":"developer","content":[{"type":"text","text":"DEV_MID","cache_control":{"type":"ephemeral"}}]},
			{"role":"user","content":"next"}
		]
	}`)
	_, out2, err := provideradapt.TranslateRequest(cfg, in2, "claude-opus-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out2, "system").Exists() {
		t.Fatalf("no leading system expected: %s", string(out2))
	}
	mid := gjson.GetBytes(out2, "messages.1")
	if mid.Get("role").String() != "system" {
		t.Fatalf("mid developer → messages role=system, got: %s", mid.Raw)
	}
	if !strings.Contains(mid.Get("content").Raw, "DEV_MID") {
		t.Fatalf("mid developer content lost: %s", mid.Raw)
	}
	// Adjacent mid-conversation system TextBlockParam[] must merge without JSON-dump garble.
	in3 := []byte(`{
		"model":"claude-opus-4-20250514",
		"messages":[
			{"role":"user","content":"hi"},
			{"role":"system","content":[{"type":"text","text":"MID_A","cache_control":{"type":"ephemeral"}}]},
			{"role":"system","content":[{"type":"text","text":"MID_B","cache_control":{"type":"ephemeral"}}]},
			{"role":"user","content":"next"}
		]
	}`)
	_, out3, err := provideradapt.TranslateRequest(cfg, in3, "claude-opus-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	wire3 := string(out3)
	if strings.Contains(wire3, `"type":"text","text":"[{`) || strings.Contains(wire3, `"text":"[{\"type\"`) {
		t.Fatalf("asBlocks JSON-dumped structured system chrome (garble): %s", wire3)
	}
	foundA, foundB := false, false
	for _, m := range gjson.GetBytes(out3, "messages").Array() {
		if m.Get("role").String() != "system" {
			continue
		}
		raw := m.Get("content").Raw
		if strings.Contains(raw, "MID_A") {
			foundA = true
		}
		if strings.Contains(raw, "MID_B") {
			foundB = true
		}
	}
	if !foundA || !foundB {
		t.Fatalf("merged mid system lost blocks: %s", wire3)
	}
}

func TestTranslateRequestPlainSystemStaysString(t *testing.T) {
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 4096,
	}
	in := []byte(`{"model":"claude-sonnet-4-20250514","messages":[{"role":"system","content":"be brief"},{"role":"user","content":"hi"}]}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	sys := gjson.GetBytes(out, "system")
	if sys.Type != gjson.String || sys.String() != "be brief" {
		t.Fatalf("plain system should stay string: %s", string(out))
	}
}

func TestMapRequestAuthBearer(t *testing.T) {
	cfg := provideradapt.AdapterConfig{
		ID:                   "anthropic",
		Dialect:              provideradapt.DialectAnthropicMessages,
		DoorLabel:            "openai_to_anthropic",
		UpstreamKind:         provideradapt.UpstreamAnthropic,
		UpstreamPath:         "/v1/messages",
		AuthMode:             provideradapt.AuthBearerToXAPIKey,
		AnthropicVersion:     "2023-06-01",
		AnthropicWorkspaceID: "wrkspc_01IgnoredFromDBAAAAAAAAAAAA",
		DefaultMaxTokens:     4096,
		MatchModelPrefixes:   []string{"claude-"},
		ModelAliases:         map[string]string{},
	}
	h := make(http.Header)
	h.Set("Authorization", "Bearer sk-ant-test")
	out, err := provideradapt.MapRequestAuth(h, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if out.Get("x-api-key") != "sk-ant-test" {
		t.Fatalf("x-api-key=%s", out.Get("x-api-key"))
	}
	if out.Get("Authorization") != "" {
		t.Fatalf("Authorization should be cleared")
	}
	if out.Get("anthropic-version") != "2023-06-01" {
		t.Fatalf("version=%s", out.Get("anthropic-version"))
	}
	// DB anthropic_workspace_id is not injected; workspace-scoped keys omit the header.
	if out.Get(provideradapt.HeaderAnthropicWorkspaceID) != "" {
		t.Fatalf("auth mapping must not inject DB workspace: %s", out.Get(provideradapt.HeaderAnthropicWorkspaceID))
	}
	provideradapt.ApplyAnthropicWorkspaceID(out, "wrkspc_01FromEnvAAAAAAAAAAAAAAAA")
	if out.Get(provideradapt.HeaderAnthropicWorkspaceID) != "wrkspc_01FromEnvAAAAAAAAAAAAAAAA" {
		t.Fatalf("env workspace=%s", out.Get(provideradapt.HeaderAnthropicWorkspaceID))
	}
	h2 := make(http.Header)
	h2.Set("Authorization", "Bearer sk-ant-test")
	h2.Set(provideradapt.HeaderAnthropicWorkspaceID, "wrkspc_01FromClientAAAAAAAAAAAAAAA")
	out2, err := provideradapt.MapRequestAuth(h2, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if out2.Get(provideradapt.HeaderAnthropicWorkspaceID) != "wrkspc_01FromClientAAAAAAAAAAAAAAA" {
		t.Fatalf("client workspace lost: %s", out2.Get(provideradapt.HeaderAnthropicWorkspaceID))
	}
	provideradapt.ApplyAnthropicWorkspaceID(out2, "wrkspc_01FromEnvAAAAAAAAAAAAAAAA")
	if out2.Get(provideradapt.HeaderAnthropicWorkspaceID) != "wrkspc_01FromClientAAAAAAAAAAAAAAA" {
		t.Fatalf("env overwrote client: %s", out2.Get(provideradapt.HeaderAnthropicWorkspaceID))
	}
}

func TestValidateAnthropicWorkspaceID(t *testing.T) {
	cfg := provideradapt.AdapterConfig{
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
	}
	if err := provideradapt.ValidateConfig(cfg); err != nil {
		t.Fatalf("empty workspace should be ok: %v", err)
	}
	cfg.AnthropicWorkspaceID = "not-a-workspace"
	if err := provideradapt.ValidateConfig(cfg); err == nil {
		t.Fatal("expected invalid workspace id")
	}
	cfg.AnthropicWorkspaceID = "wrkspc_01JwQvzr7rXLA5AGx3HKfFUJ"
	if err := provideradapt.ValidateConfig(cfg); err != nil {
		t.Fatalf("valid wrkspc_ id rejected: %v", err)
	}
}

func TestRewriteModelAliasHostScoped(t *testing.T) {
	reg, err := provideradapt.NewRegistry(
		[]provideradapt.AdapterConfig{},
		[]provideradapt.ModelAlias{{
			ClientModel:          "trim-gemini-flash",
			UpstreamModel:        "gemini-3.6-flash",
			UpstreamHostContains: "generativelanguage.googleapis.com",
			Enabled:              true,
		}},
		[]provideradapt.DiscoverableModel{},
		provideradapt.Chrome{},
	)
	if err != nil {
		t.Fatal(err)
	}
	got := reg.RewriteModel("trim-gemini-flash", "https://generativelanguage.googleapis.com/v1beta/openai")
	if got != "gemini-3.6-flash" {
		t.Fatalf("got=%s", got)
	}
	got2 := reg.RewriteModel("trim-gemini-flash", "https://api.openai.com/v1")
	if got2 != "trim-gemini-flash" {
		t.Fatalf("expected no rewrite on other host, got=%s", got2)
	}
}

func TestPipeAnthropicSSEToOpenAI(t *testing.T) {
	src := strings.NewReader(strings.Join([]string{
		`event: message_start`,
		`data: {"type":"message_start","message":{"id":"msg_s","model":"claude-sonnet-4-20250514"}}`,
		``,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hi"}}`,
		``,
		`event: message_delta`,
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"}}`,
		``,
		`event: message_stop`,
		`data: {"type":"message_stop"}`,
		``,
	}, "\n"))
	rec := httptest_recorder{h: make(http.Header), code: 200}
	if err := provideradapt.PipeAnthropicSSEToOpenAI(&rec, src, "claude-sonnet-4-20250514"); err != nil {
		t.Fatal(err)
	}
	body := rec.buf.String()
	if !strings.Contains(body, `"content":"hi"`) {
		t.Fatalf("missing text chunk: %s", body)
	}
	if !strings.Contains(body, "data: [DONE]") {
		t.Fatalf("missing DONE: %s", body)
	}
}

type httptest_recorder struct {
	h    http.Header
	code int
	buf  strings.Builder
}

func (r *httptest_recorder) Header() http.Header         { return r.h }
func (r *httptest_recorder) Write(b []byte) (int, error) { return r.buf.Write(b) }
func (r *httptest_recorder) WriteHeader(statusCode int)  { r.code = statusCode }
func (r *httptest_recorder) Flush()                      {}

func TestImageDataURLAndThinking(t *testing.T) {
	cfg := provideradapt.AdapterConfig{
		ID:                 "anthropic",
		Enabled:            true,
		Dialect:            provideradapt.DialectAnthropicMessages,
		DoorLabel:          "openai_to_anthropic",
		MatchModelPrefixes: []string{"claude-"},
		ModelAliases:       map[string]string{},
		UpstreamKind:       provideradapt.UpstreamAnthropic,
		UpstreamPath:       "/v1/messages",
		AuthMode:           provideradapt.AuthBearerToXAPIKey,
		AnthropicVersion:   "2023-06-01",
		DefaultMaxTokens:   256,
	}
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"messages":[{"role":"user","content":[
			{"type":"text","text":"see"},
			{"type":"image_url","image_url":{"url":"data:image/png;base64,aaa"}}
		]}],
		"temperature":0.7
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out, "temperature").Exists() {
		t.Fatalf("temperature must be omitted for Anthropic compat: %s", out)
	}
	if gjson.GetBytes(out, "messages.0.content.1.source.type").String() != "base64" {
		t.Fatalf("expected base64 image source: %s", out)
	}
	anth := []byte(`{
		"id":"msg_t","model":"claude-sonnet-4-20250514",
		"content":[
			{"type":"thinking","thinking":"plan"},
			{"type":"text","text":"done"}
		],
		"stop_reason":"end_turn",
		"usage":{"input_tokens":1,"output_tokens":1}
	}`)
	oa, err := provideradapt.TranslateResponseBody(cfg, anth, 200)
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(oa, "choices.0.message.reasoning_content").String() != "plan" {
		t.Fatalf("reasoning=%s", oa)
	}
	if gjson.GetBytes(oa, "choices.0.message.content").String() != "done" {
		t.Fatalf("content=%s", oa)
	}
}

func TestThinkingBlocksRoundTripOpenAIToAnthropic(t *testing.T) {
	// Official Anthropic: thinking blocks with signature must be echoed unmodified
	// on tool-use continuations. openai_to_anthropic must not drop them.
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 1024,
	}
	anth := []byte(`{
		"id":"msg_think","model":"claude-sonnet-4-20250514",
		"content":[
			{"type":"thinking","thinking":"I will call add","signature":"sig_KEEP"},
			{"type":"redacted_thinking","data":"opaque_KEEP"},
			{"type":"tool_use","id":"toolu_1","name":"add","input":{"a":1}}
		],
		"stop_reason":"tool_use",
		"usage":{"input_tokens":1,"output_tokens":1}
	}`)
	oa, err := provideradapt.TranslateResponseBody(cfg, anth, 200)
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(oa, "choices.0.message.thinking_blocks.0.signature").String() != "sig_KEEP" {
		t.Fatalf("thinking_blocks signature lost on Anthropic→OpenAI: %s", oa)
	}
	if gjson.GetBytes(oa, "choices.0.message.thinking_blocks.1.data").String() != "opaque_KEEP" {
		t.Fatalf("redacted_thinking data lost: %s", oa)
	}

	// Client sends the OpenAI-shaped history back with thinking_blocks + tool result.
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"messages":[
			{"role":"user","content":"add 1"},
			{"role":"assistant","content":null,
			 "reasoning_content":"I will call add",
			 "thinking_blocks":[
				{"type":"thinking","thinking":"I will call add","signature":"sig_KEEP"},
				{"type":"redacted_thinking","data":"opaque_KEEP"}
			 ],
			 "tool_calls":[{"id":"toolu_1","type":"function","function":{"name":"add","arguments":"{\"a\":1}"}}]
			},
			{"role":"tool","tool_call_id":"toolu_1","content":"2"}
		]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	asst := gjson.GetBytes(out, "messages.1.content").Array()
	if len(asst) < 3 {
		t.Fatalf("assistant blocks=%d raw=%s", len(asst), string(out))
	}
	if asst[0].Get("type").String() != "thinking" || asst[0].Get("signature").String() != "sig_KEEP" {
		t.Fatalf("thinking not restored first: %s", string(out))
	}
	if asst[1].Get("type").String() != "redacted_thinking" || asst[1].Get("data").String() != "opaque_KEEP" {
		t.Fatalf("redacted_thinking not restored: %s", string(out))
	}
	if asst[len(asst)-1].Get("type").String() != "tool_use" {
		t.Fatalf("tool_use must remain: %s", string(out))
	}
}

func TestAnthropicStreamSignatureDeltaEmitted(t *testing.T) {
	toolIndex := map[string]int{}
	blockToTool := map[int]int{}
	next := 0
	var thinkingBuf strings.Builder
	// Simulate thinking_delta then signature_delta (official stream order).
	_, _, err := provideradapt.AnthropicSSEEventToOpenAIChunks(
		`{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`,
		"claude-sonnet-4", toolIndex, blockToTool, &next, &thinkingBuf,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = provideradapt.AnthropicSSEEventToOpenAIChunks(
		`{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"step by step"}}`,
		"claude-sonnet-4", toolIndex, blockToTool, &next, &thinkingBuf,
	)
	if err != nil {
		t.Fatal(err)
	}
	chunks, done, err := provideradapt.AnthropicSSEEventToOpenAIChunks(
		`{"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"sig_STREAM"}}`,
		"claude-sonnet-4", toolIndex, blockToTool, &next, &thinkingBuf,
	)
	if err != nil {
		t.Fatal(err)
	}
	if done || len(chunks) != 1 {
		t.Fatalf("chunks=%v done=%v", chunks, done)
	}
	tb := gjson.Get(chunks[0], "choices.0.delta.thinking_blocks.0")
	if tb.Get("signature").String() != "sig_STREAM" {
		t.Fatalf("signature_delta not mapped: %s", chunks[0])
	}
	// Must not blank thinking text while keeping signature (Anthropic permanent 400 class).
	if tb.Get("thinking").String() != "step by step" {
		t.Fatalf("thinking text blanked on signature_delta: %s", chunks[0])
	}
}

func TestThinkingBlocksMergeReasoningContentFromStream(t *testing.T) {
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 1024,
	}
	// Client accumulated stream: reasoning_content text + empty thinking + signature.
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"thinking":{"type":"enabled","budget_tokens":1024},
		"messages":[
			{"role":"user","content":"hi"},
			{"role":"assistant","content":null,
			 "reasoning_content":"step by step plan",
			 "thinking_blocks":[{"type":"thinking","thinking":"","signature":"sig_MERGE"}],
			 "tool_calls":[{"id":"t1","type":"function","function":{"name":"x","arguments":"{}"}}]
			},
			{"role":"tool","tool_call_id":"t1","content":"ok"}
		]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out, "thinking.type").String() != "enabled" {
		t.Fatalf("thinking config not forwarded: %s", string(out))
	}
	think := gjson.GetBytes(out, "messages.1.content.0")
	if think.Get("type").String() != "thinking" || think.Get("signature").String() != "sig_MERGE" {
		t.Fatalf("thinking block lost: %s", string(out))
	}
	if think.Get("thinking").String() != "step by step plan" {
		t.Fatalf("reasoning_content not merged into signed thinking: %s", string(out))
	}
}

func TestTranslateRequestMergeAdjacentAssistantToolCalls(t *testing.T) {
	// OpenAI Chat allows consecutive assistant turns; adapter must merge tool_use blocks
	// without dropping the second turn's calls (garble / orphan tool_result class).
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 4096,
	}
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"messages":[
			{"role":"user","content":"run both"},
			{"role":"assistant","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"a","arguments":"{}"}}]},
			{"role":"assistant","content":null,"tool_calls":[{"id":"c2","type":"function","function":{"name":"b","arguments":"{}"}}]},
			{"role":"tool","tool_call_id":"c1","content":"A_OK"},
			{"role":"tool","tool_call_id":"c2","content":"B_OK"},
			{"role":"user","content":"done"}
		]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	asst := gjson.GetBytes(out, "messages.1.content").Array()
	var names []string
	for _, b := range asst {
		if b.Get("type").String() == "tool_use" {
			names = append(names, b.Get("name").String())
		}
	}
	if len(names) != 2 || names[0] != "a" || names[1] != "b" {
		t.Fatalf("merged assistant must keep both tool_use: %v raw=%s", names, string(out))
	}
}

func TestTranslateRequestDoesNotMergeToolUseWithLaterAnswer(t *testing.T) {
	// If tool results were dropped, consecutive assistants must NOT smash tool_use +
	// final answer into one turn (Anthropic requires tool_result between them).
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 4096,
	}
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"messages":[
			{"role":"user","content":"go"},
			{"role":"assistant","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"a","arguments":"{}"}}]},
			{"role":"assistant","content":"final answer after tools"}
		]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) < 3 {
		t.Fatalf("tool_use assistant must not merge into answer assistant: %s", string(out))
	}
	if msgs[1].Get("role").String() != "assistant" || !strings.Contains(msgs[1].Get("content").Raw, "tool_use") {
		t.Fatalf("expected tool_use assistant first: %s", msgs[1].Raw)
	}
	if msgs[2].Get("role").String() != "assistant" || !strings.Contains(msgs[2].Get("content").Raw, "final answer after tools") {
		t.Fatalf("expected separate answer assistant: %s", msgs[2].Raw)
	}
}

func TestSanitizeAnthropicMessagesToolOrder(t *testing.T) {
	in := []byte(`{"messages":[{"role":"user","content":[
		{"type":"text","text":"before"},
		{"type":"tool_result","tool_use_id":"1","content":"ok"},
		{"type":"text","text":"after"}
	]}]}`)
	out := provideradapt.SanitizeAnthropicMessagesToolOrder(in)
	parts := gjson.GetBytes(out, "messages.0.content").Array()
	if len(parts) != 3 {
		t.Fatalf("parts=%d %s", len(parts), string(out))
	}
	if parts[0].Get("type").String() != "tool_result" {
		t.Fatalf("tool_result must lead: %s", string(out))
	}
	if parts[1].Get("text").String() != "before" || parts[2].Get("text").String() != "after" {
		t.Fatalf("text order: %s", string(out))
	}
}

func TestSanitizeAnthropicAssistantToolUseLast(t *testing.T) {
	// Official: server_tool_use / web_search_tool_result must appear before client tool_use.
	in := []byte(`{"messages":[{"role":"assistant","content":[
		{"type":"text","text":"checking"},
		{"type":"tool_use","id":"c1","name":"lookup","input":{}},
		{"type":"server_tool_use","id":"s1","name":"web_search","input":{}},
		{"type":"web_search_tool_result","tool_use_id":"s1","content":[]}
	]}]}`)
	out := provideradapt.SanitizeAnthropicMessagesToolOrder(in)
	parts := gjson.GetBytes(out, "messages.0.content").Array()
	if len(parts) != 4 {
		t.Fatalf("parts=%d %s", len(parts), string(out))
	}
	if parts[len(parts)-1].Get("type").String() != "tool_use" {
		t.Fatalf("client tool_use must be last: %s", string(out))
	}
	var sawServerBeforeClient bool
	clientIdx, serverIdx := -1, -1
	for i, p := range parts {
		switch p.Get("type").String() {
		case "tool_use":
			clientIdx = i
		case "server_tool_use":
			serverIdx = i
		}
	}
	if serverIdx < 0 || clientIdx < 0 || serverIdx > clientIdx {
		t.Fatalf("server_tool_use must precede tool_use: %s", string(out))
	}
	_ = sawServerBeforeClient
}

func TestTranslateRequestDoesNotGarbleAudioOrDropsContainerUpload(t *testing.T) {
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic_messages", Dialect: provideradapt.DialectAnthropicMessages,
		UpstreamPath: "/v1/messages", DefaultMaxTokens: 1024,
	}
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"messages":[{"role":"user","content":[
			{"type":"text","text":"what is 2+2?"},
			{"type":"input_audio","input_audio":{"data":"AAAA","format":"wav"}},
			{"type":"container_upload","file_id":"file_123"},
			{"type":"file","file":{"file_id":"file_pdf_1"}}
		]}]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	parts := gjson.GetBytes(out, "messages.0.content").Array()
	var sawText, sawAudioAsText, sawUpload, sawDoc bool
	for _, p := range parts {
		switch p.Get("type").String() {
		case "text":
			sawText = true
			if strings.Contains(p.Get("text").String(), "input_audio") || strings.Contains(p.Get("text").String(), "AAAA") {
				sawAudioAsText = true
			}
		case "container_upload":
			sawUpload = true
			if p.Get("file_id").String() != "file_123" {
				t.Fatalf("container_upload mutated: %s", p.Raw)
			}
		case "document":
			sawDoc = true
			if p.Get("source.file_id").String() != "file_pdf_1" {
				t.Fatalf("file→document lost id: %s", p.Raw)
			}
		}
	}
	if !sawText {
		t.Fatal("text lost")
	}
	if sawAudioAsText {
		t.Fatalf("input_audio must not be smashed into text: %s", string(out))
	}
	if !sawUpload {
		t.Fatalf("container_upload must pass through: %s", string(out))
	}
	if !sawDoc {
		t.Fatalf("OpenAI file part must map to document: %s", string(out))
	}
}

func TestTranslateRequestAcceptsLegacyFunctionAndModelRoles(t *testing.T) {
	// Official OpenAI: role=function is deprecated but still in GPT histories.
	// Gemini openai_compat documents function role for backward compatibility.
	// role=model may leak from native Gemini; map to assistant.
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 4096,
	}
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"messages":[
			{"role":"user","content":"weather?"},
			{"role":"model","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"get_weather","arguments":"{}"}}]},
			{"role":"function","name":"get_weather","content":"18C"},
			{"role":"user","content":"thanks"}
		]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	wire := string(out)
	if strings.Contains(wire, `"role":"model"`) || strings.Contains(wire, `"role":"function"`) {
		t.Fatalf("model/function roles must be mapped: %s", wire)
	}
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) < 3 {
		t.Fatalf("msgs=%d %s", len(msgs), wire)
	}
	if msgs[1].Get("role").String() != "assistant" {
		t.Fatalf("model→assistant failed: %s", msgs[1].Raw)
	}
	var sawToolResult bool
	for _, m := range msgs {
		if m.Get("role").String() != "user" {
			continue
		}
		for _, p := range m.Get("content").Array() {
			if p.Get("type").String() == "tool_result" && strings.Contains(p.Raw, "18C") {
				sawToolResult = true
			}
		}
	}
	if !sawToolResult {
		t.Fatalf("legacy function result must become tool_result: %s", wire)
	}
}

func TestTranslateRequestSkipsBadImageKeepsTextAndMapsRefusal(t *testing.T) {
	// Bad image_url must not 400 the whole openai_to_anthropic turn (real question lost).
	// Chat refusal has no Messages type - map to text, never forward type=refusal.
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 4096,
	}
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"messages":[{"role":"user","content":[
			{"type":"text","text":"what is 2+2?"},
			{"type":"image_url","image_url":{"url":"data:image/png,NOT_BASE64"}},
			{"type":"refusal","refusal":"I cannot help with that prior request"}
		]}]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	wire := string(out)
	if strings.Contains(wire, `"type":"refusal"`) {
		t.Fatalf("refusal must not be forwarded as Anthropic block: %s", wire)
	}
	parts := gjson.GetBytes(out, "messages.0.content").Array()
	var sawQ, sawRefusalText, sawBadImage bool
	for _, p := range parts {
		if p.Get("type").String() == "text" {
			if strings.Contains(p.Get("text").String(), "what is 2+2?") {
				sawQ = true
			}
			if strings.Contains(p.Get("text").String(), "I cannot help with that prior request") {
				sawRefusalText = true
			}
		}
		if p.Get("type").String() == "image" {
			sawBadImage = true
		}
	}
	if !sawQ {
		t.Fatalf("real question lost: %s", wire)
	}
	if !sawRefusalText {
		t.Fatalf("refusal text must be preserved as text: %s", wire)
	}
	if sawBadImage {
		t.Fatalf("bad image_url must be skipped: %s", wire)
	}
}

func TestTranslateRequestToolResultLeadsMergedUser(t *testing.T) {
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic_messages", Dialect: provideradapt.DialectAnthropicMessages,
		UpstreamPath: "/v1/messages", DefaultMaxTokens: 1024,
	}
	// tool role + following user text become one Anthropic user message; tool_result must lead.
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"messages":[
			{"role":"assistant","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"f","arguments":"{}"}}]},
			{"role":"tool","tool_call_id":"c1","content":"TOOL_OK"},
			{"role":"user","content":"now summarize"}
		]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	// After merge: assistant, then one user with tool_result then text.
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) < 2 {
		t.Fatalf("msgs=%d %s", len(msgs), string(out))
	}
	user := msgs[len(msgs)-1]
	parts := user.Get("content").Array()
	if len(parts) < 2 {
		t.Fatalf("user parts=%s", user.Raw)
	}
	if parts[0].Get("type").String() != "tool_result" {
		t.Fatalf("tool_result must lead user content: %s", user.Raw)
	}
	if parts[0].Get("content").String() != "TOOL_OK" && !strings.Contains(parts[0].Raw, "TOOL_OK") {
		t.Fatalf("tool content lost: %s", parts[0].Raw)
	}
}

func TestTranslateRequestToolResultPreservesStructuredContentNoRawDump(t *testing.T) {
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic_messages", Dialect: provideradapt.DialectAnthropicMessages,
		UpstreamPath: "/v1/messages", DefaultMaxTokens: 1024,
	}
	// Official Anthropic: tool_result.content may be text | image | search_result | document.
	// Never stringify object payloads via .Raw (same smash/garble class as whole-body Deep).
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"messages":[
			{"role":"assistant","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"search","arguments":"{}"}}]},
			{"role":"tool","tool_call_id":"c1","content":[
				{"type":"text","text":"hits"},
				{"type":"search_result","source":"https://ex.com","title":"T","content":[{"type":"text","text":"KEEP_RAG"}]}
			]},
			{"role":"user","content":"summarize"}
		]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	wire := string(out)
	if strings.Contains(wire, `"source":"https://ex.com"`) == false && !strings.Contains(wire, "KEEP_RAG") {
		t.Fatalf("search_result structure lost: %s", wire)
	}
	// Object-shaped tool content must not become Raw JSON text inside tool_result.
	inObj := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"messages":[
			{"role":"assistant","content":null,"tool_calls":[{"id":"c2","type":"function","function":{"name":"x","arguments":"{}"}}]},
			{"role":"tool","tool_call_id":"c2","content":{"nested":{"chrome":true},"arr":[1,2,3]}},
			{"role":"user","content":"ok"}
		]
	}`)
	_, outObj, err := provideradapt.TranslateRequest(cfg, inObj, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(outObj), `"nested"`) || strings.Contains(string(outObj), `"chrome"`) {
		t.Fatalf("object tool content must not dump Raw into Anthropic body: %s", string(outObj))
	}
}

func TestToolsTranslate(t *testing.T) {
	cfg := provideradapt.AdapterConfig{
		ID:                 "anthropic",
		Enabled:            true,
		Dialect:            provideradapt.DialectAnthropicMessages,
		DoorLabel:          "openai_to_anthropic",
		MatchModelPrefixes: []string{"claude-"},
		ModelAliases:       map[string]string{},
		UpstreamKind:       provideradapt.UpstreamAnthropic,
		UpstreamPath:       "/v1/messages",
		AuthMode:           provideradapt.AuthBearerToXAPIKey,
		AnthropicVersion:   "2023-06-01",
		DefaultMaxTokens:   1024,
	}
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"messages":[{"role":"user","content":"call tool"}],
		"tools":[{"type":"function","function":{"name":"add","description":"add","parameters":{"type":"object","properties":{"a":{"type":"number"}}}}}],
		"tool_choice":"auto"
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out, "tools.0.name").String() != "add" {
		t.Fatalf("tools=%s", out)
	}
	anth := []byte(`{
		"id":"msg_2","model":"claude-sonnet-4-20250514",
		"content":[{"type":"tool_use","id":"toolu_1","name":"add","input":{"a":1}}],
		"stop_reason":"tool_use",
		"usage":{"input_tokens":1,"output_tokens":1}
	}`)
	oa, err := provideradapt.TranslateResponseBody(cfg, anth, 200)
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(oa, "choices.0.finish_reason").String() != "tool_calls" {
		t.Fatalf("%s", oa)
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(gjson.GetBytes(oa, "choices.0.message.tool_calls.0.function.arguments").String()), &args); err != nil {
		t.Fatal(err)
	}
	if args["a"].(float64) != 1 {
		t.Fatalf("args=%v", args)
	}
}

func TestOpenAICompatHostRoutingByPrefix(t *testing.T) {
	reg, err := provideradapt.NewRegistry(
		[]provideradapt.AdapterConfig{
			{
				ID: "openai", Enabled: true, SortOrder: 20,
				Dialect: provideradapt.DialectOpenAICompat, DoorLabel: "openai_compat_openai",
				MatchModelPrefixes: []string{"gpt-", "o1-"},
				ModelAliases:       map[string]string{},
				UpstreamKind:       provideradapt.UpstreamOpenAI,
				UpstreamPath:       "/v1/chat/completions",
				UpstreamBaseURL:    "https://api.openai.com",
				AuthMode:           provideradapt.AuthPassthrough,
			},
			{
				ID: "gemini", Enabled: true, SortOrder: 30,
				Dialect: provideradapt.DialectOpenAICompat, DoorLabel: "openai_compat_gemini",
				MatchModelPrefixes: []string{"gemini-", "trim-gemini-"},
				ModelAliases:       map[string]string{"trim-gemini-flash": "gemini-flash-latest"},
				UpstreamKind:       provideradapt.UpstreamOpenAI,
				UpstreamPath:       "/chat/completions",
				UpstreamBaseURL:    "https://generativelanguage.googleapis.com/v1beta/openai",
				AuthMode:           provideradapt.AuthPassthrough,
			},
			{
				ID: "deepseek", Enabled: true, SortOrder: 40,
				Dialect: provideradapt.DialectOpenAICompat, DoorLabel: "openai_compat_deepseek",
				MatchModelPrefixes: []string{"deepseek-"},
				ModelAliases:       map[string]string{},
				UpstreamKind:       provideradapt.UpstreamOpenAI,
				UpstreamPath:       "/chat/completions",
				UpstreamBaseURL:    "https://api.deepseek.com",
				AuthMode:           provideradapt.AuthPassthrough,
			},
		},
		[]provideradapt.ModelAlias{},
		[]provideradapt.DiscoverableModel{},
		provideradapt.Chrome{},
	)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		model string
		id    string
		base  string
	}{
		{"gpt-4.1", "openai", "https://api.openai.com"},
		{"trim-gemini-flash", "gemini", "https://generativelanguage.googleapis.com/v1beta/openai"},
		{"deepseek-chat", "deepseek", "https://api.deepseek.com"},
	}
	for _, tc := range cases {
		cfg, ok := reg.Match(tc.model)
		if !ok || cfg.ID != tc.id {
			t.Fatalf("model=%s match=%v id=%s", tc.model, ok, cfg.ID)
		}
		base, err := provideradapt.ResolveUpstreamBase(cfg, "https://example.invalid/openai", "https://api.anthropic.com")
		if err != nil {
			t.Fatal(err)
		}
		if base != tc.base {
			t.Fatalf("model=%s base=%s want=%s", tc.model, base, tc.base)
		}
		resolved, err := reg.ResolveUpstreamModel(cfg, tc.model)
		if err != nil {
			t.Fatal(err)
		}
		path, out, err := provideradapt.TranslateRequest(cfg, []byte(`{"model":"`+tc.model+`","messages":[{"role":"user","content":"x"}]}`), resolved)
		if err != nil {
			t.Fatal(err)
		}
		if path != cfg.UpstreamPath {
			t.Fatalf("path=%s", path)
		}
		if gjson.GetBytes(out, "model").String() != resolved {
			t.Fatalf("body model=%s resolved=%s", out, resolved)
		}
		if provideradapt.NeedsResponseTranslate(cfg) {
			t.Fatalf("openai_compat must not need response translate")
		}
	}
}

func TestOpenAICompatModelsURL(t *testing.T) {
	cases := []struct {
		base, path, want string
	}{
		{"https://api.openai.com", "/v1/chat/completions", "https://api.openai.com/v1/models"},
		{"https://api.openai.com/v1", "/chat/completions", "https://api.openai.com/v1/models"},
		{"https://generativelanguage.googleapis.com/v1beta/openai", "/chat/completions", "https://generativelanguage.googleapis.com/v1beta/openai/models"},
		{"https://api.deepseek.com", "/chat/completions", "https://api.deepseek.com/models"},
		{"https://api.mistral.ai", "/v1/chat/completions", "https://api.mistral.ai/v1/models"},
	}
	for _, tc := range cases {
		got, err := provideradapt.OpenAICompatModelsURL(tc.base, tc.path)
		if err != nil {
			t.Fatalf("base=%s err=%v", tc.base, err)
		}
		if got != tc.want {
			t.Fatalf("base=%s got=%s want=%s", tc.base, got, tc.want)
		}
	}
}

func TestListModelsDualClientShape(t *testing.T) {
	reg, err := provideradapt.NewRegistry(
		[]provideradapt.AdapterConfig{},
		[]provideradapt.ModelAlias{},
		[]provideradapt.DiscoverableModel{{
			ID: "claude-sonnet-5-5", Enabled: true, OwnedBy: "anthropic", DisplayName: "Claude Sonnet 5.5",
		}},
		provideradapt.Chrome{},
	)
	if err != nil {
		t.Fatal(err)
	}
	out := reg.ListModels(0)
	if out.Object != "list" || len(out.Data) != 1 {
		t.Fatalf("%+v", out)
	}
	item := out.Data[0]
	if item.Object != "model" || item.Type != "model" || item.DisplayName == "" || item.ID != "claude-sonnet-5-5" {
		t.Fatalf("item=%+v", item)
	}
	if out.FirstID != item.ID || out.LastID != item.ID {
		t.Fatalf("first/last=%s/%s", out.FirstID, out.LastID)
	}
}

func TestTranslateRequestLegacyAssistantFunctionCallBecomesToolUse(t *testing.T) {
	// Defense-in-depth: even if Normalize did not run, openai_to_anthropic must not
	// silently drop deprecated assistant.function_call (tool-loop smash).
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 4096,
	}
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"messages":[
			{"role":"user","content":"weather?"},
			{"role":"assistant","content":null,"function_call":{"name":"get_weather","arguments":"{\"city\":\"Paris\"}"}},
			{"role":"function","name":"get_weather","content":"18C"},
			{"role":"user","content":"thanks"}
		]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	wire := string(out)
	var sawToolUse, sawToolResult bool
	for _, m := range gjson.GetBytes(out, "messages").Array() {
		for _, p := range m.Get("content").Array() {
			switch p.Get("type").String() {
			case "tool_use":
				if p.Get("name").String() == "get_weather" {
					sawToolUse = true
				}
			case "tool_result":
				if strings.Contains(p.Raw, "18C") {
					sawToolResult = true
				}
			}
		}
	}
	if !sawToolUse {
		t.Fatalf("function_call must become tool_use: %s", wire)
	}
	if !sawToolResult {
		t.Fatalf("function result must bind to tool_use id: %s", wire)
	}
}

func TestTranslateRequestLegacyFunctionsArray(t *testing.T) {
	// Deprecated functions[] must become Anthropic tools when tools[] is absent.
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 4096,
	}
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"functions":[{"name":"get_weather","description":"Weather","parameters":{"type":"object","properties":{"city":{"type":"string"}}}}],
		"function_call":"auto",
		"messages":[{"role":"user","content":"Paris?"}]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out, "tools.0.name").String() != "get_weather" {
		t.Fatalf("functions→tools failed: %s", string(out))
	}
	if gjson.GetBytes(out, "tool_choice.type").String() != "auto" {
		t.Fatalf("function_call→tool_choice failed: %s", string(out))
	}
}

func TestOpenAICompatDeepSeekToolChoiceDowngradedUnderThinking(t *testing.T) {
	cfg := provideradapt.AdapterConfig{
		ID: "deepseek", Enabled: true, Dialect: provideradapt.DialectOpenAICompat,
		DoorLabel: "openai_compat_deepseek", MatchModelPrefixes: []string{"deepseek-"},
		UpstreamKind: provideradapt.UpstreamOpenAI, UpstreamPath: "/chat/completions",
		UpstreamBaseURL: "https://api.deepseek.com", AuthMode: provideradapt.AuthPassthrough,
	}
	// V4 thinks by default - forced tool_choice must become auto (official 400 otherwise).
	in := []byte(`{
		"model":"deepseek-v4-flash",
		"tool_choice":"required",
		"tools":[{"type":"function","function":{"name":"search","parameters":{"type":"object"}}}],
		"messages":[{"role":"user","content":"find X"}]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "deepseek-v4-flash")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out, "tool_choice").String() != "auto" {
		t.Fatalf("V4 thinking default must downgrade required→auto: %s", string(out))
	}

	// Explicit thinking disabled: keep required.
	in2 := []byte(`{
		"model":"deepseek-v4-flash",
		"thinking":{"type":"disabled"},
		"tool_choice":"required",
		"messages":[{"role":"user","content":"find X"}]
	}`)
	_, out2, err := provideradapt.TranslateRequest(cfg, in2, "deepseek-v4-flash")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out2, "tool_choice").String() != "required" {
		t.Fatalf("thinking disabled must keep required: %s", string(out2))
	}

	// Named function force under thinking.enabled → auto.
	in3 := []byte(`{
		"model":"deepseek-chat",
		"thinking":{"type":"enabled"},
		"tool_choice":{"type":"function","function":{"name":"search"}},
		"messages":[{"role":"user","content":"find X"}]
	}`)
	_, out3, err := provideradapt.TranslateRequest(cfg, in3, "deepseek-chat")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out3, "tool_choice").String() != "auto" {
		t.Fatalf("thinking.enabled must downgrade named function→auto: %s", string(out3))
	}

	// GPT / Gemini must not be touched.
	gptCfg := provideradapt.AdapterConfig{
		ID: "openai", Enabled: true, Dialect: provideradapt.DialectOpenAICompat,
		DoorLabel: "openai_compat_openai", MatchModelPrefixes: []string{"gpt-"},
		UpstreamKind: provideradapt.UpstreamOpenAI, UpstreamPath: "/v1/chat/completions",
		UpstreamBaseURL: "https://api.openai.com/v1", AuthMode: provideradapt.AuthPassthrough,
	}
	gptIn := []byte(`{"model":"gpt-4o","tool_choice":"required","messages":[{"role":"user","content":"x"}]}`)
	_, gptOut, err := provideradapt.TranslateRequest(gptCfg, gptIn, "gpt-4o")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(gptOut, "tool_choice").String() != "required" {
		t.Fatalf("non-DeepSeek must keep required: %s", string(gptOut))
	}
}

func TestOpenAICompatGeminiToolChoiceAllowedToolsMapped(t *testing.T) {
	cfg := provideradapt.AdapterConfig{
		ID: "gemini", Enabled: true, Dialect: provideradapt.DialectOpenAICompat,
		DoorLabel: "openai_compat_gemini", MatchModelPrefixes: []string{"gemini-"},
		UpstreamKind: provideradapt.UpstreamOpenAI, UpstreamPath: "/chat/completions",
		UpstreamBaseURL: "https://generativelanguage.googleapis.com/v1beta/openai",
		AuthMode:        provideradapt.AuthPassthrough,
	}
	in := []byte(`{
		"model":"gemini-2.5-flash",
		"tool_choice":{"type":"allowed_tools","mode":"required","tools":[{"type":"function","name":"search"}]},
		"messages":[{"role":"user","content":"find X"}]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "gemini-2.5-flash")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out, "tool_choice").String() != "required" {
		t.Fatalf("Gemini allowed_tools mode=required must map to required: %s", string(out))
	}

	in2 := []byte(`{
		"model":"gemini-2.5-flash",
		"tool_choice":{"type":"function","function":{"name":"search"}},
		"messages":[{"role":"user","content":"find X"}]
	}`)
	_, out2, err := provideradapt.TranslateRequest(cfg, in2, "gemini-2.5-flash")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out2, "tool_choice").String() != "required" {
		t.Fatalf("Gemini named function must map to required: %s", string(out2))
	}

	in3 := []byte(`{"model":"gemini-2.5-flash","tool_choice":"auto","messages":[{"role":"user","content":"x"}]}`)
	_, out3, err := provideradapt.TranslateRequest(cfg, in3, "gemini-2.5-flash")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out3, "tool_choice").String() != "auto" {
		t.Fatalf("Gemini string auto must stay: %s", string(out3))
	}
}

func TestTranslateRequestPreservesAnthropicToolInputExamplesAndCallers(t *testing.T) {
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 4096,
	}
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"tools":[{
			"type":"function",
			"function":{
				"name":"get_weather",
				"description":"Weather",
				"parameters":{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]},
				"input_examples":[{"city":"Paris"}],
				"allowed_callers":["direct"],
				"eager_input_streaming":true,
				"defer_loading":true,
				"strict":true
			}
		}],
		"messages":[{"role":"user","content":"Paris?"}]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	tool := gjson.GetBytes(out, "tools.0")
	if tool.Get("name").String() != "get_weather" {
		t.Fatalf("tool name lost: %s", string(out))
	}
	if tool.Get("input_examples.0.city").String() != "Paris" {
		t.Fatalf("input_examples lost: %s", string(out))
	}
	if tool.Get("allowed_callers.0").String() != "direct" {
		t.Fatalf("allowed_callers lost: %s", string(out))
	}
	if !tool.Get("eager_input_streaming").Bool() {
		t.Fatalf("eager_input_streaming lost: %s", string(out))
	}
	if !tool.Get("defer_loading").Bool() || !tool.Get("strict").Bool() {
		t.Fatalf("defer_loading/strict lost: %s", string(out))
	}
}

func TestTranslateRequestMapsMessageLevelRefusalAndCachedRefusalPart(t *testing.T) {
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 4096,
	}
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"messages":[
			{"role":"assistant","content":null,"refusal":"I cannot help with that request"},
			{"role":"user","content":"ok try something else"},
			{"role":"assistant","content":[
				{"type":"refusal","refusal":"Still cannot","prompt_cache_breakpoint":{"mode":"explicit"}}
			]},
			{"role":"user","content":"latest"}
		]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	wire := string(out)
	if strings.Contains(wire, `"type":"refusal"`) {
		t.Fatalf("refusal must not be forwarded as Anthropic block: %s", wire)
	}
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) < 4 {
		t.Fatalf("expected 4 messages, got %d: %s", len(msgs), wire)
	}
	if !strings.Contains(msgs[0].Get("content").Raw, "I cannot help with that request") {
		t.Fatalf("message-level refusal lost: %s", msgs[0].Raw)
	}
	asst2 := msgs[2].Get("content")
	found := false
	for _, p := range asst2.Array() {
		if p.Get("type").String() == "text" && p.Get("text").String() == "Still cannot" {
			found = true
			if p.Get("cache_control.type").String() != "ephemeral" {
				t.Fatalf("refusal cache breakpoint must map to cache_control: %s", p.Raw)
			}
		}
	}
	if !found {
		t.Fatalf("cached refusal part must become text: %s", msgs[2].Raw)
	}
}

func TestTranslateRequestPreservesToolsetNameAndStripsNullCaller(t *testing.T) {
	// Official Anthropic computer/browser toolsets: toolset_name must round-trip on
	// tool_use + tool_result. caller:null on replay 400s Bedrock/strict hosts.
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 4096,
	}
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"messages":[
			{"role":"user","content":"click"},
			{"role":"assistant","content":[
				{"type":"tool_use","id":"toolu_1","name":"left_click","input":{"x":10,"y":20},"toolset_name":"computer","caller":null}
			]},
			{"role":"tool","tool_call_id":"toolu_1","toolset_name":"computer","content":"OK"}
		]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	asst := gjson.GetBytes(out, "messages.1.content").Array()
	foundUse := false
	for _, p := range asst {
		if p.Get("type").String() != "tool_use" {
			continue
		}
		foundUse = true
		if p.Get("toolset_name").String() != "computer" {
			t.Fatalf("tool_use toolset_name lost: %s", p.Raw)
		}
		if p.Get("caller").Exists() {
			t.Fatalf("caller:null must be stripped on replay: %s", p.Raw)
		}
	}
	if !foundUse {
		t.Fatalf("tool_use missing: %s", string(out))
	}
	userParts := gjson.GetBytes(out, "messages.2.content").Array()
	if len(userParts) < 1 || userParts[0].Get("type").String() != "tool_result" {
		t.Fatalf("tool_result missing: %s", string(out))
	}
	if userParts[0].Get("toolset_name").String() != "computer" {
		t.Fatalf("tool_result toolset_name lost: %s", userParts[0].Raw)
	}

	// Chat tool_calls carrying toolset_name must map into Anthropic tool_use.
	in2 := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"messages":[
			{"role":"user","content":"shot"},
			{"role":"assistant","content":null,"tool_calls":[
				{"id":"toolu_2","type":"function","toolset_name":"computer","function":{"name":"screenshot","arguments":"{}"}}
			]},
			{"role":"tool","tool_call_id":"toolu_2","toolset_name":"computer","content":"img"}
		]
	}`)
	_, out2, err := provideradapt.TranslateRequest(cfg, in2, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	asst2 := gjson.GetBytes(out2, "messages.1.content").Array()
	found2 := false
	for _, p := range asst2 {
		if p.Get("type").String() == "tool_use" && p.Get("name").String() == "screenshot" {
			found2 = true
			if p.Get("toolset_name").String() != "computer" {
				t.Fatalf("tool_calls toolset_name lost: %s", p.Raw)
			}
		}
	}
	if !found2 {
		t.Fatalf("screenshot tool_use missing: %s", string(out2))
	}
}

func TestTranslateRequestOrdersToolResultsBeforeText(t *testing.T) {
	// Official Anthropic: tool_result blocks must come FIRST in a user content array.
	// Adjacent OpenAI user(text) + tool(result) merge must reorder, not text-then-tool.
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "openai_to_anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		AuthMode: provideradapt.AuthBearerToXAPIKey, AnthropicVersion: "2023-06-01",
		DefaultMaxTokens: 4096,
	}
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"messages":[
			{"role":"user","content":"start"},
			{"role":"assistant","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"lookup","arguments":"{}"}}]},
			{"role":"user","content":"please continue after the tool"},
			{"role":"tool","tool_call_id":"c1","content":"TOOL_OK"}
		]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	msgs := gjson.GetBytes(out, "messages").Array()
	// After assistant, adjacent user(text)+user(tool_result) merge into one user turn.
	var userAfterAsst gjson.Result
	for i := 1; i < len(msgs); i++ {
		if msgs[i].Get("role").String() == "user" {
			userAfterAsst = msgs[i]
			break
		}
	}
	parts := userAfterAsst.Get("content").Array()
	if len(parts) < 2 {
		t.Fatalf("expected merged tool_result+text user turn: %s", string(out))
	}
	if parts[0].Get("type").String() != "tool_result" {
		t.Fatalf("tool_result must come first, got %s: %s", parts[0].Get("type").String(), userAfterAsst.Raw)
	}
	if parts[0].Get("content").String() != "TOOL_OK" && !strings.Contains(parts[0].Raw, "TOOL_OK") {
		t.Fatalf("tool content lost: %s", parts[0].Raw)
	}
	foundText := false
	for _, p := range parts[1:] {
		if p.Get("type").String() == "text" && strings.Contains(p.Get("text").String(), "please continue") {
			foundText = true
		}
	}
	if !foundText {
		t.Fatalf("follow-up text lost after reorder: %s", userAfterAsst.Raw)
	}

	// Native mixed user array with text before tool_result must also reorder.
	// browser_state lives inside tool_result content (official browser toolset).
	in2 := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"messages":[
			{"role":"user","content":[
				{"type":"text","text":"AFTER_TOOL note"},
				{"type":"tool_result","tool_use_id":"c9","toolset_name":"browser","content":[
					{"type":"text","text":"OK"},
					{"type":"browser_state","tabs":[{"tab_id":"t1","title":"Docs","url":"https://example.com","active":true}]}
				]}
			]}
		]
	}`)
	_, out2, err := provideradapt.TranslateRequest(cfg, in2, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	parts2 := gjson.GetBytes(out2, "messages.0.content").Array()
	if len(parts2) < 2 {
		t.Fatalf("parts missing: %s", string(out2))
	}
	if parts2[0].Get("type").String() != "tool_result" {
		t.Fatalf("native mixed user must lead with tool_result: %s", string(out2))
	}
	if parts2[0].Get("toolset_name").String() != "browser" {
		t.Fatalf("toolset_name lost on reorder: %s", parts2[0].Raw)
	}
	wire := string(out2)
	if !strings.Contains(wire, `"browser_state"`) {
		t.Fatalf("browser_state must be preserved: %s", wire)
	}
	if parts2[1].Get("type").String() != "text" || !strings.Contains(parts2[1].Get("text").String(), "AFTER_TOOL") {
		t.Fatalf("text must follow tool_result: %s", string(out2))
	}
}

func TestEnsureDeepSeekToolCallReasoningContentPassback(t *testing.T) {
	cfg := provideradapt.AdapterConfig{
		ID: "deepseek", Enabled: true, Dialect: provideradapt.DialectOpenAICompat,
		DoorLabel: "openai_compat_deepseek", MatchModelPrefixes: []string{"deepseek-"},
		UpstreamKind: provideradapt.UpstreamOpenAI, UpstreamPath: "/chat/completions",
		UpstreamBaseURL: "https://api.deepseek.com", AuthMode: provideradapt.AuthPassthrough,
	}
	// Official: with tools, ALL assistant turns need reasoning_content (not only tool_calls).
	// Missing / "" → inject " " (V4 Pro rejects empty string as "not passed back").
	in := []byte(`{
		"model":"deepseek-v4-pro",
		"tools":[{"type":"function","function":{"name":"search","parameters":{"type":"object"}}}],
		"messages":[
			{"role":"user","content":"weather?"},
			{"role":"assistant","content":"","tool_calls":[{"id":"c1","type":"function","function":{"name":"search","arguments":"{}"}}]},
			{"role":"tool","tool_call_id":"c1","content":"ok"},
			{"role":"assistant","content":"done","reasoning_content":""},
			{"role":"user","content":"thanks"}
		]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "deepseek-v4-pro")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out, "messages.1.reasoning_content").String() != " " {
		t.Fatalf("tool_calls assistant missing reasoning must get space placeholder: %s", string(out))
	}
	if gjson.GetBytes(out, "messages.1.content").Type != gjson.Null {
		t.Fatalf("empty-string content on tool_calls must become null: %s", string(out))
	}
	if gjson.GetBytes(out, "messages.3.reasoning_content").String() != " " {
		t.Fatalf("empty reasoning on final assistant under tools must upgrade to space: %s", string(out))
	}

	// Preserve non-empty reasoning verbatim.
	in2 := []byte(`{
		"model":"deepseek-v4-flash",
		"tools":[{"type":"function","function":{"name":"search","parameters":{"type":"object"}}}],
		"messages":[
			{"role":"assistant","content":null,"reasoning_content":"plan A","tool_calls":[{"id":"c1","type":"function","function":{"name":"search","arguments":"{}"}}]},
			{"role":"tool","tool_call_id":"c1","content":"ok"}
		]
	}`)
	_, out2, err := provideradapt.TranslateRequest(cfg, in2, "deepseek-v4-flash")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out2, "messages.0.reasoning_content").String() != "plan A" {
		t.Fatalf("non-empty reasoning must stay: %s", string(out2))
	}

	// thinking disabled → no inject.
	in3 := []byte(`{
		"model":"deepseek-v4-pro",
		"thinking":{"type":"disabled"},
		"tools":[{"type":"function","function":{"name":"search","parameters":{"type":"object"}}}],
		"messages":[
			{"role":"assistant","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"search","arguments":"{}"}}]}
		]
	}`)
	_, out3, err := provideradapt.TranslateRequest(cfg, in3, "deepseek-v4-pro")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out3, "messages.0.reasoning_content").Exists() {
		t.Fatalf("thinking disabled must not inject reasoning_content: %s", string(out3))
	}

	// GPT must not be touched.
	gpt := []byte(`{
		"model":"gpt-4o",
		"tools":[{"type":"function","function":{"name":"search","parameters":{"type":"object"}}}],
		"messages":[{"role":"assistant","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"search","arguments":"{}"}}]}]
	}`)
	healed := provideradapt.EnsureDeepSeekToolCallReasoningContent(gpt)
	if gjson.GetBytes(healed, "messages.0.reasoning_content").Exists() {
		t.Fatalf("GPT must not get reasoning_content: %s", string(healed))
	}

	// Official Kimi Code: thinking enabled + tool_calls missing reasoning_content → 400.
	kimi := []byte(`{
		"model":"kimi-k2.6",
		"thinking":{"type":"enabled"},
		"tools":[{"type":"function","function":{"name":"search","parameters":{"type":"object"}}}],
		"messages":[
			{"role":"assistant","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"search","arguments":"{}"}}]},
			{"role":"tool","tool_call_id":"c1","content":"ok"}
		]
	}`)
	kimiOut := provideradapt.EnsureDeepSeekToolCallReasoningContent(kimi)
	if gjson.GetBytes(kimiOut, "messages.0.reasoning_content").String() != " " {
		t.Fatalf("Kimi tool_calls missing reasoning must get space: %s", string(kimiOut))
	}
	// thinking disabled → no inject for Kimi either.
	kimiOff := []byte(`{
		"model":"moonshot-v1-128k",
		"thinking":{"type":"disabled"},
		"tools":[{"type":"function","function":{"name":"search","parameters":{"type":"object"}}}],
		"messages":[{"role":"assistant","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"search","arguments":"{}"}}]}]
	}`)
	kimiOffOut := provideradapt.EnsureDeepSeekToolCallReasoningContent(kimiOff)
	if gjson.GetBytes(kimiOffOut, "messages.0.reasoning_content").Exists() {
		t.Fatalf("Kimi thinking disabled must not inject: %s", string(kimiOffOut))
	}

	// Official MiniMax M2.5: thinking always on; missing reasoning_content on tool turns → heal.
	mm := []byte(`{
		"model":"MiniMax-M2.5",
		"tools":[{"type":"function","function":{"name":"search","parameters":{"type":"object"}}}],
		"messages":[
			{"role":"assistant","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"search","arguments":"{}"}}]},
			{"role":"tool","tool_call_id":"c1","content":"ok"}
		]
	}`)
	mmOut := provideradapt.EnsureDeepSeekToolCallReasoningContent(mm)
	if gjson.GetBytes(mmOut, "messages.0.reasoning_content").String() != " " {
		t.Fatalf("MiniMax tool_calls missing reasoning must get space: %s", string(mmOut))
	}
}

func TestEnsureGeminiAssistantToolCallContent(t *testing.T) {
	in := []byte(`{
		"model":"gemini-2.5-flash",
		"messages":[
			{"role":"user","content":"weather?"},
			{"role":"assistant","content":null,"tool_calls":[
				{"id":"","type":"function","function":{"name":"get_weather","arguments":"{}"}}
			]},
			{"role":"tool","name":"get_weather","tool_call_id":"","content":"18C"}
		]
	}`)
	out := provideradapt.EnsureGeminiAssistantToolCallContent(in)
	if gjson.GetBytes(out, "messages.1.content").String() != " " {
		t.Fatalf("Gemini null content must become space: %s", string(out))
	}
	if gjson.GetBytes(out, "messages.1.tool_calls.0.id").String() != "get_weather" {
		t.Fatalf("empty tool id must become function name: %s", string(out))
	}
	if gjson.GetBytes(out, "messages.2.tool_call_id").String() != "get_weather" {
		t.Fatalf("empty tool_call_id must pair with name: %s", string(out))
	}

	// Non-Gemini must stay null.
	gpt := []byte(`{
		"model":"gpt-4o",
		"messages":[{"role":"assistant","content":null,"tool_calls":[
			{"id":"c1","type":"function","function":{"name":"f","arguments":"{}"}}
		]}]
	}`)
	gptOut := provideradapt.EnsureGeminiAssistantToolCallContent(gpt)
	if gjson.GetBytes(gptOut, "messages.0.content").Type != gjson.Null {
		t.Fatalf("GPT content null must stay: %s", string(gptOut))
	}

	// Official Gemini forum: refusal:null → 400 "Value is not a string: null".
	ref := []byte(`{
		"model":"gemini-2.5-flash",
		"messages":[
			{"role":"assistant","content":"Hi","refusal":null,"tool_calls":[]}
		]
	}`)
	refOut := provideradapt.EnsureGeminiAssistantToolCallContent(ref)
	if gjson.GetBytes(refOut, "messages.0.refusal").Exists() {
		t.Fatalf("Gemini refusal:null must be stripped: %s", string(refOut))
	}
	if gjson.GetBytes(refOut, "messages.0.tool_calls").Exists() {
		t.Fatalf("Gemini empty tool_calls[] must be stripped: %s", string(refOut))
	}
}

func TestEnsureMistralToolCallIDs(t *testing.T) {
	cfg := provideradapt.AdapterConfig{
		ID: "mistral", Enabled: true, Dialect: provideradapt.DialectOpenAICompat,
		DoorLabel: "openai_compat_mistral", MatchModelPrefixes: []string{"mistral-"},
		UpstreamKind: provideradapt.UpstreamOpenAI, UpstreamPath: "/v1/chat/completions",
		UpstreamBaseURL: "https://api.mistral.ai/v1", AuthMode: provideradapt.AuthPassthrough,
	}
	// Cursor/OpenAI-style call_… id must become exactly 9 alphanumeric chars; tool
	// message tool_call_id must stay paired. prefix:true on tool turns must clear.
	in := []byte(`{
		"model":"mistral-large-latest",
		"messages":[
			{"role":"user","content":"weather?"},
			{"role":"assistant","content":null,"prefix":true,
			 "tool_calls":[{"id":"call_PTLP8xhu3uwZk4l3nlnrrJha","type":"function",
			   "function":{"name":"get_weather","arguments":"{}"}}]},
			{"role":"tool","tool_call_id":"call_PTLP8xhu3uwZk4l3nlnrrJha","name":"get_weather","content":"18C"}
		]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "mistral-large-latest")
	if err != nil {
		t.Fatal(err)
	}
	newID := gjson.GetBytes(out, "messages.1.tool_calls.0.id").String()
	if !provideradaptValidMistralID(newID) {
		t.Fatalf("tool_calls id must be 9 alnum, got %q in %s", newID, string(out))
	}
	if gjson.GetBytes(out, "messages.2.tool_call_id").String() != newID {
		t.Fatalf("tool message id must match remapped tool_calls id: %s", string(out))
	}
	if gjson.GetBytes(out, "messages.1.prefix").Bool() {
		t.Fatalf("prefix:true must be cleared on tool_calls assistant: %s", string(out))
	}

	// Already-valid 9-char id must stay unchanged.
	in2 := []byte(`{
		"model":"codestral-latest",
		"messages":[
			{"role":"assistant","content":null,"tool_calls":[{"id":"VvvODy9mT","type":"function","function":{"name":"f","arguments":"{}"}}]},
			{"role":"tool","tool_call_id":"VvvODy9mT","content":"ok"}
		]
	}`)
	healed := provideradapt.EnsureMistralToolCallIDs(in2)
	if gjson.GetBytes(healed, "messages.0.tool_calls.0.id").String() != "VvvODy9mT" {
		t.Fatalf("valid mistral id must stay: %s", string(healed))
	}

	// Non-Mistral must not be rewritten.
	gpt := []byte(`{
		"model":"gpt-4o",
		"messages":[
			{"role":"assistant","content":null,"tool_calls":[{"id":"call_PTLP8xhu3uwZk4l3nlnrrJha","type":"function","function":{"name":"f","arguments":"{}"}}]}
		]
	}`)
	gptOut := provideradapt.EnsureMistralToolCallIDs(gpt)
	if gjson.GetBytes(gptOut, "messages.0.tool_calls.0.id").String() != "call_PTLP8xhu3uwZk4l3nlnrrJha" {
		t.Fatalf("GPT ids must stay: %s", string(gptOut))
	}
}

func TestInheritToolsetNameOnToolResultFromToolCalls(t *testing.T) {
	// Official Anthropic: tool_result must echo toolset_name from the matching tool_use.
	// Chat history often carries toolset_name only on assistant.tool_calls.
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		UpstreamBaseURL: "https://api.anthropic.com", AuthMode: provideradapt.AuthPassthrough,
		DefaultMaxTokens: 1024,
	}
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"messages":[
			{"role":"assistant","content":null,"tool_calls":[
				{"id":"toolu_1","type":"function","toolset_name":"computer",
				 "function":{"name":"screenshot","arguments":"{}"}}
			]},
			{"role":"tool","tool_call_id":"toolu_1","content":"OK"}
		]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	tr := gjson.GetBytes(out, "messages.1.content.0")
	if tr.Get("type").String() != "tool_result" {
		t.Fatalf("expected tool_result: %s", string(out))
	}
	if tr.Get("toolset_name").String() != "computer" {
		t.Fatalf("toolset_name must inherit from tool_calls onto tool_result: %s", string(out))
	}

	// Native Messages path: tool_use has toolset_name, tool_result omitted it.
	native := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"max_tokens":1024,
		"messages":[
			{"role":"assistant","content":[
				{"type":"tool_use","id":"toolu_b","name":"navigate","toolset_name":"browser","input":{}}
			]},
			{"role":"user","content":[
				{"type":"tool_result","tool_use_id":"toolu_b","content":"ok"}
			]}
		]
	}`)
	fixed := provideradapt.SanitizeAnthropicMessagesToolOrder(native)
	if gjson.GetBytes(fixed, "messages.1.content.0.toolset_name").String() != "browser" {
		t.Fatalf("native tool_result must inherit toolset_name: %s", string(fixed))
	}
}

func TestNormalizeAnthropicToolChoiceForThinking(t *testing.T) {
	cfg := provideradapt.AdapterConfig{
		ID: "anthropic", Enabled: true, Dialect: provideradapt.DialectAnthropicMessages,
		DoorLabel: "anthropic", MatchModelPrefixes: []string{"claude-"},
		UpstreamKind: provideradapt.UpstreamAnthropic, UpstreamPath: "/v1/messages",
		UpstreamBaseURL: "https://api.anthropic.com", AuthMode: provideradapt.AuthPassthrough,
		DefaultMaxTokens: 1024,
	}
	// Official: thinking + tool_choice any/tool → HTTP 400. required maps to any then auto.
	in := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"thinking":{"type":"enabled","budget_tokens":1024},
		"tool_choice":"required",
		"tools":[{"type":"function","function":{"name":"search","parameters":{"type":"object"}}}],
		"messages":[{"role":"user","content":"find X"}]
	}`)
	_, out, err := provideradapt.TranslateRequest(cfg, in, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out, "tool_choice.type").String() != "auto" {
		t.Fatalf("thinking+required must downgrade to auto: %s", string(out))
	}
	if !gjson.GetBytes(out, "thinking").Exists() {
		t.Fatalf("thinking must survive: %s", string(out))
	}

	// Official: adaptive thinking supports forced tool use on Opus 4 / Sonnet 4.
	in2 := []byte(`{
		"model":"claude-opus-4",
		"thinking":{"type":"adaptive"},
		"tool_choice":{"type":"tool","name":"search","disable_parallel_tool_use":true},
		"tools":[{"type":"function","function":{"name":"search","parameters":{"type":"object"}}}],
		"messages":[{"role":"user","content":"find X"}]
	}`)
	_, out2, err := provideradapt.TranslateRequest(cfg, in2, "claude-opus-4")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out2, "tool_choice.type").String() != "tool" {
		t.Fatalf("adaptive+tool force must survive on opus-4: %s", string(out2))
	}
	if gjson.GetBytes(out2, "tool_choice.name").String() != "search" {
		t.Fatalf("tool name must survive: %s", string(out2))
	}
	if !gjson.GetBytes(out2, "tool_choice.disable_parallel_tool_use").Bool() {
		t.Fatalf("disable_parallel_tool_use must survive: %s", string(out2))
	}

	// No thinking: keep any.
	in3 := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"tool_choice":"required",
		"tools":[{"type":"function","function":{"name":"search","parameters":{"type":"object"}}}],
		"messages":[{"role":"user","content":"find X"}]
	}`)
	_, out3, err := provideradapt.TranslateRequest(cfg, in3, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out3, "tool_choice.type").String() != "any" {
		t.Fatalf("without thinking, required→any must stay: %s", string(out3))
	}

	// Native Messages path: adaptive + any stays on Sonnet 4.
	native := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"max_tokens":1024,
		"thinking":{"type":"adaptive"},
		"tool_choice":{"type":"any"},
		"tools":[{"name":"search","input_schema":{"type":"object"}}],
		"messages":[{"role":"user","content":"x"}]
	}`)
	fixed := provideradapt.NormalizeAnthropicToolChoiceForThinking(native)
	if gjson.GetBytes(fixed, "tool_choice.type").String() != "any" {
		t.Fatalf("native adaptive+any must stay: %s", string(fixed))
	}

	// Exception: Opus 5.5 adaptive still rejects forced tools.
	exc := []byte(`{
		"model":"claude-opus-5-5",
		"max_tokens":1024,
		"thinking":{"type":"adaptive"},
		"tool_choice":{"type":"any"},
		"tools":[{"name":"search","input_schema":{"type":"object"}}],
		"messages":[{"role":"user","content":"x"}]
	}`)
	excOut := provideradapt.NormalizeAnthropicToolChoiceForThinking(exc)
	if gjson.GetBytes(excOut, "tool_choice.type").String() != "auto" {
		t.Fatalf("opus-5-5 adaptive+any must → auto: %s", string(excOut))
	}

	// Claude 4.7: enabled→adaptive migrate first, then any survives.
	in47 := []byte(`{
		"model":"claude-opus-4-7",
		"thinking":{"type":"enabled","budget_tokens":2048},
		"tool_choice":{"type":"any"},
		"tools":[{"type":"function","function":{"name":"search","parameters":{"type":"object"}}}],
		"messages":[{"role":"user","content":"find X"}]
	}`)
	_, out47, err := provideradapt.TranslateRequest(cfg, in47, "claude-opus-4-7")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out47, "thinking.type").String() != "adaptive" {
		t.Fatalf("4.7 enabled must migrate to adaptive: %s", string(out47))
	}
	if gjson.GetBytes(out47, "tool_choice.type").String() != "any" {
		t.Fatalf("4.7 adaptive+any must survive after migrate: %s", string(out47))
	}

	// thinking + temperature/top_p must be stripped (official incompatible).
	samp := []byte(`{
		"model":"claude-sonnet-4-20250514",
		"max_tokens":1024,
		"thinking":{"type":"enabled","budget_tokens":2048},
		"temperature":0.7,
		"top_p":0.9,
		"top_k":40,
		"messages":[{"role":"user","content":"x"}]
	}`)
	cleaned := provideradapt.SanitizeAnthropicThinkingIncompatibleFields(samp)
	if gjson.GetBytes(cleaned, "temperature").Exists() ||
		gjson.GetBytes(cleaned, "top_p").Exists() ||
		gjson.GetBytes(cleaned, "top_k").Exists() {
		t.Fatalf("thinking must strip sampling fields: %s", string(cleaned))
	}
	if !gjson.GetBytes(cleaned, "thinking").Exists() {
		t.Fatalf("thinking must remain: %s", string(cleaned))
	}

	// Opus 4.7 rejects sampling even without thinking.
	samp47 := []byte(`{
		"model":"claude-opus-4-7",
		"max_tokens":1024,
		"temperature":0.5,
		"top_p":0.95,
		"messages":[{"role":"user","content":"x"}]
	}`)
	samp47Out := provideradapt.SanitizeAnthropicThinkingIncompatibleFields(samp47)
	if gjson.GetBytes(samp47Out, "temperature").Exists() || gjson.GetBytes(samp47Out, "top_p").Exists() {
		t.Fatalf("opus-4-7 must strip sampling without thinking: %s", string(samp47Out))
	}
}

func TestSanitizeAnthropicTemperatureTopPMutex(t *testing.T) {
	// Official: Claude 4.x rejects temperature + top_p together.
	in := []byte(`{
		"model":"claude-opus-4-6",
		"max_tokens":1024,
		"temperature":0.2,
		"top_p":0.9,
		"messages":[{"role":"user","content":"hi"}]
	}`)
	out := provideradapt.SanitizeAnthropicTemperatureTopPMutex(in)
	if !gjson.GetBytes(out, "temperature").Exists() {
		t.Fatalf("temperature must stay: %s", string(out))
	}
	if gjson.GetBytes(out, "top_p").Exists() {
		t.Fatalf("top_p must be dropped when both set: %s", string(out))
	}

	// Claude 3 still allows both.
	old := []byte(`{
		"model":"claude-3-5-sonnet-20241022",
		"temperature":0.2,
		"top_p":0.9,
		"messages":[{"role":"user","content":"hi"}]
	}`)
	oldOut := provideradapt.SanitizeAnthropicTemperatureTopPMutex(old)
	if !gjson.GetBytes(oldOut, "temperature").Exists() || !gjson.GetBytes(oldOut, "top_p").Exists() {
		t.Fatalf("Claude 3 must keep both: %s", string(oldOut))
	}
}

func TestNormalizeAnthropicThinkingForModel47(t *testing.T) {
	// Official: Claude 4.7+ rejects thinking.type=enabled → migrate to adaptive.
	in := []byte(`{
		"model":"claude-opus-4-7",
		"max_tokens":64000,
		"thinking":{"type":"enabled","budget_tokens":32000},
		"messages":[{"role":"user","content":"hi"}]
	}`)
	out := provideradapt.NormalizeAnthropicThinkingForModel(in)
	if gjson.GetBytes(out, "thinking.type").String() != "adaptive" {
		t.Fatalf("4.7 enabled must become adaptive: %s", string(out))
	}
	if gjson.GetBytes(out, "thinking.budget_tokens").Exists() {
		t.Fatalf("budget_tokens must be removed on 4.7+: %s", string(out))
	}

	// Claude 4.5 keeps enabled+budget (adaptive not supported).
	old := []byte(`{
		"model":"claude-sonnet-4-5-20250929",
		"max_tokens":1024,
		"thinking":{"type":"enabled","budget_tokens":2048},
		"messages":[{"role":"user","content":"hi"}]
	}`)
	kept := provideradapt.NormalizeAnthropicThinkingForModel(old)
	if gjson.GetBytes(kept, "thinking.type").String() != "enabled" {
		t.Fatalf("4.5 must keep enabled: %s", string(kept))
	}
	if gjson.GetBytes(kept, "thinking.budget_tokens").Int() != 2048 {
		t.Fatalf("4.5 must keep budget_tokens: %s", string(kept))
	}
}

func TestSanitizeAnthropicTrailingAssistantPrefill(t *testing.T) {
	// Official Anthropic: Claude 4.6+ rejects trailing assistant prefill (HTTP 400).
	in := []byte(`{
		"model":"claude-sonnet-4-6",
		"max_tokens":1024,
		"messages":[
			{"role":"user","content":"Say hi"},
			{"role":"assistant","content":"Hello"}
		]
	}`)
	out := provideradapt.SanitizeAnthropicTrailingAssistantPrefill(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) != 3 {
		t.Fatalf("must append continuation user turn: %s", string(out))
	}
	if msgs[2].Get("role").String() != "user" {
		t.Fatalf("last role must be user: %s", string(out))
	}
	if !strings.Contains(msgs[2].Get("content").String(), "Hello") {
		t.Fatalf("continuation must include prior assistant text: %s", string(out))
	}

	// Claude 4.5 still allows prefill - do not rewrite.
	old := []byte(`{
		"model":"claude-sonnet-4-5-20250929",
		"max_tokens":1024,
		"messages":[
			{"role":"user","content":"hi"},
			{"role":"assistant","content":"Hello"}
		]
	}`)
	oldOut := provideradapt.SanitizeAnthropicTrailingAssistantPrefill(old)
	if len(gjson.GetBytes(oldOut, "messages").Array()) != 2 {
		t.Fatalf("4.5 must keep trailing assistant prefill: %s", string(oldOut))
	}

	// Trailing assistant with tool_use must NOT get a fake Continue (needs tool_result).
	tools := []byte(`{
		"model":"claude-opus-4-6",
		"max_tokens":1024,
		"messages":[
			{"role":"user","content":"weather?"},
			{"role":"assistant","content":[
				{"type":"tool_use","id":"t1","name":"get_weather","input":{}}
			]}
		]
	}`)
	toolsOut := provideradapt.SanitizeAnthropicTrailingAssistantPrefill(tools)
	if len(gjson.GetBytes(toolsOut, "messages").Array()) != 2 {
		t.Fatalf("tool_use trailing must not get Continue: %s", string(toolsOut))
	}

	// Already ends on user - unchanged.
	ok := []byte(`{
		"model":"claude-sonnet-4-6",
		"messages":[{"role":"user","content":"hi"}]
	}`)
	okOut := provideradapt.SanitizeAnthropicTrailingAssistantPrefill(ok)
	if len(gjson.GetBytes(okOut, "messages").Array()) != 1 {
		t.Fatalf("user-ending must stay: %s", string(okOut))
	}
}

// provideradaptValidMistralID mirrors mistral-common ^[a-zA-Z0-9]{9}$ for tests
// without exporting the unexported helper.
func provideradaptValidMistralID(id string) bool {
	if len(id) != 9 {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			continue
		}
		return false
	}
	return true
}

