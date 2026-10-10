package deepopt

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

func TestCompressOpenAIChatJSONKeepsSystem(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		if req.Engine != "v2" {
			t.Fatalf("engine=%q", req.Engine)
		}
		if req.TargetToken != 100 {
			t.Fatalf("target=%d", req.TargetToken)
		}
		// Last-user-only: must not fold assistant history into Deep text.
		if strings.Contains(req.Text, "assistant:") || strings.Contains(req.Text, "please refactor") {
			t.Fatalf("text should be last user only, got %q", req.Text)
		}
		if !strings.Contains(req.Text, "more") && req.Question != "more" {
			t.Fatalf("last user missing: text=%q q=%q", req.Text, req.Question)
		}
		return &Response{
			CompressedPrompt: "compressed-user",
			OriginTokens:     40,
			CompressedTokens: 10,
			SavingRate:       "75.0%",
			Engine:           "v2",
		}, nil
	}

	in := []byte(`{"model":"gemini-3.6-flash","messages":[{"role":"system","content":"sys"},{"role":"user","content":"please refactor"},{"role":"assistant","content":"ok"},{"role":"user","content":"more"}]}`)
	out, origin, compressed, err := CompressOpenAIChatJSON(in, EngineV2, 100, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "(prior turns compacted)",
		Unavailable: "unavailable",
	}, Runtime{
		DeviceMap: "cpu",
		V2Model:   "test-v2-model",
		V1Model:   "test-v1-model",
		LongModel: "test-long-model",
	})
	if err != nil {
		t.Fatal(err)
	}
	if origin != 40 || compressed != 10 {
		t.Fatalf("tokens %d->%d", origin, compressed)
	}
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) != 4 {
		t.Fatalf("msgs=%d raw=%s", len(msgs), string(out))
	}
	if msgs[0].Get("role").String() != "system" || msgs[0].Get("content").String() != "sys" {
		t.Fatalf("system lost: %s", msgs[0].Raw)
	}
	if msgs[1].Get("role").String() != "user" || msgs[1].Get("content").String() != "please refactor" {
		t.Fatalf("earlier user mutated: %s", msgs[1].Raw)
	}
	if msgs[2].Get("role").String() != "assistant" || msgs[2].Get("content").String() != "ok" {
		t.Fatalf("assistant lost: %s", msgs[2].Raw)
	}
	if msgs[3].Get("role").String() != "user" || msgs[3].Get("content").String() != "compressed-user" {
		t.Fatalf("user=%s", msgs[3].Raw)
	}
	if gjson.GetBytes(out, "model").String() != "gemini-3.6-flash" {
		t.Fatalf("model rewritten")
	}
}

func TestCompressOpenAIAppendedQuestionUsesComparableTokenUnits(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	// Engine reports low origin tokens; appended question must NOT be estimated with a
	// different unit that falsely trips compressed >= origin (live 0% Saved bug).
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		if req.Question == "" {
			t.Fatal("expected question split")
		}
		return &Response{
			CompressedPrompt: "short ctx",
			OriginTokens:     40,
			CompressedTokens: 8,
			Engine:           "v2",
		}, nil
	}
	filler := strings.Repeat("noise line about context ", 40)
	in := []byte(`{"model":"gpt-4.1","messages":[{"role":"user","content":"` + filler + `\nwhat is 2+2?"}]}`)
	out, origin, compressed, err := CompressOpenAIChatJSON(in, EngineV2, 300, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub",
		Unavailable: "u",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	if compressed >= origin {
		t.Fatalf("must not fail-closed after question append: %d->%d", origin, compressed)
	}
	got := gjson.GetBytes(out, "messages.0.content").String()
	if !strings.Contains(got, "short ctx") || !strings.Contains(got, "what is 2+2?") {
		t.Fatalf("expected compressed+question, got %q", got)
	}
}

func TestCompressPreservesMultimodalPartOrder(t *testing.T) {
	// Official GPT/Gemini vision: content part order matters. Deep must replace the
	// first compressible text in-place, not append after images (re-order garble class).
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		if strings.Contains(req.Text, "IMG_URL") {
			t.Fatalf("must not feed image into Deep: %q", req.Text)
		}
		return &Response{CompressedPrompt: "SHORT q", OriginTokens: 40, CompressedTokens: 5}, nil
	}
	in := []byte(`{
		"model":"gpt-4o",
		"messages":[{"role":"user","content":[
			{"type":"text","text":"FILLER A\nFILLER B\nwhat is in this image?"},
			{"type":"image_url","image_url":{"url":"IMG_URL"}},
			{"type":"text","text":"extra filler after image"}
		]}]
	}`)
	out, _, _, err := CompressOpenAIChatJSON(in, EngineV2, 50, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub", Unavailable: "u",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	parts := gjson.GetBytes(out, "messages.0.content").Array()
	if len(parts) != 2 {
		t.Fatalf("want compressed text + image (merged texts), got %d: %s", len(parts), string(out))
	}
	if parts[0].Get("type").String() != "text" || !strings.Contains(parts[0].Get("text").String(), "SHORT q") {
		t.Fatalf("compressed text must stay before image: %s", string(out))
	}
	if parts[1].Get("type").String() != "image_url" || !strings.Contains(parts[1].Raw, "IMG_URL") {
		t.Fatalf("image must remain after text: %s", string(out))
	}
	// Critical: must NOT be [image, text] (old append-at-end reorder bug).
	if parts[0].Get("type").String() == "image_url" {
		t.Fatalf("image must not precede compressed text: %s", string(out))
	}
}

