//go:build windows

package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/usetrim/trim/cli/internal/cliout"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const windowsRunValueName = "TrimProxy"

// configureDetachedStartProc: fully detach the background `trim start` worker from
// the parent console so CMD/PowerShell regain the prompt after ready banners.
// CREATE_NEW_PROCESS_GROUP alone still leaves a console process attached (shell
// looks "stuck"); CREATE_NO_WINDOW + HideWindow is the Windows CLI daemon pattern.
// Parent owns all user-facing stdout; worker stdio is redirected to NUL by the caller.
func configureDetachedStartProc(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_NO_WINDOW,
	}
}

// healWindowsDeepSidecarsForRunningExe copies optimizer.py next to the running
// LocalAppData trim.exe when npm/Scoop still have the release sidecars.
// Best-effort: never fail start/Deep prefs over a heal miss.
func healWindowsDeepSidecarsForRunningExe() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	_ = ensureWindowsDeepSidecars(filepath.Dir(exe))
	if stable, err := windowsStableDaemonExe(); err == nil {
		_ = ensureWindowsDeepSidecars(filepath.Dir(stable))
	}
}

// installWindowsTask registers per-user logon autostart via HKCU Run (no admin).
// Industry practice: CurrentVersion\Run pointing at a stable LocalAppData copy -
// never at node_modules (npm uninstall cannot delete a running vendor\trim.exe).
func installWindowsTask(exe string, chrome cliChrome) error {
	stable, err := syncWindowsStableDaemonExe(exe)
	if err != nil {
		return err
	}
	// Migrate off any leftover npm/node_modules daemon so LocalAppData owns the lock.
	stopWindowsTrimPathContains(`node_modules\@usetrim`)
	stopWindowsTrimPathContains(`/node_modules/@usetrim/`)

	if err := setWindowsRunKey(stable); err != nil {
		return err
	}
	removeLegacyWindowsScheduledTask()

	cmd := exec.Command(stable, "daemon", "run")
	configureDetachedStartProc(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start daemon: %w", err)
	}
	if chrome.DaemonWindowsOk != "" {
		cliout.Success(chrome.DaemonWindowsOk)
	}
	return nil
}

func uninstallWindowsTask(chrome cliChrome) error {
	_ = deleteWindowsRunKey()
	removeLegacyWindowsScheduledTask()
	// Stop stable + legacy npm-locked processes. Full wipe of LocalAppData\Trim is
	// skipped when daemonUninstallPreserveBinary is set (`trim uninstall` defers wipe
	// until after rename; `--keep-data` skips wipe entirely).
	if stable, err := windowsStableDaemonExe(); err == nil {
		stopWindowsTrimExactPath(stable)
		if !daemonUninstallPreserveBinary {
			_ = os.Remove(stable)
			_ = os.RemoveAll(filepath.Dir(filepath.Dir(stable))) // %LOCALAPPDATA%\Trim
		}
	}
	stopWindowsTrimPathContains(`node_modules\@usetrim`)
	stopWindowsTrimPathContains(`/node_modules/@usetrim/`)
	if chrome.DaemonWindowsRemoved != "" {
		cliout.Success(chrome.DaemonWindowsRemoved)
	}
	return nil
}

func printWindowsAutostartStatus(chrome cliChrome) {
	cmdLine, ok := windowsRunKeyCommand()
	if !ok || strings.TrimSpace(cmdLine) == "" {
		if chrome.DaemonWindowsMissing != "" {
			cliout.Muted(chrome.DaemonWindowsMissing)
		}
		return
	}
	if chrome.DaemonWindowsOk != "" {
		cliout.Success(chrome.DaemonWindowsOk)
	}
	cliout.Table([]cliout.Row{{
		Label: strings.TrimSpace(chrome.DaemonWindowsRunLabel),
		Value: cmdLine,
		Kind:  "accent",
	}})
}

// windowsStableDaemonExe is the durable autostart binary location (outside package managers).
func windowsStableDaemonExe() (string, error) {
	base := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
	if base == "" {
		cfg, err := os.UserConfigDir()
		if err != nil || strings.TrimSpace(cfg) == "" {
			return "", fmt.Errorf("windows stable daemon path: LOCALAPPDATA unset")
		}
		base = cfg
	}
	return filepath.Join(base, "Trim", "bin", "trim.exe"), nil
}

// windowsDeepSidecarNames are packed next to trim.exe in release/npm archives.
// Deep Mode (LLMLingua) resolves optimizer.py beside the running binary.
func windowsDeepSidecarNames() []string {
	return []string{"optimizer.py", "requirements-deep.txt", "requirements.txt"}
}

