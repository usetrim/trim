package deepopt

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCursorUserInfoChromeFrozenNotProse(t *testing.T) {
	blob := `<user_info>
OS Version: win32 10.0.26200
Workspace Path: C:\Users\berek\Downloads\proj
</user_info>

<agent_transcripts>
Agent transcripts live in C:\Users\berek\.cursor\projects\x/agent-transcripts
</agent_transcripts>

hi`
	prose := DeepProseFromUserText(blob)
	if strings.Contains(prose, "user_info") || strings.Contains(prose, "agent_transcripts") {
		t.Fatalf("IDE XML chrome must be frozen, prose=%q", prose)
	}
	if strings.TrimSpace(prose) != "hi" {
		t.Fatalf("only user hi should remain as prose, got %q", prose)
	}
}

func TestCursorHiStillEstimatesProseForDeep(t *testing.T) {
	// User Deep prefs control whether LLMLingua runs; we must not silently zero-out short turns.
	blob := `<user_info>
OS Version: win32
Workspace Path: C:\Users\berek\proj
</user_info>

hi`
	userJSON, err := json.Marshal(blob)
	if err != nil {
		t.Fatal(err)
	}
	in := []byte(`{"model":"trim-claude-sonnet","messages":[{"role":"user","content":` + string(userJSON) + `}]}`)
	n := EstimateLiveDeepInputTokens(in, "/v1/chat/completions")
	if n == 0 {
		t.Fatal("hi remains estimable prose after chrome freeze (Deep must be allowed when prefs say Deep)")
	}
}

func TestLongCursorProseStillEstimates(t *testing.T) {
	var b strings.Builder
	b.WriteString("<user_info>\nOS: win32\n</user_info>\n\n")
	for i := 0; i < 40; i++ {
		b.WriteString("Please explain the authentication flow in detail and list every edge case. ")
	}
	userJSON, err := json.Marshal(b.String())
	if err != nil {
		t.Fatal(err)
	}
	in := []byte(`{"model":"trim-claude-sonnet","messages":[{"role":"user","content":` + string(userJSON) + `}]}`)
	if n := EstimateLiveDeepInputTokens(in, "/v1/chat/completions"); n == 0 {
		t.Fatal("long prose must remain estimable after chrome freeze")
	}
}
