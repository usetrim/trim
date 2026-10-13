package cli

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const cloudflaredDownloadsDocs = "https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/"

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
	fmt.Fprintln(os.Stderr, "Cursor Link: downloading cloudflared to "+dest)
	client := &http.Client{Timeout: 3 * time.Minute}
	res, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("download cloudflared: %w\nInstall manually: %s\nOr set TRIM_CLOUDFLARED to a cloudflared binary", err, cloudflaredInstallHint())
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download cloudflared: HTTP %s\nInstall manually: %s\nOr set TRIM_CLOUDFLARED", res.Status, cloudflaredInstallHint())
	}
	tmp := dest + ".tmp"
	if strings.HasSuffix(url, ".tgz") {
		if err := extractCloudflaredFromTGZ(res.Body, tmp); err != nil {
			_ = os.Remove(tmp)
			return "", fmt.Errorf("extract cloudflared: %w\nRun trim cursor-link enable && trim start again, or brew install cloudflared if Homebrew exists, or set TRIM_CLOUDFLARED", err)
		}
	} else {
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
	}
	_ = os.Chmod(tmp, 0o755)
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return dest, nil
}

func extractCloudflaredFromTGZ(r io.Reader, dest string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		base := filepath.Base(hdr.Name)
		if base != "cloudflared" {
			continue
		}
		f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, tr); err != nil {
			_ = f.Close()
			return err
		}
		return f.Close()
	}
	return fmt.Errorf("cloudflared binary not found inside archive")
}

func cloudflaredDownloadURL() (string, error) {
	// Cloudflare GitHub release asset names (best-effort auto-install into ~/.trim/bin).
	const base = "https://github.com/cloudflare/cloudflared/releases/latest/download/"
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "windows/amd64", "windows/arm64":
		// Cloudflare publishes windows-amd64 only (no native windows-arm64 asset).
		// On Windows on ARM, amd64 runs under emulation.
		return base + "cloudflared-windows-amd64.exe", nil
	case "windows/386":
		return base + "cloudflared-windows-386.exe", nil
	case "darwin/amd64":
		return base + "cloudflared-darwin-amd64.tgz", nil
	case "darwin/arm64":
		return base + "cloudflared-darwin-arm64.tgz", nil
	case "linux/amd64":
		return base + "cloudflared-linux-amd64", nil
	case "linux/arm64":
		return base + "cloudflared-linux-arm64", nil
	case "linux/386":
		return base + "cloudflared-linux-386", nil
	case "linux/arm":
		return base + "cloudflared-linux-arm", nil
	default:
		return "", fmt.Errorf("unsupported platform %s/%s for auto-download; %s; or set TRIM_CLOUDFLARED", runtime.GOOS, runtime.GOARCH, cloudflaredInstallHint())
	}
}

func cloudflaredInstallHint() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows: run trim cursor-link enable && trim start (auto-download), or download cloudflared-windows-amd64.exe into %USERPROFILE%\\.trim\\bin, or winget install --id Cloudflare.cloudflared if winget exists"
	case "darwin":
		return "macOS: run trim cursor-link enable && trim start (auto-download), or brew install cloudflared if Homebrew exists, or set TRIM_CLOUDFLARED"
	case "linux":
		return "Linux: run trim cursor-link enable && trim start (auto-download), or install from " + cloudflaredDownloadsDocs + ", or set TRIM_CLOUDFLARED"
	default:
		return "install cloudflared from " + cloudflaredDownloadsDocs + " or set TRIM_CLOUDFLARED"
	}
}

func findOrInstallCloudflared() (string, error) {
	p, err := findCloudflared()
	if err == nil {
		return p, nil
	}
	if _, uerr := cloudflaredDownloadURL(); uerr != nil {
		return "", fmt.Errorf("%v\n%s", err, cloudflaredInstallHint())
	}
	dest, derr := ensureCloudflared()
	if derr != nil {
		return "", fmt.Errorf("%v\n%s", derr, cloudflaredInstallHint())
	}
	return dest, nil
}