// resolveWindowsDeepSidecarFile finds a flat sidecar file, or the nested
// GoReleaser layout name/name (v0.1.23 shipped optimizer.py/optimizer.py).
func resolveWindowsDeepSidecarFile(dir, name string) string {
	dir = strings.TrimSpace(dir)
	name = strings.TrimSpace(name)
	if dir == "" || name == "" {
		return ""
	}
	flat := filepath.Join(dir, name)
	if st, err := os.Stat(flat); err == nil && !st.IsDir() {
		return flat
	}
	nested := filepath.Join(dir, name, name)
	if st, err := os.Stat(nested); err == nil && !st.IsDir() {
		return nested
	}
	return ""
}

// syncWindowsDeepSidecars copies Deep helpers from srcDir into dstDir when present.
// Missing sources are skipped (caller may recover from another install tree).
func syncWindowsDeepSidecars(srcDir, dstDir string) error {
	srcDir = strings.TrimSpace(srcDir)
	dstDir = strings.TrimSpace(dstDir)
	if srcDir == "" || dstDir == "" {
		return nil
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("windows deep sidecar dir: %w", err)
	}
	for _, name := range windowsDeepSidecarNames() {
		src := resolveWindowsDeepSidecarFile(srcDir, name)
		if src == "" {
			continue
		}
		dst := filepath.Join(dstDir, name)
		if err := copyFileReplace(src, dst); err != nil {
			return fmt.Errorf("windows deep sidecar %s: %w", name, err)
		}
	}
	return nil
}

// windowsDeepSidecarTreeComplete requires both optimizer.py and requirements-deep.txt
// (Bootstrap needs the requirements file beside the script).
func windowsDeepSidecarTreeComplete(dir string) bool {
	return resolveWindowsDeepSidecarFile(dir, "optimizer.py") != "" &&
		resolveWindowsDeepSidecarFile(dir, "requirements-deep.txt") != ""
}

// findWindowsDeepSidecarDir locates a COMPLETE Deep helper tree (optimizer + requirements).
// skipDirs are ignored (so Always-on heal does not self-donate an incomplete Trim\bin).
func findWindowsDeepSidecarDir(prefer string, skipDirs ...string) string {
	seen := map[string]struct{}{}
	for _, s := range skipDirs {
		s = strings.ToLower(filepath.Clean(strings.TrimSpace(s)))
		if s != "" && s != "." {
			seen[s] = struct{}{}
		}
	}
	add := func(dir string) string {
		dir = filepath.Clean(strings.TrimSpace(dir))
		if dir == "" || dir == "." {
			return ""
		}
		key := strings.ToLower(dir)
		if _, ok := seen[key]; ok {
			return ""
		}
		seen[key] = struct{}{}
		if windowsDeepSidecarTreeComplete(dir) {
			return dir
		}
		return ""
	}
	if d := add(prefer); d != "" {
		return d
	}
	// Prefer package installs over Always-on Trim\bin (which may be incomplete).
	candidates := make([]string, 0, 20)
	if ra := strings.TrimSpace(os.Getenv("APPDATA")); ra != "" {
		candidates = append(candidates, filepath.Join(ra, "npm", "node_modules", "@usetrim", "trim", "vendor"))
	}
	if la := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); la != "" {
		candidates = append(candidates, filepath.Join(la, "Programs", "trim"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, "scoop", "apps", "trim", "current"),
			filepath.Join(home, "AppData", "Roaming", "npm", "node_modules", "@usetrim", "trim", "vendor"),
		)
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for i := 0; i < 6; i++ {
			candidates = append(candidates, filepath.Join(dir, "node_modules", "@usetrim", "trim", "vendor"))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	if pathEnv := os.Getenv("PATH"); pathEnv != "" {
		for _, p := range strings.Split(pathEnv, string(os.PathListSeparator)) {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			candidates = append(candidates, p, filepath.Join(p, "node_modules", "@usetrim", "trim", "vendor"))
		}
	}
	// Last resort: another complete Always-on tree (never the skip/dst dir).
	if la := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); la != "" {
		candidates = append(candidates, filepath.Join(la, "Trim", "bin"))
	}
	for _, dir := range candidates {
		if d := add(dir); d != "" {
			return d
		}
	}
	return ""
}

// ensureWindowsDeepSidecars makes sure LocalAppData Trim/bin has optimizer.py +
// requirements when any known install tree still has the release sidecars.
func ensureWindowsDeepSidecars(dstDir string) error {
	dstDir = strings.TrimSpace(dstDir)
	if dstDir == "" {
		return nil
	}
	if windowsDeepSidecarTreeComplete(dstDir) {
		return nil
	}
	// Skip dst itself so an incomplete Trim\bin cannot be chosen as its own donor.
	src := findWindowsDeepSidecarDir("", dstDir)
	if src == "" {
		return nil
	}
	return syncWindowsDeepSidecars(src, dstDir)
}

