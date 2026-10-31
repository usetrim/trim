package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
	"github.com/usetrim/trim/cli/internal/cliout"
	"github.com/usetrim/trim/cli/internal/config"
	"github.com/usetrim/trim/cli/internal/deepopt"
	"github.com/usetrim/trim/cli/internal/storage"
)

var autostartCmd = &cobra.Command{
	Use:   "autostart",
	Short: "",
}

var autostartStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		chrome := loadCLIChrome()
		p, err := deepopt.LoadPreferences()
		if err != nil {
			return err
		}
		if !p.AutoStartSet() {
			if chrome.AutostartUnset != "" {
				fmt.Println(chrome.AutostartUnset)
			}
			if chrome.AutostartEnableHint != "" {
				fmt.Println(chrome.AutostartEnableHint)
			}
			return nil
		}
		if p.AutoStartEnabled() {
			if chrome.AutostartEnabled != "" {
				fmt.Println(chrome.AutostartEnabled)
			}
			if chrome.AutostartDisableHint != "" {
				fmt.Println(chrome.AutostartDisableHint)
			}
		} else {
			if chrome.AutostartDisabled != "" {
				fmt.Println(chrome.AutostartDisabled)
			}
			if chrome.AutostartEnableHint != "" {
				fmt.Println(chrome.AutostartEnableHint)
			}
		}
		if chrome.AutostartDaemonHint != "" {
			fmt.Println(chrome.AutostartDaemonHint)
		}
		return nil
	},
}

var autostartEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		return setAutostart(true)
	},
}

var autostartDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		return setAutostart(false)
	},
}

func setAutostart(on bool) error {
	chrome := loadCLIChrome()
	if blocked, reason := deepopt.AutostartBlocked(); blocked && on {
		if reason == "dnt" && chrome.AutostartSkippedDNT != "" {
			fmt.Println(chrome.AutostartSkippedDNT)
		} else if chrome.AutostartSkippedManaged != "" {
			fmt.Println(chrome.AutostartSkippedManaged)
		}
		if chrome.AutostartManagedOffHint != "" {
			fmt.Println(chrome.AutostartManagedOffHint)
		}
		return chrome.fail("")
	}
	if on {
		_ = deepopt.ClearAutostartOffMarker()
	}
	p, err := deepopt.LoadPreferences()
	if err != nil {
		return err
	}
	p.AutoStartWithIDE = deepopt.BoolPtr(on)
	prefsChrome := deepopt.PrefsChrome{
		Unavailable:    chrome.ChromeUnavailable,
		EngineInvalid:  chrome.PrefsEngineInvalid,
		RateNonNeg:     chrome.PrefsRateNonNeg,
		TierRequired:   chrome.PrefsTierRequired,
		EngineRequired: chrome.PrefsEngineRequired,
		RateRequired:   chrome.PrefsRateRequired,
	}
	if err := deepopt.SavePreferences(p, prefsChrome); err != nil {
		return err
	}
	if err := pushAutostartPreference(on, chrome); err != nil {
		return err
	}
	// Preference is source of truth (IDE + cloud). OS login hook is best-effort:
	// never dump raw schtasks/registry errors as the only UX for worldwide users.
	osHookErr := applyDaemonForAutostart(on, chrome)
	if !on {
		// Best-effort stop of a running proxy so "uncheck" actually stops auto-trim path.
		if err := stopCmd.RunE(stopCmd, nil); err == nil && chrome.StopOnDisableOk != "" {
			cliout.Muted(chrome.StopOnDisableOk)
		}
	}
	if on {
		if chrome.AutostartEnabledOk != "" {
			cliout.Success(chrome.AutostartEnabledOk)
		}
	} else {
		if chrome.AutostartDisabledOk != "" {
			cliout.Success(chrome.AutostartDisabledOk)
		}
	}
	if osHookErr != nil {
		printAutostartOSHookGuidance(chrome, on, osHookErr)
	} else if chrome.AutostartDaemonHint != "" {
		cliout.Muted(chrome.AutostartDaemonHint)
	}
	return nil
}

func printAutostartOSHookGuidance(chrome cliChrome, enabling bool, err error) {
	if err == nil {
		return
	}
	raw := strings.TrimSpace(err.Error())
	denied := strings.Contains(strings.ToLower(raw), "access is denied") ||
		strings.Contains(strings.ToLower(raw), "access denied")

	if title := strings.TrimSpace(chrome.AutostartOSHookFailedTitle); title != "" {
		cliout.Warn(title)
	}
	if denied {
		if body := strings.TrimSpace(chrome.AutostartOSHookAccessDeniedBody); body != "" {
			cliout.Muted(body)
		}
	} else if body := strings.TrimSpace(chrome.AutostartOSHookFailedBody); body != "" {
		cliout.Muted(body)
	}
	rows := make([]cliout.Row, 0, 3)
	if enabling {
		if lab := strings.TrimSpace(chrome.AutostartOSHookNextStartLabel); lab != "" {
			rows = append(rows, cliout.Row{Label: lab, Value: "trim start", Kind: "accent"})
		}
		if lab := strings.TrimSpace(chrome.AutostartOSHookNextIDELabel); lab != "" {
			v := strings.TrimSpace(chrome.AutostartOSHookNextIDEValue)
			if v != "" {
				rows = append(rows, cliout.Row{Label: lab, Value: v, Kind: "ok"})
			}
		}
	}
	if flagVerbose && raw != "" {
		if lab := strings.TrimSpace(chrome.AutostartOSHookDetailLabel); lab != "" {
			rows = append(rows, cliout.Row{Label: lab, Value: raw})
		} else {
			cliout.Muted(raw)
		}
	}
	if len(rows) > 0 {
		cliout.Table(rows)
	}
}

// applyDaemonForAutostart installs OS login when on, uninstalls when off.
func applyDaemonForAutostart(on bool, _ cliChrome) error {
	if on {
		return daemonInstallCmd.RunE(daemonInstallCmd, nil)
	}
	return daemonUninstallCmd.RunE(daemonUninstallCmd, nil)
}

func pushAutostartPreference(on bool, chrome cliChrome) error {
	cfg, err := config.LoadLocal()
	if err != nil {
		return err
	}
	token, err := storage.GetToken()
	if err != nil || token == "" {
		return nil // local-only until login; fail-closed soft skip cloud push
	}
	body, err := json.Marshal(map[string]any{"auto_start_with_ide": on})
	if err != nil {
		return err
	}
	// Pass body into doCLIRequest so X-Trim-Signature covers the JSON (server verifies body).
	req, err := http.NewRequest(http.MethodPatch, strings.TrimRight(cfg.APIBaseURL, "/")+"/api/v1/me/preferences", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := newCLIHTTPClient(cfg)
	res, err := doCLIRequest(client, req, token, cfg.CLIHMACSecret, body)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 8192))
	if res.StatusCode != http.StatusOK {
		action := strings.TrimSpace(chrome.ConfigSyncActionAutoStart)
		if chrome.ConfigSyncFailedFmt != "" && action != "" {
			return fmt.Errorf(chrome.ConfigSyncFailedFmt, res.Status, action)
		}
		return chrome.fail("")
	}
	return nil
}

func init() {
	autostartCmd.AddCommand(autostartStatusCmd)
	autostartCmd.AddCommand(autostartEnableCmd)
	autostartCmd.AddCommand(autostartDisableCmd)
}