func TestCompressOpenAIPreservesMultimodalImageAndToolsTopLevel(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		if strings.Contains(req.Text, "image_url") || strings.Contains(req.Text, "data:image") {
			t.Fatalf("must not feed image bytes into Deep: %q", req.Text)
		}
		return &Response{CompressedPrompt: "ask short", OriginTokens: 30, CompressedTokens: 6}, nil
	}
	// GPT / Gemini multimodal + declared tools schema must survive Deep rewrite.
	in := []byte(`{"model":"gpt-5","tools":[{"type":"function","function":{"name":"lookup","parameters":{"type":"object"}}}],"messages":[{"role":"system","content":"sys"},{"role":"user","content":[{"type":"text","text":"FILLER noise A\nFILLER noise B\ndescribe this"},{"type":"image_url","image_url":{"url":"data:image/png;base64,aaa"}},{"type":"input_audio","input_audio":{"data":"AAA","format":"wav"}}]}]}`)
	out, _, _, err := CompressOpenAIChatJSON(in, EngineV2, 50, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub",
		Unavailable: "unavail",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out, "tools.0.function.name").String() != "lookup" {
		t.Fatalf("top-level tools lost: %s", string(out))
	}
	parts := gjson.GetBytes(out, "messages.1.content").Array()
	var sawImg, sawAudio, sawText bool
	for _, p := range parts {
		if p.Get("type").String() == "image_url" {
			sawImg = true
			if p.Get("image_url.url").String() != "data:image/png;base64,aaa" {
				t.Fatalf("image mutated: %s", p.Raw)
			}
		}
		if p.Get("type").String() == "input_audio" {
			sawAudio = true
		}
		if p.Get("type").String() == "text" {
			sawText = true
			if !strings.Contains(p.Get("text").String(), "ask short") {
				t.Fatalf("text=%s", p.Raw)
			}
		}
	}
	if !sawImg || !sawAudio || !sawText {
		t.Fatalf("parts=%s", gjson.GetBytes(out, "messages.1.content").Raw)
	}
}

func TestCompressMultiProviderModelsLastUserOnly(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		return &Response{CompressedPrompt: "c", OriginTokens: 20, CompressedTokens: 4}, nil
	}
	models := []string{"gpt-5", "gemini-flash-latest", "deepseek-chat", "mistral-large-latest", "claude-opus-4"}
	for _, model := range models {
		in := []byte(`{"model":"` + model + `","messages":[{"role":"system","content":"KEEP_SYSTEM"},{"role":"user","content":"old"},{"role":"assistant","content":"ok"},{"role":"user","content":"line1\nline2\nfinal q?"}]}`)
		var (
			out []byte
			err error
		)
		chrome := ChatDeepChrome{CompactStub: "stub", Unavailable: "u"}
		rt := Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"}
		if strings.HasPrefix(model, "claude") {
			out, _, _, err = CompressAnthropicMessagesJSON(in, EngineV2, 50, "", UXChrome{}, 30, chrome, rt)
		} else {
			out, _, _, err = CompressOpenAIChatJSON(in, EngineV2, 50, "", UXChrome{}, 30, chrome, rt)
		}
		if err != nil {
			t.Fatalf("%s: %v", model, err)
		}
		if gjson.GetBytes(out, "messages.0.content").String() != "KEEP_SYSTEM" {
			t.Fatalf("%s system lost", model)
		}
		if gjson.GetBytes(out, "model").String() != model {
			t.Fatalf("%s model rewritten", model)
		}
		last := gjson.GetBytes(out, "messages.3.content").String()
		if last != "c" && !strings.Contains(last, "c") {
			t.Fatalf("%s last user=%q", model, last)
		}
	}
}

func TestLastUserQuestion(t *testing.T) {
	t.Parallel()
	msgs := gjson.Parse(`[{"role":"user","content":"first"},{"role":"assistant","content":"a"},{"role":"user","content":"second"}]`).Array()
	got := lastUserQuestion(msgs)
	if got != "second" {
		t.Fatalf("got %q", got)
	}
}

func TestSplitDeepContextQuestion(t *testing.T) {
	t.Parallel()
	ctx, q := splitDeepContextQuestion("line1 junk\nline2 junk\nwhat is 2+2?")
	if ctx != "line1 junk\nline2 junk" || q != "what is 2+2?" {
		t.Fatalf("ctx=%q q=%q", ctx, q)
	}
	ctx, q = splitDeepContextQuestion("only-one-line?")
	if ctx != "only-one-line?" || q != "only-one-line?" {
		t.Fatalf("single ctx=%q q=%q", ctx, q)
	}
}

func TestCompressAnthropicMessagesJSONPreservesSystemAndReminder(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		if !strings.Contains(req.Text, "FILLER NOISE") {
			t.Fatalf("should compress filler context, got %q", req.Text)
		}
		if strings.Contains(req.Text, "what is 2+2") {
			t.Fatalf("question must be separated from Deep context text: %q", req.Text)
		}
		if req.Question != "what is 2+2? Reply short." {
			t.Fatalf("question=%q", req.Question)
		}
		if strings.Contains(req.Text, "Environment") || strings.Contains(req.Text, "system-reminder") {
			t.Fatalf("must not feed system/reminder into Deep: %q", req.Text)
		}
		return &Response{
			CompressedPrompt: "FILLER",
			OriginTokens:     40,
			CompressedTokens: 4,
			Engine:           "v2",
		}, nil
	}

	in := []byte(`{"model":"claude-opus-5-5","messages":[{"role":"user","content":[{"type":"text","text":"<system-reminder>\nkeep me\n</system-reminder>\n"},{"type":"text","text":"FILLER NOISE line A\nFILLER NOISE line B\nwhat is 2+2? Reply short."}]},{"role":"system","content":"# Environment\nhuge agent chrome"}]}`)
	out, origin, compressed, err := CompressAnthropicMessagesJSON(in, EngineV2, 100, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "(prior turns compacted)",
		Unavailable: "unavailable",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	if origin < compressed {
		t.Fatalf("tokens %d->%d", origin, compressed)
	}
	msgs := gjson.GetBytes(out, "messages").Array()
	if len(msgs) != 2 {
		t.Fatalf("msgs=%d", len(msgs))
	}
	if msgs[1].Get("role").String() != "system" || !strings.Contains(msgs[1].Get("content").String(), "Environment") {
		t.Fatalf("system lost: %s", msgs[1].Raw)
	}
	userParts := msgs[0].Get("content").Array()
	if len(userParts) != 2 {
		t.Fatalf("user parts=%d raw=%s", len(userParts), msgs[0].Raw)
	}
	if !strings.Contains(userParts[0].Get("text").String(), "<system-reminder>") {
		t.Fatalf("reminder lost: %s", userParts[0].Raw)
	}
	got := userParts[1].Get("text").String()
	if !strings.Contains(got, "FILLER") || !strings.Contains(got, "what is 2+2? Reply short.") {
		t.Fatalf("compressed+question=%s", got)
	}
}

