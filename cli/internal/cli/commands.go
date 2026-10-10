package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/usetrim/trim/cli/internal/clierr"
	"github.com/usetrim/trim/cli/internal/clihttp"
	"github.com/usetrim/trim/cli/internal/config"
	"github.com/usetrim/trim/cli/internal/fingerprint"
	"github.com/usetrim/trim/cli/internal/metrics"
	"github.com/usetrim/trim/cli/internal/storage"
	"github.com/usetrim/trim/server/pkg/localchrome"
)

// Version is set by GoReleaser ldflags or defaults for local builds.
var Version = "0.1.0"

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadLocal()
		if err != nil {
			return err
		}

		hw, err := fingerprint.HardwareUUID()
		if err != nil {
			return fmt.Errorf("hardware id: %w", err)
		}

		chrome := fetchCLIChrome(cfg)
		if phrase := chrome.providerPhrase(); phrase != "" && chrome.LoginSignInFmt != "" {
			fmt.Printf(chrome.LoginSignInFmt+"\n", phrase)
		} else if chrome.LoginOpeningBrowser != "" {
			fmt.Println(chrome.LoginOpeningBrowser)
		}

		authURLPath := strings.TrimSpace(chrome.PathCLIAuth)
		if authURLPath == "" {
			return chrome.fail(chrome.ChromeUnavailable)
		}
		if !strings.HasPrefix(authURLPath, "/") {
			authURLPath = "/" + authURLPath
		}
		authURL := fmt.Sprintf("%s%s?hardware=%s", strings.TrimRight(cfg.AppPublicURL, "/"), authURLPath, url.QueryEscape(hw))
		fmt.Println(authURL)
		_ = openBrowser(authURL, chrome)

		if chrome.LoginPasteKey != "" {
			fmt.Println(chrome.LoginPasteKey)
		}
		var key string
		_, _ = fmt.Scanln(&key)
		if key == "" {
			return chrome.fail(chrome.LoginKeyRequired)
		}
		if err := storage.SaveToken(key); err != nil {
			return err
		}
		if chrome.LoginSuccess != "" {
			fmt.Println(chrome.LoginSuccess)
		}
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadLocal()
		if err != nil {
			return err
		}
		chrome := fetchCLIChrome(cfg)
		err = storage.DeleteToken()
		if err == storage.ErrNotLoggedIn {
			if chrome.LogoutAlready != "" {
				fmt.Println(chrome.LogoutAlready)
			}
			return nil
		}
		if err != nil {
			return err
		}
		if chrome.LogoutSuccess != "" {
			fmt.Println(chrome.LogoutSuccess)
		}
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadLocal()
		if err != nil {
			return err
		}
		token, err := storage.GetToken()
		if err != nil || token == "" {
			chrome := fetchCLIChrome(cfg)
			if chrome.StatusNotLoggedIn != "" {
				fmt.Println(chrome.StatusNotLoggedIn)
			}
			return nil
		}
		req, err := http.NewRequest(http.MethodGet, cfg.APIBaseURL+"/api/v1/me/quota", nil)
		if err != nil {
			return err
		}
		client := newCLIHTTPClient(cfg)
		res, err := doCLIRequest(client, req, token, cfg.CLIHMACSecret, nil)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		chrome := fetchCLIChrome(cfg)
		if chrome.StatusAPIFmt != "" {
			fmt.Printf(chrome.StatusAPIFmt+"\n", res.Status)
		}
		body, _ := io.ReadAll(io.LimitReader(res.Body, 8192))
		fmt.Println(string(body))
		printQuotaExhaustedHint(chrome, body)
		return nil
	},
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadLocal()
		if err != nil {
			return err
		}
		chrome := fetchCLIChrome(cfg)
		token, err := storage.GetToken()
		if err != nil || token == "" {
			if chrome.StatusNotLoggedIn != "" {
				fmt.Println(chrome.StatusNotLoggedIn)
			}
			return nil
		}
		req, err := http.NewRequest(http.MethodGet, cfg.APIBaseURL+"/api/v1/me/quota", nil)
		if err != nil {
			return err
		}
		client := newCLIHTTPClient(cfg)
		res, err := doCLIRequest(client, req, token, cfg.CLIHMACSecret, nil)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(res.Body, 8192))
		upgradeURL := tierUpgradeURLFromQuotaJSON(body)
		if upgradeURL == "" {
			return chrome.fail(chrome.QuotaUpgradeURLMissing)
		}
		if chrome.QuotaExhaustedTitle != "" {
			fmt.Println(chrome.QuotaExhaustedTitle)
		}
		if chrome.QuotaExhaustedBody != "" {
			fmt.Println(chrome.QuotaExhaustedBody)
		}
		if chrome.QuotaUpgradeOpening != "" {
			fmt.Println(chrome.QuotaUpgradeOpening)
		}
		fmt.Println(upgradeURL)
		_ = openBrowser(upgradeURL, chrome)
		return nil
	},
}

func tierUpgradeURLFromQuotaJSON(body []byte) string {
	var q struct {
		TierUpgrade string `json:"tier_upgrade"`
	}
	if err := json.Unmarshal(body, &q); err != nil {
		return ""
	}
	return strings.TrimSpace(q.TierUpgrade)
}

func printQuotaExhaustedHint(chrome cliChrome, body []byte) {
	var q struct {
		Remaining   int    `json:"remaining"`
		Unlimited   bool   `json:"unlimited"`
		Code        string `json:"code"`
		TierUpgrade string `json:"tier_upgrade"`
	}
	if err := json.Unmarshal(body, &q); err != nil {
		return
	}
	if q.Unlimited {
		return
	}
	exhausted := q.Remaining <= 0 || q.Code == "QUOTA_EXHAUSTED" || q.Code == "WORKSPACE_QUOTA_EXHAUSTED"
	if !exhausted {
		return
	}
	if chrome.QuotaExhaustedTitle != "" {
		fmt.Println(chrome.QuotaExhaustedTitle)
	}
	if chrome.QuotaExhaustedBody != "" {
		fmt.Println(chrome.QuotaExhaustedBody)
	}
	url := strings.TrimSpace(q.TierUpgrade)
	if url != "" && chrome.QuotaUpgradeHintFmt != "" {
		fmt.Printf(chrome.QuotaUpgradeHintFmt+"\n", url)
	}
}

var statsTUI bool

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if statsTUI {
			return tuiCmd.RunE(cmd, args)
		}
		cfg, err := config.LoadLocal()
		if err != nil {
			return err
		}
		chrome := fetchCLIChrome(cfg)
		if chrome.StatsURLFmt == "" || !strings.Contains(chrome.StatsURLFmt, "{port}") {
			return fmt.Errorf("%s", chrome.StatsURLMissing)
		}
		url := strings.ReplaceAll(chrome.StatsURLFmt, "{port}", cfg.Port)
		res, err := http.Get(url)
		if err == nil {
			defer res.Body.Close()
			body, _ := io.ReadAll(io.LimitReader(res.Body, 8192))
			if chrome.StatsLiveHeader != "" {
				fmt.Println(chrome.StatsLiveHeader)
			}
			printLiveSavingsClarity(chrome, cfg.Port, body)
			fmt.Println(string(body))
		} else if chrome.StatsLiveOfflineFmt != "" {
			fmt.Printf(chrome.StatsLiveOfflineFmt+"\n", err)
		}
		store, err := metrics.OpenDefault()
		if err != nil {
			return fmt.Errorf("local metrics db: %w", err)
		}
		sum, err := store.TodaySummary(cfg.SavingsUsdPerMTok)
		if err != nil {
			return err
		}
		if chrome.StatsSqliteTodayFmt != "" {
			fmt.Printf(chrome.StatsSqliteTodayFmt+"\n",
				sum.Requests, sum.TokensBefore, sum.TokensAfter, sum.SavedUSD, sum.LastLatency)
		}
		series, serr := store.Series(cfg.StatsSeriesDays, cfg.SavingsUsdPerMTok)
		if serr == nil && len(series) > 0 {
			if chrome.StatsSqliteSeriesHeader != "" {
				fmt.Println(chrome.StatsSqliteSeriesHeader)
			}
			for _, p := range series {
				if chrome.StatsSqliteSeriesRowFmt != "" {
					fmt.Printf(chrome.StatsSqliteSeriesRowFmt+"\n",
						p.Day, p.Requests, p.TokensBefore, p.TokensAfter, p.SavedUSD, p.AvgLatency)
				}
			}
		}
		if chrome.StatsTuiTip != "" {
			fmt.Println(chrome.StatsTuiTip)
		}
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(Version)
	},
}

func attachCLIHeaders(req *http.Request, token string, hmacSecret string, body []byte) error {
	return clihttp.AttachAuthHeaders(req, token, hmacSecret, Version, "cli", body)
}

