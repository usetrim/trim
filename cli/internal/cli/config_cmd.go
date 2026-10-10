package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/usetrim/trim/cli/internal/config"
	"github.com/usetrim/trim/cli/internal/deepopt"
	"github.com/usetrim/trim/cli/internal/storage"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "",
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		chrome := loadCLIChrome()
		p, err := deepopt.LoadPreferences()
		if err != nil {
			return err
		}
		if len(args) == 0 {
			if chrome.ConfigGetTierFmt != "" {
				fmt.Printf(chrome.ConfigGetTierFmt+"\n", p.DefaultTier)
			}
			if chrome.ConfigGetEngineFmt != "" {
				fmt.Printf(chrome.ConfigGetEngineFmt+"\n", p.DeepEngine)
			}
			if chrome.ConfigGetTargetFmt != "" {
				fmt.Printf(chrome.ConfigGetTargetFmt+"\n", p.TargetToken)
			}
			if chrome.ConfigGetAutostartFmt != "" {
				if !p.AutoStartSet() {
					if chrome.ConfigGetAutostartUnset != "" {
						fmt.Printf(chrome.ConfigGetAutostartFmt+"\n", chrome.ConfigGetAutostartUnset)
					}
				} else {
					fmt.Printf(chrome.ConfigGetAutostartFmt+"\n", p.AutoStartEnabled())
				}
			}
			return nil
		}
		switch strings.ToLower(args[0]) {
		case "default-tier", "default-mode", "tier", "mode":
			fmt.Println(p.DefaultTier)
		case "deep-engine", "engine":
			fmt.Println(p.DeepEngine)
		case "target-token":
			fmt.Println(p.TargetToken)
		case "auto-start-with-ide", "autostart", "auto-start":
			if !p.AutoStartSet() {
				if chrome.ConfigGetAutostartUnset != "" {
					fmt.Println(chrome.ConfigGetAutostartUnset)
				}
			} else {
				fmt.Println(p.AutoStartEnabled())
			}
		default:
			if chrome.ConfigUnknownKeyFmt != "" {
				return fmt.Errorf(chrome.ConfigUnknownKeyFmt, args[0])
			}
			return chrome.fail("")
		}
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		chrome := loadCLIChrome()
		p, err := deepopt.LoadPreferences()
		if err != nil {
			return err
		}
		key := strings.ToLower(args[0])
		val := args[1]
		switch key {
		case "default-tier", "default-mode", "tier", "mode":
			tier := deepopt.NormalizeTier(val)
			if tier == "" {
				if chrome.PrefsTierRequired != "" {
					return fmt.Errorf("%s", chrome.PrefsTierRequired)
				}
				return chrome.fail("")
			}
			p.DefaultTier = tier
		case "deep-engine", "engine":
			eng := deepopt.NormalizeEngine(val)
			if eng == "" {
				if chrome.PrefsEngineInvalid != "" {
					return fmt.Errorf("%s", chrome.PrefsEngineInvalid)
				}
				return chrome.fail("")
			}
			p.DeepEngine = eng
		case "target-token":
			n, err := strconv.Atoi(val)
			if err != nil || n <= 0 {
				if chrome.ConfigTargetPositive != "" {
					return fmt.Errorf("%s", chrome.ConfigTargetPositive)
				}
				return chrome.fail("")
			}
			p.TargetToken = n
		case "auto-start-with-ide", "autostart", "auto-start":
			switch strings.ToLower(strings.TrimSpace(val)) {
			case "true", "1", "on", "yes", "enable", "enabled":
				return setAutostart(true)
			case "false", "0", "off", "no", "disable", "disabled":
				return setAutostart(false)
			default:
				if chrome.ConfigUnknownKeyFmt != "" {
					return fmt.Errorf(chrome.ConfigUnknownKeyFmt, val)
				}
				return chrome.fail("")
			}
		default:
			if chrome.ConfigUnknownKeyFmt != "" {
				return fmt.Errorf(chrome.ConfigUnknownKeyFmt, key)
			}
			return chrome.fail("")
		}
		prefsChrome := deepopt.PrefsChrome{
			Unavailable:    chrome.ChromeUnavailable,
			EngineInvalid:  chrome.PrefsEngineInvalid,
			TargetNonNeg:   chrome.PrefsTargetNonNeg,
			TierRequired:   chrome.PrefsTierRequired,
			EngineRequired: chrome.PrefsEngineRequired,
			TargetRequired: chrome.PrefsTargetRequired,
		}
		if err := deepopt.SavePreferences(p, prefsChrome); err != nil {
			return err
		}
		// Push Fast/Deep dials to cloud so trim start refresh does not overwrite local set.
		switch key {
		case "default-tier", "default-mode", "tier", "mode",
			"deep-engine", "engine", "target-token":
			if err := pushCompressPreferences(p, chrome); err != nil {
				return err
			}
		}
		if chrome.ConfigSavedFmt != "" {
			fmt.Printf(chrome.ConfigSavedFmt+"\n", key)
		}
		return nil
	},
}

