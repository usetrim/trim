package proxy

import (
	"strings"
	"testing"

	"github.com/tidwall/gjson"
	"github.com/usetrim/trim/server/pkg/trimmer"
)

func TestOptimizePayloadPreservesOpenAIToolScaffolding(t *testing.T) {
	s := NewServer(Options{CompressionMode: "aggressive", HistoryKeepTurns: 1})
	in := []byte(`{
		"model":"deepseek-chat",
		"messages":[
			{"role":"system","content":"sys keep"},
			{"role":"developer","content":"dev keep"},
			{"role":"user","content":"old turn that would normally collapse"},
			{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{}"}}]},
			{"role":"tool","tool_call_id":"call_1","content":"{\"temp\":18}"},
			{"role":"user","content":"what is 2+2?"}
		]
	}`)
	out, _, _ := s.optimizePayload(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) != 6 {
		t.Fatalf("msgs=%d raw=%s", len(msgs), string(out))
	}
	if msgs[0].Get("content").String() != "sys keep" {
		t.Fatalf("system mutated: %s", msgs[0].Raw)
	}
	if msgs[1].Get("content").String() != "dev keep" {
		t.Fatalf("developer mutated: %s", msgs[1].Raw)
	}
	if !msgs[3].Get("tool_calls").Exists() {
		t.Fatalf("assistant tool_calls lost: %s", msgs[3].Raw)
	}
	if msgs[4].Get("role").String() != "tool" || !strings.Contains(msgs[4].Get("content").String(), "temp") {
		t.Fatalf("tool result mutated: %s", msgs[4].Raw)
	}
	if strings.Contains(string(out), trimmer.MarkerHistoryCompacted) &&
		(strings.Contains(msgs[3].Get("content").Raw, trimmer.MarkerHistoryCompacted) ||
			strings.Contains(msgs[4].Get("content").String(), trimmer.MarkerHistoryCompacted)) {
		t.Fatalf("tool turns must not be history-stubbed: %s", string(out))
	}
}

func TestOptimizePayloadPreservesAnthropicToolResultTurns(t *testing.T) {
	s := NewServer(Options{CompressionMode: "aggressive", HistoryKeepTurns: 1})
	in := []byte(`{
		"model":"claude-opus-4",
		"messages":[
			{"role":"user","content":"old"},
			{"role":"assistant","content":[{"type":"tool_use","id":"1","name":"Read","input":{"path":"a.go"}}]},
			{"role":"user","content":[{"type":"tool_result","tool_use_id":"1","content":"file bytes"}]},
			{"role":"user","content":"summarize"}
		]
	}`)
	out, _, _ := s.optimizePayload(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) != 4 {
		t.Fatalf("msgs=%d", len(msgs))
	}
	if msgs[1].Get("content.0.type").String() != "tool_use" {
		t.Fatalf("tool_use lost: %s", msgs[1].Raw)
	}
	if msgs[2].Get("content.0.type").String() != "tool_result" {
		t.Fatalf("tool_result lost/stubbed: %s", msgs[2].Raw)
	}
	if strings.Contains(msgs[2].Raw, trimmer.MarkerHistoryCompacted) {
		t.Fatalf("tool_result history-stubbed")
	}
}

func TestMessageHasToolScaffolding(t *testing.T) {
	t.Parallel()
	if !messageHasToolScaffolding(gjson.Parse(`{"role":"tool","content":"x"}`)) {
		t.Fatal("role tool")
	}
	if !messageHasToolScaffolding(gjson.Parse(`{"role":"assistant","tool_calls":[{"id":"1"}]}`)) {
		t.Fatal("tool_calls")
	}
	if !messageHasToolScaffolding(gjson.Parse(`{"role":"user","content":[{"type":"tool_result","tool_use_id":"1","content":"x"}]}`)) {
		t.Fatal("anthropic tool_result")
	}
	if !messageHasToolScaffolding(gjson.Parse(`{"role":"user","content":[{"type":"mcp_tool_result","tool_use_id":"m1","content":"x"}]}`)) {
		t.Fatal("mcp_tool_result")
	}
	if !messageHasToolScaffolding(gjson.Parse(`{"role":"assistant","content":[{"type":"bash_code_execution_tool_result","tool_use_id":"s1","content":{"type":"bash_code_execution_result","stdout":"ok"}}]}`)) {
		t.Fatal("bash_code_execution_tool_result suffix")
	}
	if !messageHasToolScaffolding(gjson.Parse(`{"role":"assistant","content":[{"type":"web_fetch_tool_result","tool_use_id":"w1","content":[]}]}`)) {
		t.Fatal("web_fetch_tool_result suffix")
	}
	if messageHasToolScaffolding(gjson.Parse(`{"role":"user","content":"hi"}`)) {
		t.Fatal("plain user must not flag")
	}
}

func TestOptimizePayloadPreservesAnthropicCitations(t *testing.T) {
	s := NewServer(Options{CompressionMode: "aggressive", HistoryKeepTurns: 1})
	in := []byte(`{
		"model":"claude-opus-4",
		"messages":[
			{"role":"user","content":"old turn collapse candidate"},
			{"role":"assistant","content":[{"type":"text","text":"CITED_CLAIM keep verbatim","citations":[{"type":"char_location","cited_text":"src","document_index":0,"start_char_index":0,"end_char_index":3}]}]},
			{"role":"user","content":"next"}
		]
	}`)
	out, _, _ := s.optimizePayload(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) < 2 {
		t.Fatalf("messages missing: %s", string(out))
	}
	if strings.Contains(msgs[1].Get("content").Raw, trimmer.MarkerHistoryCompacted) {
		t.Fatalf("cited assistant must not be history-stubbed: %s", msgs[1].Raw)
	}
	if msgs[1].Get("content.0.text").String() != "CITED_CLAIM keep verbatim" {
		t.Fatalf("cited text mutated: %s", msgs[1].Raw)
	}
	if msgs[1].Get("content.0.citations.0.type").String() != "char_location" {
		t.Fatalf("citations lost: %s", msgs[1].Raw)
	}
}

func TestOptimizePayloadRequestLevelPromptCacheLastUserOnly(t *testing.T) {
	// Official Anthropic automatic caching (top-level cache_control) and OpenAI
	// prompt_cache_options / prompt_cache_key: Fast must not history-stub or mutate earlier turns.
	s := NewServer(Options{CompressionMode: "aggressive", HistoryKeepTurns: 1})
	in := []byte(`{
		"model":"claude-opus-4",
		"cache_control":{"type":"ephemeral"},
		"messages":[
			{"role":"user","content":"OLD_TURN lots of filler text that would normally collapse or trim"},
			{"role":"assistant","content":"OLD_ASSIST filler answer that would normally trim"},
			{"role":"user","content":"latest question with filler filler filler what is 2+2?"}
		]
	}`)
	out, _, _ := s.optimizePayload(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) != 3 {
		t.Fatalf("messages smashed: %s", string(out))
	}
	if strings.Contains(msgs[0].Get("content").Raw, trimmer.MarkerHistoryCompacted) ||
		strings.Contains(msgs[1].Get("content").Raw, trimmer.MarkerHistoryCompacted) {
		t.Fatalf("request-level cache_control must not history-stub prior turns: %s", string(out))
	}
	if msgs[0].Get("content").String() != "OLD_TURN lots of filler text that would normally collapse or trim" {
		t.Fatalf("prior user mutated under automatic cache: %s", msgs[0].Raw)
	}
	if msgs[1].Get("content").String() != "OLD_ASSIST filler answer that would normally trim" {
		t.Fatalf("prior assistant mutated under automatic cache: %s", msgs[1].Raw)
	}

	in2 := []byte(`{
		"model":"gpt-5.6",
		"prompt_cache_options":{"mode":"explicit","ttl":"30m"},
		"messages":[
			{"role":"developer","content":"DEV_KEEP instructions"},
			{"role":"user","content":"OLD_TURN filler that would collapse"},
			{"role":"assistant","content":"old answer filler"},
			{"role":"user","content":"latest ask with filler filler"}
		]
	}`)
	out2, _, _ := s.optimizePayload(in2)
	msgs2 := gjson.GetBytes(out2, "messages").Array()
	if strings.Contains(msgs2[1].Get("content").Raw, trimmer.MarkerHistoryCompacted) {
		t.Fatalf("prompt_cache_options must not history-stub: %s", string(out2))
	}
	if msgs2[1].Get("content").String() != "OLD_TURN filler that would collapse" {
		t.Fatalf("prior user mutated under prompt_cache_options: %s", msgs2[1].Raw)
	}

	// Official OpenAI pre-5.6 / routing: prompt_cache_key alone still means prefix must
	// stay byte-stable - history-stubbing earlier turns breaks automatic cache hits.
	in3 := []byte(`{
		"model":"gpt-4o",
		"prompt_cache_key":"session-abc",
		"messages":[
			{"role":"user","content":"OLD_TURN lots of filler text that would normally collapse"},
			{"role":"assistant","content":"OLD_ASSIST filler"},
			{"role":"user","content":"latest with filler filler"}
		]
	}`)
	out3, _, _ := s.optimizePayload(in3)
	msgs3 := gjson.GetBytes(out3, "messages").Array()
	if strings.Contains(msgs3[0].Get("content").Raw, trimmer.MarkerHistoryCompacted) {
		t.Fatalf("prompt_cache_key must not history-stub: %s", string(out3))
	}
	if msgs3[0].Get("content").String() != "OLD_TURN lots of filler text that would normally collapse" {
		t.Fatalf("prior user mutated under prompt_cache_key: %s", msgs3[0].Raw)
	}

	// Mid-tool-loop under request-level cache: body ends on assistant/tool, not user.
	// Deep ShouldSkipDeep parity - freeze ALL turns (do not Fast-mutate a non-trailing last user).
	in4 := []byte(`{
		"model":"claude-opus-4",
		"cache_control":{"type":"ephemeral"},
		"messages":[
			{"role":"user","content":"please call lookup FILLER FILLER FILLER FILLER"},
			{"role":"assistant","content":"ok","tool_calls":[{"id":"c1","type":"function","function":{"name":"lookup","arguments":"{}"}}]},
			{"role":"tool","tool_call_id":"c1","content":"TOOL_OK"}
		]
	}`)
	out4, _, _ := s.optimizePayload(in4)
	msgs4 := gjson.GetBytes(out4, "messages").Array()
	if msgs4[0].Get("content").String() != "please call lookup FILLER FILLER FILLER FILLER" {
		t.Fatalf("mid-loop under cache_control must freeze prior user: %s", msgs4[0].Raw)
	}
}

