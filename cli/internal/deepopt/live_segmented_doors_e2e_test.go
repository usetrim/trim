package deepopt

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

// Live-door E2E: OpenAI-compat (Cursor/GPT/Gemini) + Anthropic (Claude Code) share
// Segmented Allowlist Deep. Tools/system freeze; mixed prose+path compresses prose only.
func TestLiveSegmentedAllowlistDeepMultiDoorE2E(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })

	dialogue := "John: Ship Friday?\nSarah: Not without tests.\nWhat is the smallest safe plan?"
	pathWin := `C:\Users\berek\proj\app.ts`
	userBlob := dialogue + "\n\n" + pathWin
	userJSON, err := json.Marshal(userBlob)
	if err != nil {
		t.Fatal(err)
	}

	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		if strings.Contains(req.Text, "tools") || strings.Contains(req.Text, `C:\Users`) ||
			strings.Contains(req.Text, "app.ts") || strings.Contains(req.Text, "://") {
			t.Fatalf("protected span entered Deep: %q", req.Text)
		}
		if !strings.Contains(req.Text, "John") && !strings.Contains(req.Question, "plan") {
			t.Fatalf("prose missing: text=%q q=%q", req.Text, req.Question)
		}
		return &Response{CompressedPrompt: "SHORT_PROSE", OriginTokens: 90, CompressedTokens: 22}, nil
	}

	chrome := ChatDeepChrome{CompactStub: "stub", Unavailable: "u"}
	rt := Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long",
		V2ForceTokens: []string{"\n", "/", ".", "?", "!"}}

	t.Run("openai_compat_cursor_shape", func(t *testing.T) {
		in := []byte(`{
			"model":"gpt-4o",
			"tools":[{"type":"function","function":{"name":"Read","parameters":{"type":"object"}}}],
			"messages":[
				{"role":"system","content":"You are Cursor Agent. Keep tool schemas intact."},
				{"role":"user","content":` + string(userJSON) + `}
			]
		}`)
		if EstimateLiveDeepInputTokens(in, "/v1/chat/completions") == 0 {
			t.Fatal("gate must count prose islands")
		}
		out, origin, compressed, err := CompressOpenAIChatJSON(in, EngineV2, 0.7, "", UXChrome{}, 30, chrome, rt)
		if err != nil {
			t.Fatal(err)
		}
		if origin <= compressed {
			t.Fatalf("stage must shrink %d->%d", origin, compressed)
		}
		if gjson.GetBytes(out, "tools.0.function.name").String() != "Read" {
			t.Fatalf("tools frozen: %s", string(out))
		}
		if !strings.Contains(gjson.GetBytes(out, "messages.0.content").String(), "Cursor Agent") {
			t.Fatalf("system frozen: %s", string(out))
		}
		u := gjson.GetBytes(out, "messages.1.content").String()
		if !strings.Contains(u, "SHORT_PROSE") || !strings.Contains(u, pathWin) {
			t.Fatalf("prose rewritten + path kept: %s", u)
		}
		if DeepExpanded(in, out, origin, compressed) {
			t.Fatal("must not expand vs Fast input")
		}
	})

	t.Run("anthropic_claude_code_shape", func(t *testing.T) {
		in := []byte(`{
			"model":"claude-opus-4",
			"system":"You are Claude Code. Never smash tools.",
			"tools":[{"name":"Read","description":"Read a file","input_schema":{"type":"object"}}],
			"messages":[
				{"role":"user","content":[
					{"type":"text","text":` + string(userJSON) + `}
				]}
			]
		}`)
		if EstimateLiveDeepInputTokens(in, "/v1/messages") == 0 {
			t.Fatal("gate must count prose islands on Anthropic")
		}
		out, origin, compressed, err := CompressAnthropicMessagesJSON(in, EngineV2, 0.7, "", UXChrome{}, 30, chrome, rt)
		if err != nil {
			t.Fatal(err)
		}
		if origin <= compressed {
			t.Fatalf("stage must shrink %d->%d", origin, compressed)
		}
		if gjson.GetBytes(out, "tools.0.name").String() != "Read" {
			t.Fatalf("anthropic tools frozen: %s", string(out))
		}
		if gjson.GetBytes(out, "system").String() != "You are Claude Code. Never smash tools." {
			t.Fatalf("anthropic system frozen: %s", string(out))
		}
		u := gjson.GetBytes(out, "messages.0.content.0.text").String()
		if !strings.Contains(u, "SHORT_PROSE") || !strings.Contains(u, pathWin) {
			t.Fatalf("anthropic prose+path: %s", u)
		}
	})
}