func fetchCLIChrome(cfg config.Local) cliChrome {
	out := fetchCLIChromeRemote(cfg)
	if chromeAutostartUsable(out) {
		_ = writeCLIChromeCache(out)
		setPOWChrome(out)
		return out
	}
	// Last-known-good only: previously synced site_messages bodies (not invent).
	if cached, ok := readCLIChromeCache(); ok {
		setPOWChrome(cached)
		return cached
	}
	setPOWChrome(out)
	return out
}

func fetchCLIChromeRemote(cfg config.Local) cliChrome {
	base := strings.TrimRight(strings.TrimSpace(cfg.APIBaseURL), "/")
	if base == "" {
		return cliChrome{}
	}
	client := newCLIHTTPClient(cfg)
	res, err := client.Get(base + "/api/v1/public/auth-providers")
	if err != nil {
		return cliChrome{}
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return cliChrome{}
	}
	var body struct {
		Items []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
			ActionLabel string `json:"action_label"`
		} `json:"items"`
		Site struct {
			PathCLIAuth string `json:"path_cli_auth"`
		} `json:"site"`
		Local localchrome.Chrome `json:"local"`
		CLI   struct {
			LoginSignInFmt              string `json:"login_sign_in_fmt"`
			LoginOpeningBrowser         string `json:"login_opening_browser"`
			LoginPasteKey               string `json:"login_paste_key"`
			LoginKeyRequired            string `json:"login_key_required"`
			LoginSuccess                string `json:"login_success"`
			LogoutSuccess               string `json:"logout_success"`
			LogoutAlready               string `json:"logout_already"`
			ChromeUnavailable           string `json:"chrome_unavailable"`
			BrowserUnsupportedFmt       string `json:"browser_unsupported_fmt"`
			StatusNotLoggedIn           string `json:"status_not_logged_in"`
			LoginPhraseOr               string `json:"login_phrase_or"`
			LoginPhraseComma            string `json:"login_phrase_comma"`
			LoginPhraseCommaOr          string `json:"login_phrase_comma_or"`
			SetupEndpointFmt            string `json:"setup_endpoint_fmt"`
			SetupNoneFound              string `json:"setup_none_found"`
			SetupUpdatedHeader          string `json:"setup_updated_header"`
			SetupShellHeader            string `json:"setup_shell_header"`
			SetupJetbrainsHint          string `json:"setup_jetbrains_hint"`
			SetupAPIEndpointFmt         string `json:"setup_api_endpoint_fmt"`
			SetupNextHeader             string `json:"setup_next_header"`
			SetupStepStart              string `json:"setup_step_start"`
			SetupStepRestart            string `json:"setup_step_restart"`
			SetupStepLogin              string `json:"setup_step_login"`
			SetupStepDashboardFmt       string `json:"setup_step_dashboard_fmt"`
			SetupOptionalTLS            string `json:"setup_optional_tls"`
			SetupTLSThen                string `json:"setup_tls_then"`
			SetupTLSGeneratedFmt        string `json:"setup_tls_generated_fmt"`
			SetupTLSCertFmt             string `json:"setup_tls_cert_fmt"`
			SetupTLSKeyFmt              string `json:"setup_tls_key_fmt"`
			SetupTLSKeyPathFmt          string `json:"setup_tls_key_path_fmt"`
			SetupShellOpenAIFmt         string `json:"setup_shell_openai_fmt"`
			SetupShellAnthropicFmt      string `json:"setup_shell_anthropic_fmt"`
			SetupTLSStart               string `json:"setup_tls_start"`
			SetupTLSPointIDE            string `json:"setup_tls_point_ide"`
			SetupTLSEnvRequired         string `json:"setup_tls_env_required"`
			SetupTLSDaysInvalid         string `json:"setup_tls_days_invalid"`
			PowClientRequired           string `json:"pow_client_required"`
			Pow428Missing               string `json:"pow_428_missing"`
			PowSolveFailed              string `json:"pow_solve_failed"`
			PowParseFailedFmt           string `json:"pow_parse_failed_fmt"`
			SetupWroteFmt               string `json:"setup_wrote_fmt"`
			SetupProductCursor          string `json:"setup_product_cursor"`
			SetupProductVSCode          string `json:"setup_product_vscode"`
			SetupProductWindsurf        string `json:"setup_product_windsurf"`
			SetupProductContinue        string `json:"setup_product_continue"`
			SetupProductZed             string `json:"setup_product_zed"`
			SetupProductJetbrains       string `json:"setup_product_jetbrains"`
			SetupContinueNoModel        string `json:"setup_continue_no_model"`
			StartRunningFmt             string `json:"start_running_fmt"`
			StartDashboardFmt           string `json:"start_dashboard_fmt"`
			StartModeFmt                string `json:"start_mode_fmt"`
			StartPointIDEFmt            string `json:"start_point_ide_fmt"`
			StartRulesFmt               string `json:"start_rules_fmt"`
			StartStopped                string `json:"start_stopped"`
			ErrFmt                      string `json:"err_fmt"`
			ProxyFailedFmt              string `json:"proxy_failed_fmt"`
			MainErrFmt                  string `json:"main_err_fmt"`
			DaemonUnsupportedFmt        string `json:"daemon_unsupported_fmt"`
			DaemonLaunchdInstalledFmt   string `json:"daemon_launchd_installed_fmt"`
			DaemonLaunchdMissing        string `json:"daemon_launchd_missing"`
			DaemonSystemdInstalledFmt   string `json:"daemon_systemd_installed_fmt"`
			DaemonSystemdMissing        string `json:"daemon_systemd_missing"`
			DaemonWindowsMissing        string `json:"daemon_windows_missing"`
			DaemonLaunchdOkFmt          string `json:"daemon_launchd_ok_fmt"`
			DaemonLaunchdHint           string `json:"daemon_launchd_hint"`
			DaemonLaunchdRemoved        string `json:"daemon_launchd_removed"`
			DaemonSystemdOkFmt          string `json:"daemon_systemd_ok_fmt"`
			DaemonSystemdRemoved        string `json:"daemon_systemd_removed"`
			DaemonWindowsOk             string `json:"daemon_windows_ok"`
			DaemonWindowsRemoved        string `json:"daemon_windows_removed"`
			TelemetryEnabled            string `json:"telemetry_enabled"`
			TelemetryDisableHint        string `json:"telemetry_disable_hint"`
			TelemetryEnvHint            string `json:"telemetry_env_hint"`
			TelemetryDisabled           string `json:"telemetry_disabled"`
			TelemetryEnableHint         string `json:"telemetry_enable_hint"`
			TelemetryDisabledOk         string `json:"telemetry_disabled_ok"`
			TelemetryEnabledOk          string `json:"telemetry_enabled_ok"`
			AutostartEnabled            string `json:"autostart_enabled"`
			AutostartDisabled           string `json:"autostart_disabled"`
			AutostartUnset              string `json:"autostart_unset"`
			AutostartEnableHint         string `json:"autostart_enable_hint"`
			AutostartDisableHint        string `json:"autostart_disable_hint"`
			AutostartEnabledOk          string `json:"autostart_enabled_ok"`
			AutostartDisabledOk         string `json:"autostart_disabled_ok"`
			AutostartDaemonHint         string `json:"autostart_daemon_hint"`
			AutostartDaemonSkippedOff   string `json:"autostart_daemon_skipped_off"`
			AutostartDaemonSkippedUnset string `json:"autostart_daemon_skipped_unset"`
			AutostartSkippedManaged     string `json:"autostart_skipped_managed"`
			AutostartSkippedDNT         string `json:"autostart_skipped_dnt"`
			AutostartManagedOffHint     string `json:"autostart_managed_off_hint"`
			AutostartSyncDaemonOk       string `json:"autostart_sync_daemon_ok"`
			DefaultAutoStartWithIDE     string `json:"default_auto_start_with_ide"`
			AutostartPrefPollMs         string `json:"autostart_pref_poll_ms"`
			AutostartEnforcerStopped    string `json:"autostart_enforcer_stopped"`
			AutostartPrefPollMinMs      string `json:"autostart_pref_poll_min_ms"`
			AutostartPrefPollMaxMs      string `json:"autostart_pref_poll_max_ms"`
			AutostartTimeoutMinMs       string `json:"autostart_timeout_min_ms"`
			AutostartTimeoutMaxMs       string `json:"autostart_timeout_max_ms"`
			DaemonRestartSec            string `json:"daemon_restart_sec"`
			DaemonRestartMinSec         string `json:"daemon_restart_min_sec"`
			DaemonRestartMaxSec         string `json:"daemon_restart_max_sec"`
			StopOnDisableOk             string `json:"stop_on_disable_ok"`
			ProxyHealthURL              string `json:"proxy_health_url"`
			ProxyShutdownURL            string `json:"proxy_shutdown_url"`
			ProxyHealthTimeoutMs        string `json:"proxy_health_timeout_ms"`
			ProxyShutdownTimeoutMs      string `json:"proxy_shutdown_timeout_ms"`
			HTTPShutdownTimeoutMs       string `json:"http_shutdown_timeout_ms"`
			ProxyFallbackUncompressed   string `json:"proxy_fallback_uncompressed"`
			ProxyActiveFileProtection   string `json:"proxy_active_file_protection"`
			StopOk                      string `json:"stop_ok"`
			StopFailedFmt               string `json:"stop_failed_fmt"`
			StopURLMissing              string `json:"stop_url_missing"`
			ConfigGetAutostartUnset     string `json:"config_get_autostart_unset"`
			CompressFileRequired        string `json:"compress_file_required"`
			CompressModeRequired        string `json:"compress_mode_required"`
			CompressModeInvalid         string `json:"compress_mode_invalid"`
			TreesitterRequired          string `json:"treesitter_required"`
			CompressDeepPrefsRequired   string `json:"compress_deep_prefs_required"`
			CompressDeepResultFmt       string `json:"compress_deep_result_fmt"`
			CompressWroteFmt            string `json:"compress_wrote_fmt"`
			StatsLiveHeader             string `json:"stats_live_header"`
			StatsLiveOfflineFmt         string `json:"stats_live_offline_fmt"`
			StatsURLFmt                 string `json:"stats_url_fmt"`
			StatsURLMissing             string `json:"stats_url_missing"`
			StatsSqliteTodayFmt         string `json:"stats_sqlite_today_fmt"`
			StatsSqliteSeriesHeader     string `json:"stats_sqlite_series_header"`
			StatsSqliteSeriesRowFmt     string `json:"stats_sqlite_series_row_fmt"`
			StatsTuiTip                 string `json:"stats_tui_tip"`
			StatsDeepStatusFmt          string `json:"stats_deep_status_fmt"`
			StatsDeepStageFmt           string `json:"stats_deep_stage_fmt"`
			StatsLastRequestFmt         string `json:"stats_last_request_fmt"`
			StatsDoorFmt                string `json:"stats_door_fmt"`
			StatsDashboardTipFmt        string `json:"stats_dashboard_tip_fmt"`
			StatusAPIFmt                string `json:"status_api_fmt"`
			QuotaExhaustedTitle         string `json:"quota_exhausted_title"`
			QuotaExhaustedBody          string `json:"quota_exhausted_body"`
			QuotaUpgradeHintFmt         string `json:"quota_upgrade_hint_fmt"`
			QuotaUpgradeOpening         string `json:"quota_upgrade_opening"`
			QuotaUpgradeURLMissing      string `json:"quota_upgrade_url_missing"`
			DeepBootstrapReqs           string `json:"deep_bootstrap_reqs"`
			DeepBootstrapPip            string `json:"deep_bootstrap_pip"`
			DeepRequirementsMissing     string `json:"deep_requirements_missing"`
			DeepAutoInstall             string `json:"deep_auto_install"`
			DeepBinEnvMissingFmt        string `json:"deep_bin_env_missing_fmt"`
			DeepBinMissing              string `json:"deep_bin_missing"`
			DeepPyEnvMissingFmt         string `json:"deep_py_env_missing_fmt"`
			DeepPyMissing               string `json:"deep_py_missing"`
			DeepLLMMissingFmt           string `json:"deep_llm_missing_fmt"`
			DeepPipFailed               string `json:"deep_pip_failed"`
			DeepTargetRequired          string `json:"deep_target_required"`
			DeepEngineRequired          string `json:"deep_engine_required"`
			DeepQuestionRequired        string `json:"deep_question_required"`
			DeepOptimizeFmt             string `json:"deep_optimize_fmt"`
			DeepTimeout                 string `json:"deep_timeout"`
			DeepParseFmt                string `json:"deep_parse_fmt"`
			ProxyDeepFailedFmt          string `json:"proxy_deep_failed_fmt"`
			ProxyLiveDeepModeFmt        string `json:"proxy_live_deep_mode_fmt"`
			ProxyLiveDeepEnabledFmt     string `json:"proxy_live_deep_enabled_fmt"`
			ProxyDeepCompactStub        string `json:"proxy_deep_compact_stub"`
			ProxyDeepChromeRequired     string `json:"proxy_deep_chrome_required"`
			ProxyDeepPrefsRequired      string `json:"proxy_deep_prefs_required"`
			ProxyDeepSkippedMinFmt      string `json:"proxy_deep_skipped_min_fmt"`
			ProxyDeepSkippedStream      string `json:"proxy_deep_skipped_stream"`
			ProxyDeepOOMSkipped         string `json:"proxy_deep_oom_skipped"`
			ProxyDeepRuntimeRequired    string `json:"proxy_deep_runtime_required"`
			ProxyAdapterRequired        string `json:"proxy_adapter_required"`
			ProxyAdapterUnknownDialectFmt string `json:"proxy_adapter_unknown_dialect_fmt"`
			ProxyAdapterModelRequired   string `json:"proxy_adapter_model_required"`
			ProxyAdapterAliasRequiredFmt string `json:"proxy_adapter_alias_required_fmt"`
			ProxyAdapterTranslateFmt    string `json:"proxy_adapter_translate_fmt"`
			ProxyAdapterAuthFmt         string `json:"proxy_adapter_auth_fmt"`
			ProxyAdapterResponseFmt     string `json:"proxy_adapter_response_fmt"`
			ProxyAdapterEnabledFmt      string `json:"proxy_adapter_enabled_fmt"`
			ProxyModelsNotSynced        string `json:"proxy_models_not_synced"`
			ProxyModelsMethod           string `json:"proxy_models_method"`
			ProxyErrBaseURLDoubleV1     string `json:"proxy_err_base_url_double_v1"`
			ProxyErrModelNotFoundFmt    string `json:"proxy_err_model_not_found_fmt"`
			ProxyErrUnknownModelFmt     string `json:"proxy_err_unknown_model_fmt"`
			ProxyErrUpstreamAuthFmt     string `json:"proxy_err_upstream_auth_fmt"`
			ProxyErrUpstreamQuotaFmt    string `json:"proxy_err_upstream_quota_fmt"`
			ProxyErrUpstreamUnavailableFmt string `json:"proxy_err_upstream_unavailable_fmt"`
			ProxyModelsUpstreamAdapterRequired string `json:"proxy_models_upstream_adapter_required"`
			ProxyModelsUpstreamNotFoundFmt     string `json:"proxy_models_upstream_not_found_fmt"`
			ProxyModelsUpstreamAuthRequired    string `json:"proxy_models_upstream_auth"`
			ProxyModelsUpstreamUnsupportedFmt  string `json:"proxy_models_upstream_unsupported_fmt"`
			SetupClaudeCodeBlockFmt     string `json:"setup_claude_code_block_fmt"`
			SetupContinueBlockFmt       string `json:"setup_continue_block_fmt"`
			SetupVSCodeChatBlockFmt     string `json:"setup_vscode_chat_block_fmt"`
			ProxyDoorOpenAI             string `json:"proxy_door_openai"`
			ProxyDoorAnthropic          string `json:"proxy_door_anthropic"`
			PrefsEngineInvalid          string `json:"prefs_engine_invalid"`
			PrefsTargetNonNeg           string `json:"prefs_target_nonneg"`
			PrefsTierRequired           string `json:"prefs_tier_required"`
			PrefsEngineRequired         string `json:"prefs_engine_required"`
			PrefsTargetRequired         string `json:"prefs_target_required"`
			ConfigGetTierFmt            string `json:"config_get_tier_fmt"`
			ConfigGetEngineFmt          string `json:"config_get_engine_fmt"`
			ConfigGetTargetFmt          string `json:"config_get_target_fmt"`
			ConfigGetAutostartFmt       string `json:"config_get_autostart_fmt"`
			ConfigUnknownKeyFmt         string `json:"config_unknown_key_fmt"`
			ConfigTargetPositive        string `json:"config_target_positive"`
			ConfigSavedFmt              string `json:"config_saved_fmt"`
			ConfigNotLoggedIn           string `json:"config_not_logged_in"`
			ConfigSyncFailedFmt         string `json:"config_sync_failed_fmt"`
			ConfigSyncedFmt             string `json:"config_synced_fmt"`
			HelpRootShort               string `json:"help_root_short"`
			HelpRootLong                string `json:"help_root_long"`
			HelpGroupEveryday           string `json:"help_group_everyday"`
			HelpGroupAdvanced           string `json:"help_group_advanced"`
			HelpStartShort              string `json:"help_start_short"`
			HelpStartLong               string `json:"help_start_long"`
			HelpStartFlagPort           string `json:"help_start_flag_port"`
			HelpLoginShort              string `json:"help_login_short"`
			HelpLogoutShort             string `json:"help_logout_short"`
			HelpUninstallShort          string `json:"help_uninstall_short"`
			HelpUninstallLong           string `json:"help_uninstall_long"`
			HelpUninstallFlagKeepData   string `json:"help_uninstall_flag_keep_data"`
			UninstallStarting           string `json:"uninstall_starting"`
			UninstallStopped            string `json:"uninstall_stopped"`
			UninstallDaemonOk           string `json:"uninstall_daemon_ok"`
			UninstallLogoutOk           string `json:"uninstall_logout_ok"`
			UninstallSetupRevertedFmt   string `json:"uninstall_setup_reverted_fmt"`
			UninstallSetupNone          string `json:"uninstall_setup_none"`
			UninstallConfigRemovedFmt   string `json:"uninstall_config_removed_fmt"`
			UninstallLogsRemoved        string `json:"uninstall_logs_removed"`
			UninstallDeepOk             string `json:"uninstall_deep_ok"`
			UninstallDeepSkipped        string `json:"uninstall_deep_skipped"`
			UninstallSidecarRemovedFmt  string `json:"uninstall_sidecar_removed_fmt"`
			UninstallBinaryRemovedFmt   string `json:"uninstall_binary_removed_fmt"`
			UninstallBinaryManualFmt    string `json:"uninstall_binary_manual_fmt"`
			UninstallPathRemoved        string `json:"uninstall_path_removed"`
			UninstallDone               string `json:"uninstall_done"`
			UninstallNextExt            string `json:"uninstall_next_ext"`
			UninstallNextPkgBrew        string `json:"uninstall_next_pkg_brew"`
			UninstallNextPkgScoop       string `json:"uninstall_next_pkg_scoop"`
			UninstallNextPkgWinget      string `json:"uninstall_next_pkg_winget"`
			UninstallHfHubDirs          string `json:"uninstall_hf_hub_dirs"`
			UninstallHfCacheRel         string `json:"uninstall_hf_cache_rel"`
			UninstallHfHubSubdir        string `json:"uninstall_hf_hub_subdir"`
			UninstallHomeDirsRel        string `json:"uninstall_home_dirs_rel"`
			UninstallHomeDirsMissing    string `json:"uninstall_home_dirs_missing"`
			UninstallDarwinLogRels      string `json:"uninstall_darwin_log_rels"`
			UninstallPipPackages        string `json:"uninstall_pip_packages"`
			UninstallPipBins            string `json:"uninstall_pip_bins"`
			UninstallPythonBins         string `json:"uninstall_python_bins"`
			UninstallPipModule          string `json:"uninstall_pip_module"`
			UninstallSidecarNames       string `json:"uninstall_sidecar_names"`
			UninstallArpDisplayName     string `json:"uninstall_arp_display_name"`
			UninstallArpPublisher       string `json:"uninstall_arp_publisher"`
			UninstallArpRegKey          string `json:"uninstall_arp_reg_key"`
			UninstallArpRemoved         string `json:"uninstall_arp_removed"`
			HelpStatusShort             string `json:"help_status_short"`
			HelpStatusLong              string `json:"help_status_long"`
			HelpUpgradeShort            string `json:"help_upgrade_short"`
			HelpStatsShort              string `json:"help_stats_short"`
			HelpStatsLong               string `json:"help_stats_long"`
			HelpStatsFlagTui            string `json:"help_stats_flag_tui"`
			HelpSetupShort              string `json:"help_setup_short"`
			HelpSetupLong               string `json:"help_setup_long"`
			HelpSetupFlagTls            string `json:"help_setup_flag_tls"`
			HelpVersionShort            string `json:"help_version_short"`
			HelpDaemonShort             string `json:"help_daemon_short"`
			HelpDaemonInstallShort      string `json:"help_daemon_install_short"`
			HelpDaemonUninstallShort    string `json:"help_daemon_uninstall_short"`
			HelpDaemonStatusShort       string `json:"help_daemon_status_short"`
			HelpDaemonRunShort          string `json:"help_daemon_run_short"`
			HelpDaemonServiceDesc       string `json:"help_daemon_service_desc"`
			HelpTelemetryShort          string `json:"help_telemetry_short"`
			HelpTelemetryStatusShort    string `json:"help_telemetry_status_short"`
			HelpTelemetryDisableShort   string `json:"help_telemetry_disable_short"`
			HelpTelemetryEnableShort    string `json:"help_telemetry_enable_short"`
			HelpAutostartShort          string `json:"help_autostart_short"`
			HelpAutostartStatusShort    string `json:"help_autostart_status_short"`
			HelpAutostartEnableShort    string `json:"help_autostart_enable_short"`
			HelpAutostartDisableShort   string `json:"help_autostart_disable_short"`
			HelpStopShort               string `json:"help_stop_short"`
			HelpTuiShort                string `json:"help_tui_short"`
			HelpTuiLong                 string `json:"help_tui_long"`
			HelpCompressShort           string `json:"help_compress_short"`
			HelpCompressLong            string `json:"help_compress_long"`
			HelpCompressFlagMode        string `json:"help_compress_flag_mode"`
			HelpCompressFlagDeep        string `json:"help_compress_flag_deep"`
			HelpCompressFlagEngine      string `json:"help_compress_flag_engine"`
			HelpCompressFlagQuestion    string `json:"help_compress_flag_question"`
			HelpCompressFlagTarget      string `json:"help_compress_flag_target"`
			HelpCompressFlagBootstrap   string `json:"help_compress_flag_bootstrap"`
			HelpCompressFlagOut         string `json:"help_compress_flag_out"`
			HelpConfigShort             string `json:"help_config_short"`
			HelpConfigGetShort          string `json:"help_config_get_short"`
			HelpConfigSetShort          string `json:"help_config_set_short"`
			HelpConfigSyncShort         string `json:"help_config_sync_short"`
		} `json:"cli"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return cliChrome{}
	}
	labels := make([]string, 0, len(body.Items))
	for _, it := range body.Items {
		// Fail closed: only API display_name (no invent-parse of action_label).
		name := strings.TrimSpace(it.DisplayName)
		if name != "" {
			labels = append(labels, name)
		}
	}
	out := cliChrome{
		PathCLIAuth:                 strings.TrimSpace(body.Site.PathCLIAuth),
		LoginSignInFmt:              body.CLI.LoginSignInFmt,
		LoginOpeningBrowser:         body.CLI.LoginOpeningBrowser,
		LoginPasteKey:               body.CLI.LoginPasteKey,
		LoginKeyRequired:            body.CLI.LoginKeyRequired,
		LoginSuccess:                body.CLI.LoginSuccess,
		LogoutSuccess:               body.CLI.LogoutSuccess,
		LogoutAlready:               body.CLI.LogoutAlready,
		ChromeUnavailable:           body.CLI.ChromeUnavailable,
		BrowserUnsupportedFmt:       body.CLI.BrowserUnsupportedFmt,
		StatusNotLoggedIn:           body.CLI.StatusNotLoggedIn,
		LoginPhraseOr:               body.CLI.LoginPhraseOr,
		LoginPhraseComma:            body.CLI.LoginPhraseComma,
		LoginPhraseCommaOr:          body.CLI.LoginPhraseCommaOr,
		SetupEndpointFmt:            body.CLI.SetupEndpointFmt,
		SetupNoneFound:              body.CLI.SetupNoneFound,
		SetupUpdatedHeader:          body.CLI.SetupUpdatedHeader,
		SetupShellHeader:            body.CLI.SetupShellHeader,
		SetupJetbrainsHint:          body.CLI.SetupJetbrainsHint,
		SetupAPIEndpointFmt:         body.CLI.SetupAPIEndpointFmt,
		SetupNextHeader:             body.CLI.SetupNextHeader,
		SetupStepStart:              body.CLI.SetupStepStart,
		SetupStepRestart:            body.CLI.SetupStepRestart,
		SetupStepLogin:              body.CLI.SetupStepLogin,
		SetupStepDashboardFmt:       body.CLI.SetupStepDashboardFmt,
		SetupOptionalTLS:            body.CLI.SetupOptionalTLS,
		SetupTLSThen:                body.CLI.SetupTLSThen,
		SetupTLSGeneratedFmt:        body.CLI.SetupTLSGeneratedFmt,
		SetupTLSCertFmt:             body.CLI.SetupTLSCertFmt,
		SetupTLSKeyFmt:              body.CLI.SetupTLSKeyFmt,
		SetupTLSKeyPathFmt:          body.CLI.SetupTLSKeyPathFmt,
		SetupShellOpenAIFmt:         body.CLI.SetupShellOpenAIFmt,
		SetupShellAnthropicFmt:      body.CLI.SetupShellAnthropicFmt,
		SetupTLSStart:               body.CLI.SetupTLSStart,
		SetupTLSPointIDE:            body.CLI.SetupTLSPointIDE,
		SetupTLSEnvRequired:         body.CLI.SetupTLSEnvRequired,
		SetupTLSDaysInvalid:         body.CLI.SetupTLSDaysInvalid,
		PowClientRequired:           body.CLI.PowClientRequired,
		Pow428Missing:               body.CLI.Pow428Missing,
		PowSolveFailed:              body.CLI.PowSolveFailed,
		PowParseFailedFmt:           body.CLI.PowParseFailedFmt,
		SetupWroteFmt:               body.CLI.SetupWroteFmt,
		SetupProductCursor:          body.CLI.SetupProductCursor,
		SetupProductVSCode:          body.CLI.SetupProductVSCode,
		SetupProductWindsurf:        body.CLI.SetupProductWindsurf,
		SetupProductContinue:        body.CLI.SetupProductContinue,
		SetupProductZed:             body.CLI.SetupProductZed,
		SetupProductJetbrains:       body.CLI.SetupProductJetbrains,
		SetupContinueNoModel:        body.CLI.SetupContinueNoModel,
		StartRunningFmt:             body.CLI.StartRunningFmt,
		StartDashboardFmt:           body.CLI.StartDashboardFmt,
		StartModeFmt:                body.CLI.StartModeFmt,
		StartPointIDEFmt:            body.CLI.StartPointIDEFmt,
		StartRulesFmt:               body.CLI.StartRulesFmt,
		StartStopped:                body.CLI.StartStopped,
		ErrFmt:                      body.CLI.ErrFmt,
		ProxyFailedFmt:              body.CLI.ProxyFailedFmt,
		MainErrFmt:                  body.CLI.MainErrFmt,
		DaemonUnsupportedFmt:        body.CLI.DaemonUnsupportedFmt,
		DaemonLaunchdInstalledFmt:   body.CLI.DaemonLaunchdInstalledFmt,
		DaemonLaunchdMissing:        body.CLI.DaemonLaunchdMissing,
		DaemonSystemdInstalledFmt:   body.CLI.DaemonSystemdInstalledFmt,
		DaemonSystemdMissing:        body.CLI.DaemonSystemdMissing,
		DaemonWindowsMissing:        body.CLI.DaemonWindowsMissing,
		DaemonLaunchdOkFmt:          body.CLI.DaemonLaunchdOkFmt,
		DaemonLaunchdHint:           body.CLI.DaemonLaunchdHint,
		DaemonLaunchdRemoved:        body.CLI.DaemonLaunchdRemoved,
		DaemonSystemdOkFmt:          body.CLI.DaemonSystemdOkFmt,
		DaemonSystemdRemoved:        body.CLI.DaemonSystemdRemoved,
		DaemonWindowsOk:             body.CLI.DaemonWindowsOk,
		DaemonWindowsRemoved:        body.CLI.DaemonWindowsRemoved,
		TelemetryEnabled:            body.CLI.TelemetryEnabled,
		TelemetryDisableHint:        body.CLI.TelemetryDisableHint,
		TelemetryEnvHint:            body.CLI.TelemetryEnvHint,
		TelemetryDisabled:           body.CLI.TelemetryDisabled,
		TelemetryEnableHint:         body.CLI.TelemetryEnableHint,
		TelemetryDisabledOk:         body.CLI.TelemetryDisabledOk,
		TelemetryEnabledOk:          body.CLI.TelemetryEnabledOk,
		AutostartEnabled:            body.CLI.AutostartEnabled,
		AutostartDisabled:           body.CLI.AutostartDisabled,
		AutostartUnset:              body.CLI.AutostartUnset,
		AutostartEnableHint:         body.CLI.AutostartEnableHint,
		AutostartDisableHint:        body.CLI.AutostartDisableHint,
		AutostartEnabledOk:          body.CLI.AutostartEnabledOk,
		AutostartDisabledOk:         body.CLI.AutostartDisabledOk,
		AutostartDaemonHint:         body.CLI.AutostartDaemonHint,
		AutostartDaemonSkippedOff:   body.CLI.AutostartDaemonSkippedOff,
		AutostartDaemonSkippedUnset: body.CLI.AutostartDaemonSkippedUnset,
		AutostartSkippedManaged:     body.CLI.AutostartSkippedManaged,
		AutostartSkippedDNT:         body.CLI.AutostartSkippedDNT,
		AutostartManagedOffHint:     body.CLI.AutostartManagedOffHint,
		AutostartSyncDaemonOk:       body.CLI.AutostartSyncDaemonOk,
		DefaultAutoStartWithIDE:     body.CLI.DefaultAutoStartWithIDE,
		AutostartPrefPollMs:         body.CLI.AutostartPrefPollMs,
		AutostartEnforcerStopped:    body.CLI.AutostartEnforcerStopped,
		AutostartPrefPollMinMs:      body.CLI.AutostartPrefPollMinMs,
		AutostartPrefPollMaxMs:      body.CLI.AutostartPrefPollMaxMs,
		AutostartTimeoutMinMs:       body.CLI.AutostartTimeoutMinMs,
		AutostartTimeoutMaxMs:       body.CLI.AutostartTimeoutMaxMs,
		DaemonRestartSec:            body.CLI.DaemonRestartSec,
		DaemonRestartMinSec:         body.CLI.DaemonRestartMinSec,
		DaemonRestartMaxSec:         body.CLI.DaemonRestartMaxSec,
		StopOnDisableOk:             body.CLI.StopOnDisableOk,
		ProxyHealthURL:              body.CLI.ProxyHealthURL,
		ProxyShutdownURL:            body.CLI.ProxyShutdownURL,
		ProxyHealthTimeoutMs:        body.CLI.ProxyHealthTimeoutMs,
		ProxyShutdownTimeoutMs:      body.CLI.ProxyShutdownTimeoutMs,
		HTTPShutdownTimeoutMs:       body.CLI.HTTPShutdownTimeoutMs,
		ProxyFallbackUncompressed:   body.CLI.ProxyFallbackUncompressed,
		ProxyActiveFileProtection:   body.CLI.ProxyActiveFileProtection,
		StopOk:                      body.CLI.StopOk,
		StopFailedFmt:               body.CLI.StopFailedFmt,
		StopURLMissing:              body.CLI.StopURLMissing,
		ConfigGetAutostartUnset:     body.CLI.ConfigGetAutostartUnset,
		CompressFileRequired:        body.CLI.CompressFileRequired,
		CompressModeRequired:        body.CLI.CompressModeRequired,
		CompressModeInvalid:         body.CLI.CompressModeInvalid,
		TreesitterRequired:          body.CLI.TreesitterRequired,
		CompressDeepPrefsRequired:   body.CLI.CompressDeepPrefsRequired,
		CompressDeepResultFmt:       body.CLI.CompressDeepResultFmt,
		CompressWroteFmt:            body.CLI.CompressWroteFmt,
		StatsLiveHeader:             body.CLI.StatsLiveHeader,
		StatsLiveOfflineFmt:         body.CLI.StatsLiveOfflineFmt,
		StatsURLFmt:                 body.CLI.StatsURLFmt,
		StatsURLMissing:             body.CLI.StatsURLMissing,
		StatsSqliteTodayFmt:         body.CLI.StatsSqliteTodayFmt,
		StatsSqliteSeriesHeader:     body.CLI.StatsSqliteSeriesHeader,
		StatsSqliteSeriesRowFmt:     body.CLI.StatsSqliteSeriesRowFmt,
		StatsTuiTip:                 body.CLI.StatsTuiTip,
		StatsDeepStatusFmt:          body.CLI.StatsDeepStatusFmt,
		StatsDeepStageFmt:           body.CLI.StatsDeepStageFmt,
		StatsLastRequestFmt:         body.CLI.StatsLastRequestFmt,
		StatsDoorFmt:                body.CLI.StatsDoorFmt,
		StatsDashboardTipFmt:        body.CLI.StatsDashboardTipFmt,
		StatusAPIFmt:                body.CLI.StatusAPIFmt,
		QuotaExhaustedTitle:         body.CLI.QuotaExhaustedTitle,
		QuotaExhaustedBody:          body.CLI.QuotaExhaustedBody,
		QuotaUpgradeHintFmt:         body.CLI.QuotaUpgradeHintFmt,
		QuotaUpgradeOpening:         body.CLI.QuotaUpgradeOpening,
		QuotaUpgradeURLMissing:      body.CLI.QuotaUpgradeURLMissing,
		DeepBootstrapReqs:           body.CLI.DeepBootstrapReqs,
		DeepBootstrapPip:            body.CLI.DeepBootstrapPip,
		DeepRequirementsMissing:     body.CLI.DeepRequirementsMissing,
		DeepAutoInstall:             body.CLI.DeepAutoInstall,
		DeepBinEnvMissingFmt:        body.CLI.DeepBinEnvMissingFmt,
		DeepBinMissing:              body.CLI.DeepBinMissing,
		DeepPyEnvMissingFmt:         body.CLI.DeepPyEnvMissingFmt,
		DeepPyMissing:               body.CLI.DeepPyMissing,
		DeepLLMMissingFmt:           body.CLI.DeepLLMMissingFmt,
		DeepPipFailed:               body.CLI.DeepPipFailed,
		DeepTargetRequired:          body.CLI.DeepTargetRequired,
		DeepEngineRequired:          body.CLI.DeepEngineRequired,
		DeepQuestionRequired:        body.CLI.DeepQuestionRequired,
		DeepOptimizeFmt:             body.CLI.DeepOptimizeFmt,
		DeepTimeout:                 body.CLI.DeepTimeout,
		DeepParseFmt:                body.CLI.DeepParseFmt,
		ProxyDeepFailedFmt:          body.CLI.ProxyDeepFailedFmt,
		ProxyLiveDeepModeFmt:        body.CLI.ProxyLiveDeepModeFmt,
		ProxyLiveDeepEnabledFmt:     body.CLI.ProxyLiveDeepEnabledFmt,
		ProxyDeepCompactStub:        body.CLI.ProxyDeepCompactStub,
		ProxyDeepChromeRequired:     body.CLI.ProxyDeepChromeRequired,
		ProxyDeepPrefsRequired:      body.CLI.ProxyDeepPrefsRequired,
		ProxyDeepSkippedMinFmt:      body.CLI.ProxyDeepSkippedMinFmt,
		ProxyDeepSkippedStream:      body.CLI.ProxyDeepSkippedStream,
		ProxyDeepOOMSkipped:         body.CLI.ProxyDeepOOMSkipped,
		ProxyDeepRuntimeRequired:    body.CLI.ProxyDeepRuntimeRequired,
		ProxyAdapterRequired:        body.CLI.ProxyAdapterRequired,
		ProxyAdapterUnknownDialectFmt: body.CLI.ProxyAdapterUnknownDialectFmt,
		ProxyAdapterModelRequired:   body.CLI.ProxyAdapterModelRequired,
		ProxyAdapterAliasRequiredFmt: body.CLI.ProxyAdapterAliasRequiredFmt,
		ProxyAdapterTranslateFmt:    body.CLI.ProxyAdapterTranslateFmt,
		ProxyAdapterAuthFmt:         body.CLI.ProxyAdapterAuthFmt,
		ProxyAdapterResponseFmt:     body.CLI.ProxyAdapterResponseFmt,
		ProxyAdapterEnabledFmt:      body.CLI.ProxyAdapterEnabledFmt,
		ProxyModelsNotSynced:        body.CLI.ProxyModelsNotSynced,
		ProxyModelsMethod:           body.CLI.ProxyModelsMethod,
		ProxyErrBaseURLDoubleV1:     body.CLI.ProxyErrBaseURLDoubleV1,
		ProxyErrModelNotFoundFmt:    body.CLI.ProxyErrModelNotFoundFmt,
		ProxyErrUnknownModelFmt:     body.CLI.ProxyErrUnknownModelFmt,
		ProxyErrUpstreamAuthFmt:     body.CLI.ProxyErrUpstreamAuthFmt,
		ProxyErrUpstreamQuotaFmt:    body.CLI.ProxyErrUpstreamQuotaFmt,
		ProxyErrUpstreamUnavailableFmt: body.CLI.ProxyErrUpstreamUnavailableFmt,
		ProxyModelsUpstreamAdapterRequired: body.CLI.ProxyModelsUpstreamAdapterRequired,
		ProxyModelsUpstreamNotFoundFmt:     body.CLI.ProxyModelsUpstreamNotFoundFmt,
		ProxyModelsUpstreamAuthRequired:    body.CLI.ProxyModelsUpstreamAuthRequired,
		ProxyModelsUpstreamUnsupportedFmt:  body.CLI.ProxyModelsUpstreamUnsupportedFmt,
		SetupClaudeCodeBlockFmt:     body.CLI.SetupClaudeCodeBlockFmt,
		SetupContinueBlockFmt:       body.CLI.SetupContinueBlockFmt,
		SetupVSCodeChatBlockFmt:     body.CLI.SetupVSCodeChatBlockFmt,
		ProxyDoorOpenAI:             body.CLI.ProxyDoorOpenAI,
		ProxyDoorAnthropic:          body.CLI.ProxyDoorAnthropic,
		PrefsEngineInvalid:          body.CLI.PrefsEngineInvalid,
		PrefsTargetNonNeg:           body.CLI.PrefsTargetNonNeg,
		PrefsTierRequired:           body.CLI.PrefsTierRequired,
		PrefsEngineRequired:         body.CLI.PrefsEngineRequired,
		PrefsTargetRequired:         body.CLI.PrefsTargetRequired,
		ConfigGetTierFmt:            body.CLI.ConfigGetTierFmt,
		ConfigGetEngineFmt:          body.CLI.ConfigGetEngineFmt,
		ConfigGetTargetFmt:          body.CLI.ConfigGetTargetFmt,
		ConfigGetAutostartFmt:       body.CLI.ConfigGetAutostartFmt,
		ConfigUnknownKeyFmt:         body.CLI.ConfigUnknownKeyFmt,
		ConfigTargetPositive:        body.CLI.ConfigTargetPositive,
		ConfigSavedFmt:              body.CLI.ConfigSavedFmt,
		ConfigNotLoggedIn:           body.CLI.ConfigNotLoggedIn,
		ConfigSyncFailedFmt:         body.CLI.ConfigSyncFailedFmt,
		ConfigSyncedFmt:             body.CLI.ConfigSyncedFmt,
		HelpRootShort:               body.CLI.HelpRootShort,
		HelpRootLong:                body.CLI.HelpRootLong,
		HelpGroupEveryday:           body.CLI.HelpGroupEveryday,
		HelpGroupAdvanced:           body.CLI.HelpGroupAdvanced,
		HelpStartShort:              body.CLI.HelpStartShort,
		HelpStartLong:               body.CLI.HelpStartLong,
		HelpStartFlagPort:           body.CLI.HelpStartFlagPort,
		HelpLoginShort:              body.CLI.HelpLoginShort,
		HelpLogoutShort:             body.CLI.HelpLogoutShort,
		HelpUninstallShort:          body.CLI.HelpUninstallShort,
		HelpUninstallLong:           body.CLI.HelpUninstallLong,
		HelpUninstallFlagKeepData:   body.CLI.HelpUninstallFlagKeepData,
		UninstallStarting:           body.CLI.UninstallStarting,
		UninstallStopped:            body.CLI.UninstallStopped,
		UninstallDaemonOk:           body.CLI.UninstallDaemonOk,
		UninstallLogoutOk:           body.CLI.UninstallLogoutOk,
		UninstallSetupRevertedFmt:   body.CLI.UninstallSetupRevertedFmt,
		UninstallSetupNone:          body.CLI.UninstallSetupNone,
		UninstallConfigRemovedFmt:   body.CLI.UninstallConfigRemovedFmt,
		UninstallLogsRemoved:        body.CLI.UninstallLogsRemoved,
		UninstallDeepOk:             body.CLI.UninstallDeepOk,
		UninstallDeepSkipped:        body.CLI.UninstallDeepSkipped,
		UninstallSidecarRemovedFmt:  body.CLI.UninstallSidecarRemovedFmt,
		UninstallBinaryRemovedFmt:   body.CLI.UninstallBinaryRemovedFmt,
		UninstallBinaryManualFmt:    body.CLI.UninstallBinaryManualFmt,
		UninstallPathRemoved:        body.CLI.UninstallPathRemoved,
		UninstallDone:               body.CLI.UninstallDone,
		UninstallNextExt:            body.CLI.UninstallNextExt,
		UninstallNextPkgBrew:        body.CLI.UninstallNextPkgBrew,
		UninstallNextPkgScoop:       body.CLI.UninstallNextPkgScoop,
		UninstallNextPkgWinget:      body.CLI.UninstallNextPkgWinget,
		UninstallHfHubDirs:          body.CLI.UninstallHfHubDirs,
		UninstallHfCacheRel:         body.CLI.UninstallHfCacheRel,
		UninstallHfHubSubdir:        body.CLI.UninstallHfHubSubdir,
		UninstallHomeDirsRel:        body.CLI.UninstallHomeDirsRel,
		UninstallHomeDirsMissing:    body.CLI.UninstallHomeDirsMissing,
		UninstallDarwinLogRels:      body.CLI.UninstallDarwinLogRels,
		UninstallPipPackages:        body.CLI.UninstallPipPackages,
		UninstallPipBins:            body.CLI.UninstallPipBins,
		UninstallPythonBins:         body.CLI.UninstallPythonBins,
		UninstallPipModule:          body.CLI.UninstallPipModule,
		UninstallSidecarNames:       body.CLI.UninstallSidecarNames,
		UninstallArpDisplayName:     body.CLI.UninstallArpDisplayName,
		UninstallArpPublisher:       body.CLI.UninstallArpPublisher,
		UninstallArpRegKey:          body.CLI.UninstallArpRegKey,
		UninstallArpRemoved:         body.CLI.UninstallArpRemoved,
		HelpStatusShort:             body.CLI.HelpStatusShort,
		HelpStatusLong:              body.CLI.HelpStatusLong,
		HelpUpgradeShort:            body.CLI.HelpUpgradeShort,
		HelpStatsShort:              body.CLI.HelpStatsShort,
		HelpStatsLong:               body.CLI.HelpStatsLong,
		HelpStatsFlagTui:            body.CLI.HelpStatsFlagTui,
		HelpSetupShort:              body.CLI.HelpSetupShort,
		HelpSetupLong:               body.CLI.HelpSetupLong,
		HelpSetupFlagTls:            body.CLI.HelpSetupFlagTls,
		HelpVersionShort:            body.CLI.HelpVersionShort,
		HelpDaemonShort:             body.CLI.HelpDaemonShort,
		HelpDaemonInstallShort:      body.CLI.HelpDaemonInstallShort,
		HelpDaemonUninstallShort:    body.CLI.HelpDaemonUninstallShort,
		HelpDaemonStatusShort:       body.CLI.HelpDaemonStatusShort,
		HelpDaemonRunShort:          body.CLI.HelpDaemonRunShort,
		HelpDaemonServiceDesc:       body.CLI.HelpDaemonServiceDesc,
		HelpTelemetryShort:          body.CLI.HelpTelemetryShort,
		HelpTelemetryStatusShort:    body.CLI.HelpTelemetryStatusShort,
		HelpTelemetryDisableShort:   body.CLI.HelpTelemetryDisableShort,
		HelpTelemetryEnableShort:    body.CLI.HelpTelemetryEnableShort,
		HelpAutostartShort:          body.CLI.HelpAutostartShort,
		HelpAutostartStatusShort:    body.CLI.HelpAutostartStatusShort,
		HelpAutostartEnableShort:    body.CLI.HelpAutostartEnableShort,
		HelpAutostartDisableShort:   body.CLI.HelpAutostartDisableShort,
		HelpStopShort:               body.CLI.HelpStopShort,
		HelpTuiShort:                body.CLI.HelpTuiShort,
		HelpTuiLong:                 body.CLI.HelpTuiLong,
		HelpCompressShort:           body.CLI.HelpCompressShort,
		HelpCompressLong:            body.CLI.HelpCompressLong,
		HelpCompressFlagMode:        body.CLI.HelpCompressFlagMode,
		HelpCompressFlagDeep:        body.CLI.HelpCompressFlagDeep,
		HelpCompressFlagEngine:      body.CLI.HelpCompressFlagEngine,
		HelpCompressFlagQuestion:    body.CLI.HelpCompressFlagQuestion,
		HelpCompressFlagTarget:      body.CLI.HelpCompressFlagTarget,
		HelpCompressFlagBootstrap:   body.CLI.HelpCompressFlagBootstrap,
		HelpCompressFlagOut:         body.CLI.HelpCompressFlagOut,
		HelpConfigShort:             body.CLI.HelpConfigShort,
		HelpConfigGetShort:          body.CLI.HelpConfigGetShort,
		HelpConfigSetShort:          body.CLI.HelpConfigSetShort,
		HelpConfigSyncShort:         body.CLI.HelpConfigSyncShort,
		providerLabels:              labels,
	}
	return out
}

func setPOWChrome(c cliChrome) {
	powClientRequired = strings.TrimSpace(c.PowClientRequired)
	pow428Missing = strings.TrimSpace(c.Pow428Missing)
	powSolveFailed = strings.TrimSpace(c.PowSolveFailed)
	powParseFailedFmt = strings.TrimSpace(c.PowParseFailedFmt)
}

func loadCLIChrome() cliChrome {
	cfg, err := config.LoadLocal()
	if err != nil {
		return cliChrome{}
	}
	return fetchCLIChrome(cfg)
}

func fetchLocalChrome(cfg config.Local) localchrome.Chrome {
	base := strings.TrimRight(strings.TrimSpace(cfg.APIBaseURL), "/")
	if base == "" {
		return localchrome.Chrome{}
	}
	client := newCLIHTTPClient(cfg)
	res, err := client.Get(base + "/api/v1/public/auth-providers")
	if err != nil {
		return localchrome.Chrome{}
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return localchrome.Chrome{}
	}
	var body struct {
		Local localchrome.Chrome `json:"local"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return localchrome.Chrome{}
	}
	return body.Local
}

