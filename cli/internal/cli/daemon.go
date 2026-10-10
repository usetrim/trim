package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/usetrim/trim/cli/internal/config"
	"github.com/usetrim/trim/cli/internal/deepopt"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "",
}

// daemonRunCmd is the OS login entrypoint: honors auto_start_with_ide (fail-closed).
// Manual `trim start` stays unrestricted so users can run the proxy with auto-start off.
var daemonRunCmd = &cobra.Command{
	Use:   "run",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		chrome := loadCLIChrome()
		if blocked, reason := deepopt.AutostartBlocked(); blocked {
			if reason == "dnt" && chrome.AutostartSkippedDNT != "" {
				fmt.Println(chrome.AutostartSkippedDNT)
			} else if chrome.AutostartSkippedManaged != "" {
				fmt.Println(chrome.AutostartSkippedManaged)
			}
			if chrome.AutostartManagedOffHint != "" {
				fmt.Println(chrome.AutostartManagedOffHint)
			}
			// Drop OS login item so KeepAlive does not restart-loop while managed-off.
			releaseDaemonLoginItem(chrome)
			return nil
		}
		p, err := deepopt.LoadPreferences()
		if err != nil {
			return err
		}
		if !p.AutoStartSet() {
			// First install before login: honor site_messages DEFAULT_AUTO_START_WITH_IDE (same as IDE).
			def := strings.ToLower(strings.TrimSpace(chrome.DefaultAutoStartWithIDE))
			if def == "true" {
				// continue to health/start
			} else if def == "false" {
				if chrome.AutostartDaemonSkippedOff != "" {
					fmt.Println(chrome.AutostartDaemonSkippedOff)
				}
				releaseDaemonLoginItem(chrome)
				return nil
			} else {
				if chrome.AutostartDaemonSkippedUnset != "" {
					fmt.Println(chrome.AutostartDaemonSkippedUnset)
				}
				releaseDaemonLoginItem(chrome)
				return nil
			}
		} else if !p.AutoStartEnabled() {
			if chrome.AutostartDaemonSkippedOff != "" {
				fmt.Println(chrome.AutostartDaemonSkippedOff)
			}
			releaseDaemonLoginItem(chrome)
			return nil
		}
		// Prefer attach: if a healthy proxy already owns the port, do not double-bind.
		// Stay as a sidecar enforcer so dashboard uncheck still stops that proxy.
		healthURL := strings.TrimSpace(chrome.ProxyHealthURL)
		if healthURL != "" {
			to, err := requireChromeTimeout(chrome.ProxyHealthTimeoutMs, "CLI_PROXY_HEALTH_TIMEOUT_MS", chrome)
			if err != nil {
				return err
			}
			if proxyHealthOK(healthURL, to) {
				cfg, err := config.LoadLocal()
				if err != nil {
					return err
				}
				return runAutostartAttachEnforcer(cfg, chrome, healthURL)
			}
		}
		// Mark this process as the OS-login enforcer so start will re-check pref while running.
		// Manual `trim start` does not set this (user-owned process stays up until stop).
		_ = os.Setenv("TRIM_AUTOSTART_ENFORCER", "1")
		return startCmd.RunE(startCmd, args)
	},
}

// releaseDaemonLoginItem best-effort uninstalls OS autostart so KeepAlive/login
// agents do not restart thrash while auto-start is off or managed-off.
func releaseDaemonLoginItem(chrome cliChrome) {
	_ = applyDaemonForAutostart(false, chrome)
}

var daemonInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		exe, err = filepath.Abs(exe)
		if err != nil {
			return err
		}
		chrome := loadCLIChrome()

		switch runtime.GOOS {
		case "darwin":
			return installLaunchd(exe, chrome)
		case "linux":
			return installSystemdUser(exe, chrome)
		case "windows":
			return installWindowsTask(exe, chrome)
		default:
			return chrome.failFmt(chrome.DaemonUnsupportedFmt, "install", runtime.GOOS)
		}
	},
}

var daemonUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		chrome := loadCLIChrome()
		switch runtime.GOOS {
		case "darwin":
			return uninstallLaunchd(chrome)
		case "linux":
			return uninstallSystemdUser(chrome)
		case "windows":
			return uninstallWindowsTask(chrome)
		default:
			return chrome.failFmt(chrome.DaemonUnsupportedFmt, "uninstall", runtime.GOOS)
		}
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		chrome := loadCLIChrome()
		switch runtime.GOOS {
		case "darwin":
			plist := launchdPlistPath()
			if _, err := os.Stat(plist); err == nil {
				if chrome.DaemonLaunchdInstalledFmt != "" {
					fmt.Printf(chrome.DaemonLaunchdInstalledFmt+"\n", plist)
				}
				if u, err := currentUID(); err == nil {
					_ = runQuiet("launchctl", "print", "gui/"+u+"/ai.usetrim.trim")
				}
				return nil
			}
			if chrome.DaemonLaunchdMissing != "" {
				fmt.Println(chrome.DaemonLaunchdMissing)
			}
		case "linux":
			unit := systemdUnitPath()
			if _, err := os.Stat(unit); err == nil {
				if chrome.DaemonSystemdInstalledFmt != "" {
					fmt.Printf(chrome.DaemonSystemdInstalledFmt+"\n", unit)
				}
				_ = runQuiet("systemctl", "--user", "status", "trim.service")
				return nil
			}
			if chrome.DaemonSystemdMissing != "" {
				fmt.Println(chrome.DaemonSystemdMissing)
			}
		case "windows":
			out, err := exec.Command("schtasks", "/Query", "/TN", "TrimProxy").CombinedOutput()
			if err != nil {
				if chrome.DaemonWindowsMissing != "" {
					fmt.Println(chrome.DaemonWindowsMissing)
				}
				return nil
			}
			fmt.Println(string(out))
		default:
			return chrome.failFmt(chrome.DaemonUnsupportedFmt, "status", runtime.GOOS)
		}
		return nil
	},
}

func init() {
	daemonCmd.AddCommand(daemonRunCmd)
	daemonCmd.AddCommand(daemonInstallCmd)
	daemonCmd.AddCommand(daemonUninstallCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
}

func launchdPlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", "ai.usetrim.trim.plist")
}

