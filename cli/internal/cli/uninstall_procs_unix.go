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

// stopOtherTrimProcesses stops other Trim CLI processes so package-manager
// uninstall can unlock binaries. Never signals this PID.
func stopOtherTrimProcesses(chrome cliChrome) {
	self := os.Getpid()
	out, err := exec.Command("ps", "-ax", "-o", "pid=,command=").Output()
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(out), "\n") {
		pid, cmdLine, ok := parsePSLine(line)
		if !ok || pid <= 1 || pid == self {
			continue
		}
		if !unixLooksLikeTrimCLI(cmdLine) {
			continue
		}
		_ = syscall.Kill(pid, syscall.SIGTERM)
	}
	_ = exec.Command("sleep", "0.4").Run()
	out2, err := exec.Command("ps", "-ax", "-o", "pid=,command=").Output()
	if err != nil {
		if chrome.UninstallProcessesStopped != "" {
			fmt.Println(chrome.UninstallProcessesStopped)
		}
		return
	}
	for _, line := range strings.Split(string(out2), "\n") {
		pid, cmdLine, ok := parsePSLine(line)
		if !ok || pid <= 1 || pid == self {
			continue
		}
		if !unixLooksLikeTrimCLI(cmdLine) {
			continue
		}
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}
	if chrome.UninstallProcessesStopped != "" {
		fmt.Println(chrome.UninstallProcessesStopped)
	}
}

func parsePSLine(line string) (pid int, cmdLine string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return 0, "", false
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0, "", false
	}
	pid, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, "", false
	}
	return pid, strings.Join(fields[1:], " "), true
}

func unixLooksLikeTrimCLI(cmdLine string) bool {
	lower := strings.ToLower(strings.TrimSpace(cmdLine))
	if lower == "" {
		return false
	}
	first := strings.Fields(lower)[0]
	base := filepath.Base(first)
	if base != "trim" {
		return false
	}
	switch {
	case strings.Contains(lower, "/node_modules/@usetrim/trim/"):
		return true
	case strings.Contains(lower, "/cellar/trim/"):
		return true
	case strings.Contains(lower, "/.local/bin/trim"):
		return true
	case strings.Contains(lower, "/usr/local/bin/trim"):
		return true
	case strings.Contains(lower, "/opt/homebrew/bin/trim"):
		return true
	case first == "trim":
		return true
	case strings.HasSuffix(first, "/trim"):
		return true
	default:
		return false
	}
}
