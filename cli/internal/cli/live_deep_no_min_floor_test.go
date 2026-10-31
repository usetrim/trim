package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/usetrim/trim/cli/internal/deepopt"
)

// Regression: live Deep must not hardcode a min-token skip. User prefs (DefaultTier=deep)
// plus estimable prose after chrome freeze decide; est==0 → skipped_empty only.
func TestLiveDeepSourceHasNoMinTokenFloor(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	src, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "live_deep.go"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	for _, ban := range []string{
		"LiveDeepMinProseTokens",
		"DeepStatusSkippedMin",
	} {
		if strings.Contains(body, ban) {
			t.Fatalf("live_deep.go must not reference %q (user prefs only)", ban)
		}
	}
	if strings.Contains(body, "est <") {
		t.Fatal("live_deep.go must not compare est against a min floor")
	}
}

func TestShortCursorHiEstimatesProseSoDeepCanRun(t *testing.T) {
	blob := `<user_info>
OS Version: win32
</user_info>

hi`
	userJSON, err := json.Marshal(blob)
	if err != nil {
		t.Fatal(err)
	}
	in := []byte(`{"model":"x","messages":[{"role":"user","content":` + string(userJSON) + `}]}`)
	n := deepopt.EstimateLiveDeepInputTokens(in, "/v1/chat/completions")
	if n == 0 {
		t.Fatal("short user prose after chrome freeze must remain estimable so Deep prefs can apply")
	}
}
