package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCloudflaredDownloadURLSupported(t *testing.T) {
	url, err := cloudflaredDownloadURL()
	if err != nil {
		// Unsupported GOOS/GOARCH in exotic test hosts is OK to skip.
		if strings.Contains(err.Error(), "unsupported platform") {
			t.Skip(err.Error())
		}
		t.Fatal(err)
	}
	if !strings.HasPrefix(url, "https://github.com/cloudflare/cloudflared/releases/latest/download/") {
		t.Fatalf("unexpected url: %s", url)
	}
	switch runtime.GOOS {
	case "windows":
		if !strings.HasSuffix(url, ".exe") {
			t.Fatalf("windows asset should be .exe: %s", url)
		}
		// Cloudflare has no windows-arm64 asset; arm64 hosts must use amd64.
		if runtime.GOARCH == "arm64" && !strings.Contains(url, "windows-amd64") {
			t.Fatalf("windows/arm64 must use amd64 asset: %s", url)
		}
	case "darwin":
		if !strings.HasSuffix(url, ".tgz") {
			t.Fatalf("darwin asset should be .tgz: %s", url)
		}
	case "linux":
		// Linux release assets are extensionless binaries (…/cloudflared-linux-amd64).
		// Do not scan the full URL for "." - github.com always contains dots.
		base := filepath.Base(url)
		if !strings.HasPrefix(base, "cloudflared-linux-") {
			t.Fatalf("linux asset unexpected: %s", url)
		}
		if strings.Contains(base, ".") {
			t.Fatalf("linux asset should have no file extension: %s", url)
		}
	}
}

func TestExtractCloudflaredFromTGZ(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	body := []byte("fake-cloudflared-binary")
	hdr := &tar.Header{
		Name: "cloudflared",
		Mode: 0o755,
		Size: int64(len(body)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	dest := filepath.Join(dir, "cloudflared")
	if err := extractCloudflaredFromTGZ(bytes.NewReader(buf.Bytes()), dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("extracted mismatch: %q", got)
	}
}

func TestFindCloudflaredHonorsEnv(t *testing.T) {
	dir := t.TempDir()
	name := "cloudflared"
	if runtime.GOOS == "windows" {
		name = "cloudflared.exe"
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TRIM_CLOUDFLARED", path)
	got, err := findCloudflared()
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Fatalf("got %q want %q", got, path)
	}
}

func TestCloudflaredInstallHintNonEmpty(t *testing.T) {
	chrome := cliChrome{}
	applyHumanChromeFallbacks(&chrome)
	h := cloudflaredInstallHint(chrome)
	if strings.TrimSpace(h) == "" {
		t.Fatal("empty hint")
	}
	_ = io.Discard
}

func TestEnsureCloudflaredAutoDownload(t *testing.T) {
	if testing.Short() {
		t.Skip("network download")
	}
	if _, err := cloudflaredDownloadURL(); err != nil {
		t.Skip(err.Error())
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("TRIM_CLOUDFLARED", "")
	// Avoid finding a system cloudflared via PATH / Program Files.
	t.Setenv("PATH", filepath.Join(home, "empty-bin"))
	t.Setenv("ProgramFiles", filepath.Join(home, "pf"))
	t.Setenv("ProgramFiles(x86)", filepath.Join(home, "pf86"))
	t.Setenv("LOCALAPPDATA", filepath.Join(home, "local"))
	_ = os.MkdirAll(filepath.Join(home, "empty-bin"), 0o755)

	got, err := ensureCloudflared()
	if err != nil {
		t.Skipf("cloudflared download unavailable: %v", err)
	}
	wantName := "cloudflared"
	if runtime.GOOS == "windows" {
		wantName = "cloudflared.exe"
	}
	if filepath.Base(got) != wantName {
		t.Fatalf("got %q", got)
	}
	st, err := os.Stat(got)
	if err != nil || st.IsDir() || st.Size() < 1000 {
		t.Fatalf("downloaded binary invalid: %v size=%d", err, st.Size())
	}
	// Second call should reuse without re-download errors.
	got2, err := ensureCloudflared()
	if err != nil {
		t.Fatal(err)
	}
	if got2 != got {
		t.Fatalf("reuse path mismatch: %q vs %q", got2, got)
	}
}
