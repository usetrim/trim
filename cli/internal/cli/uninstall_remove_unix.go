//go:build !windows

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// removeCLIInstall removes the CLI binary and schedules package-manager uninstall
// (and deferred rm if the binary was locked) after this process exits.
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
	channels := detectPkgChannels(exe)
	detected = append(detected, channels...)
	writeUninstallMarkers(exe)

	scheduled := false
	needSchedule := false
	if err := os.Remove(exe); err != nil {
		needSchedule = true
		if err := scheduleUnixPostExitCleanup(exe, channels); err != nil {
			if chrome.UninstallBinaryManualFmt != "" {
				fmt.Printf(chrome.UninstallBinaryManualFmt+"\n", exe)
			}
			// Keep markers while the binary is still on disk (blocks npm self-heal).
			if _, statErr := os.Stat(exe); os.IsNotExist(statErr) {
				clearUninstallMarkers(exe)
			}
		} else {
			scheduled = true
			if chrome.UninstallBinaryScheduledFmt != "" {
				fmt.Printf(chrome.UninstallBinaryScheduledFmt+"\n", exe)
			}
			for _, ch := range channels {
				if chrome.UninstallPkgScheduledFmt != "" {
					fmt.Printf(chrome.UninstallPkgScheduledFmt+"\n", channelPkgLabel(ch))
				}
			}
		}
	} else {
		if chrome.UninstallBinaryRemovedFmt != "" {
			fmt.Printf(chrome.UninstallBinaryRemovedFmt+"\n", exe)
		}
		_ = os.Remove(filepath.Dir(exe))
		if len(channels) > 0 {
			needSchedule = true
			if err := scheduleUnixPostExitCleanup("", channels); err == nil {
				scheduled = true
				for _, ch := range channels {
					if chrome.UninstallPkgScheduledFmt != "" {
						fmt.Printf(chrome.UninstallPkgScheduledFmt+"\n", channelPkgLabel(ch))
					}
				}
			} else {
				// Binary already unlinked but package schedule failed - keep markers
				// while an npm (or other) package tree may still self-heal.
				if !npmGlobalHasUsetrimTrim() {
					clearUninstallMarkers(exe)
				}
			}
		}
	}
	if !needSchedule {
		clearUninstallMarkers(exe)
	}
	_ = scheduled
	return handled, detected
}

func scheduleUnixPostExitCleanup(asidePath string, channels []installChannel) error {
	parts := []string{
		"pid=" + strconv.Itoa(os.Getpid()),
		`while kill -0 "$pid" 2>/dev/null; do sleep 0.2; done`,
		"sleep 1",
	}
	if asidePath != "" {
		parts = append(parts, "rm -f "+shellQuote(asidePath))
		parts = append(parts, "rmdir "+shellQuote(filepath.Dir(asidePath))+" 2>/dev/null || true")
	}
	for _, ch := range channels {
		switch ch {
		case channelNPM:
			parts = append(parts, "command -v npm >/dev/null 2>&1 && npm uninstall -g @usetrim/trim || true")
		case channelBrew:
			parts = append(parts, "command -v brew >/dev/null 2>&1 && brew uninstall trim || true")
		case channelScoop:
			parts = append(parts, "command -v scoop >/dev/null 2>&1 && scoop uninstall trim || true")
		case channelWinget:
			parts = append(parts, "command -v winget >/dev/null 2>&1 && winget uninstall --id Trim.CLI -e --silent || true")
		}
	}
	// Legacy TEMP only; keep package-local .trim-uninstalling if npm uninstall failed.
	parts = append(parts, `rm -f "${TMPDIR:-/tmp}/trim-uninstalling" 2>/dev/null || true`)
	script := strings.Join(parts, "; ")
	cmd := exec.Command("sh", "-c", script)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd.Start()
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
