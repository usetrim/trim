//go:build windows

package cli

func unixPIDAlive(pid int) bool {
	return pid > 0
}