func TestOptimizePayloadPreservesOpenAIPromptCacheBreakpoint(t *testing.T) {
	s := NewServer(Options{CompressionMode: "aggressive", HistoryKeepTurns: 1})
	in := []byte(`{
		"model":"gpt-5.6",
		"messages":[
			{"role":"user","content":"old turn collapse candidate"},
			{"role":"user","content":[{"type":"text","text":"CACHED_PREFIX keep verbatim","prompt_cache_breakpoint":{"mode":"explicit"}},{"type":"text","text":"later filler that could shrink"}]},
			{"role":"user","content":"next"}
		]
	}`)
	out, _, _ := s.optimizePayload(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) < 2 {
		t.Fatalf("messages missing: %s", string(out))
	}
	// Message with breakpoint must not be history-stubbed.
	if strings.Contains(msgs[1].Get("content").Raw, trimmer.MarkerHistoryCompacted) {
		t.Fatalf("prompt_cache_breakpoint message must not be history-stubbed: %s", msgs[1].Raw)
	}
	parts := msgs[1].Get("content").Array()
	if len(parts) < 1 || parts[0].Get("text").String() != "CACHED_PREFIX keep verbatim" {
		t.Fatalf("cached prefix mutated: %s", msgs[1].Raw)
	}
	if parts[0].Get("prompt_cache_breakpoint.mode").String() != "explicit" {
		t.Fatalf("prompt_cache_breakpoint lost: %s", msgs[1].Raw)
	}
}

func TestOptimizePayloadFreezesAnthropicCacheControlMessage(t *testing.T) {
	// Official Anthropic prompt caching: part-level cache_control makes the whole
	// message a cache-prefix participant. Fast must not mutate pre-breakpoint text
	// (byte change invalidates the reusable prefix) - parity with OpenAI breakpoints.
	s := NewServer(Options{CompressionMode: "aggressive", HistoryKeepTurns: 1})
	in := []byte(`{
		"model":"claude-opus-4",
		"messages":[
			{"role":"user","content":"old turn collapse candidate"},
			{"role":"user","content":[
				{"type":"text","text":"BEFORE_BP keep verbatim for cache prefix"},
				{"type":"text","text":"CACHED_PREFIX keep","cache_control":{"type":"ephemeral"}},
				{"type":"text","text":"AFTER_BP also frozen by Fast for fail-closed cache safety"}
			]},
			{"role":"user","content":"next"}
		]
	}`)
	out, _, _ := s.optimizePayload(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) < 2 {
		t.Fatalf("messages missing: %s", string(out))
	}
	if strings.Contains(msgs[1].Get("content").Raw, trimmer.MarkerHistoryCompacted) {
		t.Fatalf("cache_control message must not be history-stubbed: %s", msgs[1].Raw)
	}
	wire := msgs[1].Raw
	if !strings.Contains(wire, "BEFORE_BP keep verbatim for cache prefix") {
		t.Fatalf("pre-breakpoint text mutated: %s", wire)
	}
	if !strings.Contains(wire, `"cache_control"`) || !strings.Contains(wire, "CACHED_PREFIX keep") {
		t.Fatalf("cache_control / cached text lost: %s", wire)
	}
	if !strings.Contains(wire, "AFTER_BP also frozen by Fast for fail-closed cache safety") {
		t.Fatalf("post-breakpoint text mutated under Fast freeze: %s", wire)
	}
}

