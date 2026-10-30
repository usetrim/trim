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
	// Stop stable + legacy npm-locked processes, then best-effort remove stable binary.
	if stable, err := windowsStableDaemonExe(); err == nil {
		stopWindowsTrimExactPath(stable)
		_ = os.Remove(stable)
		_ = os.Remove(filepath.Dir(stable))               // bin/
		_ = os.Remove(filepath.Dir(filepath.Dir(stable))) // Trim/ if empty
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

// syncWindowsStableDaemonExe copies the current CLI into LocalAppData so OS logon
// does not lock npm/Scoop/Homebrew install trees.
func syncWindowsStableDaemonExe(src string) (string, error) {
	src, err := filepath.Abs(src)
	if err != nil {
		return "", err
	}
	dst, err := windowsStableDaemonExe()
	if err != nil {
		return "", err
	}
	srcEval, _ := filepath.EvalSymlinks(src)
	dstEval, _ := filepath.EvalSymlinks(dst)
	if srcEval != "" && dstEval != "" && strings.EqualFold(srcEval, dstEval) {
		return dst, nil
	}
	if strings.EqualFold(src, dst) {
		return dst, nil
	}

	stopWindowsTrimExactPath(dst)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
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
