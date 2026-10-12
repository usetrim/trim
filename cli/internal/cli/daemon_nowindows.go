//go:build !windows

package cli

import (
	"fmt"
	"os/exec"
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
	_ = chrome
	return fmt.Errorf("windows autostart is only available on Windows")
}

func uninstallWindowsTask(chrome cliChrome) error {
	_ = chrome
	return fmt.Errorf("windows autostart is only available on Windows")
}

func printWindowsAutostartStatus(chrome cliChrome) {
	_ = chrome
}
