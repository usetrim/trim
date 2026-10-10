package deepopt

import (
	"fmt"
	"testing"
)

func TestIsDeepOOMError(t *testing.T) {
	t.Parallel()
	if !IsDeepOOMError(fmt.Errorf("CUDA out of memory")) {
		t.Fatal("expected oom")
	}
	if !IsDeepOOMError(fmt.Errorf("RuntimeError: torch.cuda.OutOfMemoryError: tried to allocate")) {
		t.Fatal("expected torch oom")
	}
	if IsDeepOOMError(fmt.Errorf("connection refused")) {
		t.Fatal("false positive")
	}
	if IsDeepOOMError(fmt.Errorf("bedroom unavailable")) {
		t.Fatal("substring oom false positive")
	}
}

func TestEstimateLiveDeepInputTokens(t *testing.T) {
	t.Parallel()
	in := []byte(`{"model":"x","messages":[{"role":"system","content":"sys"},{"role":"user","content":"hello world please"}]}`)
	n := EstimateLiveDeepInputTokens(in, "/v1/chat/completions")
	if n < 1 {
		t.Fatalf("expected compressible tokens, got %d", n)
	}
	// System text must not be counted (Claude Code agent chrome gate).
	sysOnly := []byte(`{"messages":[{"role":"system","content":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`)
	if EstimateLiveDeepInputTokens(sysOnly, "/v1/messages") != 0 {
		t.Fatal("system-only should be 0")
	}
	if EstimateLiveDeepInputTokens([]byte(`{}`), "/v1/chat/completions") != 0 {
		t.Fatal("empty messages should be 0")
	}
	if EstimateLiveDeepInputTokens(nil, "/v1/chat/completions") != 0 {
		t.Fatal("nil body should be 0")
	}
}

func TestRequestWantsStream(t *testing.T) {
	t.Parallel()
	if !RequestWantsStream([]byte(`{"stream":true,"messages":[]}`)) {
		t.Fatal("expected stream")
	}
	if RequestWantsStream([]byte(`{"stream":false}`)) {
		t.Fatal("expected non-stream")
	}
}