func TestCompressPreservesStringEmbeddedSystemReminder(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		if strings.Contains(req.Text, "system-reminder") {
			t.Fatalf("must not feed reminder into Deep: %q", req.Text)
		}
		return &Response{CompressedPrompt: "SHORT_Q", OriginTokens: 40, CompressedTokens: 5}, nil
	}
	// Claude Code sometimes uses a single string content with embedded reminder.
	in := []byte(`{"model":"claude-opus-4","messages":[{"role":"user","content":"<system-reminder>\nkeep me\n</system-reminder>\nFILLER A\nFILLER B\nwhat is 2+2?"}]}`)
	out, _, _, err := CompressAnthropicMessagesJSON(in, EngineV2, 50, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub",
		Unavailable: "unavail",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	got := gjson.GetBytes(out, "messages.0.content").String()
	if !strings.Contains(got, "<system-reminder>") || !strings.Contains(got, "keep me") {
		t.Fatalf("reminder lost from string content: %q", got)
	}
	if !strings.Contains(got, "SHORT_Q") {
		t.Fatalf("compressed text missing: %q", got)
	}
}

func TestCompressPreservesGeminiThoughtSignatureParts(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		if strings.Contains(req.Text, "SIGNED") {
			t.Fatalf("must not feed signed part into Deep: %q", req.Text)
		}
		return &Response{CompressedPrompt: "ask", OriginTokens: 30, CompressedTokens: 4}, nil
	}
	in := []byte(`{"model":"gemini-2.5-flash","messages":[{"role":"user","content":[{"type":"text","text":"SIGNED prefix","thought_signature":"SIG123"},{"type":"text","text":"FILLER noise\nfinal q?"}]}]}`)
	out, _, _, err := CompressOpenAIChatJSON(in, EngineV2, 50, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub",
		Unavailable: "unavail",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out, "messages.0.content.0.thought_signature").String() != "SIG123" {
		t.Fatalf("thought_signature lost: %s", string(out))
	}
	if gjson.GetBytes(out, "messages.0.content.0.text").String() != "SIGNED prefix" {
		t.Fatalf("signed text mutated: %s", string(out))
	}
}

func TestCompressSkipsWhenOnlyFrozenSignedParts(t *testing.T) {
	// Official Gemini: thought_signature parts must stay verbatim. If the last user
	// turn is ONLY signed text, Deep must not run (old openaiCompatFallback fed it
	// into LLMLingua and duplicated chrome - same smash class as Claude garble).
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		t.Fatalf("Deep must not run on frozen-only user turn: %q", req.Text)
		return nil, nil
	}
	in := []byte(`{"model":"gemini-2.5-flash","messages":[{"role":"user","content":[{"type":"text","text":"SIGNED only chrome","thought_signature":"SIG_ONLY"}]}]}`)
	if n := EstimateLiveDeepInputTokens(in, "/v1/chat/completions"); n != 0 {
		t.Fatalf("live gate must ignore signed-only text, got %d", n)
	}
	out, origin, compressed, err := CompressOpenAIChatJSON(in, EngineV2, 50, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub",
		Unavailable: "unavail",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(in) {
		t.Fatalf("body mutated: %s", string(out))
	}
	if origin != compressed {
		t.Fatalf("passthrough tokens %d!=%d", origin, compressed)
	}
}

func TestCompressPreservesAnthropicCitationsParts(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		if strings.Contains(req.Text, "CITED_CLAIM") {
			t.Fatalf("must not feed cited text into Deep: %q", req.Text)
		}
		if !strings.Contains(req.Text, "FILLER") {
			t.Fatalf("should compress non-cited filler: %q", req.Text)
		}
		return &Response{CompressedPrompt: "SHORT", OriginTokens: 40, CompressedTokens: 5}, nil
	}
	in := []byte(`{"model":"claude-opus-4","messages":[{"role":"user","content":[{"type":"text","text":"CITED_CLAIM keep","citations":[{"type":"char_location","cited_text":"src","document_index":0,"start_char_index":0,"end_char_index":3}]},{"type":"text","text":"FILLER A\nFILLER B\nwhat is 2+2?"}]}]}`)
	out, _, _, err := CompressAnthropicMessagesJSON(in, EngineV2, 50, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub", Unavailable: "u",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	parts := gjson.GetBytes(out, "messages.0.content").Array()
	if len(parts) < 2 {
		t.Fatalf("parts=%d raw=%s", len(parts), string(out))
	}
	if parts[0].Get("text").String() != "CITED_CLAIM keep" {
		t.Fatalf("cited text mutated: %s", parts[0].Raw)
	}
	if parts[0].Get("citations.0.type").String() != "char_location" {
		t.Fatalf("citations lost: %s", parts[0].Raw)
	}
}

func TestCompressPreservesOpenAIPromptCacheBreakpoint(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		if strings.Contains(req.Text, "CACHED_PREFIX") {
			t.Fatalf("must not feed prompt_cache_breakpoint text into Deep: %q", req.Text)
		}
		if !strings.Contains(req.Text, "FILLER") {
			t.Fatalf("should compress non-cached filler: %q", req.Text)
		}
		return &Response{CompressedPrompt: "SHORT", OriginTokens: 40, CompressedTokens: 5}, nil
	}
	// OpenAI official: prompt_cache_breakpoint on content parts (GPT-5.6+ explicit caching).
	in := []byte(`{"model":"gpt-5.6","messages":[{"role":"developer","content":[{"type":"text","text":"CACHED_PREFIX keep","prompt_cache_breakpoint":{"mode":"explicit"}}]},{"role":"user","content":[{"type":"text","text":"CACHED_PREFIX keep","prompt_cache_breakpoint":{"mode":"explicit"}},{"type":"text","text":"FILLER A\nFILLER B\nwhat is 2+2?"}]}]}`)
	out, _, _, err := CompressOpenAIChatJSON(in, EngineV2, 50, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub", Unavailable: "u",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out, "messages.0.content.0.prompt_cache_breakpoint.mode").String() != "explicit" {
		t.Fatalf("developer breakpoint lost: %s", string(out))
	}
	parts := gjson.GetBytes(out, "messages.1.content").Array()
	if len(parts) < 2 {
		t.Fatalf("parts=%d raw=%s", len(parts), string(out))
	}
	if parts[0].Get("text").String() != "CACHED_PREFIX keep" {
		t.Fatalf("cached text mutated: %s", parts[0].Raw)
	}
	if parts[0].Get("prompt_cache_breakpoint.mode").String() != "explicit" {
		t.Fatalf("user breakpoint lost: %s", parts[0].Raw)
	}
}

