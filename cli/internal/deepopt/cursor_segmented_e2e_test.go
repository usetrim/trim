package deepopt

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

// Cursor-shaped Agent body: large tools/system chrome + John/Sarah prose + one path.
// Segmented Allowlist Deep must compress prose only; freeze tools/path; meter stage savings.
func TestCursorShapedSegmentedAllowlistDeepE2E(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })

	dialogue := strings.TrimSpace(`
John: I think we should ship Friday without more tests.
Sarah: That timeline feels rushed. We still have flaky coverage on payments.
John: Fair point. What is the smallest safe plan before we merge?
Sarah: Add the two missing integration tests, then decide.
`) + "\n\nPlease summarize the plan carefully."

	userBlob := dialogue + "\n\nC:\\Users\\berek\\proj\\checkout.ts"
	userJSON, err := json.Marshal(userBlob)
	if err != nil {
		t.Fatal(err)
	}

	in := []byte(`{
		"model":"gpt-4o",
		"tools":[{"type":"function","function":{"name":"Read","description":"Read a file","parameters":{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}}}],
		"messages":[
			{"role":"system","content":"You are Cursor Agent. Never invent tool schemas. Follow MCP protocols exactly."},
			{"role":"user","content":` + string(userJSON) + `}
		]
	}`)

	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		if strings.Contains(req.Text, "tools") || strings.Contains(req.Text, "Read") {
			t.Fatalf("tools chrome must not enter Deep: %q", req.Text)
		}
		if strings.Contains(req.Text, `C:\Users`) || strings.Contains(req.Text, "checkout.ts") {
			t.Fatalf("path must not enter Deep: %q", req.Text)
		}
		if !strings.Contains(req.Text, "John") && !strings.Contains(req.Question, "plan") {
			t.Fatalf("dialogue prose missing from Deep: text=%q q=%q", req.Text, req.Question)
		}
		return &Response{CompressedPrompt: "SHORT_CURSOR_PROSE", OriginTokens: 120, CompressedTokens: 30}, nil
	}

	if n := EstimateLiveDeepInputTokens(in, "/v1/chat/completions"); n == 0 {
		t.Fatal("live gate must see Cursor prose islands (not chrome-only skip)")
	}

	out, origin, compressed, err := CompressOpenAIChatJSON(in, EngineV2, 0.7, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub", Unavailable: "u",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	if origin <= compressed {
		t.Fatalf("Deep stage must shrink on prose %d->%d", origin, compressed)
	}
	if gjson.GetBytes(out, "tools.0.function.name").String() != "Read" {
		t.Fatalf("tools must stay verbatim: %s", string(out))
	}
	if gjson.GetBytes(out, "messages.0.content").String() == "" ||
		!strings.Contains(gjson.GetBytes(out, "messages.0.content").String(), "Cursor Agent") {
		t.Fatalf("system must stay: %s", string(out))
	}
	userOut := gjson.GetBytes(out, "messages.1.content").String()
	if !strings.Contains(userOut, "SHORT_CURSOR_PROSE") {
		t.Fatalf("prose must Deep-rewrite: %s", userOut)
	}
	if !strings.Contains(userOut, `C:\Users\berek\proj\checkout.ts`) {
		t.Fatalf("path must remain verbatim beside compressed prose: %s", userOut)
	}
}
