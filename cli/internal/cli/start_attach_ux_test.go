package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/usetrim/trim/cli/internal/cliout"
)

func TestPrintStartReadyUI_Attached(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	var buf bytes.Buffer
	prev := cliout.Writer
	cliout.Writer = &buf
	t.Cleanup(func() { cliout.Writer = prev })
	t.Setenv("NO_COLOR", "1")

	ch := cliChrome{}
	applyHumanChromeFallbacks(&ch)
	printStartReadyUI(ch, "http", "8888", true)
	out := buf.String()
	for _, want := range []string{
		"already running",
		"127.0.0.1:8888",
		"Local Base URL",
		"/v1",
		"Dashboard",
		"/dashboard",
		"Attached to the existing local proxy",
		"Cursor Agent",
		"cursor-link",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	// Same design language as status/autostart: no blue Info ℹ line.
	if strings.Contains(out, "ℹ") || strings.Contains(out, "Local dashboard:") {
		t.Fatalf("dashboard should be a Table row, not Info:\n%s", out)
	}
}

func TestPrintStartReadyUI_FreshStart(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	var buf bytes.Buffer
	prev := cliout.Writer
	cliout.Writer = &buf
	t.Cleanup(func() { cliout.Writer = prev })
	t.Setenv("NO_COLOR", "1")

	ch := cliChrome{}
	applyHumanChromeFallbacks(&ch)
	printStartReadyUI(ch, "http", "8888", false)
	out := buf.String()
	for _, want := range []string{
		"Trim proxy running",
		"Local Base URL",
		"127.0.0.1:8888/v1",
		"On-device clients",
		"Cursor Agent",
		"cursor-link",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "already running") {
		t.Fatalf("fresh start should not say already running:\n%s", out)
	}
}