func TestOptimizePayloadPreservesReasoningContent(t *testing.T) {
	s := NewServer(Options{CompressionMode: "aggressive", HistoryKeepTurns: 1})
	in := []byte(`{
		"model":"deepseek-chat",
		"messages":[
			{"role":"user","content":"old turn collapse candidate"},
			{"role":"assistant","content":"answer","reasoning_content":"hidden chain of thought keep me"},
			{"role":"user","content":"next"}
		]
	}`)
	out, _, _ := s.optimizePayload(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if msgs[1].Get("reasoning_content").String() != "hidden chain of thought keep me" {
		t.Fatalf("reasoning_content lost: %s", msgs[1].Raw)
	}
	if strings.Contains(msgs[1].Get("content").String(), trimmer.MarkerHistoryCompacted) {
		t.Fatalf("reasoning assistant must not be history-stubbed: %s", msgs[1].Raw)
	}
}

func TestOptimizePayloadPreservesGeminiThoughtSignatures(t *testing.T) {
	s := NewServer(Options{CompressionMode: "aggressive", HistoryKeepTurns: 1})
	in := []byte(`{
		"model":"gemini-2.5-flash",
		"messages":[
			{"role":"user","content":"old turn"},
			{"role":"assistant","content":[{"type":"text","text":"prior answer","thought_signature":"SIG_KEEP"}],"tool_calls":[{"id":"c1","type":"function","function":{"name":"f","arguments":"{}"},"extra_content":{"google":{"thought_signature":"TC_SIG"}}}]},
			{"role":"tool","tool_call_id":"c1","content":"ok"},
			{"role":"user","content":"next"}
		]
	}`)
	out, _, _ := s.optimizePayload(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if msgs[1].Get("content.0.thought_signature").String() != "SIG_KEEP" {
		t.Fatalf("part thought_signature lost: %s", msgs[1].Raw)
	}
	if msgs[1].Get("tool_calls.0.extra_content.google.thought_signature").String() != "TC_SIG" {
		t.Fatalf("tool_call thought_signature lost: %s", msgs[1].Raw)
	}
	if strings.Contains(msgs[1].Get("content").Raw, trimmer.MarkerHistoryCompacted) {
		t.Fatalf("signed assistant must not be history-stubbed: %s", msgs[1].Raw)
	}
}

func TestOptimizePayloadFreezesGeminiThoughtSummaryParts(t *testing.T) {
	// Official Gemini thinking: thought:true marks a thought-summary part. Fast must
	// not mutate or history-stub those turns (same smash class as thought_signature).
	s := NewServer(Options{CompressionMode: "aggressive", HistoryKeepTurns: 1})
	in := []byte(`{
		"model":"gemini-2.5-flash",
		"messages":[
			{"role":"user","content":"old turn collapse candidate"},
			{"role":"assistant","content":[
				{"type":"text","text":"THOUGHT_SUMMARY keep verbatim FILLER FILLER","thought":true},
				{"type":"text","text":"final answer FILLER FILLER"}
			]},
			{"role":"user","content":"next"}
		]
	}`)
	out, _, _ := s.optimizePayload(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if strings.Contains(msgs[1].Get("content").Raw, trimmer.MarkerHistoryCompacted) {
		t.Fatalf("thought-summary assistant must not be history-stubbed: %s", msgs[1].Raw)
	}
	if msgs[1].Get("content.0.text").String() != "THOUGHT_SUMMARY keep verbatim FILLER FILLER" {
		t.Fatalf("thought:true part mutated: %s", msgs[1].Raw)
	}
	if !msgs[1].Get("content.0.thought").Bool() {
		t.Fatalf("thought flag lost: %s", msgs[1].Raw)
	}
}

func TestOptimizePayloadPreservesMultimodalHistory(t *testing.T) {
	s := NewServer(Options{CompressionMode: "aggressive", HistoryKeepTurns: 1})
	in := []byte(`{
		"model":"gpt-4o",
		"messages":[
			{"role":"user","content":[{"type":"text","text":"old vision turn"},{"type":"image_url","image_url":{"url":"data:image/png;base64,aaa"}}]},
			{"role":"assistant","content":"saw it"},
			{"role":"user","content":"follow up only"}
		]
	}`)
	out, _, _ := s.optimizePayload(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if msgs[0].Get("content.1.type").String() != "image_url" {
		t.Fatalf("image_url must not be history-stubbed: %s", msgs[0].Raw)
	}
	if strings.Contains(msgs[0].Get("content").Raw, trimmer.MarkerHistoryCompacted) {
		t.Fatalf("multimodal user must not be stubbed: %s", msgs[0].Raw)
	}
}

func TestOptimizePayloadPreservesInputImageHistoryDefense(t *testing.T) {
	// Defense in depth: Responses input_image before migrate must not be history-stubbed
	// (GPT/Gemini vision turns on openai_compat doors).
	s := NewServer(Options{CompressionMode: "aggressive", HistoryKeepTurns: 1})
	in := []byte(`{
		"model":"gpt-4o",
		"messages":[
			{"role":"user","content":[
				{"type":"input_text","text":"old vision"},
				{"type":"input_image","image_url":"https://example.com/a.png"}
			]},
			{"role":"assistant","content":"saw it"},
			{"role":"user","content":"follow up"}
		]
	}`)
	out, _, _ := s.optimizePayload(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if msgs[0].Get("content.1.type").String() != "input_image" {
		t.Fatalf("input_image must not be history-stubbed: %s", msgs[0].Raw)
	}
	if strings.Contains(msgs[0].Get("content").Raw, trimmer.MarkerHistoryCompacted) {
		t.Fatalf("input_image turn must not be stubbed: %s", msgs[0].Raw)
	}
}

func TestOptimizePayloadPreservesRefusalHistory(t *testing.T) {
	// Official Chat Completions refusal parts must not be history-stubbed (same freeze
	// class as Deep contentPartFrozenForDeep refusal).
	s := NewServer(Options{CompressionMode: "aggressive", HistoryKeepTurns: 1})
	in := []byte(`{
		"model":"gpt-4o",
		"messages":[
			{"role":"assistant","content":[{"type":"refusal","refusal":"I cannot help with that","prompt_cache_breakpoint":{"mode":"explicit"}}]},
			{"role":"user","content":"ok try something else"},
			{"role":"assistant","content":"sure"},
			{"role":"user","content":"latest"}
		]
	}`)
	out, _, _ := s.optimizePayload(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if msgs[0].Get("content.0.type").String() != "refusal" {
		t.Fatalf("refusal must not be history-stubbed: %s", msgs[0].Raw)
	}
	if msgs[0].Get("content.0.refusal").String() != "I cannot help with that" {
		t.Fatalf("refusal text lost: %s", msgs[0].Raw)
	}
	if strings.Contains(msgs[0].Get("content").Raw, trimmer.MarkerHistoryCompacted) {
		t.Fatalf("refusal message must not be stubbed: %s", msgs[0].Raw)
	}
}

func TestOptimizePayloadPreservesMessageLevelRefusalAndAnnotations(t *testing.T) {
	// Official Chat Completions: assistant may carry top-level refusal + annotations[]
	// (url_citation) outside content[]. Fast must not history-stub or mutate those turns.
	s := NewServer(Options{CompressionMode: "aggressive", HistoryKeepTurns: 1})
	in := []byte(`{
		"model":"gpt-4o",
		"messages":[
			{"role":"assistant","content":"I looked it up FILLER FILLER FILLER","refusal":null,
			 "annotations":[{"type":"url_citation","url_citation":{"url":"https://example.com","title":"Ex","start_index":0,"end_index":12}}]},
			{"role":"user","content":"ok"},
			{"role":"assistant","content":null,"refusal":"I cannot help with that request"},
			{"role":"user","content":"latest"}
		]
	}`)
	out, _, _ := s.optimizePayload(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if msgs[0].Get("annotations.0.type").String() != "url_citation" {
		t.Fatalf("message-level annotations must not be history-stubbed: %s", msgs[0].Raw)
	}
	if msgs[0].Get("content").String() != "I looked it up FILLER FILLER FILLER" {
		t.Fatalf("annotated assistant content must stay verbatim: %s", msgs[0].Raw)
	}
	if msgs[2].Get("refusal").String() != "I cannot help with that request" {
		t.Fatalf("message-level refusal lost: %s", msgs[2].Raw)
	}
	if strings.Contains(msgs[0].Raw, trimmer.MarkerHistoryCompacted) ||
		strings.Contains(msgs[2].Raw, trimmer.MarkerHistoryCompacted) {
		t.Fatalf("annotated/refusal turns must not be stubbed: %s", string(out))
	}
}

func TestOptimizePayloadPreservesAssistantAudioHistory(t *testing.T) {
	// Official Chat Completions: message.audio is outside content[]. Stubbing destroys it.
	// Round-trip also pairs audio.id with content=transcript - Fast must not mutate that text.
	s := NewServer(Options{CompressionMode: "aggressive", HistoryKeepTurns: 1})
	in := []byte(`{
		"model":"gpt-4o-audio-preview",
		"messages":[
			{"role":"assistant","content":null,"audio":{"id":"audio_1","transcript":"hello there"}},
			{"role":"user","content":"ok"},
			{"role":"assistant","content":"sure FILLER FILLER FILLER transcript keep","audio":{"id":"audio_2"}},
			{"role":"user","content":"latest"}
		]
	}`)
	out, _, _ := s.optimizePayload(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if msgs[0].Get("audio.id").String() != "audio_1" {
		t.Fatalf("assistant audio must not be history-stubbed: %s", msgs[0].Raw)
	}
	if msgs[0].Get("audio.transcript").String() != "hello there" {
		t.Fatalf("audio transcript lost: %s", msgs[0].Raw)
	}
	if msgs[2].Get("content").String() != "sure FILLER FILLER FILLER transcript keep" {
		t.Fatalf("audio-paired assistant content must stay verbatim: %s", msgs[2].Raw)
	}
	if msgs[2].Get("audio.id").String() != "audio_2" {
		t.Fatalf("audio id lost on content+audio turn: %s", msgs[2].Raw)
	}
}

func TestOptimizePayloadPreservesServerToolResultHistory(t *testing.T) {
	s := NewServer(Options{CompressionMode: "aggressive", HistoryKeepTurns: 1})
	in := []byte(`{
		"model":"claude-opus-4",
		"messages":[
			{"role":"user","content":"run ls"},
			{"role":"assistant","content":[{"type":"server_tool_use","id":"s1","name":"bash","input":{}},{"type":"bash_code_execution_tool_result","tool_use_id":"s1","content":{"type":"bash_code_execution_result","stdout":"KEEP_OUT","stderr":"","return_code":0}}]},
			{"role":"user","content":"thanks"}
		]
	}`)
	out, _, _ := s.optimizePayload(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if strings.Contains(msgs[1].Get("content").Raw, trimmer.MarkerHistoryCompacted) {
		t.Fatalf("server tool assistant must not be history-stubbed: %s", msgs[1].Raw)
	}
	if !strings.Contains(msgs[1].Get("content").Raw, "KEEP_OUT") {
		t.Fatalf("bash result lost: %s", msgs[1].Raw)
	}
}

func TestOptimizePayloadPreservesSearchResultHistory(t *testing.T) {
	s := NewServer(Options{CompressionMode: "aggressive", HistoryKeepTurns: 1})
	in := []byte(`{
		"model":"claude-opus-4",
		"messages":[
			{"role":"user","content":[{"type":"search_result","source":"https://example.com","title":"Ex","content":[{"type":"text","text":"KEEP_RAG"}]},{"type":"text","text":"old q"}]},
			{"role":"assistant","content":"ok"},
			{"role":"user","content":"follow up"}
		]
	}`)
	out, _, _ := s.optimizePayload(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if strings.Contains(msgs[0].Get("content").Raw, trimmer.MarkerHistoryCompacted) {
		t.Fatalf("search_result user must not be history-stubbed: %s", msgs[0].Raw)
	}
	if !strings.Contains(msgs[0].Get("content").Raw, "KEEP_RAG") {
		t.Fatalf("search_result lost: %s", msgs[0].Raw)
	}
}

func TestNormalizeCursorBYOKMapsResponsesToolLoop(t *testing.T) {
	in := []byte(`{
		"model":"gemini-flash-latest",
		"max_output_tokens":128,
		"input":[
			{"type":"message","role":"user","content":"weather in Paris?"},
			{"type":"function_call","call_id":"call_1","name":"get_weather","arguments":"{\"city\":\"Paris\"}"},
			{"type":"function_call_output","call_id":"call_1","output":"{\"temp\":18}"},
			{"type":"reasoning","summary":[{"type":"summary_text","text":"skip me"}]},
			{"type":"message","role":"user","content":"thanks; what is 2+2?"}
		],
		"tools":[{"name":"get_weather","description":"weather","parameters":{"type":"object"}}]
	}`)
	out := normalizeCursorBYOKChatBody(in)
	if gjson.GetBytes(out, "input").Exists() {
		t.Fatalf("input should be removed: %s", string(out))
	}
	if gjson.GetBytes(out, "max_tokens").Int() != 128 {
		t.Fatalf("max_output_tokens not mapped: %s", string(out))
	}
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) != 4 {
		t.Fatalf("msgs=%d raw=%s", len(msgs), string(out))
	}
	if msgs[0].Get("role").String() != "user" {
		t.Fatalf("msg0=%s", msgs[0].Raw)
	}
	if msgs[1].Get("role").String() != "assistant" || !msgs[1].Get("tool_calls").Exists() {
		t.Fatalf("function_call must become assistant.tool_calls: %s", msgs[1].Raw)
	}
	if msgs[1].Get("tool_calls.0.function.name").String() != "get_weather" {
		t.Fatalf("tool name lost: %s", msgs[1].Raw)
	}
	if msgs[2].Get("role").String() != "tool" || msgs[2].Get("tool_call_id").String() != "call_1" {
		t.Fatalf("function_call_output mapping: %s", msgs[2].Raw)
	}
	if msgs[2].Get("name").String() != "get_weather" {
		t.Fatalf("tool name must be enriched from prior function_call: %s", msgs[2].Raw)
	}
	if !strings.Contains(msgs[2].Get("content").String(), "temp") {
		t.Fatalf("output field lost: %s", msgs[2].Raw)
	}
	// DeepSeek: reasoning summary merges into prior assistant as reasoning_content
	// (never becomes a user turn that would trigger Deep).
	if msgs[1].Get("reasoning_content").String() != "skip me" {
		t.Fatalf("reasoning must merge into assistant: %s", msgs[1].Raw)
	}
	if msgs[3].Get("content").String() != "thanks; what is 2+2?" {
		t.Fatalf("last user=%s", msgs[3].Raw)
	}
	if gjson.GetBytes(out, "tools.0.type").String() != "function" {
		t.Fatalf("flat tools not normalized: %s", string(out))
	}
	// After normalize, Deep gate must see trailing user (not tool chrome as user).
	estBody := out
	// Import cycle avoided: spot-check shape only here; deepopt tested separately.
	if msgs[len(msgs)-1].Get("role").String() != "user" {
		t.Fatal("trailing user required for Deep eligibility")
	}
	_ = estBody
}

func TestNormalizeCursorBYOKStringInput(t *testing.T) {
	out := normalizeCursorBYOKChatBody([]byte(`{"model":"gpt-5","input":"hello there"}`))
	if gjson.GetBytes(out, "messages.0.content").String() != "hello there" {
		t.Fatalf("%s", string(out))
	}
}

func TestNormalizeCursorBYOKMapsInstructionsToSystem(t *testing.T) {
	// Official Responses→Chat: instructions become system/developer guidance.
	// GPT / Gemini / DeepSeek / Mistral openai_compat doors need this inside messages.
	in := []byte(`{
		"model":"gpt-4o",
		"instructions":"KEEP_SYSTEM be concise",
		"max_output_tokens":64,
		"input":[{"type":"message","role":"user","content":"what is 2+2?"}]
	}`)
	out := normalizeCursorBYOKChatBody(in)
	if gjson.GetBytes(out, "instructions").Exists() {
		t.Fatalf("instructions must be removed after map: %s", string(out))
	}
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) != 2 {
		t.Fatalf("msgs=%d raw=%s", len(msgs), string(out))
	}
	if msgs[0].Get("role").String() != "system" || msgs[0].Get("content").String() != "KEEP_SYSTEM be concise" {
		t.Fatalf("system from instructions lost: %s", msgs[0].Raw)
	}
	if msgs[1].Get("role").String() != "user" || msgs[1].Get("content").String() != "what is 2+2?" {
		t.Fatalf("user lost: %s", msgs[1].Raw)
	}
	if gjson.GetBytes(out, "max_tokens").Int() != 64 {
		t.Fatalf("max_tokens: %s", string(out))
	}
}

func TestEnrichOpenAIToolMessageNames(t *testing.T) {
	// Chat Completions bodies that omit tool.name (common from some IDEs) get
	// filled from assistant.tool_calls for Mistral / openai_compat compatibility.
	in := []byte(`{
		"model":"mistral-large-latest",
		"messages":[
			{"role":"user","content":"weather?"},
			{"role":"assistant","tool_calls":[{"id":"c1","type":"function","function":{"name":"get_weather","arguments":"{}"}}]},
			{"role":"tool","tool_call_id":"c1","content":"18C"},
			{"role":"user","content":"thanks"}
		]
	}`)
	out := enrichOpenAIToolMessageNames(in)
	if gjson.GetBytes(out, "messages.2.name").String() != "get_weather" {
		t.Fatalf("name not enriched: %s", string(out))
	}
	// Already-present name must not be overwritten.
	in2 := []byte(`{
		"messages":[
			{"role":"assistant","tool_calls":[{"id":"c1","type":"function","function":{"name":"A","arguments":"{}"}}]},
			{"role":"tool","tool_call_id":"c1","name":"KEEP_ME","content":"x"}
		]
	}`)
	out2 := enrichOpenAIToolMessageNames(in2)
	if gjson.GetBytes(out2, "messages.1.name").String() != "KEEP_ME" {
		t.Fatalf("existing name overwritten: %s", string(out2))
	}
}

func TestNormalizeCursorBYOKPreservesGeminiThoughtSignature(t *testing.T) {
	in := []byte(`{
		"model":"gemini-3-flash",
		"input":[
			{"type":"message","role":"user","content":"fly?"},
			{"type":"function_call","call_id":"c1","name":"check_flight",
			 "arguments":"{}",
			 "extra_content":{"google":{"thought_signature":"SIG_KEEP"}}}
		]
	}`)
	out := normalizeCursorBYOKChatBody(in)
	sig := gjson.GetBytes(out, "messages.1.tool_calls.0.extra_content.google.thought_signature").String()
	if sig != "SIG_KEEP" {
		t.Fatalf("Gemini thought_signature dropped on function_call normalize: %s", string(out))
	}
}

func TestNormalizeCursorBYOKParallelFunctionCallsOneAssistant(t *testing.T) {
	// Official OpenAI Chat Completions + Gemini: parallel Responses function_call items
	// must become one assistant.tool_calls array. Interleaving assistant/tool pairs
	// causes Gemini HTTP 400 (must be FC1+sig, FC2, FR1, FR2).
	in := []byte(`{
		"model":"gemini-3-flash",
		"input":[
			{"type":"message","role":"user","content":"temps in Paris and London?"},
			{"type":"function_call","call_id":"c1","name":"get_temperature","arguments":"{\"location\":\"Paris\"}",
			 "extra_content":{"google":{"thought_signature":"SIG_A"}}},
			{"type":"function_call","call_id":"c2","name":"get_temperature","arguments":"{\"location\":\"London\"}"},
			{"type":"function_call_output","call_id":"c1","output":"18C"},
			{"type":"function_call_output","call_id":"c2","output":"14C"},
			{"type":"message","role":"user","content":"thanks; what is 2+2?"}
		]
	}`)
	out := normalizeCursorBYOKChatBody(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	// user + one assistant (2 tool_calls) + 2 tools + trailing user = 5
	if len(msgs) != 5 {
		t.Fatalf("want 5 msgs (user,asst,tool,tool,user), got %d: %s", len(msgs), string(out))
	}
	if msgs[1].Get("role").String() != "assistant" {
		t.Fatalf("msg1=%s", msgs[1].Raw)
	}
	tcs := msgs[1].Get("tool_calls").Array()
	if len(tcs) != 2 {
		t.Fatalf("parallel tool_calls lost: %s", msgs[1].Raw)
	}
	if tcs[0].Get("extra_content.google.thought_signature").String() != "SIG_A" {
		t.Fatalf("first-call thought_signature lost: %s", tcs[0].Raw)
	}
	if tcs[1].Get("id").String() != "c2" {
		t.Fatalf("second call lost: %s", msgs[1].Raw)
	}
	if msgs[2].Get("role").String() != "tool" || msgs[2].Get("tool_call_id").String() != "c1" {
		t.Fatalf("tool1=%s", msgs[2].Raw)
	}
	if msgs[3].Get("role").String() != "tool" || msgs[3].Get("tool_call_id").String() != "c2" {
		t.Fatalf("tool2=%s", msgs[3].Raw)
	}
	if msgs[4].Get("content").String() != "thanks; what is 2+2?" {
		t.Fatalf("trailing user lost: %s", msgs[4].Raw)
	}
	if msgs[2].Get("name").String() != "get_temperature" || msgs[3].Get("name").String() != "get_temperature" {
		t.Fatalf("tool names not enriched: %s / %s", msgs[2].Raw, msgs[3].Raw)
	}
}

func TestNormalizeCursorBYOKAttachesFunctionCallToPriorAssistantText(t *testing.T) {
	in := []byte(`{
		"model":"gpt-4o",
		"input":[
			{"type":"message","role":"user","content":"weather?"},
			{"type":"message","role":"assistant","content":"Let me check."},
			{"type":"function_call","call_id":"c1","name":"get_weather","arguments":"{}"},
			{"type":"function_call_output","call_id":"c1","output":"18C"}
		]
	}`)
	out := normalizeCursorBYOKChatBody(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) != 3 {
		t.Fatalf("want user+assistant+tool, got %d: %s", len(msgs), string(out))
	}
	if msgs[1].Get("content").String() != "Let me check." {
		t.Fatalf("assistant text lost: %s", msgs[1].Raw)
	}
	if msgs[1].Get("tool_calls.0.function.name").String() != "get_weather" {
		t.Fatalf("tool_calls not attached to prior assistant: %s", msgs[1].Raw)
	}
}

func TestNormalizeResponsesContentPartsToChatCompletions(t *testing.T) {
	// Official migrate: input_text→text, input_image→image_url, input_file→file.
	// Leaving Responses types on openai_compat doors (GPT/Gemini/DeepSeek/Mistral) → 400 / Deep blind.
	in := []byte(`{
		"model":"gpt-4o",
		"input":[
			{"type":"message","role":"user","content":[
				{"type":"input_text","text":"FILLER\nwhat is in this?"},
				{"type":"input_image","image_url":"https://example.com/a.png","detail":"high"},
				{"type":"input_file","file_id":"file_pdf_1"}
			]},
			{"type":"web_search_call","id":"ws_1"},
			{"type":"message","role":"assistant","content":[
				{"type":"output_text","text":"looks fine"}
			]}
		]
	}`)
	out := normalizeCursorBYOKChatBody(in)
	if gjson.GetBytes(out, "messages.0.content.0.type").String() != "text" {
		t.Fatalf("input_text not mapped: %s", string(out))
	}
	if gjson.GetBytes(out, "messages.0.content.0.text").String() != "FILLER\nwhat is in this?" {
		t.Fatalf("text lost: %s", string(out))
	}
	if gjson.GetBytes(out, "messages.0.content.1.type").String() != "image_url" {
		t.Fatalf("input_image not mapped: %s", string(out))
	}
	if gjson.GetBytes(out, "messages.0.content.1.image_url.url").String() != "https://example.com/a.png" {
		t.Fatalf("image url lost: %s", string(out))
	}
	if gjson.GetBytes(out, "messages.0.content.2.type").String() != "file" {
		t.Fatalf("input_file not mapped: %s", string(out))
	}
	if gjson.GetBytes(out, "messages.0.content.2.file.file_id").String() != "file_pdf_1" {
		t.Fatalf("file_id lost: %s", string(out))
	}
	// web_search_call must not become a user message.
	for _, m := range gjson.GetBytes(out, "messages").Array() {
		if strings.Contains(m.Raw, "web_search_call") || strings.Contains(m.Get("content").String(), "ws_1") {
			t.Fatalf("web_search_call leaked into chat messages: %s", string(out))
		}
	}
	if gjson.GetBytes(out, "messages.1.content.0.type").String() != "text" {
		t.Fatalf("output_text not mapped: %s", string(out))
	}
}

func TestNormalizeCursorBYOKSkipsRawAndUnknownTypedChrome(t *testing.T) {
	// Never dump item.Raw / unknown typed Responses chrome as role=user (garble class).
	in := []byte(`{
		"model":"gpt-4o",
		"input":[
			{"type":"message","role":"user","content":"real question"},
			{"type":"future_server_widget","id":"w1","payload":{"huge":"chrome"}},
			{"role":"user"},
			{"foo":"bar","baz":[1,2,3]}
		]
	}`)
	out := normalizeCursorBYOKChatBody(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) != 1 {
		t.Fatalf("expected only real user message, got %d: %s", len(msgs), string(out))
	}
	if msgs[0].Get("content").String() != "real question" {
		t.Fatalf("user lost: %s", string(out))
	}
	wire := string(out)
	if strings.Contains(wire, "future_server_widget") || strings.Contains(wire, `"foo":"bar"`) {
		t.Fatalf("unknown/raw chrome leaked into chat messages: %s", wire)
	}
}

func TestNormalizeOpenAICompatMessageContentPartsInPlace(t *testing.T) {
	// Already-messages body with Responses content types (no top-level input).
	in := []byte(`{
		"model":"gemini-flash-latest",
		"messages":[
			{"role":"user","content":[{"type":"input_text","text":"hello vision"},{"type":"input_image","image_url":"data:image/png;base64,aaa"}]}
		]
	}`)
	out := normalizeOpenAICompatMessageContentParts(in)
	if gjson.GetBytes(out, "messages.0.content.0.type").String() != "text" {
		t.Fatalf("in-place input_text: %s", string(out))
	}
	if gjson.GetBytes(out, "messages.0.content.1.type").String() != "image_url" {
		t.Fatalf("in-place input_image: %s", string(out))
	}
}

func TestNormalizeCursorBYOKMapsTextFormatToResponseFormat(t *testing.T) {
	// Official: Responses text.format → Chat Completions response_format.
	// Prior bug: we deleted text and silently dropped Structured Outputs for GPT.
	in := []byte(`{
		"model":"gpt-4o",
		"input":[{"type":"message","role":"user","content":"Jane, 54"}],
		"text":{"format":{
			"type":"json_schema",
			"name":"person",
			"strict":true,
			"schema":{"type":"object","properties":{"name":{"type":"string"},"age":{"type":"number"}},"required":["name","age"],"additionalProperties":false}
		}},
		"store":false,
		"previous_response_id":"resp_omit_me"
	}`)
	out := normalizeCursorBYOKChatBody(in)
	if gjson.GetBytes(out, "text").Exists() {
		t.Fatalf("text must be removed: %s", string(out))
	}
	// previous_response_id is Responses-only; store is valid Chat Completions (official OpenAI).
	if gjson.GetBytes(out, "previous_response_id").Exists() {
		t.Fatalf("previous_response_id must be dropped: %s", string(out))
	}
	if !gjson.GetBytes(out, "store").Exists() {
		t.Fatalf("Chat Completions store must be kept: %s", string(out))
	}
	if gjson.GetBytes(out, "response_format.type").String() != "json_schema" {
		t.Fatalf("response_format missing: %s", string(out))
	}
	if gjson.GetBytes(out, "response_format.json_schema.name").String() != "person" {
		t.Fatalf("json_schema.name lost: %s", string(out))
	}
	if !gjson.GetBytes(out, "response_format.json_schema.strict").Bool() {
		t.Fatalf("strict lost: %s", string(out))
	}
	if gjson.GetBytes(out, "response_format.json_schema.schema.type").String() != "object" {
		t.Fatalf("schema lost: %s", string(out))
	}

	in2 := []byte(`{"model":"gpt-4o","input":"hi","text":{"format":{"type":"json_object"}}}`)
	out2 := normalizeCursorBYOKChatBody(in2)
	if gjson.GetBytes(out2, "response_format.type").String() != "json_object" {
		t.Fatalf("json_object map failed: %s", string(out2))
	}
}

func TestNormalizeResponsesKeepsChatCacheAndSafetyFields(t *testing.T) {
	// Official OpenAI Chat Completions: prompt_cache_key + safety_identifier + service_tier
	// are valid on /v1/chat/completions. Responses→Chat must not strip them (GPT/Gemini/DeepSeek/Mistral).
	in := []byte(`{
		"model":"gpt-5.6",
		"input":[{"role":"user","content":[{"type":"input_text","text":"hi"}]}],
		"prompt_cache_key":"workflow-v1",
		"safety_identifier":"hashed-user-1",
		"service_tier":"auto",
		"prompt_cache_options":{"mode":"explicit","ttl":"30m"},
		"store":false,
		"stream":true,
		"stream_options":{"include_usage":true,"include_obfuscation":false},
		"previous_response_id":"resp_should_drop"
	}`)
	out := NormalizeOpenAICompatRequestBody(in)
	if !gjson.GetBytes(out, "messages").IsArray() {
		t.Fatalf("expected messages: %s", string(out))
	}
	if gjson.GetBytes(out, "prompt_cache_key").String() != "workflow-v1" {
		t.Fatalf("prompt_cache_key lost: %s", string(out))
	}
	if gjson.GetBytes(out, "safety_identifier").String() != "hashed-user-1" {
		t.Fatalf("safety_identifier lost: %s", string(out))
	}
	if gjson.GetBytes(out, "service_tier").String() != "auto" {
		t.Fatalf("service_tier lost: %s", string(out))
	}
	if gjson.GetBytes(out, "prompt_cache_options.mode").String() != "explicit" {
		t.Fatalf("prompt_cache_options lost: %s", string(out))
	}
	if !gjson.GetBytes(out, "store").Exists() || gjson.GetBytes(out, "store").Bool() {
		t.Fatalf("Chat Completions store must be kept (false): %s", string(out))
	}
	if !gjson.GetBytes(out, "stream_options.include_usage").Bool() {
		t.Fatalf("stream_options.include_usage lost: %s", string(out))
	}
	if gjson.GetBytes(out, "stream_options.include_obfuscation").Bool() {
		t.Fatalf("stream_options.include_obfuscation should stay false: %s", string(out))
	}
	if gjson.GetBytes(out, "previous_response_id").Exists() {
		t.Fatalf("Responses-only previous_response_id must be dropped: %s", string(out))
	}
	if gjson.GetBytes(out, "input").Exists() {
		t.Fatalf("input must be removed after normalize: %s", string(out))
	}
}

func TestNormalizeResponsesMapsTextVerbosityAndDropsResponsesOnlyKnobs(t *testing.T) {
	// Official migrate: text.verbosity → verbosity; max_tool_calls/background/context_management
	// are Responses-only (no Chat Completions equivalent).
	in := []byte(`{
		"model":"gpt-5",
		"input":"hi",
		"text":{"verbosity":"low","format":{"type":"json_object"}},
		"max_tool_calls":3,
		"background":true,
		"context_management":[{"type":"compaction","compact_threshold":200000}],
		"parallel_tool_calls":false,
		"metadata":{"app":"trim"}
	}`)
	out := NormalizeOpenAICompatRequestBody(in)
	if gjson.GetBytes(out, "text").Exists() {
		t.Fatalf("Responses text must be dropped: %s", string(out))
	}
	if gjson.GetBytes(out, "verbosity").String() != "low" {
		t.Fatalf("text.verbosity must map to verbosity: %s", string(out))
	}
	if gjson.GetBytes(out, "response_format.type").String() != "json_object" {
		t.Fatalf("text.format must still map: %s", string(out))
	}
	if gjson.GetBytes(out, "max_tool_calls").Exists() ||
		gjson.GetBytes(out, "background").Exists() ||
		gjson.GetBytes(out, "context_management").Exists() {
		t.Fatalf("Responses-only knobs must be dropped: %s", string(out))
	}
	if gjson.GetBytes(out, "parallel_tool_calls").Bool() {
		t.Fatalf("Chat parallel_tool_calls must be kept (false): %s", string(out))
	}
	if gjson.GetBytes(out, "metadata.app").String() != "trim" {
		t.Fatalf("Chat metadata must be kept: %s", string(out))
	}
}

func TestNormalizeToolChoiceResponsesFlatToChatNested(t *testing.T) {
	// Official: Responses {"type":"function","name":"x"} → Chat {"type":"function","function":{"name":"x"}}.
	// Leaving the flat shape 400s GPT/Gemini/DeepSeek/Mistral openai_compat.
	in := []byte(`{
		"model":"gpt-4o",
		"input":"weather?",
		"tools":[{"type":"function","name":"get_weather","parameters":{"type":"object"}}],
		"tool_choice":{"type":"function","name":"get_weather"}
	}`)
	out := NormalizeOpenAICompatRequestBody(in)
	if gjson.GetBytes(out, "tool_choice.type").String() != "function" {
		t.Fatalf("tool_choice.type: %s", string(out))
	}
	if gjson.GetBytes(out, "tool_choice.function.name").String() != "get_weather" {
		t.Fatalf("flat tool_choice not nested: %s", string(out))
	}
	if gjson.GetBytes(out, "tool_choice.name").Exists() {
		t.Fatalf("Responses flat name must not remain at top level: %s", string(out))
	}
	// Already Chat-shaped must stay.
	in2 := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}],"tool_choice":{"type":"function","function":{"name":"keep_me"}}}`)
	out2 := NormalizeOpenAICompatRequestBody(in2)
	if gjson.GetBytes(out2, "tool_choice.function.name").String() != "keep_me" {
		t.Fatalf("Chat-shaped tool_choice mutated: %s", string(out2))
	}
	// Responses-only hosted tool_choice → auto (no Chat equivalent).
	in3 := []byte(`{"model":"gpt-4o","input":"hi","tool_choice":{"type":"web_search_preview"}}`)
	out3 := NormalizeOpenAICompatRequestBody(in3)
	if gjson.GetBytes(out3, "tool_choice").String() != "auto" {
		t.Fatalf("hosted tool_choice must fall back to auto: %s", string(out3))
	}
	// allowed_tools: keep function refs, drop hosted refs.
	in4 := []byte(`{
		"model":"gpt-4o",
		"input":"hi",
		"tool_choice":{"type":"allowed_tools","mode":"required","tools":[
			{"type":"function","name":"add"},
			{"type":"web_search_preview"},
			{"type":"mcp","server_label":"deepwiki"}
		]}
	}`)
	out4 := NormalizeOpenAICompatRequestBody(in4)
	if gjson.GetBytes(out4, "tool_choice.type").String() != "allowed_tools" {
		t.Fatalf("allowed_tools lost: %s", string(out4))
	}
	if gjson.GetBytes(out4, "tool_choice.mode").String() != "required" {
		t.Fatalf("mode lost: %s", string(out4))
	}
	refs := gjson.GetBytes(out4, "tool_choice.tools").Array()
	if len(refs) != 1 || refs[0].Get("name").String() != "add" {
		t.Fatalf("hosted refs must be stripped from allowed_tools: %s", string(out4))
	}
	// Official Chat: custom tool_choice nests under custom.name - never force to auto.
	in5 := []byte(`{
		"model":"gpt-5",
		"input":"run code",
		"tools":[{"type":"custom","name":"code_exec","description":"exec"}],
		"tool_choice":{"type":"custom","name":"code_exec"}
	}`)
	out5 := NormalizeOpenAICompatRequestBody(in5)
	if gjson.GetBytes(out5, "tool_choice.type").String() != "custom" {
		t.Fatalf("custom tool_choice forced away: %s", string(out5))
	}
	if gjson.GetBytes(out5, "tool_choice.custom.name").String() != "code_exec" {
		t.Fatalf("flat custom tool_choice not nested: %s", string(out5))
	}
	if gjson.GetBytes(out5, "tool_choice").String() == "auto" {
		t.Fatalf("custom must not become auto: %s", string(out5))
	}
	// Flat Responses custom tool → nested Chat custom.custom.
	if gjson.GetBytes(out5, "tools.0.type").String() != "custom" {
		t.Fatalf("custom tool type lost: %s", string(out5))
	}
	if gjson.GetBytes(out5, "tools.0.custom.name").String() != "code_exec" {
		t.Fatalf("flat custom tool not nested: %s", string(out5))
	}
	// allowed_tools keeps custom refs alongside function refs.
	in6 := []byte(`{
		"model":"gpt-5",
		"input":"hi",
		"tool_choice":{"type":"allowed_tools","mode":"auto","tools":[
			{"type":"function","name":"add"},
			{"type":"custom","name":"code_exec"},
			{"type":"web_search_preview"}
		]}
	}`)
	out6 := NormalizeOpenAICompatRequestBody(in6)
	refs6 := gjson.GetBytes(out6, "tool_choice.tools").Array()
	if len(refs6) != 2 {
		t.Fatalf("want function+custom kept, got %d: %s", len(refs6), string(out6))
	}
}

func TestNormalizeCustomToolCallToChatCustomShape(t *testing.T) {
	// Official: Responses custom_tool_call → Chat {type:"custom", custom:{name,input}}.
	// Mapping to type:function with JSON arguments smashes custom-tool protocol.
	in := []byte(`{
		"model":"gpt-5",
		"input":[
			{"type":"message","role":"user","content":"print hello"},
			{"type":"custom_tool_call","call_id":"call_c1","name":"code_exec","input":"print(\"hello\")"},
			{"type":"custom_tool_call_output","call_id":"call_c1","name":"code_exec","output":"hello"}
		]
	}`)
	out := NormalizeOpenAICompatRequestBody(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) < 3 {
		t.Fatalf("want user+assistant+tool, got %d: %s", len(msgs), string(out))
	}
	tc := msgs[1].Get("tool_calls.0")
	if tc.Get("type").String() != "custom" {
		t.Fatalf("custom_tool_call must become type=custom, got: %s", tc.Raw)
	}
	if tc.Get("custom.name").String() != "code_exec" {
		t.Fatalf("custom.name lost: %s", tc.Raw)
	}
	if tc.Get("custom.input").String() != `print("hello")` {
		t.Fatalf("custom.input lost: %s", tc.Raw)
	}
	if tc.Get("function").Exists() {
		t.Fatalf("must not smash custom into function: %s", tc.Raw)
	}
	toolMsg := msgs[2]
	if toolMsg.Get("role").String() != "tool" || toolMsg.Get("tool_call_id").String() != "call_c1" {
		t.Fatalf("custom output → role=tool: %s", toolMsg.Raw)
	}
	if toolMsg.Get("name").String() != "code_exec" {
		t.Fatalf("tool name from custom.name: %s", toolMsg.Raw)
	}
}

func TestNormalizeDropsResponsesBuiltinToolsKeepsFunctions(t *testing.T) {
	// Official: Responses built-ins (web_search_preview, …) ≠ Chat Completions function tools.
	// Forwarding them 400s openai_compat doors; must drop, keep callable functions + strict.
	in := []byte(`{
		"model":"gpt-4o",
		"input":"search then calc",
		"tools":[
			{"type":"web_search_preview"},
			{"type":"file_search","vector_store_ids":["vs_1"]},
			{"type":"function","name":"add","strict":true,"parameters":{"type":"object","properties":{"a":{"type":"number"}},"required":["a"],"additionalProperties":false}},
			{"name":"flat_lookup","description":"d","parameters":{"type":"object"},"cache_control":{"type":"ephemeral"}}
		]
	}`)
	out := NormalizeOpenAICompatRequestBody(in)
	tools := gjson.GetBytes(out, "tools").Array()
	if len(tools) != 2 {
		t.Fatalf("want 2 function tools (builtins dropped), got %d: %s", len(tools), string(out))
	}
	for _, ttool := range tools {
		if ttool.Get("type").String() != "function" || !ttool.Get("function").Exists() {
			t.Fatalf("expected Chat function tool: %s", ttool.Raw)
		}
		name := ttool.Get("function.name").String()
		if name != "add" && name != "flat_lookup" {
			t.Fatalf("unexpected tool name %q: %s", name, string(out))
		}
		if name == "add" && !ttool.Get("function.strict").Bool() {
			t.Fatalf("strict lost on add: %s", ttool.Raw)
		}
		if name == "flat_lookup" && ttool.Get("cache_control.type").String() != "ephemeral" {
			t.Fatalf("cache_control lost on flat_lookup: %s", ttool.Raw)
		}
	}
	wire := string(out)
	if strings.Contains(wire, "web_search_preview") || strings.Contains(wire, "file_search") {
		t.Fatalf("Responses builtins must not leak: %s", wire)
	}
}

func TestNormalizeResponsesMapsReasoningEffortAndMaxCompletionTokens(t *testing.T) {
	// Official OpenAI: o-series / GPT-5 reject max_tokens - use max_completion_tokens only.
	in := []byte(`{
		"model":"o4-mini",
		"input":"solve 2+2",
		"max_output_tokens":4096,
		"reasoning":{"effort":"medium"}
	}`)
	out := NormalizeOpenAICompatRequestBody(in)
	if gjson.GetBytes(out, "reasoning").Exists() {
		t.Fatalf("Responses-only reasoning object must be dropped: %s", string(out))
	}
	if gjson.GetBytes(out, "reasoning_effort").String() != "medium" {
		t.Fatalf("reasoning.effort must map to reasoning_effort: %s", string(out))
	}
	if gjson.GetBytes(out, "max_output_tokens").Exists() {
		t.Fatalf("max_output_tokens must be dropped: %s", string(out))
	}
	if gjson.GetBytes(out, "max_tokens").Exists() {
		t.Fatalf("o-series must not keep max_tokens: %s", string(out))
	}
	if gjson.GetBytes(out, "max_completion_tokens").Int() != 4096 {
		t.Fatalf("max_completion_tokens missing for o-series: %s", string(out))
	}
	// Existing Chat reasoning_effort must win; max_tokens migrates away on o-series.
	in2 := []byte(`{
		"model":"o4-mini",
		"input":"hi",
		"reasoning_effort":"low",
		"reasoning":{"effort":"high"},
		"max_tokens":100,
		"max_output_tokens":999
	}`)
	out2 := NormalizeOpenAICompatRequestBody(in2)
	if gjson.GetBytes(out2, "reasoning_effort").String() != "low" {
		t.Fatalf("existing reasoning_effort must win: %s", string(out2))
	}
	if gjson.GetBytes(out2, "max_tokens").Exists() {
		t.Fatalf("o-series max_tokens must be removed: %s", string(out2))
	}
	if gjson.GetBytes(out2, "max_completion_tokens").Int() != 999 {
		t.Fatalf("max_completion_tokens should fill from max_output_tokens: %s", string(out2))
	}
	// Non-reasoning openai_compat: max_tokens only (no max_completion_tokens injection).
	in3 := []byte(`{"model":"deepseek-chat","input":"hi","max_output_tokens":64}`)
	out3 := NormalizeOpenAICompatRequestBody(in3)
	if gjson.GetBytes(out3, "max_tokens").Int() != 64 {
		t.Fatalf("max_tokens map failed: %s", string(out3))
	}
	if gjson.GetBytes(out3, "max_completion_tokens").Exists() {
		t.Fatalf("must not inject max_completion_tokens without reasoning_effort: %s", string(out3))
	}

	// GPT-5 Chat body with max_tokens only → max_completion_tokens.
	in4 := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hi"}],"max_tokens":256,"stop":["\n"]}`)
	out4 := NormalizeOpenAICompatRequestBody(in4)
	if gjson.GetBytes(out4, "max_tokens").Exists() {
		t.Fatalf("gpt-5 must drop max_tokens: %s", string(out4))
	}
	if gjson.GetBytes(out4, "max_completion_tokens").Int() != 256 {
		t.Fatalf("gpt-5 must move to max_completion_tokens: %s", string(out4))
	}
}

func TestSanitizeOpenAIReasoningUnsupportedSampling(t *testing.T) {
	in := []byte(`{
		"model":"gpt-5.4",
		"messages":[{"role":"user","content":"hi"}],
		"temperature":0.2,
		"top_p":0.9,
		"presence_penalty":0.5,
		"frequency_penalty":0.1,
		"logprobs":true,
		"top_logprobs":2,
		"logit_bias":{"42":1},
		"stop":["\n"],
		"seed":42,
		"n":2,
		"best_of":3,
		"max_tokens":100
	}`)
	out := NormalizeOpenAICompatRequestBody(in)
	for _, key := range []string{
		"temperature", "top_p", "presence_penalty", "frequency_penalty",
		"logprobs", "top_logprobs", "logit_bias", "stop", "max_tokens",
		"seed", "n", "best_of",
	} {
		if gjson.GetBytes(out, key).Exists() {
			t.Fatalf("%s must be stripped for gpt-5 reasoning: %s", key, string(out))
		}
	}
	if gjson.GetBytes(out, "max_completion_tokens").Int() != 100 {
		t.Fatalf("max_tokens must move to max_completion_tokens: %s", string(out))
	}

	// gpt-4o keeps temperature.
	legacy := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}],"temperature":0.2}`)
	out2 := NormalizeOpenAICompatRequestBody(legacy)
	if gjson.GetBytes(out2, "temperature").Float() != 0.2 {
		t.Fatalf("gpt-4o must keep temperature: %s", string(out2))
	}

	// Official OpenAI: gpt-5-chat-latest is non-reasoning and keeps temperature/max_tokens.
	chat := []byte(`{
		"model":"gpt-5-chat-latest",
		"messages":[{"role":"user","content":"hi"}],
		"temperature":0.4,
		"max_tokens":128
	}`)
	chatOut := NormalizeOpenAICompatRequestBody(chat)
	if gjson.GetBytes(chatOut, "temperature").Float() != 0.4 {
		t.Fatalf("gpt-5-chat-latest must keep temperature: %s", string(chatOut))
	}
	if gjson.GetBytes(chatOut, "max_tokens").Int() != 128 {
		t.Fatalf("gpt-5-chat-latest must keep max_tokens: %s", string(chatOut))
	}
	if gjson.GetBytes(chatOut, "max_completion_tokens").Exists() {
		t.Fatalf("gpt-5-chat-latest must not migrate to max_completion_tokens: %s", string(chatOut))
	}
}

func TestSanitizeGeminiOpenAICompatUnsupportedFields(t *testing.T) {
	in := []byte(`{
		"model":"gemini-2.5-flash",
		"messages":[{"role":"user","content":"hi"}],
		"store":false,
		"metadata":{"user":"x"},
		"logprobs":true,
		"top_logprobs":3,
		"verbosity":"high",
		"service_tier":"auto",
		"safety_identifier":"u1",
		"prompt_cache_key":"k1",
		"parallel_tool_calls":true,
		"presence_penalty":0.1,
		"frequency_penalty":0.2,
		"seed":42,
		"logit_bias":{"1":1}
	}`)
	out := NormalizeOpenAICompatRequestBody(in)
	for _, key := range []string{
		"store", "metadata", "logprobs", "top_logprobs",
		"verbosity", "service_tier", "safety_identifier", "prompt_cache_key",
		"parallel_tool_calls", "presence_penalty", "frequency_penalty", "seed", "logit_bias",
	} {
		if gjson.GetBytes(out, key).Exists() {
			t.Fatalf("Gemini must strip %s: %s", key, string(out))
		}
	}
	// Non-Gemini keeps store (OpenAI accepts it).
	gpt := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}],"store":false}`)
	out2 := NormalizeOpenAICompatRequestBody(gpt)
	if !gjson.GetBytes(out2, "store").Exists() {
		t.Fatalf("GPT must keep store: %s", string(out2))
	}
}

func TestMapDeveloperRoleForOpenAICompat(t *testing.T) {
	// Official DeepSeek/Gemini/Mistral: role=developer → 400. Map to system.
	for _, model := range []string{"deepseek-v4-pro", "gemini-2.5-flash", "mistral-large-latest", "codestral-latest", "MiniMax-M2.5", "kimi-k2.6"} {
		in := []byte(`{"model":"` + model + `","messages":[{"role":"developer","content":"DEV_KEEP"},{"role":"user","content":"hi"}]}`)
		out := NormalizeOpenAICompatRequestBody(in)
		if gjson.GetBytes(out, "messages.0.role").String() != "system" {
			t.Fatalf("%s developer must → system: %s", model, string(out))
		}
		if gjson.GetBytes(out, "messages.0.content").String() != "DEV_KEEP" {
			t.Fatalf("%s developer content lost: %s", model, string(out))
		}
	}
	// Official OpenAI: GPT keeps developer.
	gpt := []byte(`{"model":"gpt-5.4","messages":[{"role":"developer","content":"DEV_KEEP"},{"role":"user","content":"hi"}]}`)
	gptOut := NormalizeOpenAICompatRequestBody(gpt)
	if gjson.GetBytes(gptOut, "messages.0.role").String() != "developer" {
		t.Fatalf("GPT must keep developer: %s", string(gptOut))
	}
}

func TestSanitizeMistralOpenAICompatRequest(t *testing.T) {
	in := []byte(`{
		"model":"mistral-large-latest",
		"messages":[{"role":"user","content":"hi"}],
		"store":false,
		"max_completion_tokens":512,
		"logit_bias":{"1":1},
		"logprobs":true,
		"tool_choice":"required",
		"temperature":1.7,
		"presence_penalty":0.1,
		"reasoning_effort":"high"
	}`)
	out := NormalizeOpenAICompatRequestBody(in)
	for _, key := range []string{"store", "max_completion_tokens", "logit_bias", "logprobs"} {
		if gjson.GetBytes(out, key).Exists() {
			t.Fatalf("Mistral must strip %s: %s", key, string(out))
		}
	}
	if gjson.GetBytes(out, "max_tokens").Int() != 512 {
		t.Fatalf("max_completion_tokens must migrate to max_tokens: %s", string(out))
	}
	if gjson.GetBytes(out, "tool_choice").String() != "any" {
		t.Fatalf("required must map to any: %s", string(out))
	}
	if gjson.GetBytes(out, "temperature").Float() != 1.0 {
		t.Fatalf("temperature >1 must clamp to 1: %s", string(out))
	}
	// Official Mistral docs: keep presence_penalty + reasoning_effort.
	if !gjson.GetBytes(out, "presence_penalty").Exists() {
		t.Fatalf("presence_penalty must stay: %s", string(out))
	}
	if gjson.GetBytes(out, "reasoning_effort").String() != "high" {
		t.Fatalf("reasoning_effort must stay: %s", string(out))
	}
	// GPT must not be rewritten by Mistral sanitize.
	gpt := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}],"store":false,"tool_choice":"required","temperature":1.7}`)
	gptOut := NormalizeOpenAICompatRequestBody(gpt)
	if !gjson.GetBytes(gptOut, "store").Exists() {
		t.Fatalf("GPT store must stay: %s", string(gptOut))
	}
	if gjson.GetBytes(gptOut, "tool_choice").String() != "required" {
		t.Fatalf("GPT tool_choice required must stay: %s", string(gptOut))
	}
}

func TestSanitizeGPT54ReasoningEffortWithTools(t *testing.T) {
	// Official OpenAI: GPT-5.4+ Chat Completions rejects tools + reasoning_effort≠none (HTTP 400).
	in := []byte(`{
		"model":"gpt-5.4",
		"reasoning_effort":"high",
		"tools":[{"type":"function","function":{"name":"lookup","parameters":{"type":"object"}}}],
		"messages":[{"role":"user","content":"hi"}]
	}`)
	out := NormalizeOpenAICompatRequestBody(in)
	if gjson.GetBytes(out, "reasoning_effort").String() != "none" {
		t.Fatalf("gpt-5.4+tools must force reasoning_effort=none: %s", string(out))
	}

	// Responses→Chat injects reasoning.effort then must sanitize under tools.
	in2 := []byte(`{
		"model":"gpt-5.5-mini",
		"input":"use the tool",
		"reasoning":{"effort":"medium"},
		"tools":[{"type":"function","name":"lookup","parameters":{"type":"object"}}]
	}`)
	out2 := NormalizeOpenAICompatRequestBody(in2)
	if gjson.GetBytes(out2, "reasoning_effort").String() != "none" {
		t.Fatalf("Responses migrate must sanitize gpt-5.5+tools: %s", string(out2))
	}

	// Without tools, effort must survive.
	in3 := []byte(`{"model":"gpt-5.4","reasoning_effort":"high","messages":[{"role":"user","content":"hi"}]}`)
	out3 := NormalizeOpenAICompatRequestBody(in3)
	if gjson.GetBytes(out3, "reasoning_effort").String() != "high" {
		t.Fatalf("gpt-5.4 without tools must keep effort: %s", string(out3))
	}

	// Older GPT-5.x (5.2/5.3) must keep effort+tools.
	in4 := []byte(`{
		"model":"gpt-5.2",
		"reasoning_effort":"high",
		"tools":[{"type":"function","function":{"name":"lookup","parameters":{"type":"object"}}}],
		"messages":[{"role":"user","content":"hi"}]
	}`)
	out4 := NormalizeOpenAICompatRequestBody(in4)
	if gjson.GetBytes(out4, "reasoning_effort").String() != "high" {
		t.Fatalf("gpt-5.2+tools must keep effort: %s", string(out4))
	}

	// Azure gpt-5.6+: tools without reasoning_effort still 400 (default medium).
	// Must inject none even when the field was omitted.
	in5 := []byte(`{
		"model":"gpt-5.6",
		"tools":[{"type":"function","function":{"name":"lookup","parameters":{"type":"object"}}}],
		"messages":[{"role":"user","content":"hi"}]
	}`)
	out5 := NormalizeOpenAICompatRequestBody(in5)
	if gjson.GetBytes(out5, "reasoning_effort").String() != "none" {
		t.Fatalf("gpt-5.6+tools with omitted effort must inject none: %s", string(out5))
	}
}

func TestEnsureContiguousToolResultsAfterToolCalls(t *testing.T) {
	// Interleaved user between parallel tool results (Gemini/OpenAI-strict 400 class).
	in := []byte(`{
		"model":"gemini-3-flash",
		"messages":[
			{"role":"user","content":"weather?"},
			{"role":"assistant","content":null,"tool_calls":[
				{"id":"c1","type":"function","function":{"name":"get_weather","arguments":"{}"}},
				{"id":"c2","type":"function","function":{"name":"get_weather","arguments":"{}"}}
			]},
			{"role":"user","content":"hurry"},
			{"role":"tool","tool_call_id":"c1","content":"18C"},
			{"role":"tool","tool_call_id":"c2","content":"14C"}
		]
	}`)
	out := NormalizeOpenAICompatRequestBody(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) < 5 {
		t.Fatalf("messages lost: %s", string(out))
	}
	if msgs[1].Get("role").String() != "assistant" {
		t.Fatalf("assistant must stay: %s", string(out))
	}
	if msgs[2].Get("role").String() != "tool" || msgs[2].Get("tool_call_id").String() != "c1" {
		t.Fatalf("first tool must follow assistant immediately: %s", string(out))
	}
	if msgs[3].Get("role").String() != "tool" || msgs[3].Get("tool_call_id").String() != "c2" {
		t.Fatalf("second tool must follow first tool: %s", string(out))
	}
	if msgs[4].Get("role").String() != "user" || !strings.Contains(msgs[4].Get("content").String(), "hurry") {
		t.Fatalf("interleaved user must move after tools: %s", string(out))
	}

	// Already contiguous: unchanged.
	ok := []byte(`{
		"model":"gpt-4o",
		"messages":[
			{"role":"assistant","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"f","arguments":"{}"}}]},
			{"role":"tool","tool_call_id":"c1","content":"ok"},
			{"role":"user","content":"thanks"}
		]
	}`)
	out2 := EnsureContiguousToolResultsAfterToolCalls(ok)
	if string(out2) != string(ok) {
		t.Fatalf("contiguous history must stay byte-stable: %s", string(out2))
	}
}

func TestNormalizeResponsesPreservesAssistantPhase(t *testing.T) {
	// Official OpenAI GPT-5.4+/5.5: assistant phase must survive Responses→Chat replay.
	in := []byte(`{
		"model":"gpt-5.5",
		"input":[
			{"type":"message","role":"user","content":[{"type":"input_text","text":"hi"}]},
			{"type":"message","role":"assistant","phase":"commentary","content":[{"type":"output_text","text":"checking tools…"}]},
			{"type":"message","role":"assistant","phase":"final_answer","content":[{"type":"output_text","text":"done"}]}
		]
	}`)
	out := NormalizeOpenAICompatRequestBody(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) < 3 {
		t.Fatalf("messages missing: %s", string(out))
	}
	if msgs[1].Get("phase").String() != "commentary" {
		t.Fatalf("commentary phase lost: %s", msgs[1].Raw)
	}
	if msgs[2].Get("phase").String() != "final_answer" {
		t.Fatalf("final_answer phase lost: %s", msgs[2].Raw)
	}
}

func TestOptimizePayloadFreezesAssistantPhase(t *testing.T) {
	s := NewServer(Options{CompressionMode: "aggressive", HistoryKeepTurns: 1})
	in := []byte(`{
		"model":"gpt-5.5",
		"messages":[
			{"role":"user","content":"old turn collapse candidate"},
			{"role":"assistant","phase":"commentary","content":"I will check FILLER FILLER FILLER now"},
			{"role":"user","content":"next"}
		]
	}`)
	out, _, _ := s.optimizePayload(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if strings.Contains(msgs[1].Get("content").Raw, trimmer.MarkerHistoryCompacted) {
		t.Fatalf("phased assistant must not be history-stubbed: %s", msgs[1].Raw)
	}
	if msgs[1].Get("content").String() != "I will check FILLER FILLER FILLER now" {
		t.Fatalf("phased assistant content mutated: %s", msgs[1].Raw)
	}
	if msgs[1].Get("phase").String() != "commentary" {
		t.Fatalf("phase lost: %s", msgs[1].Raw)
	}
}

func TestOptimizePayloadFreezesMistralPrefixAssistant(t *testing.T) {
	// Official Mistral: assistant prefix:true content is prepended to the completion.
	// Fast must not mutate or history-stub that text (forced-prefix contract).
	s := NewServer(Options{CompressionMode: "aggressive", HistoryKeepTurns: 1})
	in := []byte(`{
		"model":"mistral-large-latest",
		"messages":[
			{"role":"user","content":"old turn collapse candidate"},
			{"role":"assistant","content":"The Capital of France is FILLER FILLER","prefix":true},
			{"role":"user","content":"next"}
		]
	}`)
	out, _, _ := s.optimizePayload(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if strings.Contains(msgs[1].Get("content").Raw, trimmer.MarkerHistoryCompacted) {
		t.Fatalf("prefix assistant must not be history-stubbed: %s", msgs[1].Raw)
	}
	if msgs[1].Get("content").String() != "The Capital of France is FILLER FILLER" {
		t.Fatalf("prefix assistant content mutated: %s", msgs[1].Raw)
	}
	if !msgs[1].Get("prefix").Bool() {
		t.Fatalf("prefix flag lost: %s", msgs[1].Raw)
	}
}

func TestNormalizeFlatToolsPreservesAnthropicDatedServerTools(t *testing.T) {
	// Dated Anthropic server tools must never become Chat function tools.
	// Client input_schema tools still wrap on Chat door (openai_compat / openai_to_anthropic);
	// native /v1/messages is protected by the path gate (never calls this normalizer).
	in := []byte(`{
		"model":"gpt-4o",
		"messages":[{"role":"user","content":"hi"}],
		"tools":[
			{"name":"lookup","description":"d","input_schema":{"type":"object"},"cache_control":{"type":"ephemeral"}},
			{"type":"bash_20250124","name":"bash"}
		]
	}`)
	out := NormalizeOpenAICompatRequestBody(in)
	if gjson.GetBytes(out, "tools.0.type").String() != "function" {
		t.Fatalf("Chat door should wrap client input_schema into function: %s", string(out))
	}
	if gjson.GetBytes(out, "tools.0.function.name").String() != "lookup" {
		t.Fatalf("function name lost: %s", string(out))
	}
	if gjson.GetBytes(out, "tools.0.cache_control.type").String() != "ephemeral" {
		t.Fatalf("cache_control lost: %s", string(out))
	}
	if gjson.GetBytes(out, "tools.1.type").String() != "bash_20250124" {
		t.Fatalf("Anthropic dated server tool smashed: %s", string(out))
	}
	if gjson.GetBytes(out, "tools.1.function").Exists() {
		t.Fatalf("dated server tool must not wrap into function: %s", string(out))
	}
}

func TestNormalizeFlatToolsPreservesStrict(t *testing.T) {
	// Official OpenAI Responses + DeepSeek: strict lives at tool top-level (Responses)
	// or function.strict (Chat). Dropping it silently disables strict tool mode.
	in := []byte(`{
		"model":"gpt-4o",
		"input":"weather?",
		"tools":[{
			"type":"function",
			"name":"get_weather",
			"description":"Get weather",
			"strict":true,
			"parameters":{"type":"object","properties":{"location":{"type":"string"}},"required":["location"],"additionalProperties":false}
		}]
	}`)
	out := NormalizeOpenAICompatRequestBody(in)
	if gjson.GetBytes(out, "tools.0.type").String() != "function" {
		t.Fatalf("expected Chat function wrap: %s", string(out))
	}
	if gjson.GetBytes(out, "tools.0.function.name").String() != "get_weather" {
		t.Fatalf("name lost: %s", string(out))
	}
	if !gjson.GetBytes(out, "tools.0.function.strict").Bool() {
		t.Fatalf("strict must nest under function: %s", string(out))
	}
}

func TestNormalizePreservesRefusalAndMultimodalCacheBreakpoints(t *testing.T) {
	// Official OpenAI Chat Completions: prompt_cache_breakpoint on image_url / file / refusal.
	in := []byte(`{
		"model":"gpt-5.6",
		"input":[{
			"type":"message","role":"user","content":[
				{"type":"input_image","image_url":"https://example.com/a.png","prompt_cache_breakpoint":{"mode":"explicit"}},
				{"type":"input_file","file_id":"file_1","prompt_cache_breakpoint":{"mode":"explicit"}},
				{"type":"refusal","refusal":"I cannot help with that","prompt_cache_breakpoint":{"mode":"explicit"}}
			]
		}]
	}`)
	out := NormalizeOpenAICompatRequestBody(in)
	parts := gjson.GetBytes(out, "messages.0.content").Array()
	if len(parts) < 3 {
		t.Fatalf("parts missing: %s", string(out))
	}
	if parts[0].Get("type").String() != "image_url" ||
		parts[0].Get("prompt_cache_breakpoint.mode").String() != "explicit" {
		t.Fatalf("image_url cache breakpoint lost: %s", parts[0].Raw)
	}
	if parts[1].Get("type").String() != "file" ||
		parts[1].Get("prompt_cache_breakpoint.mode").String() != "explicit" {
		t.Fatalf("file cache breakpoint lost: %s", parts[1].Raw)
	}
	if parts[2].Get("type").String() != "refusal" ||
		parts[2].Get("refusal").String() != "I cannot help with that" ||
		parts[2].Get("prompt_cache_breakpoint.mode").String() != "explicit" {
		t.Fatalf("refusal must stay typed with cache breakpoint: %s", parts[2].Raw)
	}
}

func TestNormalizeResponsesMergesToolCallsOntoModelRole(t *testing.T) {
	// Gemini-style role=model in Responses input must merge subsequent function_call
	// onto the same assistant turn (not invent a second empty assistant).
	in := []byte(`{
		"model":"gemini-flash-latest",
		"input":[
			{"type":"message","role":"user","content":[{"type":"input_text","text":"weather?"}]},
			{"type":"message","role":"model","content":[{"type":"output_text","text":"calling tool"}]},
			{"type":"function_call","call_id":"c1","name":"get_weather","arguments":"{}"},
			{"type":"function_call_output","call_id":"c1","output":"18C"}
		]
	}`)
	out := NormalizeOpenAICompatRequestBody(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) < 3 {
		t.Fatalf("msgs=%d %s", len(msgs), string(out))
	}
	if msgs[1].Get("role").String() != "assistant" {
		t.Fatalf("model→assistant: %s", msgs[1].Raw)
	}
	if !msgs[1].Get("tool_calls").Exists() {
		t.Fatalf("function_call must merge onto model turn: %s", string(out))
	}
	if msgs[1].Get("tool_calls.0.function.name").String() != "get_weather" {
		t.Fatalf("tool name lost: %s", msgs[1].Raw)
	}
	// Must not create a third assistant solely for the tool_call.
	asstCount := 0
	for _, m := range msgs {
		if m.Get("role").String() == "assistant" {
			asstCount++
		}
	}
	if asstCount != 1 {
		t.Fatalf("want 1 assistant after merge, got %d: %s", asstCount, string(out))
	}
	if msgs[2].Get("role").String() != "tool" || msgs[2].Get("content").String() != "18C" {
		t.Fatalf("tool output: %s", string(out))
	}
}

func TestNormalizeOpenAICompatMapsModelAndFunctionRoles(t *testing.T) {
	// Gemini native role=model and OpenAI legacy role=function must become Chat roles
	// before GPT/Gemini/DeepSeek/Mistral openai_compat upstream (400 on unknown roles).
	in := []byte(`{
		"model":"gemini-flash-latest",
		"messages":[
			{"role":"user","content":"hi"},
			{"role":"model","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"f","arguments":"{}"}}]},
			{"role":"function","name":"f","tool_call_id":"c1","content":"ok"}
		]
	}`)
	out := NormalizeOpenAICompatRequestBody(in)
	if gjson.GetBytes(out, "messages.1.role").String() != "assistant" {
		t.Fatalf("model→assistant: %s", string(out))
	}
	if gjson.GetBytes(out, "messages.2.role").String() != "tool" {
		t.Fatalf("function→tool: %s", string(out))
	}
	if gjson.GetBytes(out, "messages.1.tool_calls.0.id").String() != "c1" {
		t.Fatalf("tool_calls lost: %s", string(out))
	}
}

func TestOptimizePayloadFreezesGeminiModelToolTurns(t *testing.T) {
	// Fast must not mutate role=model tool turns (Gemini assistant alias) - same freeze
	// class as role=assistant + tool_calls.
	s := NewServer(Options{CompressionMode: "aggressive", HistoryKeepTurns: 2})
	in := []byte(`{
		"model":"gemini-2.5-flash",
		"messages":[
			{"role":"user","content":"weather in FILLER city NAME please now"},
			{"role":"model","content":"I will call the tool FILLER noise keep stable","tool_calls":[{"id":"c1","type":"function","function":{"name":"get_weather","arguments":"{}"}}]},
			{"role":"tool","tool_call_id":"c1","content":"18C"},
			{"role":"user","content":"thanks"}
		]
	}`)
	out, _, _ := s.optimizePayload(in)
	asst := gjson.GetBytes(out, "messages.1")
	if asst.Get("content").String() != "I will call the tool FILLER noise keep stable" {
		t.Fatalf("model tool turn content mutated: %s", asst.Raw)
	}
	if !asst.Get("tool_calls").Exists() {
		t.Fatalf("tool_calls lost: %s", asst.Raw)
	}
}

func TestNormalizeOpenAICompatRequestBodyPublic(t *testing.T) {
	// Public entry used by live proxy + CLI compress - must normalize Responses input.
	in := []byte(`{
		"model":"deepseek-chat",
		"input":[
			{"type":"message","role":"user","content":[{"type":"input_text","text":"hi"}]},
			{"type":"function_call","call_id":"c1","name":"ping","arguments":"{}"},
			{"type":"function_call_output","call_id":"c1","output":"pong"}
		]
	}`)
	out := NormalizeOpenAICompatRequestBody(in)
	if gjson.GetBytes(out, "input").Exists() {
		t.Fatalf("input remain: %s", string(out))
	}
	if gjson.GetBytes(out, "messages.0.content.0.type").String() != "text" {
		t.Fatalf("input_text not mapped: %s", string(out))
	}
	if gjson.GetBytes(out, "messages.2.name").String() != "ping" {
		t.Fatalf("tool name not enriched: %s", string(out))
	}
}

func TestOptimizePayloadFreezesAnthropicThinkingBlocks(t *testing.T) {
	s := NewServer(Options{CompressionMode: "balanced", HistoryKeepTurns: 2})
	in := []byte(`{
		"model":"claude-opus-4",
		"messages":[
			{"role":"user","content":"old"},
			{"role":"assistant","content":[
				{"type":"thinking","thinking":"KEEP_THINK","signature":"sig_abc"},
				{"type":"text","text":"answer filler that would normally trim"}
			]},
			{"role":"user","content":"next"}
		]
	}`)
	out, _, _ := s.optimizePayload(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if msgs[1].Get("content.0.thinking").String() != "KEEP_THINK" {
		t.Fatalf("thinking mutated: %s", msgs[1].Raw)
	}
	if msgs[1].Get("content.0.signature").String() != "sig_abc" {
		t.Fatalf("thinking signature lost: %s", msgs[1].Raw)
	}
	// Whole assistant with thinking is frozen (official: blocks must remain as they were).
	if msgs[1].Get("content.1.text").String() != "answer filler that would normally trim" {
		t.Fatalf("assistant text under thinking turn must stay verbatim: %s", msgs[1].Raw)
	}
}

func TestOptimizePayloadFreezesOpenAICompatThinkingBlocksField(t *testing.T) {
	// Stream adapter emits thinking_blocks on the OpenAI message; Fast must not history-stub
	// or mutate that assistant turn (official Anthropic: return signed blocks unchanged).
	s := NewServer(Options{CompressionMode: "balanced", HistoryKeepTurns: 1})
	in := []byte(`{
		"model":"claude-sonnet-4",
		"messages":[
			{"role":"user","content":"old question with lots of filler text to trim"},
			{"role":"assistant","content":"answer filler that would normally trim or stub",
			 "reasoning_content":"step plan KEEP",
			 "thinking_blocks":[{"type":"thinking","thinking":"","signature":"sig_TB"}]},
			{"role":"user","content":"next"}
		]
	}`)
	out, _, _ := s.optimizePayload(in)
	asst := gjson.GetBytes(out, "messages.1")
	if asst.Get("thinking_blocks.0.signature").String() != "sig_TB" {
		t.Fatalf("thinking_blocks lost: %s", asst.Raw)
	}
	if asst.Get("reasoning_content").String() != "step plan KEEP" {
		t.Fatalf("reasoning_content lost: %s", asst.Raw)
	}
	if asst.Get("content").String() != "answer filler that would normally trim or stub" {
		t.Fatalf("assistant with thinking_blocks must stay verbatim: %s", asst.Raw)
	}
}

func TestNormalizeCursorBYOKReasoningBeforeAssistant(t *testing.T) {
	in := []byte(`{
		"model":"deepseek-chat",
		"input":[
			{"type":"message","role":"user","content":"hi"},
			{"type":"reasoning","content":"think hard"},
			{"type":"message","role":"assistant","content":"hello"}
		]
	}`)
	out := normalizeCursorBYOKChatBody(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) != 2 {
		t.Fatalf("msgs=%d raw=%s", len(msgs), string(out))
	}
	if msgs[1].Get("role").String() != "assistant" || msgs[1].Get("reasoning_content").String() != "think hard" {
		t.Fatalf("pending reasoning not applied: %s", msgs[1].Raw)
	}
	if msgs[0].Get("role").String() == "user" && strings.Contains(msgs[0].Get("content").String(), "think hard") {
		t.Fatalf("reasoning leaked into user: %s", msgs[0].Raw)
	}
}

func TestNormalizeLegacyFunctionCallToToolCallsAndEnrichToolCallID(t *testing.T) {
	// Official OpenAI deprecated shape still appears in GPT/Cursor histories:
	// assistant.function_call + role=function{name} without tool_call_id.
	// Must become tool_calls + role=tool + tool_call_id for Gemini/DeepSeek/Mistral.
	in := []byte(`{
		"model":"gpt-4o",
		"messages":[
			{"role":"user","content":"weather in Paris?"},
			{"role":"assistant","content":null,"function_call":{"name":"get_weather","arguments":"{\"city\":\"Paris\"}"}},
			{"role":"function","name":"get_weather","content":"18C"},
			{"role":"user","content":"thanks"}
		]
	}`)
	out := NormalizeOpenAICompatRequestBody(in)
	asst := gjson.GetBytes(out, "messages.1")
	if asst.Get("function_call").Exists() {
		t.Fatalf("function_call must be removed: %s", asst.Raw)
	}
	if asst.Get("tool_calls.0.type").String() != "function" {
		t.Fatalf("want tool_calls: %s", asst.Raw)
	}
	if asst.Get("tool_calls.0.function.name").String() != "get_weather" {
		t.Fatalf("name lost: %s", asst.Raw)
	}
	id := asst.Get("tool_calls.0.id").String()
	if id == "" {
		t.Fatalf("synthetic tool_call id required: %s", asst.Raw)
	}
	tool := gjson.GetBytes(out, "messages.2")
	if tool.Get("role").String() != "tool" {
		t.Fatalf("function→tool: %s", tool.Raw)
	}
	if tool.Get("tool_call_id").String() != id {
		t.Fatalf("tool_call_id enrich failed want=%s got=%s raw=%s", id, tool.Get("tool_call_id").String(), tool.Raw)
	}
	if tool.Get("name").String() != "get_weather" {
		t.Fatalf("name must stay for Mistral: %s", tool.Raw)
	}
}

func TestNormalizeLegacyFunctionsArrayAndFunctionCallChoice(t *testing.T) {
	// Deprecated top-level functions[] + function_call must become tools[] + tool_choice
	// so GPT/Gemini/DeepSeek/Mistral openai_compat actually receive callable tools.
	in := []byte(`{
		"model":"gpt-4o",
		"functions":[{"name":"get_weather","description":"Weather","parameters":{"type":"object","properties":{"city":{"type":"string"}}}}],
		"function_call":{"name":"get_weather"},
		"messages":[{"role":"user","content":"Paris weather?"}]
	}`)
	out := NormalizeOpenAICompatRequestBody(in)
	if gjson.GetBytes(out, "functions").Exists() {
		t.Fatalf("functions must be removed after migrate: %s", string(out))
	}
	if gjson.GetBytes(out, "function_call").Exists() {
		t.Fatalf("function_call must be removed after migrate: %s", string(out))
	}
	if gjson.GetBytes(out, "tools.0.type").String() != "function" {
		t.Fatalf("tools migrate failed: %s", string(out))
	}
	if gjson.GetBytes(out, "tools.0.function.name").String() != "get_weather" {
		t.Fatalf("tool name lost: %s", string(out))
	}
	if gjson.GetBytes(out, "tool_choice.type").String() != "function" {
		t.Fatalf("tool_choice migrate failed: %s", string(out))
	}
	if gjson.GetBytes(out, "tool_choice.function.name").String() != "get_weather" {
		t.Fatalf("tool_choice name lost: %s", string(out))
	}
}

