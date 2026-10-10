package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeContinueConfigYAML_EmptyModelsWritesTemplate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("name: Main Config\nversion: 1.0.0\nschema: v1\nmodels: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TRIM_SETUP_DEFAULT_MODEL", "")
	t.Setenv("TRIM_SETUP_DEFAULT_MODEL_TITLE", "")
	if err := mergeContinueConfigYAML(path, "http://127.0.0.1:8888/v1", "no-model", "chrome-down"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{
		"apiBase: http://127.0.0.1:8888/v1",
		"model: trim-claude-sonnet",
		"provider: openai",
		"useResponsesApi: false",
		"tool_use",
		"REPLACE_WITH_PROVIDER_KEY",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
}

func TestMergeContinueConfigYAML_UpdatesExistingAPIBase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	body := `name: Main Config
version: 1.0.0
schema: v1
models:
  - name: Claude via Trim
    provider: openai
    model: trim-claude-sonnet
    apiBase: http://127.0.0.1:9999/v1
    apiKey: sk-test
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := mergeContinueConfigYAML(path, "http://127.0.0.1:8888/v1", "no-model", "chrome-down"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, "apiBase: http://127.0.0.1:8888/v1") {
		t.Fatalf("apiBase not updated:\n%s", text)
	}
	if strings.Contains(text, "9999") {
		t.Fatalf("old apiBase still present:\n%s", text)
	}
	if !strings.Contains(text, "apiKey: sk-test") {
		t.Fatalf("apiKey should be preserved:\n%s", text)
	}
	for _, want := range []string{"tool_use", "useResponsesApi: false"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing agent flag %q in:\n%s", want, text)
		}
	}
}

func TestEnsureContinueTrimAgentFlags_Idempotent(t *testing.T) {
	in := `models:
  - name: Claude via Trim
    provider: openai
    model: trim-claude-sonnet
    apiBase: http://127.0.0.1:8888/v1
    capabilities:
      - tool_use
    useResponsesApi: false
`
	out := ensureContinueTrimAgentFlags(in)
	if strings.Count(out, "tool_use") != 1 {
		t.Fatalf("tool_use duplicated:\n%s", out)
	}
	if strings.Count(out, "useResponsesApi:") != 1 {
		t.Fatalf("useResponsesApi duplicated:\n%s", out)
	}
}

func TestContinueConfigPath_PrefersYAML(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home) // Windows
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".continue")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"models":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("models: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// continueConfigPath uses os.UserHomeDir - override via HOME may not work on Windows
	// if USERPROFILE is used. Call logic inline by temporarily chdir? Better: test via
	// rewriting - we export nothing. Skip if UserHomeDir != home.
	gotHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if gotHome != home {
		t.Skipf("cannot override UserHomeDir (got %s want %s)", gotHome, home)
	}
	p, err := continueConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(p, "config.yaml") {
		t.Fatalf("want config.yaml, got %s", p)
	}
}

func TestClearContinueTrimBaseYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	body := "models:\n  - name: X\n    apiBase: http://127.0.0.1:8888/v1\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := clearContinueTrimBaseYAML(path, "http://127.0.0.1:8888/v1"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if strings.Contains(string(raw), "8888") {
		t.Fatalf("trim base should be cleared:\n%s", raw)
	}
}
