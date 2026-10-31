package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/usetrim/trim/cli/internal/cliout"
	"github.com/usetrim/trim/cli/internal/deepopt"
)

func TestBootstrapChromeFallbacksAndHelp(t *testing.T) {
	ch := cliChrome{}
	applyHumanChromeFallbacks(&ch)
	if !strings.Contains(ch.HelpCompressFlagBootstrap, "llmlingua") {
		t.Fatalf("HelpCompressFlagBootstrap missing llmlingua: %q", ch.HelpCompressFlagBootstrap)
	}
	if !strings.Contains(ch.HelpCompressLong, "trim compress --bootstrap") {
		t.Fatalf("HelpCompressLong missing bootstrap command")
	}
	if !strings.Contains(ch.HelpCompressLong, "green banner") {
		t.Fatalf("HelpCompressLong missing green banner success UX: %q", ch.HelpCompressLong)
	}
	if !strings.Contains(ch.HelpRootLong, "trim compress --bootstrap") {
		t.Fatalf("HelpRootLong missing bootstrap command")
	}
	if strings.TrimSpace(ch.DeepBootstrapOk) == "" {
		t.Fatal("DeepBootstrapOk fallback empty")
	}
	if strings.TrimSpace(ch.DeepBootstrapOkDetail) == "" {
		t.Fatal("DeepBootstrapOkDetail fallback empty")
	}
	if !strings.Contains(ch.DeepBootstrapOkDetail, "Not installed in Trim cloud") {
		t.Fatalf("DeepBootstrapOkDetail should stress local-only: %q", ch.DeepBootstrapOkDetail)
	}

	// Stale cloud chrome: Long present but missing bootstrap - ensureContains must patch.
	stale := cliChrome{
		HelpRootLong:     "Trim keeps a local Fast Mode proxy.\n\nRun trim start.",
		HelpCompressLong: "Fast Mode and Deep Mode for files.",
		HelpCompressFlagBootstrap: "install Deep Mode dependencies on this machine",
	}
	applyHumanChromeFallbacks(&stale)
	if !strings.Contains(stale.HelpRootLong, "compress --bootstrap") {
		t.Fatalf("stale HelpRootLong not patched: %q", stale.HelpRootLong)
	}
	if !strings.Contains(stale.HelpCompressLong, "compress --bootstrap") {
		t.Fatalf("stale HelpCompressLong not patched: %q", stale.HelpCompressLong)
	}
	if !strings.Contains(stale.HelpCompressFlagBootstrap, "llmlingua") && !strings.Contains(stale.HelpCompressFlagBootstrap, "LLMLingua") {
		t.Fatalf("weak HelpCompressFlagBootstrap not upgraded: %q", stale.HelpCompressFlagBootstrap)
	}
}

func TestDeepUXNotifyHooksUseCliout(t *testing.T) {
	ch := cliChrome{}
	applyHumanChromeFallbacks(&ch)
	ux := deepUXFrom(ch)
	if ux.NotifySuccess == nil || ux.NotifyInfo == nil || ux.NotifyMuted == nil {
		t.Fatal("deepUXFrom must wire cliout Notify* hooks for live Deep auto-bootstrap green UX")
	}
	var buf bytes.Buffer
	prev := cliout.Writer
	cliout.Writer = &buf
	t.Cleanup(func() { cliout.Writer = prev })
	ux.NotifySuccess("Deep Mode dependencies installed on this machine")
	out := buf.String()
	if !strings.Contains(out, "●") || !strings.Contains(out, "Deep Mode dependencies installed") {
		t.Fatalf("NotifySuccess should print green ● success chrome, got: %q", out)
	}
}

func TestCompressBootstrapBypassesLoadLocal(t *testing.T) {
	// compressCmd RunE must honor --bootstrap before config.LoadLocal (HMAC bake).
	prev := compressBootstrap
	compressBootstrap = true
	t.Cleanup(func() { compressBootstrap = prev })

	// Force empty chrome path: heal is a no-op on non-Windows / when no sidecars.
	// We only assert the command returns without CONFIG_ENV_MISSING / HMAC errors.
	err := compressCmd.RunE(compressCmd, nil)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "HMAC") || strings.Contains(msg, "CONFIG_ENV") || strings.Contains(msg, "HostedCLIHMAC") {
			t.Fatalf("bootstrap must not require Trim Cloud HMAC: %v", err)
		}
		// Missing optimizer.py / python is acceptable in CI without release layout.
		if !strings.Contains(msg, "optimizer") &&
			!strings.Contains(msg, "requirements") &&
			!strings.Contains(msg, "Python") &&
			!strings.Contains(msg, "pip") &&
			!strings.Contains(strings.ToLower(msg), "deep") {
			t.Fatalf("unexpected bootstrap error: %v", err)
		}
	}
}

func TestRunCompressBootstrapSuccessChrome(t *testing.T) {
	// Skip real pip when Deep sidecars are unavailable in the test sandbox.
	ux := deepopt.UXChrome{
		RequirementsMissing: "requirements missing",
		PyMissing:           "optimizer missing",
	}
	if _, err := deepopt.ScriptPath(ux); err != nil {
		t.Skip("optimizer.py not beside test binary; chrome-only path covered above")
	}

	var buf bytes.Buffer
	prev := cliout.Writer
	cliout.Writer = &buf
	t.Cleanup(func() { cliout.Writer = prev })

	ch := cliChrome{}
	applyHumanChromeFallbacks(&ch)
	if err := runCompressBootstrap(ch, deepUXFrom(ch)); err != nil {
		t.Fatalf("runCompressBootstrap: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"●",
		ch.DeepBootstrapOk,
		"llmlingua",
		"local Python",
		"requirements-deep.txt",
		"trim compress",
		"Not installed in Trim cloud",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("bootstrap chrome missing %q in:\n%s", want, out)
		}
	}
}
