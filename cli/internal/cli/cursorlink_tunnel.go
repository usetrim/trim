package cli

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	tryCloudflareURLRe = regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)
	cursorLinkMu       sync.Mutex
)

func findCloudflared() (string, error) {
	if p := strings.TrimSpace(os.Getenv("TRIM_CLOUDFLARED")); p != "" {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
		return "", fmt.Errorf("TRIM_CLOUDFLARED points to a missing binary: %s", p)
	}
	if p, err := exec.LookPath("cloudflared"); err == nil {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cloudflared not found on PATH (install Cloudflare cloudflared, or set TRIM_CLOUDFLARED)")
	}
	name := "cloudflared"
	if runtime.GOOS == "windows" {
		name = "cloudflared.exe"
	}
	candidates := []string{
		filepath.Join(home, ".trim", "bin", name),
		filepath.Join(home, ".trim", name),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, nil
		}
	}
	return "", fmt.Errorf("cloudflared not found on PATH (install from https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/ or set TRIM_CLOUDFLARED)")
}

func parseTryCloudflareURL(line string) string {
	return tryCloudflareURLRe.FindString(line)
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	// Prefer tunnel /health when we have a public origin; PID alone is best-effort.
	// Unix: signal 0. Windows: assume PID still set until health fails (restart path clears it).
	if runtime.GOOS != "windows" {
		return unixPIDAlive(pid)
	}
	return true
}

func cursorLinkTunnelHealthy(origin string) bool {
	origin = strings.TrimRight(strings.TrimSpace(origin), "/")
	if origin == "" {
		return false
	}
	client := &http.Client{Timeout: 3 * time.Second}
	res, err := client.Get(origin + "/health")
	if err != nil {
		return false
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 2048))
	return res.StatusCode == http.StatusOK
}

// ensureCursorLink starts (or reuses) a Cloudflare quick tunnel to local Trim when enabled.
// Public hostname only - compression still runs on this PC.
func ensureCursorLink(localPort string) (string, error) {
	cursorLinkMu.Lock()
	defer cursorLinkMu.Unlock()

	localPort = strings.TrimSpace(localPort)
	if localPort == "" {
		return "", fmt.Errorf("local Trim port is required for Cursor Link")
	}

	st, err := loadCursorLinkState()
	if err != nil {
		return "", err
	}
	if !st.Enabled {
		return "", nil
	}
	if err := ensureCursorLinkAccessToken(&st); err != nil {
		return "", err
	}
	if err := saveCursorLinkState(st); err != nil {
		return "", err
	}

	if st.PID > 0 && strings.TrimSpace(st.PublicOrigin) != "" && processAlive(st.PID) {
		if cursorLinkTunnelHealthy(st.PublicOrigin) || strings.TrimSpace(st.PublicBaseURL) != "" {
			// Refresh tokenized Base URL in case token rotated while tunnel stayed up.
			st.PublicBaseURL = cursorLinkPublicBaseURL(st)
			_ = saveCursorLinkState(st)
			return st.PublicBaseURL, nil
		}
	}
	if st.PID > 0 {
		_ = killPID(st.PID)
		st.PID = 0
		st.PublicOrigin = ""
		st.PublicBaseURL = ""
	}

	bin, err := findOrInstallCloudflared()
	if err != nil {
		return "", err
	}

	target := fmt.Sprintf("http://127.0.0.1:%s", localPort)
	cmd := exec.Command(bin, "tunnel", "--url", target, "--no-autoupdate")
	cmd.Env = os.Environ()
	configureDetachedStartProc(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start cloudflared: %w", err)
	}

	originCh := make(chan string, 1)
	scan := func(r io.Reader) {
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			if u := parseTryCloudflareURL(sc.Text()); u != "" {
				select {
				case originCh <- u:
				default:
				}
				return
			}
		}
	}
	go scan(stdout)
	go scan(stderr)

	var origin string
	select {
	case origin = <-originCh:
	case <-time.After(45 * time.Second):
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		return "", fmt.Errorf("cloudflared did not publish a trycloudflare.com URL in time")
	}

	origin = strings.TrimRight(origin, "/")
	if err := ensureCursorLinkAccessToken(&st); err != nil {
		_ = cmd.Process.Kill()
		return "", err
	}
	st.CloudflaredBin = bin
	st.LocalPort = localPort
	st.PID = cmd.Process.Pid
	st.PublicOrigin = origin
	st.PublicBaseURL = origin + "/cl/" + st.AccessToken + "/v1"
	if err := saveCursorLinkState(st); err != nil {
		_ = cmd.Process.Kill()
		return "", err
	}

	// Detach wait so the process is not reaped as a zombie on Unix.
	go func() { _, _ = cmd.Process.Wait() }()

	return st.PublicBaseURL, nil
}

func stopCursorLink() error {
	cursorLinkMu.Lock()
	defer cursorLinkMu.Unlock()

	st, err := loadCursorLinkState()
	if err != nil {
		return err
	}
	if st.PID > 0 {
		_ = killPID(st.PID)
	}
	st.PID = 0
	st.PublicOrigin = ""
	st.PublicBaseURL = ""
	st.LocalPort = ""
	st.CloudflaredBin = ""
	// Keep Enabled so the next trim start brings the tunnel back.
	return saveCursorLinkState(st)
}

func killPID(pid int) error {
	if pid <= 0 {
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil
	}
	_ = proc.Kill()
	_, _ = proc.Wait()
	return nil
}

func setCursorLinkEnabled(on bool) error {
	st, err := loadCursorLinkState()
	if err != nil {
		return err
	}
	st.Enabled = on
	if on {
		if err := ensureCursorLinkAccessToken(&st); err != nil {
			return err
		}
	} else {
		if st.PID > 0 {
			_ = killPID(st.PID)
		}
		st.PID = 0
		st.PublicOrigin = ""
		st.PublicBaseURL = ""
		st.LocalPort = ""
		// Keep AccessToken so re-enable can rotate explicitly; clear on disable for revoke.
		st.AccessToken = ""
	}
	return saveCursorLinkState(st)
}

func cursorLinkLocalPortFallback(cfgPort string) string {
	st, _ := loadCursorLinkState()
	if p := strings.TrimSpace(st.LocalPort); p != "" {
		return p
	}
	if p := strings.TrimSpace(cfgPort); p != "" {
		return p
	}
	return "8888"
}

func formatPID(pid int) string {
	if pid <= 0 {
		return ""
	}
	return strconv.Itoa(pid)
}