type cliChrome struct {
	PathCLIAuth                 string
	LoginSignInFmt              string
	LoginOpeningBrowser         string
	LoginPasteKey               string
	LoginKeyRequired            string
	LoginSuccess                string
	LogoutSuccess               string
	LogoutAlready               string
	ChromeUnavailable           string
	BrowserUnsupportedFmt       string
	StatusNotLoggedIn           string
	LoginPhraseOr               string
	LoginPhraseComma            string
	LoginPhraseCommaOr          string
	SetupEndpointFmt            string
	SetupNoneFound              string
	SetupUpdatedHeader          string
	SetupShellHeader            string
	SetupJetbrainsHint          string
	SetupAPIEndpointFmt         string
	SetupNextHeader             string
	SetupStepStart              string
	SetupStepRestart            string
	SetupStepLogin              string
	SetupStepDashboardFmt       string
	SetupOptionalTLS            string
	SetupTLSThen                string
	SetupTLSGeneratedFmt        string
	SetupTLSCertFmt             string
	SetupTLSKeyFmt              string
	SetupTLSKeyPathFmt          string
	SetupShellOpenAIFmt         string
	SetupShellAnthropicFmt      string
	SetupTLSStart               string
	SetupTLSPointIDE            string
	SetupTLSEnvRequired         string
	SetupTLSDaysInvalid         string
	PowClientRequired           string
	Pow428Missing               string
	PowSolveFailed              string
	PowParseFailedFmt           string
	SetupWroteFmt               string
	SetupProductCursor          string
	SetupProductVSCode          string
	SetupProductWindsurf        string
	SetupProductContinue        string
	SetupProductZed             string
	SetupProductJetbrains       string
	SetupContinueNoModel        string
	StartRunningFmt             string
	StartDashboardFmt           string
	StartModeFmt                string
	StartPointIDEFmt            string
	StartRulesFmt               string
	StartStopped                string
	ErrFmt                      string
	ProxyFailedFmt              string
	MainErrFmt                  string
	DaemonUnsupportedFmt        string
	DaemonLaunchdInstalledFmt   string
	DaemonLaunchdMissing        string
	DaemonSystemdInstalledFmt   string
	DaemonSystemdMissing        string
	DaemonWindowsMissing        string
	DaemonLaunchdOkFmt          string
	DaemonLaunchdHint           string
	DaemonLaunchdRemoved        string
	DaemonSystemdOkFmt          string
	DaemonSystemdRemoved        string
	DaemonWindowsOk             string
	DaemonWindowsRemoved        string
	TelemetryEnabled            string
	TelemetryDisableHint        string
	TelemetryEnvHint            string
	TelemetryDisabled           string
	TelemetryEnableHint         string
	TelemetryDisabledOk         string
	TelemetryEnabledOk          string
	AutostartEnabled            string
	AutostartDisabled           string
	AutostartUnset              string
	AutostartEnableHint         string
	AutostartDisableHint        string
	AutostartEnabledOk          string
	AutostartDisabledOk         string
	AutostartDaemonHint         string
	AutostartDaemonSkippedOff   string
	AutostartDaemonSkippedUnset string
	AutostartSkippedManaged     string
	AutostartSkippedDNT         string
	AutostartManagedOffHint     string
	AutostartSyncDaemonOk       string
	DefaultAutoStartWithIDE     string
	AutostartPrefPollMs         string
	AutostartEnforcerStopped    string
	AutostartPrefPollMinMs      string
	AutostartPrefPollMaxMs      string
	AutostartTimeoutMinMs       string
	AutostartTimeoutMaxMs       string
	DaemonRestartSec            string
	DaemonRestartMinSec         string
	DaemonRestartMaxSec         string
	StopOnDisableOk             string
	ProxyHealthURL              string
	ProxyShutdownURL            string
	ProxyHealthTimeoutMs        string
	ProxyShutdownTimeoutMs      string
	HTTPShutdownTimeoutMs       string
	ProxyFallbackUncompressed   string
	ProxyActiveFileProtection   string
	StopOk                      string
	StopFailedFmt               string
	StopURLMissing              string
	ConfigGetAutostartUnset     string
	CompressFileRequired        string
	CompressModeRequired        string
	CompressModeInvalid         string
	TreesitterRequired          string
	CompressDeepPrefsRequired   string
	CompressDeepResultFmt       string
	CompressWroteFmt            string
	StatsLiveHeader             string
	StatsLiveOfflineFmt         string
	StatsURLFmt                 string
	StatsURLMissing             string
	StatsSqliteTodayFmt         string
	StatsSqliteSeriesHeader     string
	StatsSqliteSeriesRowFmt     string
	StatsTuiTip                 string
	StatsDeepStatusFmt          string
	StatsDeepStageFmt           string
	StatsLastRequestFmt         string
	StatsDoorFmt                string
	StatsDashboardTipFmt        string
	StatusAPIFmt                string
	QuotaExhaustedTitle         string
	QuotaExhaustedBody          string
	QuotaUpgradeHintFmt         string
	QuotaUpgradeOpening         string
	QuotaUpgradeURLMissing      string
	DeepBootstrapReqs           string
	DeepBootstrapPip            string
	DeepRequirementsMissing     string
	DeepAutoInstall             string
	DeepBinEnvMissingFmt        string
	DeepBinMissing              string
	DeepPyEnvMissingFmt         string
	DeepPyMissing               string
	DeepLLMMissingFmt           string
	DeepPipFailed               string
	DeepTargetRequired          string
	DeepEngineRequired          string
	DeepQuestionRequired        string
	DeepOptimizeFmt             string
	DeepTimeout                 string
	DeepParseFmt                string
	ProxyDeepFailedFmt          string
	ProxyLiveDeepModeFmt        string
	ProxyLiveDeepEnabledFmt     string
	ProxyDeepCompactStub        string
	ProxyDeepChromeRequired     string
	ProxyDeepPrefsRequired      string
	ProxyDeepSkippedMinFmt      string
	ProxyDeepSkippedStream      string
	ProxyDeepOOMSkipped         string
	ProxyDeepRuntimeRequired    string
	ProxyAdapterRequired        string
	ProxyAdapterUnknownDialectFmt string
	ProxyAdapterModelRequired   string
	ProxyAdapterAliasRequiredFmt string
	ProxyAdapterTranslateFmt    string
	ProxyAdapterAuthFmt         string
	ProxyAdapterResponseFmt     string
	ProxyAdapterEnabledFmt      string
	ProxyModelsNotSynced        string
	ProxyModelsMethod           string
	ProxyErrBaseURLDoubleV1     string
	ProxyErrModelNotFoundFmt    string
	ProxyErrUnknownModelFmt     string
	ProxyErrUpstreamAuthFmt     string
	ProxyErrUpstreamQuotaFmt    string
	ProxyErrUpstreamUnavailableFmt string
	ProxyModelsUpstreamAdapterRequired string
	ProxyModelsUpstreamNotFoundFmt     string
	ProxyModelsUpstreamAuthRequired    string
	ProxyModelsUpstreamUnsupportedFmt  string
	SetupClaudeCodeBlockFmt     string
	SetupContinueBlockFmt       string
	SetupVSCodeChatBlockFmt     string
	ProxyDoorOpenAI             string
	ProxyDoorAnthropic          string
	PrefsEngineInvalid          string
	PrefsTargetNonNeg           string
	PrefsTierRequired           string
	PrefsEngineRequired         string
	PrefsTargetRequired         string
	ConfigGetTierFmt            string
	ConfigGetEngineFmt          string
	ConfigGetTargetFmt          string
	ConfigGetAutostartFmt       string
	ConfigUnknownKeyFmt         string
	ConfigTargetPositive        string
	ConfigSavedFmt              string
	ConfigNotLoggedIn           string
	ConfigSyncFailedFmt         string
	ConfigSyncedFmt             string
	HelpRootShort               string
	HelpRootLong                string
	HelpGroupEveryday           string
	HelpGroupAdvanced           string
	HelpStartShort              string
	HelpStartLong               string
	HelpStartFlagPort           string
	HelpLoginShort              string
	HelpLogoutShort             string
	HelpUninstallShort          string
	HelpUninstallLong           string
	HelpUninstallFlagKeepData   string
	UninstallStarting           string
	UninstallStopped            string
	UninstallDaemonOk           string
	UninstallLogoutOk           string
	UninstallSetupRevertedFmt   string
	UninstallSetupNone          string
	UninstallConfigRemovedFmt   string
	UninstallLogsRemoved        string
	UninstallDeepOk             string
	UninstallDeepSkipped        string
	UninstallSidecarRemovedFmt  string
	UninstallBinaryRemovedFmt   string
	UninstallBinaryManualFmt    string
	UninstallPathRemoved        string
	UninstallDone               string
	UninstallNextExt            string
	UninstallNextPkgBrew        string
	UninstallNextPkgScoop       string
	UninstallNextPkgWinget      string
	UninstallHfHubDirs          string
	UninstallHfCacheRel         string
	UninstallHfHubSubdir        string
	UninstallHomeDirsRel        string
	UninstallHomeDirsMissing    string
	UninstallDarwinLogRels      string
	UninstallPipPackages        string
	UninstallPipBins            string
	UninstallPythonBins         string
	UninstallPipModule          string
	UninstallSidecarNames       string
	UninstallArpDisplayName     string
	UninstallArpPublisher       string
	UninstallArpRegKey          string
	UninstallArpRemoved         string
	HelpStatusShort             string
	HelpStatusLong              string
	HelpUpgradeShort            string
	HelpStatsShort              string
	HelpStatsLong               string
	HelpStatsFlagTui            string
	HelpSetupShort              string
	HelpSetupLong               string
	HelpSetupFlagTls            string
	HelpVersionShort            string
	HelpDaemonShort             string
	HelpDaemonInstallShort      string
	HelpDaemonUninstallShort    string
	HelpDaemonStatusShort       string
	HelpDaemonRunShort          string
	HelpDaemonServiceDesc       string
	HelpTelemetryShort          string
	HelpTelemetryStatusShort    string
	HelpTelemetryDisableShort   string
	HelpTelemetryEnableShort    string
	HelpAutostartShort          string
	HelpAutostartStatusShort    string
	HelpAutostartEnableShort    string
	HelpAutostartDisableShort   string
	HelpStopShort               string
	HelpTuiShort                string
	HelpTuiLong                 string
	HelpCompressShort           string
	HelpCompressLong            string
	HelpCompressFlagMode        string
	HelpCompressFlagDeep        string
	HelpCompressFlagEngine      string
	HelpCompressFlagQuestion    string
	HelpCompressFlagTarget      string
	HelpCompressFlagBootstrap   string
	HelpCompressFlagOut         string
	HelpConfigShort             string
	HelpConfigGetShort          string
	HelpConfigSetShort          string
	HelpConfigSyncShort         string
	providerLabels              []string
}

