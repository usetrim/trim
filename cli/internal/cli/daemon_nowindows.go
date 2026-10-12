//go:build !windows

package cli

import "fmt"

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