func TestLastUserUnsafeDirTreeWithoutExt(t *testing.T) {
	if !LastUserUnsafeForDeepLLMLingua("please inspect pkg/proxy and explain the gate") {
		t.Fatal("coding dir tree without file ext must skip Deep")
	}
	if LastUserUnsafeForDeepLLMLingua("FILLER A\nFILLER B\nwhat is 2+2?") {
		t.Fatal("plain prose must remain Deep-eligible")
	}
}

func TestLastUserUnsafeUNCAndWindowsRoots(t *testing.T) {
	// UNC / extended Windows paths must fail-closed (LLMLingua mangles backslash runs).
	if !LastUserUnsafeForDeepLLMLingua(`please open \\fileserver\share\src\main.go and summarize`) {
		t.Fatal("UNC path must be unsafe for Deep")
	}
	if !LastUserUnsafeForDeepLLMLingua(`look at \Users\berek\project\app.ts`) {
		t.Fatal(`\Users\ root must be unsafe for Deep`)
	}
	if LastUserUnsafeForDeepLLMLingua("FILLER A\nFILLER B\nwhat is 2+2?") {
		t.Fatal("plain prose must remain Deep-eligible")
	}
}

func TestCompressSkipsPathAndCodeFenceUnsafeLastUser(t *testing.T) {
	// LLMLingua lossiness: paths/fences must not enter Deep (all doors).
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		t.Fatalf("Deep must not run on path/code-heavy turn: %q", req.Text)
		return nil, nil
	}
	cases := []string{
		`{"model":"gpt-4o","messages":[{"role":"user","content":"see C:\\Users\\berek\\proj\\main.go and fix it please now"}]}`,
		`{"model":"claude-opus-4","messages":[{"role":"user","content":"read /Users/berek/proj/app.ts then explain"}]}`,
		"{\"model\":\"gemini-flash\",\"messages\":[{\"role\":\"user\",\"content\":\"```go\\npackage main\\n```\\nwhat does this do?\"}]}",
		`{"model":"deepseek-chat","messages":[{"role":"user","content":"open https://example.com/docs and summarize"}]}`,
		`{"model":"mistral-large","messages":[{"role":"user","content":"please edit src/proxy/server.go and explain"}]}`,
		`{"model":"gpt-4o","messages":[{"role":"user","content":"please inspect pkg/proxy and explain the gate"}]}`,
	}
	for _, raw := range cases {
		in := []byte(raw)
		if n := EstimateLiveDeepInputTokens(in, "/v1/chat/completions"); n != 0 {
			t.Fatalf("gate should skip unsafe last-user, est=%d body=%s", n, raw)
		}
		out, origin, compressed, err := CompressOpenAIChatJSON(in, EngineV2, 50, "", UXChrome{}, 30, ChatDeepChrome{
			CompactStub: "stub", Unavailable: "u",
		}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
		if err != nil {
			t.Fatal(err)
		}
		if string(out) != string(in) || origin != compressed {
			t.Fatalf("unsafe turn must stay Fast-only: %s", string(out))
		}
	}
	if LastUserUnsafeForDeepLLMLingua("FILLER A\nFILLER B\nwhat is 2+2?") {
		t.Fatal("plain prose must remain Deep-eligible")
	}
}

func TestCompressAnthropicPreservesCacheControlParts(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		if strings.Contains(req.Text, "CACHED_PREFIX") {
			t.Fatalf("must not feed cache_control text into Deep: %q", req.Text)
		}
		if !strings.Contains(req.Text, "FILLER") {
			t.Fatalf("should compress non-cached filler: %q", req.Text)
		}
		return &Response{CompressedPrompt: "SHORT", OriginTokens: 40, CompressedTokens: 5}, nil
	}
	// Anthropic prompt-caching: cache_control breakpoint parts must stay byte-identical.
	// Filler AFTER the breakpoint may still Deep (not in the reusable prefix).
	in := []byte(`{"model":"claude-opus-4","messages":[{"role":"user","content":[{"type":"text","text":"CACHED_PREFIX keep","cache_control":{"type":"ephemeral"}},{"type":"text","text":"FILLER A\nFILLER B\nwhat is 2+2?"}]}]}`)
	out, _, _, err := CompressAnthropicMessagesJSON(in, EngineV2, 50, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub",
		Unavailable: "unavail",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	parts := gjson.GetBytes(out, "messages.0.content").Array()
	if len(parts) < 2 {
		t.Fatalf("parts=%d raw=%s", len(parts), string(out))
	}
	if parts[0].Get("text").String() != "CACHED_PREFIX keep" {
		t.Fatalf("cached text mutated: %s", parts[0].Raw)
	}
	if parts[0].Get("cache_control.type").String() != "ephemeral" {
		t.Fatalf("cache_control lost: %s", parts[0].Raw)
	}
	if !strings.Contains(parts[len(parts)-1].Get("text").String(), "SHORT") {
		t.Fatalf("expected compressed last text: %s", parts[len(parts)-1].Raw)
	}
}

