package deepopt

import (
	"slices"
	"testing"
)

func TestApplyEngineUsesBillingForceTokensOnly(t *testing.T) {
	rt := Runtime{
		DeviceMap:     "cpu",
		V1Model:       "v1-model",
		V2Model:       "v2-model",
		LongModel:     "long-model",
		V2ForceTokens: []string{"KEEPME", "/", "\n", ".", "?", "!"},
	}
	for _, eng := range []Engine{EngineV1, EngineV2, EngineLong} {
		req := &Request{Text: "x", Rate: 0.7, Engine: string(eng)}
		if err := rt.ApplyEngine(req, eng); err != nil {
			t.Fatalf("%s: %v", eng, err)
		}
		if !slices.Contains(req.ForceTokens, "KEEPME") {
			t.Fatalf("%s missing billing force token: %v", eng, req.ForceTokens)
		}
		// Must not invent tokens beyond billing_settings sync.
		if len(req.ForceTokens) != len(rt.V2ForceTokens) {
			t.Fatalf("%s force_tokens invent/extra: got %v want %v", eng, req.ForceTokens, rt.V2ForceTokens)
		}
		if eng == EngineV2 && !req.UseLLMLingua2 {
			t.Fatal("v2 must set use_llmlingua2")
		}
		if eng != EngineV2 && req.UseLLMLingua2 {
			t.Fatalf("%s must not set use_llmlingua2", eng)
		}
	}
}

func TestApplyEngineNoInventForceTokensWhenBillingEmpty(t *testing.T) {
	rt := Runtime{DeviceMap: "cpu", V2Model: "v2-model", V2ForceTokens: nil}
	req := &Request{Text: "x", Rate: 0.7, Engine: "v2"}
	if err := rt.ApplyEngine(req, EngineV2); err != nil {
		t.Fatal(err)
	}
	if len(req.ForceTokens) != 0 {
		t.Fatalf("empty billing must not invent force_tokens: %v", req.ForceTokens)
	}
	if req.ForceReserveDigit != nil || req.DropConsecutive != nil || req.UseSentenceLevelFilter != nil {
		t.Fatal("unsynced dials must not invent force_reserve_digit/drop_consecutive/use_sentence_level_filter")
	}
}

func TestApplyEngineEmitsBillingCodingDials(t *testing.T) {
	rt := Runtime{
		DeviceMap: "cpu", V2Model: "v2-model", V2ForceTokens: []string{"/"},
		ForceReserveDigit: true, DropConsecutive: true, UseSentenceLevelFilter: false,
		LLMLinguaCodingDials: true,
	}
	req := &Request{Text: "x", Rate: 0.7, Engine: "v2"}
	if err := rt.ApplyEngine(req, EngineV2); err != nil {
		t.Fatal(err)
	}
	if req.ForceReserveDigit == nil || !*req.ForceReserveDigit {
		t.Fatal("expected billing force_reserve_digit=true")
	}
	if req.DropConsecutive == nil || !*req.DropConsecutive {
		t.Fatal("expected billing drop_consecutive=true")
	}
	if req.UseSentenceLevelFilter == nil || *req.UseSentenceLevelFilter {
		t.Fatal("expected billing use_sentence_level_filter=false (coding default)")
	}
}

func TestLastUserHasRelativeCodePath(t *testing.T) {
	if !lastUserHasRelativeCodePath("please edit src/proxy/server.go and explain") {
		t.Fatal("expected relative path hit")
	}
	if !lastUserHasRelativeCodePath(`fix .\pkg\bar.ts`) {
		t.Fatal("expected windows relative path hit")
	}
	if !lastUserHasRelativeCodePath("inspect pkg/proxy carefully") {
		t.Fatal("expected coding dir tree hit")
	}
	if lastUserHasRelativeCodePath("FILLER A\nFILLER B\nwhat is 2+2?") {
		t.Fatal("plain prose must not look like a code path")
	}
	if lastUserHasRelativeCodePath("node.js is fine without a slash") {
		t.Fatal("extension alone must not trip")
	}
}
