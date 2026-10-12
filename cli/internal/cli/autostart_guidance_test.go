package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/usetrim/trim/cli/internal/cliout"
)

func TestPrintAutostartOSHookGuidance_AccessDenied(t *testing.T) {
	var buf bytes.Buffer
	prev := cliout.Writer
	cliout.Writer = &buf
	t.Cleanup(func() { cliout.Writer = prev })
	t.Setenv("NO_COLOR", "1")

	ch := cliChrome{}
	applyHumanChromeFallbacks(&ch)
	printAutostartOSHookGuidance(ch, true, errString("schtasks: exit status 1: ERROR: Access is denied."))
	out := buf.String()
	for _, want := range []string{
		"Preference saved",
		"Access denied",
		"trim start",
		"Manual start",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "schtasks: exit status") && !flagVerbose {
		// Raw detail only with --verbose
		t.Fatalf("raw schtasks should not show by default:\n%s", out)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