// pushCompressPreferences PATCHes profiles compression dials (same contract as Dashboard Save).
func pushCompressPreferences(p deepopt.Preferences, chrome cliChrome) error {
	cfg, err := config.LoadLocal()
	if err != nil {
		return err
	}
	token, err := storage.GetToken()
	if err != nil || token == "" {
		return nil // local-only until login
	}
	body, err := json.Marshal(map[string]any{
		"compression_tier":  string(p.DefaultTier),
		"deep_engine":       string(p.DeepEngine),
		"deep_target_token": p.TargetToken,
	})
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
		if chrome.ConfigSyncFailedFmt != "" {
			return fmt.Errorf(chrome.ConfigSyncFailedFmt, res.Status, "patch compression preferences")
		}
		return chrome.fail("")
	}
	return nil
}

var configSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadLocal()
		if err != nil {
			return err
		}
		chrome := fetchCLIChrome(cfg)
		token, err := storage.GetToken()
		if err != nil || token == "" {
			if chrome.ConfigNotLoggedIn != "" {
				return fmt.Errorf("%s", chrome.ConfigNotLoggedIn)
			}
			return chrome.fail("")
		}
		req, err := http.NewRequest(http.MethodGet, cfg.APIBaseURL+"/api/v1/me/preferences", nil)
		if err != nil {
			return err
		}
		client := newCLIHTTPClient(cfg)
		res, err := doCLIRequest(client, req, token, cfg.CLIHMACSecret, nil)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		body, err := io.ReadAll(io.LimitReader(res.Body, 262144))
		if err != nil {
			return err
		}
		if res.StatusCode != http.StatusOK {
			if chrome.ConfigSyncFailedFmt != "" {
				return fmt.Errorf(chrome.ConfigSyncFailedFmt, res.Status, strings.TrimSpace(string(body)))
			}
			return chrome.fail("")
		}
		var remote remotePreferences
		if err := json.Unmarshal(body, &remote); err != nil {
			return err
		}
		cur, _ := deepopt.LoadPreferences()
		p := preferencesFromRemote(remote, cur)
		prefsChrome := deepopt.PrefsChrome{
			Unavailable:    chrome.ChromeUnavailable,
			EngineInvalid:  chrome.PrefsEngineInvalid,
			TargetNonNeg:   chrome.PrefsTargetNonNeg,
			TierRequired:   chrome.PrefsTierRequired,
			EngineRequired: chrome.PrefsEngineRequired,
			TargetRequired: chrome.PrefsTargetRequired,
		}
		if err := deepopt.SavePreferences(p, prefsChrome); err != nil {
			return err
		}
		if blocked, reason := deepopt.AutostartBlocked(); blocked {
			if reason == "dnt" && chrome.AutostartSkippedDNT != "" {
				fmt.Println(chrome.AutostartSkippedDNT)
			} else if chrome.AutostartSkippedManaged != "" {
				fmt.Println(chrome.AutostartSkippedManaged)
			}
			// Still save prefs from cloud, but do not install daemon / start under managed-off.
			_ = applyDaemonForAutostart(false, chrome)
			_ = stopCmd.RunE(stopCmd, nil)
		} else if err := applyDaemonForAutostart(p.AutoStartEnabled(), chrome); err != nil {
			// Preferences already saved from cloud; OS scheduler may be denied on locked-down hosts.
			fmt.Fprintf(os.Stderr, "%v\n", err)
		} else if !p.AutoStartEnabled() {
			if err := stopCmd.RunE(stopCmd, nil); err == nil && chrome.StopOnDisableOk != "" {
				fmt.Println(chrome.StopOnDisableOk)
			}
		}
		if chrome.AutostartSyncDaemonOk != "" {
			fmt.Println(chrome.AutostartSyncDaemonOk)
		}
		if chrome.ConfigSyncedFmt != "" {
			fmt.Printf(chrome.ConfigSyncedFmt+"\n", p.DefaultTier, p.DeepEngine, p.TargetToken)
		}
		return nil
	},
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configSyncCmd)
}
