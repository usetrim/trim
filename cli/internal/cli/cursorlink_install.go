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
	chrome := loadCLIChrome()
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
	if fmtStr := strings.TrimSpace(chrome.CursorLinkDownloadingFmt); fmtStr != "" {
		fmt.Fprintf(os.Stderr, fmtStr+"\n", dest)
	}
	client := &http.Client{Timeout: 3 * time.Minute}
	res, err := client.Get(url)
	if err != nil {
		fmtStr := strings.TrimSpace(chrome.CursorLinkDownloadErrFmt)
		if fmtStr == "" {
			return "", chrome.fail("")
		}
		return "", fmt.Errorf(fmtStr, err, cloudflaredInstallHint(chrome))
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		fmtStr := strings.TrimSpace(chrome.CursorLinkDownloadHTTPFmt)
		if fmtStr == "" {
			return "", chrome.fail("")
		}
		return "", fmt.Errorf(fmtStr, res.Status, cloudflaredInstallHint(chrome))
	}
	tmp := dest + ".tmp"
	if strings.HasSuffix(url, ".tgz") {
		if err := extractCloudflaredFromTGZ(res.Body, tmp); err != nil {
			_ = os.Remove(tmp)
			fmtStr := strings.TrimSpace(chrome.CursorLinkExtractErrFmt)
			if fmtStr == "" {
				return "", chrome.fail("")
			}
			return "", fmt.Errorf(fmtStr, err)
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
	chrome := loadCLIChrome()
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
	msg := strings.TrimSpace(chrome.CursorLinkArchiveBinMissing)
	if msg == "" {
		return chrome.fail("")
	}
	return fmt.Errorf("%s", msg)
}

func cloudflaredDownloadURL() (string, error) {
	chrome := loadCLIChrome()
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
		fmtStr := strings.TrimSpace(chrome.CursorLinkUnsupportedPlatformFmt)
		if fmtStr == "" {
			return "", chrome.fail("")
		}
		return "", fmt.Errorf(fmtStr, runtime.GOOS, runtime.GOARCH, cloudflaredInstallHint(chrome))
	}
}

func cloudflaredInstallHint(chrome cliChrome) string {
	switch runtime.GOOS {
	case "windows":
		return strings.TrimSpace(chrome.CursorLinkHintWindows)
	case "darwin":
		return strings.TrimSpace(chrome.CursorLinkHintDarwin)
	case "linux":
		return strings.TrimSpace(chrome.CursorLinkHintLinux)
	default:
		return strings.TrimSpace(chrome.CursorLinkHintOther)
	}
}

func findOrInstallCloudflared() (string, error) {
	chrome := loadCLIChrome()
	p, err := findCloudflared()
	if err == nil {
		return p, nil
	}
	hint := cloudflaredInstallHint(chrome)
	combine := strings.TrimSpace(chrome.CursorLinkCombineErrHintFmt)
	if _, uerr := cloudflaredDownloadURL(); uerr != nil {
		if combine == "" || hint == "" {
			return "", chrome.fail("")
		}
		return "", fmt.Errorf(combine, err, hint)
	}
	dest, derr := ensureCloudflared()
	if derr != nil {
		if combine == "" || hint == "" {
			return "", chrome.fail("")
		}
		return "", fmt.Errorf(combine, derr, hint)
	}
	return dest, nil
}
