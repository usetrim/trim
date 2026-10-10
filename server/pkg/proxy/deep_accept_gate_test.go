package proxy

import (
	"strings"
	"testing"
)

func TestShouldAcceptDeepResultWireVsWire(t *testing.T) {
	t.Parallel()
	fast := []byte(`{"messages":[{"role":"system","content":"` + strings.Repeat("S", 400) + `"},{"role":"user","content":"` + strings.Repeat("F", 200) + ` what is 2+2?"}]}`)
	// Deep only rewrote last user to a short string (realistic last-user-only).
	deep := []byte(`{"messages":[{"role":"system","content":"` + strings.Repeat("S", 400) + `"},{"role":"user","content":"2+2?"}]}`)

	// Fast text-sum `after` can be tiny vs full-body estimate - the old bug.
	textSumAfter := 30
	fullDeepEst := estimate(deep)
	if fullDeepEst <= textSumAfter {
		t.Fatalf("fixture broken: fullDeepEst=%d textSumAfter=%d", fullDeepEst, textSumAfter)
	}
	// Old (wrong) gate would reject:
	oldGate := fullDeepEst <= textSumAfter && len(deep) <= len(fast)
	if oldGate {
		t.Fatal("expected old gate to reject this realistic Deep shrink")
	}
	// New gate must accept (wire shrank + Deep-stage tokens shrank).
	if !shouldAcceptDeepResult(fast, deep, 80, 12) {
		t.Fatal("should accept Deep when wire+Deep-stage both shrink")
	}
	if shouldAcceptDeepResult(fast, []byte(string(deep)+strings.Repeat("X", len(fast))), 80, 12) {
		t.Fatal("must reject wire expansion")
	}
	if shouldAcceptDeepResult(fast, deep, 10, 50) {
		t.Fatal("must reject Deep-stage token expansion")
	}
	if shouldAcceptDeepResult(fast, nil, 80, 12) {
		t.Fatal("must reject empty deep body")
	}
}

func TestAcceptGateParityWithDeepExpandedLogic(t *testing.T) {
	t.Parallel()
	// shouldAcceptDeepResult must stay the logical inverse of deepopt.DeepExpanded.
	fast := []byte(`{"messages":[{"role":"user","content":"hello world please"}]}`)
	good := []byte(`{"messages":[{"role":"user","content":"hi"}]}`)
	cases := []struct {
		deep       []byte
		origin     int
		compressed int
	}{
		{good, 80, 12},
		{[]byte(string(good) + strings.Repeat("X", len(fast))), 80, 12},
		{good, 10, 50},
		{nil, 80, 12},
		{good, 0, 0},
	}
	for i, c := range cases {
		accept := shouldAcceptDeepResult(fast, c.deep, c.origin, c.compressed)
		expanded := false
		if len(c.deep) == 0 {
			expanded = true
		} else if len(c.deep) > len(fast) {
			expanded = true
		} else if c.origin > 0 && c.compressed >= c.origin {
			expanded = true
		} else if estimate(c.deep) > estimate(fast) {
			expanded = true
		}
		if accept == expanded {
			t.Fatalf("case %d parity broken accept=%v expanded=%v", i, accept, expanded)
		}
	}
}
