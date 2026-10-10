package deepopt

import (
	"slices"
	"testing"
)

func TestApplyEngineMergesForceTokensAllEngines(t *testing.T) {
	rt := Runtime{
		DeviceMap:     "cpu",
		V1Model:       "v1-model",
		V2Model:       "v2-model",
		LongModel:     "long-model",
		V2ForceTokens: []string{"KEEPME", "/"},
	}
	for _, eng := range []Engine{EngineV1, EngineV2, EngineLong} {
		req := &Request{Text: "x", TargetToken: 10, Engine: string(eng)}
		if err := rt.ApplyEngine(req, eng); err != nil {
			t.Fatalf("%s: %v", eng, err)
		}
		if !slices.Contains(req.ForceTokens, "KEEPME") {
			t.Fatalf("%s missing billing force token: %v", eng, req.ForceTokens)
		}
		for _, need := range []string{"\n", "/", ".", "?", "!"} {
			if !slices.Contains(req.ForceTokens, need) {
				t.Fatalf("%s missing coding force token %q: %v", eng, need, req.ForceTokens)
			}
		}
		if eng == EngineV2 && !req.UseLLMLingua2 {
			t.Fatal("v2 must set use_llmlingua2")
		}
		if eng != EngineV2 && req.UseLLMLingua2 {
			t.Fatalf("%s must not set use_llmlingua2", eng)
		}
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