// syncWindowsStableDaemonExe copies the current CLI into LocalAppData so OS logon
// does not lock npm/Scoop/Homebrew install trees. Also copies Deep Mode sidecars
// (optimizer.py + requirements) so live Deep does not fail-closed with engine error.
func syncWindowsStableDaemonExe(src string) (string, error) {
	src, err := filepath.Abs(src)
	if err != nil {
		return "", err
	}
	dst, err := windowsStableDaemonExe()
	if err != nil {
		return "", err
	}
	dstDir := filepath.Dir(dst)
	srcDir := filepath.Dir(src)
	srcEval, _ := filepath.EvalSymlinks(src)
	dstEval, _ := filepath.EvalSymlinks(dst)
	same := (srcEval != "" && dstEval != "" && strings.EqualFold(srcEval, dstEval)) ||
		strings.EqualFold(src, dst)
	if same {
		// Already the stable binary; still heal missing Deep sidecars from npm/PATH.
		if err := ensureWindowsDeepSidecars(dstDir); err != nil {
			return "", err
		}
		return dst, nil
	}

	stopWindowsTrimExactPath(dst)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return "", fmt.Errorf("windows stable daemon dir: %w", err)
	}

	tmp := dst + ".tmp"
	_ = os.Remove(tmp)
	if err := copyFileReplace(src, tmp); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("windows stable daemon copy: %w", err)
	}
	_ = os.Remove(dst) // may fail if still locked; rename often still works after stop
	if err := os.Rename(tmp, dst); err != nil {
		// Fallback: direct overwrite after stop.
		_ = os.Remove(tmp)
		if err2 := copyFileReplace(src, dst); err2 != nil {
			return "", fmt.Errorf("windows stable daemon install: %w", err2)
		}
	}
	// Prefer sidecars beside the source install (npm vendor / Scoop); fall back to PATH heal.
	if err := syncWindowsDeepSidecars(srcDir, dstDir); err != nil {
		return "", err
	}
	if err := ensureWindowsDeepSidecars(dstDir); err != nil {
		return "", err
	}
	return dst, nil
}

func copyFileReplace(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func setWindowsRunKey(exe string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("windows run key: %w", err)
	}
	defer k.Close()
	val := fmt.Sprintf(`"%s" daemon run`, exe)
	if err := k.SetStringValue(windowsRunValueName, val); err != nil {
		return fmt.Errorf("windows run key set: %w", err)
	}
	return nil
}

func deleteWindowsRunKey() error {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return nil
	}
	defer k.Close()
	_ = k.DeleteValue(windowsRunValueName)
	return nil
}

func windowsRunKeyCommand() (string, bool) {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE)
	if err != nil {
		return "", false
	}
	defer k.Close()
	val, _, err := k.GetStringValue(windowsRunValueName)
	if err != nil {
		return "", false
	}
	return val, true
}

func removeLegacyWindowsScheduledTask() {
	_, _ = exec.Command("schtasks", "/Delete", "/F", "/TN", "TrimProxy").CombinedOutput()
}

func stopWindowsTrimExactPath(exe string) {
	exe = strings.TrimSpace(exe)
	if exe == "" {
		return
	}
	// Never kill this process (install/uninstall may be the same binary path).
	// Prefer stopping OS-logon "daemon run" holders so vendor trees unlock.
	stopWindowsTrimPS(fmt.Sprintf(
		`$exact = '%s'; $self = %d; Get-CimInstance Win32_Process | Where-Object { $_.Name -eq 'trim.exe' -and $_.ProcessId -ne $self } | ForEach-Object { $p = [string]$_.ExecutablePath; $c = [string]$_.CommandLine; if ($p -and ($p -ieq $exact)) { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue } }; Start-Sleep -Milliseconds 600`,
		powershellSingleQuote(exe),
		os.Getpid(),
	))
}

func stopWindowsTrimPathContains(substr string) {
	substr = strings.TrimSpace(substr)
	if substr == "" {
		return
	}
	// Target legacy npm-locked daemons only. Do not kill this CLI (e.g. trim
	// daemon install running from node_modules\@usetrim\...\vendor\trim.exe).
	stopWindowsTrimPS(fmt.Sprintf(
		`$sub = '%s'; $self = %d; Get-CimInstance Win32_Process | Where-Object { $_.Name -eq 'trim.exe' -and $_.ProcessId -ne $self } | ForEach-Object { $p = [string]$_.ExecutablePath; $c = [string]$_.CommandLine; $pathHit = ($p -and ($p -like ('*'+$sub+'*'))) -or ($c -and ($c -like ('*'+$sub+'*'))); $daemon = ($c -and ($c -match '(?i)\bdaemon\s+run\b')); if ($pathHit -and $daemon) { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue } }; Start-Sleep -Milliseconds 400`,
		powershellSingleQuote(substr),
		os.Getpid(),
	))
}

func powershellSingleQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func stopWindowsTrimPS(command string) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", command)
	cmd.Stdout = nil
	cmd.Stderr = nil
	_ = cmd.Run()
	time.Sleep(200 * time.Millisecond)
}