func installLaunchd(exe string, chrome cliChrome) error {
	plist := launchdPlistPath()
	if err := os.MkdirAll(filepath.Dir(plist), 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>ai.usetrim.trim</string>
  <key>ProgramArguments</key>
  <array>
    <string>%s</string>
    <string>daemon</string>
    <string>run</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <dict>
    <key>SuccessfulExit</key>
    <false/>
  </dict>
  <key>StandardOutPath</key>
  <string>%s/Library/Logs/trim.log</string>
  <key>StandardErrorPath</key>
  <string>%s/Library/Logs/trim.err.log</string>
</dict>
</plist>
`, exe, os.Getenv("HOME"), os.Getenv("HOME"))
	if err := os.WriteFile(plist, []byte(content), 0o644); err != nil {
		return err
	}
	_ = exec.Command("launchctl", "unload", plist).Run()
	if err := exec.Command("launchctl", "load", plist).Run(); err != nil {
		return fmt.Errorf("launchctl load: %w", err)
	}
	if chrome.DaemonLaunchdOkFmt != "" {
		fmt.Printf(chrome.DaemonLaunchdOkFmt+"\n", plist)
	}
	if chrome.DaemonLaunchdHint != "" {
		fmt.Println(chrome.DaemonLaunchdHint)
	}
	return nil
}

func uninstallLaunchd(chrome cliChrome) error {
	plist := launchdPlistPath()
	_ = exec.Command("launchctl", "unload", plist).Run()
	_ = os.Remove(plist)
	if chrome.DaemonLaunchdRemoved != "" {
		fmt.Println(chrome.DaemonLaunchdRemoved)
	}
	return nil
}

func systemdUnitPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user", "trim.service")
}

func installSystemdUser(exe string, chrome cliChrome) error {
	unit := systemdUnitPath()
	if err := os.MkdirAll(filepath.Dir(unit), 0o755); err != nil {
		return err
	}
	desc := chrome.HelpDaemonServiceDesc
	restartSec := strings.TrimSpace(chrome.DaemonRestartSec)
	if restartSec == "" {
		return chrome.fail("")
	}
	minSec, errMin := strconv.Atoi(strings.TrimSpace(chrome.DaemonRestartMinSec))
	maxSec, errMax := strconv.Atoi(strings.TrimSpace(chrome.DaemonRestartMaxSec))
	if errMin != nil || errMax != nil || minSec < 1 || maxSec < 1 || minSec > maxSec {
		return chrome.fail("")
	}
	n, err := strconv.Atoi(restartSec)
	if err != nil || n < minSec || n > maxSec {
		return chrome.fail("")
	}
	content := fmt.Sprintf(`[Unit]
Description=%s
After=network.target

[Service]
Type=simple
ExecStart=%s daemon run
Restart=on-failure
RestartSec=%s

[Install]
WantedBy=default.target
`, desc, exe, restartSec)
	if err := os.WriteFile(unit, []byte(content), 0o644); err != nil {
		return err
	}
	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	if err := exec.Command("systemctl", "--user", "enable", "--now", "trim.service").Run(); err != nil {
		return fmt.Errorf("systemctl enable: %w (is lingering enabled? loginctl enable-linger $USER)", err)
	}
	if chrome.DaemonSystemdOkFmt != "" {
		fmt.Printf(chrome.DaemonSystemdOkFmt+"\n", unit)
	}
	return nil
}

func uninstallSystemdUser(chrome cliChrome) error {
	_ = exec.Command("systemctl", "--user", "disable", "--now", "trim.service").Run()
	_ = os.Remove(systemdUnitPath())
	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	if chrome.DaemonSystemdRemoved != "" {
		fmt.Println(chrome.DaemonSystemdRemoved)
	}
	return nil
}

func installWindowsTask(exe string, chrome cliChrome) error {
	cmd := exec.Command("schtasks", "/Create", "/F", "/SC", "ONLOGON", "/RL", "LIMITED",
		"/TN", "TrimProxy", "/TR", fmt.Sprintf("\"%s\" daemon run", exe))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("schtasks: %w: %s", err, strings.TrimSpace(string(out)))
	}
	// Start immediately (Create alone only registers ONLOGON; macOS/Linux enable --now parity).
	runOut, runErr := exec.Command("schtasks", "/Run", "/TN", "TrimProxy").CombinedOutput()
	if runErr != nil {
		return fmt.Errorf("schtasks /Run: %w: %s", runErr, strings.TrimSpace(string(runOut)))
	}
	if chrome.DaemonWindowsOk != "" {
		fmt.Println(chrome.DaemonWindowsOk)
	}
	return nil
}

func uninstallWindowsTask(chrome cliChrome) error {
	out, err := exec.Command("schtasks", "/Delete", "/F", "/TN", "TrimProxy").CombinedOutput()
	if err != nil {
		return fmt.Errorf("schtasks delete: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if chrome.DaemonWindowsRemoved != "" {
		fmt.Println(chrome.DaemonWindowsRemoved)
	}
	return nil
}

func currentUID() (string, error) {
	out, err := exec.Command("id", "-u").Output()
	if err != nil {
		return "", err
	}
	u := strings.TrimSpace(string(out))
	if u == "" {
		return "", fmt.Errorf("id -u returned empty")
	}
	return u, nil
}

func runQuiet(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func proxyHealthOK(healthURL string, timeout time.Duration) bool {
	if timeout <= 0 {
		return false
	}
	client := &http.Client{Timeout: timeout}
	res, err := client.Get(healthURL)
	if err != nil {
		return false
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return false
	}
	var body struct {
		Status string `json:"status"`
		Proxy  string `json:"proxy"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 4096)).Decode(&body); err != nil {
		return false
	}
	return body.Status == "ok" && body.Proxy == "trim"
}
