//go:build windows

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// removeCLIInstall stops leftover locks, removes/schedules the running binary,
// clears LocalAppData Trim + Programs\trim trees, and schedules package-manager
// uninstall for every detected channel.
//
// Windows cannot DeleteFile a mapped running exe. Industry practice:
// rename aside → spawn detached post-exit helper → helper waits for our PID,
// force-stops leftover Trim, deletes the aside + install trees, then runs
// npm/scoop/winget/brew uninstall. Always also register MoveFileEx
// DELAY_UNTIL_REBOOT on the aside so a killed helper cannot leave trim.exe.
func removeCLIInstall(chrome cliChrome) (handled map[string]bool, detected []installChannel) {
	handled = map[string]bool{}
	exe, err := os.Executable()
	if err != nil {
		return handled, detected
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return handled, detected
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil && resolved != "" {
		exe = resolved
	}
	installDir := filepath.Dir(exe)
	channels := detectPkgChannels(exe)
	detected = append(detected, channels...)
	writeUninstallMarkers(exe)

	// Stop stable daemon holders first (do not wipe trees yet - we may be running from them).
	if stable, err := windowsStableDaemonExe(); err == nil {
		stopWindowsTrimExactPath(stable)
	}

	removedNow := false
	aside := ""
	if err := os.Remove(exe); err == nil {
		removedNow = true
		if chrome.UninstallBinaryRemovedFmt != "" {
			fmt.Printf(chrome.UninstallBinaryRemovedFmt+"\n", exe)
		}
	} else {
		aside = exe + ".uninstalling"
		_ = os.Remove(aside)
		if err := os.Rename(exe, aside); err != nil {
			aside = exe
		}
	}

	// After this process image is unlinked/renamed, wipe durable Windows install roots.
	wipeWindowsKnownTrimRoots()

	needSchedule := !removedNow || len(channels) > 0
	scheduled := false
	if needSchedule {
		primary := channelStandalone
		if len(channels) > 0 {
			primary = channels[0]
		}
		if err := scheduleWindowsPostExitCleanup(aside, installDir, channels, primary); err != nil {
			if !removedNow && chrome.UninstallBinaryManualFmt != "" {
				fmt.Printf(chrome.UninstallBinaryManualFmt+"\n", exe)
			}
			// Keep markers whenever package channels still need removal (blocks npm
			// self-heal). Only clear when there is no remaining pkg work AND the
			// locked image/aside is already gone. Also keep if a global npm tree
			// exists but was somehow missed by channel detection.
			if len(channels) == 0 && !npmGlobalHasUsetrimTrim() {
				if removedNow {
					clearUninstallMarkers(exe)
				} else if aside != "" {
					if _, statErr := os.Stat(aside); os.IsNotExist(statErr) {
						clearUninstallMarkers(exe)
					}
				} else {
					clearUninstallMarkers(exe)
				}
			}
		} else {
			scheduled = true
			if !removedNow {
				target := aside
				if target == "" {
					target = exe
				}
				if chrome.UninstallBinaryScheduledFmt != "" {
					fmt.Printf(chrome.UninstallBinaryScheduledFmt+"\n", target)
				}
			}
			// Do not mark channels handled here: schedule is optimistic. Keep
			// fallback next-steps visible if the helper/package uninstall fails later.
			for _, ch := range channels {
				if chrome.UninstallPkgScheduledFmt != "" {
					fmt.Printf(chrome.UninstallPkgScheduledFmt+"\n", channelPkgLabel(ch))
				}
			}
		}
	}
	// Belt-and-suspenders: even when the helper started, register reboot delete for
	// any still-present locked image so a killed helper cannot leave vendor\trim.exe.
	if !removedNow {
		if aside != "" {
			_ = scheduleWindowsDeleteOnReboot(aside)
		}
		if aside != exe {
			_ = scheduleWindowsDeleteOnReboot(exe)
		}
	}
	// Clear markers only when no post-exit work was needed. If schedule failed while
	// the aside/image remains, the err branch above keeps markers on purpose - a
	// blanket `if !scheduled` clear previously undid that and let npm self-heal
	// vendor\trim.exe. When scheduled=true, markers stay until npm removes the package.
	if !needSchedule {
		clearUninstallMarkers(exe)
	}
	_ = scheduled

	if removeWindowsUserPathEntry(installDir) && chrome.UninstallPathRemoved != "" {
		fmt.Println(chrome.UninstallPathRemoved)
	}
	// Strip durable Windows install dirs from user PATH when present.
	if local := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); local != "" {
		_ = removeWindowsUserPathEntry(filepath.Join(local, "Programs", "trim"))
		_ = removeWindowsUserPathEntry(filepath.Join(local, "Trim", "bin"))
	}
	_ = os.Remove(installDir)
	return handled, detected
}

