//go:build windows

package cli

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/usetrim/trim/cli/internal/cliout"
	"golang.org/x/sys/windows/registry"
)

const windowsRunValueName = "TrimProxy"

// installWindowsTask registers per-user logon autostart via HKCU Run (no admin).
// Industry practice (VS Code, browsers, Slack): CurrentVersion\Run - not schtasks ONLOGON.
// Legacy TrimProxy scheduled tasks are best-effort removed so upgrades migrate cleanly.
func installWindowsTask(exe string, chrome cliChrome) error {
	if err := setWindowsRunKey(exe); err != nil {
		return err
	}
	removeLegacyWindowsScheduledTask()
	// Start now (parity with old schtasks /Run and macOS/Linux enable --now).
	cmd := exec.Command(exe, "daemon", "run")
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

func setWindowsRunKey(exe string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("windows run key: %w", err)
	}
	defer k.Close()
	// Quote path so spaces in Program Files work.
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