func TestCompressSkipsTextBeforeCacheBreakpoint(t *testing.T) {
	// Official Anthropic/OpenAI: mutating bytes before a cache breakpoint invalidates
	// the reusable prefix. Filler BEFORE cache_control / prompt_cache_breakpoint must
	// stay verbatim (Claude + GPT + Gemini openai_compat).
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	called := false
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		called = true
		if strings.Contains(req.Text, "BEFORE_BP") || strings.Contains(req.Text, "CACHED_PREFIX") {
			t.Fatalf("must not feed pre-breakpoint / cached text into Deep: %q", req.Text)
		}
		return &Response{CompressedPrompt: "SHORT", OriginTokens: 40, CompressedTokens: 5}, nil
	}

	cases := []struct {
		name string
		in   string
		fn   func([]byte, Engine, int, string, UXChrome, int, ChatDeepChrome, Runtime) ([]byte, int, int, error)
	}{
		{
			name: "anthropic_cache_control_before",
			fn:   CompressAnthropicMessagesJSON,
			in: `{"model":"claude-opus-4","messages":[{"role":"user","content":[
				{"type":"text","text":"BEFORE_BP filler must stay"},
				{"type":"text","text":"CACHED_PREFIX keep","cache_control":{"type":"ephemeral"}}
			]}]}`,
		},
		{
			name: "openai_prompt_cache_breakpoint_before",
			fn:   CompressOpenAIChatJSON,
			in: `{"model":"gpt-5.6","messages":[{"role":"user","content":[
				{"type":"text","text":"BEFORE_BP filler must stay"},
				{"type":"text","text":"CACHED_PREFIX keep","prompt_cache_breakpoint":{"mode":"explicit"}}
			]}]}`,
		},
	}
	chrome := ChatDeepChrome{CompactStub: "stub", Unavailable: "u"}
	rt := Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called = false
			in := []byte(tc.in)
			out, _, _, err := tc.fn(in, EngineV2, 50, "", UXChrome{}, 30, chrome, rt)
			if err != nil {
				t.Fatal(err)
			}
			if called {
				t.Fatal("Deep must not run when only pre-breakpoint text exists")
			}
			if !strings.Contains(string(out), "BEFORE_BP filler must stay") {
				t.Fatalf("pre-breakpoint text lost: %s", string(out))
			}
			if !strings.Contains(string(out), "CACHED_PREFIX keep") {
				t.Fatalf("cached prefix lost: %s", string(out))
			}
			if n := EstimateLiveDeepInputTokens(in, "/v1/chat/completions"); n != 0 {
				t.Fatalf("live gate must be 0 for pre-breakpoint-only turn, got %d", n)
			}
		})
	}
}

func TestCompressSkipsMessageLevelCacheControl(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	called := false
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		called = true
		return &Response{CompressedPrompt: "x", OriginTokens: 10, CompressedTokens: 1}, nil
	}
	in := []byte(`{"messages":[{"role":"user","content":"big filler text for deep","cache_control":{"type":"ephemeral"}}]}`)
	out, _, _, err := CompressAnthropicMessagesJSON(in, EngineV2, 50, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub",
		Unavailable: "unavail",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("message-level cache_control must skip Deep")
	}
	if string(out) != string(in) {
		t.Fatal("body mutated")
	}
	if n := EstimateLiveDeepInputTokens(in, "/v1/messages"); n != 0 {
		t.Fatalf("live gate must be 0 for message cache_control, got %d", n)
	}
}

func TestCompressSkipsMessageLevelAnnotationsAndRefusal(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	called := false
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		called = true
		return &Response{CompressedPrompt: "x", OriginTokens: 10, CompressedTokens: 1}, nil
	}
	ann := []byte(`{"messages":[{"role":"user","content":"FILLER A\nFILLER B\nwhat is 2+2?",
		"annotations":[{"type":"url_citation","url_citation":{"url":"https://example.com","start_index":0,"end_index":1}}]}]}`)
	out, _, _, err := CompressOpenAIChatJSON(ann, EngineV2, 50, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub", Unavailable: "unavail",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("message-level annotations must skip Deep")
	}
	if string(out) != string(ann) {
		t.Fatal("annotations body mutated")
	}
	if n := EstimateLiveDeepInputTokens(ann, "/v1/chat/completions"); n != 0 {
		t.Fatalf("live gate must be 0 for annotations, got %d", n)
	}

	called = false
	ref := []byte(`{"messages":[{"role":"user","content":"FILLER A\nFILLER B\nwhat is 2+2?","refusal":"blocked"}]}`)
	out2, _, _, err := CompressOpenAIChatJSON(ref, EngineV2, 50, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub", Unavailable: "unavail",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("message-level refusal must skip Deep")
	}
	if string(out2) != string(ref) {
		t.Fatal("refusal body mutated")
	}

	called = false
	audio := []byte(`{"messages":[{"role":"user","content":"FILLER A\nFILLER B\nwhat is 2+2?","audio":{"id":"audio_u1"}}]}`)
	out3, _, _, err := CompressOpenAIChatJSON(audio, EngineV2, 50, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub", Unavailable: "unavail",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("message-level audio must skip Deep")
	}
	if string(out3) != string(audio) {
		t.Fatal("audio body mutated")
	}
	if n := EstimateLiveDeepInputTokens(audio, "/v1/chat/completions"); n != 0 {
		t.Fatalf("live gate must be 0 for message audio, got %d", n)
	}
}

func TestCompressOpenAIPreservesDeveloperAndCacheControl(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		if strings.Contains(req.Text, "DEV_KEEP") || strings.Contains(req.Text, "CACHED") {
			t.Fatalf("must not feed developer/cache into Deep: %q", req.Text)
		}
		return &Response{CompressedPrompt: "ask short", OriginTokens: 30, CompressedTokens: 6}, nil
	}
	// GPT o1+/Cursor: developer role + optional cache_control on content parts.
	in := []byte(`{"model":"gpt-5","messages":[{"role":"developer","content":"DEV_KEEP"},{"role":"user","content":[{"type":"text","text":"CACHED","cache_control":{"type":"ephemeral"}},{"type":"text","text":"FILLER noise\nfinal q?"}]}]}`)
	out, _, _, err := CompressOpenAIChatJSON(in, EngineV2, 50, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub",
		Unavailable: "unavail",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out, "messages.0.content").String() != "DEV_KEEP" {
		t.Fatalf("developer lost: %s", string(out))
	}
	if gjson.GetBytes(out, "messages.1.content.0.cache_control.type").String() != "ephemeral" {
		t.Fatalf("cache_control lost: %s", string(out))
	}
}

