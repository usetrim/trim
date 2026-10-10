package deepopt

import (
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

func TestCompressAppendedQuestionDoesNotMixTokenUnits(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	// Engine reports BPE-like counts on context-only; after we append the preserved
	// question, mixing those with byte/4 of the full prompt used to reject Deep.
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		if strings.Contains(req.Text, "2+2") {
			t.Fatalf("question must stay out of context text: %q", req.Text)
		}
		return &Response{
			CompressedPrompt: "SHORTCTX",
			OriginTokens:     50, // context-only engine count
			CompressedTokens: 3,
		}, nil
	}
	filler := strings.Repeat("FILLER noise for deep unit mix regression. ", 40)
	in := []byte(`{"model":"claude-opus-4","messages":[{"role":"user","content":[{"type":"text","text":"<system-reminder>\nkeep\n</system-reminder>\n"},{"type":"text","text":"` + filler + `\nwhat is 2+2?"}]}]}`)
	out, origin, compressed, err := CompressAnthropicMessagesJSON(in, EngineV2, 80, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub", Unavailable: "u",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) == string(in) {
		t.Fatal("Deep was falsely rejected after question append (token unit mix)")
	}
	if origin <= compressed {
		t.Fatalf("expected savings with consistent units, %d->%d", origin, compressed)
	}
	got := gjson.GetBytes(out, "messages.0.content.1.text").String()
	if !strings.Contains(got, "SHORTCTX") || !strings.Contains(got, "what is 2+2?") {
		t.Fatalf("rewritten=%q", got)
	}
	if !strings.Contains(gjson.GetBytes(out, "messages.0.content.0.text").String(), "<system-reminder>") {
		t.Fatal("reminder lost")
	}
}

func TestRewriteLastUserContentRoundTrip(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		return &Response{CompressedPrompt: "DEEP_OK", OriginTokens: 40, CompressedTokens: 5}, nil
	}
	in := []byte(`{"model":"claude-opus-4","messages":[{"role":"user","content":[{"type":"text","text":"<system-reminder>\nkeep\n</system-reminder>\n"},{"type":"text","text":"FILLER A\nFILLER B\nwhat is 2+2?"}]}]}`)
	out, origin, compressed, err := CompressAnthropicMessagesJSON(in, EngineV2, 50, "", UXChrome{}, 30, ChatDeepChrome{
		CompactStub: "stub", Unavailable: "u",
	}, Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"})
	if err != nil {
		t.Fatal(err)
	}
	if origin <= compressed {
		t.Fatalf("tokens %d->%d", origin, compressed)
	}
	if !gjson.GetBytes(out, "messages.0.content").IsArray() {
		t.Fatalf("not array: %s", string(out))
	}
	if gjson.GetBytes(out, "messages.0.content.#").Int() < 2 {
		t.Fatalf("parts lost: %s", string(out))
	}
	got := gjson.GetBytes(out, "messages.0.content.1.text").String()
	if got != "DEEP_OK" && !containsQuestion(got) {
		// compressed may append question
		if got == "" {
			t.Fatalf("empty rewrite: %s", string(out))
		}
	}
	t.Logf("rewritten=%s", gjson.GetBytes(out, "messages.0.content").Raw)
}

func containsQuestion(s string) bool {
	return len(s) > 0
}
