//go:build windows

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func TestConfigureDetachedStartProc_NoConsole(t *testing.T) {
	cmd := exec.Command("trim")
	configureDetachedStartProc(cmd)
	if cmd.SysProcAttr == nil {
		t.Fatal("expected SysProcAttr")
	}
	want := uint32(windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_NO_WINDOW)
	if cmd.SysProcAttr.CreationFlags != want {
		t.Fatalf("CreationFlags=%#x want %#x", cmd.SysProcAttr.CreationFlags, want)
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Fatal("HideWindow should be true so the worker does not hold the shell console")
	}
}

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
	// Release/npm sidecars beside the source binary must follow into LocalAppData.
	for _, name := range []string{"optimizer.py", "requirements-deep.txt", "requirements.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name+"-body"), 0o644); err != nil {
			t.Fatal(err)
		}
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
	stableDir := filepath.Dir(stable)
	for _, name := range windowsDeepSidecarNames() {
		got, err := os.ReadFile(filepath.Join(stableDir, name))
		if err != nil {
			t.Fatalf("sidecar %s: %v", name, err)
		}
		if string(got) != name+"-body" {
			t.Fatalf("sidecar %s content %q", name, got)
		}
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

func TestResolveWindowsDeepSidecarFileNestedArchive(t *testing.T) {
	dir := t.TempDir()
	// v0.1.23 GoReleaser bug: optimizer.py/optimizer.py
	nestedDir := filepath.Join(dir, "optimizer.py")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(nestedDir, "optimizer.py")
	if err := os.WriteFile(nested, []byte("nested-opt"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := resolveWindowsDeepSidecarFile(dir, "optimizer.py")
	if got != nested {
		t.Fatalf("nested resolve got %q want %q", got, nested)
	}
	dst := t.TempDir()
	if err := syncWindowsDeepSidecars(dir, dst); err != nil {
		t.Fatal(err)
	}
	flat := filepath.Join(dst, "optimizer.py")
	b, err := os.ReadFile(flat)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "nested-opt" {
		t.Fatalf("flattened content %q", b)
	}
}

func TestEnsureWindowsDeepSidecarsSkipsIncompleteSelfDonor(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOCALAPPDATA", filepath.Join(dir, "Local"))
	t.Setenv("APPDATA", filepath.Join(dir, "Roaming"))
	t.Setenv("USERPROFILE", dir)
	t.Setenv("HOME", dir)
	t.Setenv("PATH", dir)

	stableDir := filepath.Join(dir, "Local", "Trim", "bin")
	if err := os.MkdirAll(stableDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Incomplete Always-on tree (optimizer only) must not block heal from npm vendor.
	if err := os.WriteFile(filepath.Join(stableDir, "optimizer.py"), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	donor := filepath.Join(dir, "Roaming", "npm", "node_modules", "@usetrim", "trim", "vendor")
	if err := os.MkdirAll(donor, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(donor, "optimizer.py"), []byte("good-opt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(donor, "requirements-deep.txt"), []byte("good-req"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ensureWindowsDeepSidecars(stableDir); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(stableDir, "optimizer.py"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "good-opt" {
		t.Fatalf("expected npm donor, got %q", got)
	}
	req, err := os.ReadFile(filepath.Join(stableDir, "requirements-deep.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(req) != "good-req" {
		t.Fatalf("requirements %q", req)
	}
}

func TestEnsureWindowsDeepSidecarsHealsStableBin(t *testing.T) {
	dir := t.TempDir()
	// Isolate from real machine donors (LocalAppData / AppData npm / Scoop).
	t.Setenv("LOCALAPPDATA", filepath.Join(dir, "Local"))
	t.Setenv("APPDATA", filepath.Join(dir, "Roaming"))
	t.Setenv("USERPROFILE", dir)
	t.Setenv("HOME", dir)
	stableDir := filepath.Join(dir, "stable-bin")
	if err := os.MkdirAll(stableDir, 0o755); err != nil {
		t.Fatal(err)
	}
	donor := filepath.Join(dir, "npm-vendor")
	if err := os.MkdirAll(donor, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(donor, "optimizer.py"), []byte("opt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(donor, "requirements-deep.txt"), []byte("req"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", donor)
	if err := ensureWindowsDeepSidecars(stableDir); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(stableDir, "optimizer.py"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "opt" {
		t.Fatalf("healed optimizer %q", got)
	}
	req, err := os.ReadFile(filepath.Join(stableDir, "requirements-deep.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(req) != "req" {
		t.Fatalf("healed requirements %q", req)
	}
}