func TestCompressAnthropicMessagesJSONSkipsToolTurns(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	called := false
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		called = true
		return &Response{CompressedPrompt: "x", OriginTokens: 10, CompressedTokens: 1}, nil
	}
	in := []byte(`{"messages":[{"role":"user","content":"hi"},{"role":"assistant","content":[{"type":"tool_use","id":"1","name":"Read","input":{}}]}]}`)
	out, _, _, err := CompressAnthropicMessagesJSON(in, EngineV2, 50, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub",
		Unavailable: "unavail",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("Deep must not run on tool_use turns")
	}
	if string(out) != string(in) {
		t.Fatalf("body mutated")
	}
}

func TestCompressSkipsAnthropicMCPToolResultTurn(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	called := false
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		called = true
		return &Response{CompressedPrompt: "x", OriginTokens: 10, CompressedTokens: 1}, nil
	}
	// Official Anthropic MCP connector: mcp_tool_result must stay verbatim (like tool_result).
	in := []byte(`{"messages":[{"role":"assistant","content":[{"type":"mcp_tool_use","id":"m1","name":"echo","server_name":"s","input":{}}]},{"role":"user","content":[{"type":"mcp_tool_result","tool_use_id":"m1","content":[{"type":"text","text":"ok"}]}]}]}`)
	out, _, _, err := CompressAnthropicMessagesJSON(in, EngineV2, 50, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub",
		Unavailable: "unavail",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("Deep must not run on mcp_tool_result turns")
	}
	if string(out) != string(in) {
		t.Fatal("MCP body mutated")
	}
}

func TestCompressAnthropicToolResultPlusUserTextDeep(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		if strings.Contains(req.Text, "tool_result") || strings.Contains(req.Text, "TOOL_KEEP") {
			t.Fatalf("must not feed tool_result into Deep: %q", req.Text)
		}
		if !strings.Contains(req.Text, "FILLER") {
			t.Fatalf("should compress follow-up filler: %q", req.Text)
		}
		return &Response{CompressedPrompt: "SHORT_Q", OriginTokens: 40, CompressedTokens: 5}, nil
	}
	// Official Anthropic: tool_result first, then user text in the same user turn.
	in := []byte(`{"model":"claude-opus-4","messages":[{"role":"assistant","content":[{"type":"tool_use","id":"1","name":"Read","input":{}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"1","content":"TOOL_KEEP"},{"type":"text","text":"FILLER A\nFILLER B\nwhat is 2+2?"}]}]}`)
	out, origin, compressed, err := CompressAnthropicMessagesJSON(in, EngineV2, 50, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub",
		Unavailable: "unavail",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	if origin <= compressed {
		t.Fatalf("tokens %d->%d", origin, compressed)
	}
	parts := gjson.GetBytes(out, "messages.1.content").Array()
	if len(parts) < 2 {
		t.Fatalf("parts=%s", string(out))
	}
	if parts[0].Get("type").String() != "tool_result" {
		t.Fatalf("tool_result must remain first: %s", string(out))
	}
	if !strings.Contains(parts[0].Raw, "TOOL_KEEP") {
		t.Fatalf("tool_result mutated: %s", parts[0].Raw)
	}
	last := parts[len(parts)-1].Get("text").String()
	if !strings.Contains(last, "SHORT_Q") {
		t.Fatalf("compressed text missing: %s", string(out))
	}
	if n := EstimateLiveDeepInputTokens(in, "/v1/messages"); n == 0 {
		t.Fatal("live gate must count follow-up text after tool_result")
	}
}

func TestCompressOpenAIChatJSONSkipsToolTurns(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	called := false
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		called = true
		return &Response{CompressedPrompt: "x", OriginTokens: 10, CompressedTokens: 1}, nil
	}
	// GPT / Gemini / DeepSeek / Cursor Agent OpenAI-compat tool loop (ends on tool).
	in := []byte(`{"model":"gpt-5","messages":[{"role":"user","content":"weather?"},{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{}"}}]},{"role":"tool","tool_call_id":"call_1","content":"{\"ok\":true}"}]}`)
	out, _, _, err := CompressOpenAIChatJSON(in, EngineV2, 50, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub",
		Unavailable: "unavail",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("Deep must not run on OpenAI tool_calls / role=tool turns")
	}
	if string(out) != string(in) {
		t.Fatalf("body mutated")
	}
}

func TestCompressOpenAIChatJSONAllowsDeepAfterToolHistory(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		if strings.Contains(req.Text, "tool") || strings.Contains(req.Question, "call_") {
			t.Fatalf("must not feed tool chrome into Deep: text=%q q=%q", req.Text, req.Question)
		}
		if req.Question != "now what is 2+2?" && req.Text != "now what is 2+2?" {
			t.Fatalf("expected last user question, text=%q q=%q", req.Text, req.Question)
		}
		return &Response{CompressedPrompt: "compressed-q", OriginTokens: 40, CompressedTokens: 8}, nil
	}
	// After a completed tool loop, a fresh user turn must still be Deep-eligible
	// (GPT / Gemini / DeepSeek / Cursor) - last-user-only never feeds tools to LLMLingua.
	in := []byte(`{"model":"gemini-flash-latest","messages":[{"role":"user","content":"weather?"},{"role":"assistant","tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{}"}}]},{"role":"tool","tool_call_id":"call_1","content":"18C"},{"role":"user","content":"now what is 2+2?"}]}`)
	out, origin, compressed, err := CompressOpenAIChatJSON(in, EngineV2, 50, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub",
		Unavailable: "unavail",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	if origin <= compressed {
		t.Fatalf("tokens %d->%d", origin, compressed)
	}
	msgs := gjson.GetBytes(out, "messages").Array()
	if !msgs[1].Get("tool_calls").Exists() || msgs[2].Get("role").String() != "tool" {
		t.Fatalf("tool history must stay: %s", string(out))
	}
	if msgs[3].Get("content").String() != "compressed-q" {
		t.Fatalf("last user=%s", msgs[3].Raw)
	}
}

