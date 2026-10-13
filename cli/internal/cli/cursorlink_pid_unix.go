//go:build !windows

package cli

import "syscall"

func unixPIDAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil
}
