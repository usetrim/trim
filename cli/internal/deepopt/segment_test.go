package deepopt

import (
	"strings"
	"testing"
)

func TestDeepProseFromUserTextSplitsPathIslands(t *testing.T) {
	blob := "please inspect pkg/proxy and explain the gate carefully"
	prose := DeepProseFromUserText(blob)
	if prose == "" {
		t.Fatal("expected prose islands")
	}
	if strings.Contains(prose, "pkg/proxy") {
		t.Fatalf("path must stay out of prose: %q", prose)
	}
	if !strings.Contains(prose, "please inspect") || !strings.Contains(prose, "explain") {
		t.Fatalf("prose missing: %q", prose)
	}
}

func TestDeepProseFromUserTextPureProse(t *testing.T) {
	blob := "FILLER A\nFILLER B\nwhat is 2+2?"
	if DeepProseFromUserText(blob) != blob {
		t.Fatalf("pure prose must pass through, got %q", DeepProseFromUserText(blob))
	}
}

func TestDeepProseFromUserTextProtectedOnly(t *testing.T) {
	if DeepProseFromUserText(`C:\Users\berek\proj\main.go`) != "" {
		t.Fatal("drive path only must yield empty prose")
	}
	if DeepProseFromUserText("https://example.com/docs") != "" {
		t.Fatal("URL only must yield empty prose")
	}
	if DeepProseFromUserText("```go\npackage main\n```") != "" {
		t.Fatal("fence only must yield empty prose")
	}
}

func TestReassembleDeepSegmentsKeepsPath(t *testing.T) {
	blob := "please open C:\\Users\\berek\\a.go and summarize carefully"
	spans := segmentUserTextForDeep(blob)
	out := reassembleDeepSegments(spans, "SHORT")
	if !strings.Contains(out, "SHORT") {
		t.Fatalf("compressed prose missing: %q", out)
	}
	if !strings.Contains(out, `C:\Users\berek\a.go`) {
		t.Fatalf("path must remain: %q", out)
	}
}

func TestJoinDeepSpansPreservesPathBetweenIslands(t *testing.T) {
	blob := "please open C:\\Users\\berek\\a.go and summarize carefully"
	spans := segmentUserTextForDeep(blob)
	overrides := map[int]string{}
	for i, s := range spans {
		if s.protected || strings.TrimSpace(s.text) == "" {
			continue
		}
		if len(overrides) == 0 {
			overrides[i] = "OPEN"
		} else {
			overrides[i] = "SUM"
		}
	}
	if len(overrides) < 2 {
		t.Fatalf("expected two prose islands around path, spans=%v", spans)
	}
	out := joinDeepSpansWithOverrides(spans, overrides)
	pathIdx := strings.Index(out, `C:\Users\berek\a.go`)
	openIdx := strings.Index(out, "OPEN")
	sumIdx := strings.Index(out, "SUM")
	if pathIdx < 0 || openIdx < 0 || sumIdx < 0 {
		t.Fatalf("missing pieces: %q", out)
	}
	if !(openIdx < pathIdx && pathIdx < sumIdx) {
		t.Fatalf("path must stay between prose islands: %q", out)
	}
}

func TestApplySegmentedAllowlistDeepPreservesPathOrder(t *testing.T) {
	prev := compressFn
	t.Cleanup(func() { compressFn = prev })
	calls := 0
	compressFn = func(req Request, _ UXChrome, _ int) (*Response, error) {
		calls++
		short := "A"
		if calls > 1 {
			short = "B"
		}
		return &Response{CompressedPrompt: short, OriginTokens: 40, CompressedTokens: 5}, nil
	}
	blob := "please open C:\\Users\\berek\\a.go and summarize carefully"
	spans := segmentUserTextForDeep(blob)
	out, origin, compressed, applied, err := applySegmentedAllowlistDeep(
		spans, EngineV2, 0.7, "", UXChrome{}, 30,
		Runtime{DeviceMap: "cpu", V2Model: "v2", V1Model: "v1", LongModel: "long"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !applied || origin <= compressed {
		t.Fatalf("expected applied shrink origin=%d compressed=%d", origin, compressed)
	}
	if calls < 2 {
		t.Fatalf("each prose island must Deep separately, calls=%d", calls)
	}
	pathIdx := strings.Index(out, `C:\Users\berek\a.go`)
	aIdx := strings.Index(out, "A")
	bIdx := strings.Index(out, "B")
	if pathIdx < 0 || aIdx < 0 || bIdx < 0 {
		t.Fatalf("missing pieces: %q", out)
	}
	if !(aIdx < pathIdx && pathIdx < bIdx) {
		t.Fatalf("path order broken after Segmented Deep: %q", out)
	}
}
