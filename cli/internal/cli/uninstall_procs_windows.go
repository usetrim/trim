//go:build windows

package cli

import (
	"fmt"
	"os"
)

// stopOtherTrimProcesses force-stops every other trim.exe (proxy, daemon, stale
// npm locks) so uninstall can delete install trees. Never kills this process.
// Does not kill node.exe shims here: the npm parent may be hosting this uninstall
// (`node bin/trim.js uninstall` → vendor\trim.exe). The post-exit helper stops
// leftover @usetrim/trim node hosts after this PID exits.
func stopOtherTrimProcesses(chrome cliChrome) {
	stopWindowsTrimPS(fmt.Sprintf(
		`$self = %d; Get-CimInstance Win32_Process -Filter "Name = 'trim.exe'" | Where-Object { $_.ProcessId -ne $self } | ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }; Start-Sleep -Milliseconds 1000`,
		os.Getpid(),
	))
	if chrome.UninstallProcessesStopped != "" {
		fmt.Println(chrome.UninstallProcessesStopped)
	}
}
