package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ensureCloudflared returns a usable cloudflared path, downloading into ~/.trim/bin when missing.
func ensureCloudflared() (string, error) {
	if p, err := findCloudflared(); err == nil {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".trim", "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := "cloudflared"
	if runtime.GOOS == "windows" {
		name = "cloudflared.exe"
	}
	dest := filepath.Join(dir, name)
	url, err := cloudflaredDownloadURL()
	if err != nil {
		return "", err
	}
	clioutInfo := func(s string) {
		// Avoid import cycle noise: print plain when cliout not wanted in tests.
		fmt.Fprintln(os.Stderr, s)
	}
	clioutInfo("Cursor Link: downloading cloudflared to " + dest)
	client := &http.Client{Timeout: 3 * time.Minute}
	res, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("download cloudflared: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download cloudflared: HTTP %s", res.Status)
	}
	tmp := dest + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, res.Body); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return dest, nil
}

func cloudflaredDownloadURL() (string, error) {
	// Pinned-ish Cloudflare GitHub release asset names (best-effort auto-install).
	const base = "https://github.com/cloudflare/cloudflared/releases/latest/download/"
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "windows/amd64":
		return base + "cloudflared-windows-amd64.exe", nil
	case "windows/arm64":
		return base + "cloudflared-windows-arm64.exe", nil
	case "darwin/amd64":
		return base + "cloudflared-darwin-amd64.tgz", nil
	case "darwin/arm64":
		return base + "cloudflared-darwin-arm64.tgz", nil
	case "linux/amd64":
		return base + "cloudflared-linux-amd64", nil
	case "linux/arm64":
		return base + "cloudflared-linux-arm64", nil
	default:
		return "", fmt.Errorf("unsupported platform %s/%s for auto-download; install cloudflared or set TRIM_CLOUDFLARED", runtime.GOOS, runtime.GOARCH)
	}
}

func findOrInstallCloudflared() (string, error) {
	p, err := findCloudflared()
	if err == nil {
		return p, nil
	}
	// Auto-download only for raw binary assets (not .tgz) to keep install simple.
	url, uerr := cloudflaredDownloadURL()
	if uerr != nil {
		return "", err
	}
	if strings.HasSuffix(url, ".tgz") {
		return "", fmt.Errorf("%v (on macOS prefer: brew install cloudflared)", err)
	}
	return ensureCloudflared()
}
