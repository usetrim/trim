//go:build !windows

package deepopt

import "os/exec"

func configureHiddenChildProc(cmd *exec.Cmd) {
	_ = cmd
}