func TestEstimateLiveDeepInputTokensSkipsOpenAITools(t *testing.T) {
	t.Parallel()
	in := []byte(`{"messages":[{"role":"user","content":"` + strings.Repeat("filler ", 200) + `"},{"role":"assistant","tool_calls":[{"id":"1","type":"function","function":{"name":"x","arguments":"{}"}}]},{"role":"tool","tool_call_id":"1","content":"ok"}]}`)
	n := EstimateLiveDeepInputTokens(in, "/v1/chat/completions")
	if n != 0 {
		t.Fatalf("mid-tool-loop must skip live Deep gate, got %d", n)
	}
}

func TestEstimateLiveDeepInputTokensAllowsUserAfterTools(t *testing.T) {
	t.Parallel()
	filler := strings.Repeat("long filler context line ", 80)
	in := []byte(`{"messages":[{"role":"assistant","tool_calls":[{"id":"1","type":"function","function":{"name":"x","arguments":"{}"}}]},{"role":"tool","tool_call_id":"1","content":"ok"},{"role":"user","content":"` + filler + `what is 2+2?"}]}`)
	n := EstimateLiveDeepInputTokens(in, "/v1/chat/completions")
	if n < 50 {
		t.Fatalf("fresh user after tools should count for Deep gate, got %d", n)
	}
}

func TestCompressAnthropicMessagesJSONRejectsExpansion(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		return &Response{
			CompressedPrompt: strings.Repeat("EXPAND ", 50) + req.Text,
			OriginTokens:     5,
			CompressedTokens: 500,
			Engine:           "v2",
		}, nil
	}
	in := []byte(`{"messages":[{"role":"user","content":"short ask"}]}`)
	out, origin, compressed, err := CompressAnthropicMessagesJSON(in, EngineV2, 50, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub",
		Unavailable: "unavail",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(in) {
		t.Fatalf("expected original on expansion")
	}
	if origin != compressed {
		t.Fatalf("passthrough tokens %d!=%d", origin, compressed)
	}
}

func TestEstimateLiveDeepInputTokensIgnoresSystemOnMessages(t *testing.T) {
	t.Parallel()
	hugeSys := strings.Repeat("agent chrome ", 500)
	in := []byte(`{"messages":[{"role":"system","content":"` + hugeSys + `"},{"role":"user","content":"hi"}]}`)
	n := EstimateLiveDeepInputTokens(in, "/v1/messages")
	if n > 20 {
		t.Fatalf("system must not dominate live Deep gate, got %d", n)
	}
}

func TestEstimateLiveDeepInputTokensLastUserOnly(t *testing.T) {
	t.Parallel()
	// Early huge user turn must not dominate the live Deep gate; only last user counts.
	early := strings.Repeat("old filler ", 400)
	in := []byte(`{"messages":[{"role":"user","content":"` + early + `"},{"role":"assistant","content":"ok"},{"role":"user","content":"hi again"}]}`)
	n := EstimateLiveDeepInputTokens(in, "/v1/messages")
	if n > 20 {
		t.Fatalf("expected last-user-only estimate, got %d", n)
	}
}

func TestSkipLiveDeepForRequestPath(t *testing.T) {
	if !SkipLiveDeepForRequestPath("/v1/messages/count_tokens") {
		t.Fatal("count_tokens must skip live Deep")
	}
	if SkipLiveDeepForRequestPath("/v1/messages") {
		t.Fatal("messages must allow live Deep")
	}
	if SkipLiveDeepForRequestPath("/v1/chat/completions") {
		t.Fatal("chat completions must allow live Deep")
	}
}

func TestDeepExpanded(t *testing.T) {
	t.Parallel()
	fast := []byte(`{"a":1}`)
	deep := []byte(`{"a":1,"bigger":true}`)
	if !DeepExpanded(fast, deep, 10, 20) {
		t.Fatal("expected expanded")
	}
	if DeepExpanded(deep, fast, 20, 10) {
		t.Fatal("shrink should not flag")
	}
}

func TestCompressOpenAIUnderstandsInputTextParts(t *testing.T) {
	// Defense in depth: if Responses input_text somehow reaches Deep, still compress.
	orig := compressFn
	t.Cleanup(func() { compressFn = orig })
	compressFn = func(req Request, ux UXChrome, timeoutSec int) (*Response, error) {
		return &Response{CompressedPrompt: "SHORT", OriginTokens: 40, CompressedTokens: 5}, nil
	}
	in := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":[{"type":"input_text","text":"FILLER A\nFILLER B\nwhat is 2+2?"},{"type":"image_url","image_url":{"url":"data:image/png;base64,aaa"}}]}]}`)
	out, origin, compressed, err := CompressOpenAIChatJSON(in, EngineV2, 50, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub", Unavailable: "u",
	}, Runtime{DeviceMap: "cpu", V2Model: "test-v2-model"})
	if err != nil {
		t.Fatal(err)
	}
	if origin <= compressed {
		t.Fatalf("expected savings origin=%d compressed=%d", origin, compressed)
	}
	parts := gjson.GetBytes(out, "messages.0.content").Array()
	if len(parts) < 2 {
		t.Fatalf("parts=%s", string(out))
	}
	foundText, foundImg := false, false
	for _, p := range parts {
		if p.Get("type").String() == "text" && strings.Contains(p.Get("text").String(), "SHORT") {
			foundText = true
		}
		if p.Get("type").String() == "image_url" {
			foundImg = true
		}
	}
	if !foundText || !foundImg {
		t.Fatalf("input_text Deep rewrite / image preserve failed: %s", string(out))
	}
}