func wipeWindowsKnownTrimRoots() {
	local := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
	if local == "" {
		return
	}
	_ = os.RemoveAll(filepath.Join(local, "Trim"))
	_ = os.RemoveAll(filepath.Join(local, "Programs", "trim"))
}

// scheduleWindowsDeleteOnReboot registers a pending file delete (PendingFileRenameOperations).
func scheduleWindowsDeleteOnReboot(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("empty path")
	}
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(p, nil, windows.MOVEFILE_DELAY_UNTIL_REBOOT)
}

func scheduleWindowsPostExitCleanup(asidePath, installDir string, channels []installChannel, primary installChannel) error {
	return scheduleWindowsPostExitCleanupFor(os.Getpid(), asidePath, installDir, channels, primary)
}

func scheduleWindowsPostExitCleanupFor(waitPID int, asidePath, installDir string, channels []installChannel, primary installChannel) error {
	if waitPID <= 0 {
		return fmt.Errorf("invalid wait pid")
	}
	if _, err := exec.LookPath("powershell.exe"); err != nil {
		return fmt.Errorf("powershell.exe not found: %w", err)
	}

	channelCSV := make([]string, 0, len(channels))
	for _, ch := range channels {
		channelCSV = append(channelCSV, string(ch))
	}
	if len(channelCSV) == 0 {
		channelCSV = append(channelCSV, string(primary))
	}

	// Write a .ps1 file and run with -File. Large inline -Command scripts are
	// fragile (quoting / CreateProcess arg limits) and were a root cause of
	// "binary stayed behind" when the helper never actually ran.
	script := strings.TrimSpace(`
$ErrorActionPreference = 'SilentlyContinue'
$selfPid = [int]$env:TRIM_UNINSTALL_PID
$aside = [string]$env:TRIM_UNINSTALL_ASIDE
$installDir = [string]$env:TRIM_UNINSTALL_INSTALL_DIR
$channels = @(($env:TRIM_UNINSTALL_CHANNELS -split ',') | Where-Object { $_ })
$log = Join-Path $env:TEMP 'trim-uninstall-cleanup.log'
function Log([string]$m) { Add-Content -LiteralPath $log -Value ("[{0}] {1}" -f (Get-Date -Format o), $m) -ErrorAction SilentlyContinue }
function Remove-Retry([string]$path, [int]$tries = 40) {
  if (-not $path) { return }
  for ($i = 0; $i -lt $tries; $i++) {
    if (-not (Test-Path -LiteralPath $path)) { return }
    Remove-Item -LiteralPath $path -Recurse -Force -ErrorAction SilentlyContinue
    if (-not (Test-Path -LiteralPath $path)) { return }
    Start-Sleep -Milliseconds 400
  }
}
Log "cleanup start pid=$selfPid channels=$($channels -join ',')"
for ($i = 0; $i -lt 120; $i++) {
  if (-not (Get-Process -Id $selfPid -ErrorAction SilentlyContinue)) { break }
  Start-Sleep -Milliseconds 250
}
Start-Sleep -Seconds 2
Get-CimInstance Win32_Process -Filter "Name = 'trim.exe'" | ForEach-Object {
  Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue
}
Start-Sleep -Milliseconds 800
if ($channels -contains 'npm') {
  Get-CimInstance Win32_Process -Filter "Name = 'node.exe'" | ForEach-Object {
    $c = [string]$_.CommandLine
    if ($c -and (
      ($c -match '[\\/]@usetrim[\\/]trim([\\/]|$)') -or
      ($c -match '(?i)[\\/]trim\.js\b') -or
      ($c -match '(?i)node_modules[\\/]@usetrim[\\/]trim')
    )) {
      Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue
    }
  }
  Start-Sleep -Milliseconds 800
}
# Delete the renamed running image first (unlocks package trees).
Remove-Retry $aside
if ($aside) { Log "removed aside=$aside exists=$(Test-Path -LiteralPath $aside)" }
# Package managers next (before wiping installDir) so scoop/winget still see their tree.
foreach ($channel in $channels) {
  switch ($channel) {
    'npm' {
      $npm = Get-Command npm -ErrorAction SilentlyContinue
      if ($npm) {
        & npm uninstall -g '@usetrim/trim' *>> $log
        Log "npm uninstall exit=$LASTEXITCODE"
      } else { Log "npm missing" }
    }
    'scoop' {
      $scoop = Get-Command scoop -ErrorAction SilentlyContinue
      if ($scoop) {
        & scoop uninstall trim *>> $log
        Log "scoop uninstall exit=$LASTEXITCODE"
      }
    }
    'winget' {
      $winget = Get-Command winget -ErrorAction SilentlyContinue
      if ($winget) {
        & winget uninstall --id Trim.CLI -e --silent *>> $log
        Log "winget uninstall exit=$LASTEXITCODE"
      }
    }
    'brew' {
      $brew = Get-Command brew -ErrorAction SilentlyContinue
      if ($brew) {
        & brew uninstall trim *>> $log
        Log "brew uninstall exit=$LASTEXITCODE"
      }
    }
  }
}
Remove-Retry $installDir
if ($installDir) { Log "removed installDir=$installDir exists=$(Test-Path -LiteralPath $installDir)" }
$local = [string]$env:LOCALAPPDATA
if ($local) {
  Remove-Retry (Join-Path $local 'Trim')
  Remove-Retry (Join-Path $local 'Programs\trim')
}
# Legacy TEMP marker only (older builds). Keep package-local .trim-uninstalling if
# npm uninstall failed so the shim cannot self-heal the binary back.
Remove-Item -LiteralPath (Join-Path $env:TEMP 'trim-uninstalling') -Force -ErrorAction SilentlyContinue
if ($env:TRIM_UNINSTALL_SCRIPT -and (Test-Path -LiteralPath $env:TRIM_UNINSTALL_SCRIPT)) {
  Remove-Item -LiteralPath $env:TRIM_UNINSTALL_SCRIPT -Force -ErrorAction SilentlyContinue
}
Log "cleanup done"
`) + "\n"

	scriptPath := filepath.Join(os.TempDir(), fmt.Sprintf("trim-uninstall-cleanup-%d.ps1", waitPID))
	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		return err
	}

	cmd := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-File", scriptPath)
	cmd.Env = append(os.Environ(),
		"TRIM_UNINSTALL_PID="+strconv.Itoa(waitPID),
		"TRIM_UNINSTALL_ASIDE="+asidePath,
		"TRIM_UNINSTALL_INSTALL_DIR="+installDir,
		"TRIM_UNINSTALL_CHANNELS="+strings.Join(channelCSV, ","),
		"TRIM_UNINSTALL_SCRIPT="+scriptPath,
	)
	cmd.Stdout = nil
	cmd.Stderr = nil
	// Do NOT combine DETACHED_PROCESS with CREATE_NO_WINDOW (MSDN forbids it).
	// Same detach pattern as daemon/start workers that already work in production.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_NO_WINDOW,
	}
	if err := cmd.Start(); err != nil {
		_ = os.Remove(scriptPath)
		return err
	}
	helperPID := cmd.Process.Pid
	// Release so the CLI exit does not keep a waitable handle on the helper.
	_ = cmd.Process.Release()

	// Confirm the helper is alive briefly; if it died instantly while we are still
	// running, fail so callers surface manual/reboot paths instead of a false "scheduled".
	time.Sleep(300 * time.Millisecond)
	if !windowsProcessAlive(helperPID) && windowsProcessAlive(waitPID) {
		_ = os.Remove(scriptPath)
		return fmt.Errorf("post-exit cleanup helper exited immediately")
	}
	return nil
}

func windowsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	const stillActive = 259 // STILL_ACTIVE
	return code == stillActive
}
