//go:build !windows

package cli

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

// configureDetachedStartProc: new session so the worker survives parent exit / Ctrl+C.
func configureDetachedStartProc(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

func installWindowsTask(exe string, chrome cliChrome) error {
	_ = exe
	msg := strings.TrimSpace(chrome.DaemonWindowsOnly)
	if msg == "" {
		return chrome.fail("")
	}
	return fmt.Errorf("%s", msg)
}

func uninstallWindowsTask(chrome cliChrome) error {
	msg := strings.TrimSpace(chrome.DaemonWindowsOnly)
	if msg == "" {
		return chrome.fail("")
	}
	return fmt.Errorf("%s", msg)
}

func printWindowsAutostartStatus(chrome cliChrome) {
	_ = chrome
}

func healWindowsDeepSidecarsForRunningExe() {}