// TestRootCauseMatrixNoWholeBodySmash encodes the official failure chain for every door:
// system/tools/chrome must never enter LLMLingua; only last-user compressible text may.
// Covers Anthropic / GPT / Gemini / DeepSeek / Mistral OpenAI-compat shapes.
func TestRootCauseMatrixNoWholeBodySmash(t *testing.T) {
	t.Parallel()
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })

	cases := []struct {
		name string
		in   string
		fn   func([]byte, Engine, int, string, UXChrome, int, ChatDeepChrome, Runtime) ([]byte, int, int, error)
	}{
		{
			name: "anthropic_claude_code_chrome",
			fn:   CompressAnthropicMessagesJSON,
			in: `{
				"model":"claude-opus-4",
				"system":"` + strings.Repeat("AGENT_SYSTEM_CHROME ", 200) + `",
				"tools":[{"name":"Bash","input_schema":{"type":"object"}}],
				"messages":[
					{"role":"user","content":[
						{"type":"text","text":"<system-reminder>KEEP_REM</system-reminder>"},
						{"type":"text","text":"FILLER A\nFILLER B\nwhat is 2+2?"}
					]}
				]
			}`,
		},
		{
			name: "openai_gpt_tools_developer",
			fn:   CompressOpenAIChatJSON,
			in: `{
				"model":"gpt-4o",
				"tools":[{"type":"function","function":{"name":"lookup","parameters":{"type":"object"}}}],
				"messages":[
					{"role":"system","content":"` + strings.Repeat("SYS ", 100) + `"},
					{"role":"developer","content":"DEV_KEEP"},
					{"role":"user","content":"FILLER A\nFILLER B\nwhat is 2+2?"}
				]
			}`,
		},
		{
			name: "gemini_thought_signature",
			fn:   CompressOpenAIChatJSON,
			in: `{
				"model":"gemini-2.5-flash",
				"messages":[{"role":"user","content":[
					{"type":"text","text":"SIGNED","thought_signature":"SIG_KEEP"},
					{"type":"text","text":"FILLER A\nFILLER B\nwhat is 2+2?"}
				]}]
			}`,
		},
		{
			name: "deepseek_reasoning_history",
			fn:   CompressOpenAIChatJSON,
			in: `{
				"model":"deepseek-chat",
				"tools":[{"type":"function","function":{"name":"x","parameters":{"type":"object"}}}],
				"messages":[
					{"role":"user","content":"q1"},
					{"role":"assistant","content":"a1","reasoning_content":"REASON_KEEP"},
					{"role":"user","content":"FILLER A\nFILLER B\nwhat is 2+2?"}
				]
			}`,
		},
		{
			name: "anthropic_mid_conversation_system",
			fn:   CompressAnthropicMessagesJSON,
			in: `{
				"model":"claude-opus-4",
				"system":"STABLE_PREFIX",
				"messages":[
					{"role":"user","content":"hi"},
					{"role":"assistant","content":"ok"},
					{"role":"system","content":"MID_SYS_KEEP"},
					{"role":"user","content":"FILLER A\nFILLER B\nwhat is 2+2?"}
				]
			}`,
		},
		{
			name: "mistral_shape_same_path",
			fn:   CompressOpenAIChatJSON,
			in: `{
				"model":"mistral-large-latest",
				"messages":[
					{"role":"system","content":"SYS_KEEP"},
					{"role":"user","content":"FILLER A\nFILLER B\nwhat is 2+2?"}
				]
			}`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
				// Exact root cause guard: never feed system/tools/reminders/signatures/reasoning.
				bad := []string{
					"AGENT_SYSTEM", "SYS ", "SYS_KEEP", "DEV_KEEP", "KEEP_REM",
					"SIGNED", "SIG_KEEP", "REASON_KEEP", "MID_SYS_KEEP", "Bash", "lookup",
					`"tools"`, "input_schema",
				}
				blob := req.Text + "\n" + req.Question
				for _, b := range bad {
					if strings.Contains(blob, b) {
						t.Fatalf("LLMLingua must not see chrome %q in %q", b, blob)
					}
				}
				if !strings.Contains(blob, "2+2") && !strings.Contains(req.Question, "2+2") {
					t.Fatalf("real question lost: text=%q q=%q", req.Text, req.Question)
				}
				return &Response{CompressedPrompt: "SHORT q", OriginTokens: 40, CompressedTokens: 5}, nil
			}

			in := []byte(tc.in)
			// Live gate must ignore huge system chrome (Claude Code 2+2? false trigger).
			est := EstimateLiveDeepInputTokens(in, "/v1/chat/completions")
			if est > 200 {
				t.Fatalf("live gate dominated by chrome: est=%d", est)
			}

			out, origin, compressed, err := tc.fn(in, EngineV2, 50, "", UXChrome{}, 30, ChatDeepChrome{
				CompactStub: "stub", Unavailable: "u",
			}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
			if err != nil {
				t.Fatal(err)
			}
			if origin <= compressed {
				t.Fatalf("expected Deep savings origin=%d compressed=%d", origin, compressed)
			}
			// Must not smash message list into one garbled user message.
			msgs := gjson.GetBytes(out, "messages")
			if !msgs.IsArray() || len(msgs.Array()) < 1 {
				t.Fatalf("messages smashed: %s", string(out))
			}
			wire := string(out)
			if gjson.GetBytes(in, "system").Exists() && !gjson.GetBytes(out, "system").Exists() {
				t.Fatalf("top-level system deleted: %s", wire)
			}
			if gjson.GetBytes(in, "tools").Exists() && !gjson.GetBytes(out, "tools").Exists() {
				t.Fatalf("tools deleted: %s", wire)
			}
			if strings.Contains(string(in), "SIG_KEEP") && !strings.Contains(wire, "SIG_KEEP") {
				t.Fatalf("thought_signature lost: %s", wire)
			}
			if strings.Contains(string(in), "REASON_KEEP") && !strings.Contains(wire, "REASON_KEEP") {
				t.Fatalf("reasoning_content lost: %s", wire)
			}
			if strings.Contains(string(in), "KEEP_REM") && !strings.Contains(wire, "KEEP_REM") {
				t.Fatalf("system-reminder lost: %s", wire)
			}
			if strings.Contains(string(in), "DEV_KEEP") && !strings.Contains(wire, "DEV_KEEP") {
				t.Fatalf("developer lost: %s", wire)
			}
			if strings.Contains(string(in), "MID_SYS_KEEP") && !strings.Contains(wire, "MID_SYS_KEEP") {
				t.Fatalf("mid-conversation system lost: %s", wire)
			}
			// Fail-closed: expansion must not be accepted.
			if DeepExpanded(in, out, origin, compressed) {
				t.Fatalf("DeepExpanded true after successful compress")
			}
			bloated := append(append([]byte{}, out...), bytes.Repeat([]byte("X"), len(in)+100)...)
			if !DeepExpanded(in, bloated, origin, origin+50) {
				t.Fatal("DeepExpanded must catch body expansion")
			}
		})
	}
}
