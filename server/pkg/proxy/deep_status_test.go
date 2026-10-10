package proxy

import "testing"

func TestDeepStageSavedPercent(t *testing.T) {
	if got := DeepStageSavedPercent(0, 0); got != 0 {
		t.Fatalf("empty origin: got %v", got)
	}
	if got := DeepStageSavedPercent(100, 40); got < 59.9 || got > 60.1 {
		t.Fatalf("100->40: got %v want ~60", got)
	}
	if got := DeepStageSavedPercent(100, 100); got != 0 {
		t.Fatalf("no shrink: got %v", got)
	}
	if got := DeepStageSavedPercent(100, 120); got != 0 {
		t.Fatalf("expand: got %v", got)
	}
}

func TestRecordWithPreviewTracksDeepStage(t *testing.T) {
	s := NewServer(Options{SavingsUsdPerMTok: 3, PreviewMaxChars: 200})
	s.recordWithPreview(1000, 400, 12.5, `{"a":1}`, `{"a":2}`, "anthropic", false, DeepStatusApplied, 800, 200)
	st := s.Stats()
	if st.LastDeepStatus != DeepStatusApplied {
		t.Fatalf("status=%q", st.LastDeepStatus)
	}
	if st.LastDeepStageBefore != 800 || st.LastDeepStageAfter != 200 {
		t.Fatalf("stage=%d->%d", st.LastDeepStageBefore, st.LastDeepStageAfter)
	}
	if st.LastDeepStageSavedPct < 74.9 || st.LastDeepStageSavedPct > 75.1 {
		t.Fatalf("stage pct=%v", st.LastDeepStageSavedPct)
	}
	if st.LastDoor != "anthropic" {
		t.Fatalf("door=%q", st.LastDoor)
	}
}

func TestRecordWithPreviewTracksFailClosedAndSkippedStatuses(t *testing.T) {
	s := NewServer(Options{SavingsUsdPerMTok: 3, PreviewMaxChars: 200})

	s.recordWithPreview(200, 200, 1, `{"a":1}`, `{"a":1}`, "anthropic", false, DeepStatusFailClosedExpand, 180, 220)
	st := s.Stats()
	if st.LastDeepStatus != DeepStatusFailClosedExpand {
		t.Fatalf("fail_closed status=%q", st.LastDeepStatus)
	}
	// Expansion fail-closed keeps Fast body; stage after can be >= before → saved % stays 0.
	if st.LastDeepStageSavedPct != 0 {
		t.Fatalf("fail_closed stage pct=%v want 0", st.LastDeepStageSavedPct)
	}

	s.recordWithPreview(80, 80, 1, `{"a":1}`, `{"a":1}`, "anthropic", false, DeepStatusSkippedMin, 40, 40)
	st = s.Stats()
	if st.LastDeepStatus != DeepStatusSkippedMin {
		t.Fatalf("skipped_min status=%q", st.LastDeepStatus)
	}

	s.recordWithPreview(50, 50, 1, `{"a":1}`, `{"a":1}`, "anthropic", false, DeepStatusSkippedEmpty, 0, 0)
	st = s.Stats()
	if st.LastDeepStatus != DeepStatusSkippedEmpty {
		t.Fatalf("skipped_empty status=%q", st.LastDeepStatus)
	}
}
