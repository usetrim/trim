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
		chrome := loadCLIChrome()
		fmtStr := strings.TrimSpace(chrome.CursorLinkEnvMissingFmt)
		if fmtStr == "" {
			return "", chrome.fail("")
		}
		return "", fmt.Errorf(fmtStr, p)
	}
	if p, err := exec.LookPath("cloudflared"); err == nil {
		return p, nil
	}
	home, err := os.UserHomeDir()
	name := "cloudflared"
	if runtime.GOOS == "windows" {
		name = "cloudflared.exe"
	}
	var candidates []string
	if err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".trim", "bin", name),
			filepath.Join(home, ".trim", name),
		)
	}
	// Official Windows installer paths (often not on PATH for the same shell).
	if runtime.GOOS == "windows" {
		if pf86 := strings.TrimSpace(os.Getenv("ProgramFiles(x86)")); pf86 != "" {
			candidates = append(candidates, filepath.Join(pf86, "cloudflared", "cloudflared.exe"))
		}
		if pf := strings.TrimSpace(os.Getenv("ProgramFiles")); pf != "" {
			candidates = append(candidates, filepath.Join(pf, "cloudflared", "cloudflared.exe"))
		}
		if local := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); local != "" {
			candidates = append(candidates, filepath.Join(local, "cloudflared", "cloudflared.exe"))
		}
	}
	// Homebrew defaults when PATH is minimal (CI / non-interactive shells).
	if runtime.GOOS == "darwin" {
		candidates = append(candidates,
			"/opt/homebrew/bin/cloudflared",
			"/usr/local/bin/cloudflared",
		)
	}
	for _, c := range candidates {
		if strings.TrimSpace(c) == "" {
			continue
		}
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, nil
		}
	}
	chrome := loadCLIChrome()
	fmtStr := strings.TrimSpace(chrome.CursorLinkNotFoundFmt)
	if fmtStr == "" {
		return "", chrome.fail("")
	}
	return "", fmt.Errorf(fmtStr, cloudflaredDownloadsDocs)
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
		chrome := loadCLIChrome()
		msg := strings.TrimSpace(chrome.CursorLinkPortRequired)
		if msg == "" {
			return "", chrome.fail("")
		}
		return "", fmt.Errorf("%s", msg)
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
		chrome := loadCLIChrome()
		fmtStr := strings.TrimSpace(chrome.CursorLinkStartCloudflaredFmt)
		if fmtStr == "" {
			return "", chrome.fail("")
		}
		return "", fmt.Errorf(fmtStr, err)
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
		chrome := loadCLIChrome()
		msg := strings.TrimSpace(chrome.CursorLinkTunnelTimeout)
		if msg == "" {
			return "", chrome.fail("")
		}
		return "", fmt.Errorf("%s", msg)
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
