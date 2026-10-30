//go:build windows

package deepopt

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// configureHiddenChildProc hides console windows for Deep optimizer / Python
// child processes so Live Deep and warmup stay silent in the background.
func configureHiddenChildProc(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_NO_WINDOW,
	}
}