func (c cliChrome) fail(msg string) error {
	if msg != "" {
		return fmt.Errorf("%s", msg)
	}
	if c.ChromeUnavailable != "" {
		return fmt.Errorf("%s", c.ChromeUnavailable)
	}
	return clierr.ErrChromeUnavailable
}

func (c cliChrome) failFmt(fmtStr string, args ...any) error {
	if fmtStr != "" {
		return fmt.Errorf(fmtStr, args...)
	}
	return c.fail("")
}

func (c cliChrome) providerPhrase() string {
	labels := c.providerLabels
	if len(labels) == 0 {
		return ""
	}
	or := c.LoginPhraseOr
	comma := c.LoginPhraseComma
	commaOr := c.LoginPhraseCommaOr
	if or == "" || comma == "" || commaOr == "" {
		// Incomplete chrome: do not invent joiner copy.
		return strings.Join(labels, " ")
	}
	switch len(labels) {
	case 1:
		return labels[0]
	case 2:
		return labels[0] + or + labels[1]
	default:
		return strings.Join(labels[:len(labels)-1], comma) + commaOr + labels[len(labels)-1]
	}
}

func openBrowser(url string, chrome cliChrome) error {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		return chrome.failFmt(chrome.BrowserUnsupportedFmt, runtime.GOOS)
	}
	return err
}

func init() {
	_ = os.Stderr
	statsCmd.Flags().BoolVar(&statsTUI, "tui", false, "")
}
