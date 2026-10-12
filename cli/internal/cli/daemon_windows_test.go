//go:build windows

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsStableDaemonExe(t *testing.T) {
	t.Setenv("LOCALAPPDATA", `C:\Users\test\AppData\Local`)
	got, err := windowsStableDaemonExe()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(`C:\Users\test\AppData\Local`, "Trim", "bin", "trim.exe")
	if !strings.EqualFold(got, want) {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestCopyFileReplace(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.exe")
	dst := filepath.Join(dir, "dst.exe")
	if err := os.WriteFile(src, []byte("trim-binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFileReplace(src, dst); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "trim-binary" {
		t.Fatalf("dst content %q", b)
	}
}

func TestSyncWindowsStableDaemonExeIdempotent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOCALAPPDATA", dir)
	src := filepath.Join(dir, "source-trim.exe")
	if err := os.WriteFile(src, []byte("v1"), 0o755); err != nil {
		t.Fatal(err)
	}
	stable, err := syncWindowsStableDaemonExe(src)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "Trim", "bin", "trim.exe")
	if !strings.EqualFold(stable, want) {
		t.Fatalf("stable path %q want %q", stable, want)
	}
	b, err := os.ReadFile(stable)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "v1" {
		t.Fatalf("content %q", b)
	}
	// Re-sync from same path should succeed.
	stable2, err := syncWindowsStableDaemonExe(src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(stable, stable2) {
		t.Fatalf("resync path drift %q vs %q", stable, stable2)
	}
}
