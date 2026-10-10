package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/usetrim/trim/cli/internal/config"
	"github.com/usetrim/trim/cli/internal/deepopt"
	"github.com/usetrim/trim/cli/internal/metrics"
	"github.com/usetrim/trim/cli/internal/projectconfig"
	"github.com/usetrim/trim/cli/internal/storage"
	"github.com/usetrim/trim/cli/internal/telemetry"
	"github.com/usetrim/trim/server/pkg/proxy"
	"github.com/usetrim/trim/server/pkg/provideradapt"
	"github.com/usetrim/trim/server/pkg/trimmer"
)

var portFlag int

var startCmd = &cobra.Command{
	Use:     "start",
	Aliases: []string{"proxy"},
	Short:   "",
	Long:    "",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadLocal()
		if err != nil {
			return err
		}
		listenPort := cfg.Port
		if cmd.Flags().Changed("port") {
			listenPort = fmt.Sprintf("%d", portFlag)
		}

		proj, err := projectconfig.Load("")
		if err != nil {
			return err
		}
		chrome := fetchCLIChrome(cfg)
		ensureWindowsARPRegistration(chrome)
		// Prefer attach: if a healthy proxy already owns the port, do not double-bind.
		healthURL := strings.TrimSpace(chrome.ProxyHealthURL)
		if healthURL != "" {
			to, err := requireChromeTimeout(chrome.ProxyHealthTimeoutMs, "CLI_PROXY_HEALTH_TIMEOUT_MS", chrome)
			if err != nil {
				return err
			}
			if proxyHealthOK(healthURL, to) {
				// Extension/daemon enforcer path: stay as sidecar so dashboard uncheck still stops.
				// Manual `trim start` exits 0 (user-owned process already up).
				if os.Getenv("TRIM_AUTOSTART_ENFORCER") == "1" {
					return runAutostartAttachEnforcer(cfg, chrome, healthURL)
				}
				return nil
			}
		}
		if len(proj.CustomQueries) > 0 && !trimmer.TreeSitterAvailable() {
			if chrome.TreesitterRequired != "" {
				return fmt.Errorf("%s", chrome.TreesitterRequired)
			}
			return chrome.fail("")
		}

		mode := projectconfig.NormalizeMode(proj.Mode)
		if mode == "" {
			mode = projectconfig.NormalizeMode(cfg.CompressionMode)
		}
		resolved := projectconfig.Config{
			Mode:                 mode,
			CustomMinLines:       proj.CustomMinLines,
			CustomLogsOnly:       proj.CustomLogsOnly,
			ActiveFileProtection: proj.ActiveFileProtection,
			MaxLogBytes:          proj.MaxLogBytes,
			LogCompactMinBytes:   proj.LogCompactMinBytes,
			LogCompactMaxLines:   proj.LogCompactMaxLines,
			LogNoiseSubstrings:   proj.LogNoiseSubstrings,
			IgnorePaths:          proj.IgnorePaths,
			NeverTrimPaths:       proj.NeverTrimPaths,
			CustomQueries:        proj.CustomQueries,
			HistoryKeepTurns:     proj.HistoryKeepTurns,
		}
		historyKeep := proj.HistoryKeepTurns
		if historyKeep == 0 {
			historyKeep = cfg.HistoryKeepTurns
		}
		activeFile := proj.ActiveFileProtection
		if !proj.ActiveFileProtectionSet {
			activeFile = chromeProxyActiveFileProtection(chrome)
		}

		// Pull account preferences (compression_tier / deep_* / adapters) before binding Deep.
		_ = refreshLocalPreferencesFromCloud(cfg, chrome)
		liveMode, deepOptimize, errDeepFmt, liveErr := prepareLiveDeep(cfg, chrome)
		if liveErr != nil {
			return liveErr
		}
		adapterReg, _ := loadProviderAdapterRegistry(chrome)
		if adapterReg != nil && chrome.ProxyAdapterEnabledFmt != "" {
			fmt.Printf(chrome.ProxyAdapterEnabledFmt+"\n", adapterReg.EnabledCount())
		}

		httpServer := &http.Server{
			Addr:              ":" + listenPort,
			ReadHeaderTimeout: time.Duration(cfg.HTTPReadHeaderTimeoutSec) * time.Second,
		}
		srv := proxy.NewServer(proxy.Options{
			UpstreamOpenAI:            cfg.UpstreamOpenAI,
			UpstreamAnthropic:         cfg.UpstreamAnthropic,
			UpstreamOpenAIFailover:    cfg.UpstreamOpenAIFailover,
			UpstreamAnthropicFailover: cfg.UpstreamAnthropicFailover,
			NeverTrim:                 proj.MustNeverTrim,
			MaxLogBytes:               proj.MaxLogBytes,
			LogCompactMinBytes:        proj.LogCompactMinBytes,
			LogCompactMaxLines:        proj.LogCompactMaxLines,
			LogNoiseSubstrings:        proj.LogNoiseSubstrings,
			ActiveFileProtection:      activeFile,
			CompressionMode:           mode,
			MinLines:                  resolved.MinLinesForMode(),
			DisableSkeletonize:        !resolved.SkeletonizeEnabled(),
			AlwaysCompactLogs:         resolved.AlwaysCompactLogs(),
			CustomQueries:             proj.CustomQueries,
			FallbackUncompressed:      chromeProxyFallbackUncompressed(chrome),
			CheapModel:                cfg.CheapModel,
			RouteMaxTokens:            cfg.RouteMaxTokens,
			SavingsUsdPerMTok:         cfg.SavingsUsdPerMTok,
			HistoryKeepTurns:          historyKeep,
			UpstreamHTTPTimeout:       time.Duration(cfg.ProxyHTTPTimeoutSec) * time.Second,
			LiveModeLabel:             liveMode,
			DeepOptimize:              deepOptimize,
			ErrDeepFmt:                errDeepFmt,
			ProviderAdapters:          adapterReg,
			AnthropicWorkspaceID:      cfg.AnthropicWorkspaceID,
			OpenAIOrganization:        cfg.OpenAIOrganization,
			OpenAIProject:             cfg.OpenAIProject,
			DoorOpenAI:                chrome.ProxyDoorOpenAI,
			DoorAnthropic:             chrome.ProxyDoorAnthropic,
			AdapterTranslateFmt:       chrome.ProxyAdapterTranslateFmt,
			AdapterAuthFmt:            chrome.ProxyAdapterAuthFmt,
			AdapterResponseFmt:        chrome.ProxyAdapterResponseFmt,
			ModelsNotSynced:           chrome.ProxyModelsNotSynced,
			ModelsMethod:              chrome.ProxyModelsMethod,
			BaseURLDoubleV1:           chrome.ProxyErrBaseURLDoubleV1,
			ModelNotFoundFmt:          chrome.ProxyErrModelNotFoundFmt,
			UnknownModelFmt:           chrome.ProxyErrUnknownModelFmt,
			UpstreamAuthFmt:           chrome.ProxyErrUpstreamAuthFmt,
			UpstreamQuotaFmt:          chrome.ProxyErrUpstreamQuotaFmt,
			UpstreamUnavailableFmt:    chrome.ProxyErrUpstreamUnavailableFmt,
			ModelsUpstreamAdapterRequired: chrome.ProxyModelsUpstreamAdapterRequired,
			ModelsUpstreamNotFoundFmt:     chrome.ProxyModelsUpstreamNotFoundFmt,
			ModelsUpstreamAuthRequired:    chrome.ProxyModelsUpstreamAuthRequired,
			ModelsUpstreamUnsupportedFmt:  chrome.ProxyModelsUpstreamUnsupportedFmt,
			OnEvent: func(_ *http.Request, model string, before, after int, latencyMs float64, status, mode, _ string) {
				go func() {
					if store, err := metrics.OpenDefault(); err == nil && store != nil {
						_ = store.Record(model, before, after, latencyMs, status, mode)
					}
					reportEvent(cfg, model, before, after, latencyMs, status, mode)
				}()
			},
			SeriesProvider: func(days int) (any, error) {
				store, err := metrics.OpenDefault()
				if err != nil || store == nil {
					return nil, err
				}
				return store.Series(days, cfg.SavingsUsdPerMTok)
			},
			DefaultSeriesDays: cfg.StatsSeriesDays,
			PreviewMaxChars:   cfg.ProxyPreviewMaxChars,
			Chrome:            fetchLocalChrome(cfg),
			OnLocalShutdown: func() {
				to, err := requireChromeTimeout(chrome.HTTPShutdownTimeoutMs, "CLI_HTTP_SHUTDOWN_TIMEOUT_MS", chrome)
				if err != nil || to <= 0 {
					return
				}
				ctx, cancel := context.WithTimeout(context.Background(), to)
				defer cancel()
				_ = httpServer.Shutdown(ctx)
			},
		})
		httpServer.Handler = srv.Handler()

		telemetry.CaptureEvent(cfg.APIBaseURL, "proxy_start", map[string]interface{}{
			"cli_version": Version,
			"port":        listenPort,
		})

		go func() {
			scheme := "http"
			certFile := strings.TrimSpace(os.Getenv("TRIM_TLS_CERT"))
			keyFile := strings.TrimSpace(os.Getenv("TRIM_TLS_KEY"))
			useTLS := certFile != "" && keyFile != ""
			if useTLS {
				scheme = "https"
			}
			if chrome.StartRunningFmt != "" {
				fmt.Printf(chrome.StartRunningFmt+"\n", scheme, listenPort)
			}
			if chrome.StartDashboardFmt != "" {
				fmt.Printf(chrome.StartDashboardFmt+"\n", scheme, listenPort)
			}
			if chrome.StartModeFmt != "" {
				fmt.Printf(chrome.StartModeFmt+"\n", projectconfig.NormalizeMode(proj.Mode))
			}
			if chrome.StartPointIDEFmt != "" {
				fmt.Printf(chrome.StartPointIDEFmt+"\n", scheme, listenPort)
			}
			if len(proj.NeverTrimPaths) > 0 || len(proj.IgnorePaths) > 0 {
				if chrome.StartRulesFmt != "" {
					fmt.Printf(chrome.StartRulesFmt+"\n",
						len(proj.NeverTrimPaths), len(proj.IgnorePaths))
				}
			}
			var listenErr error
			if useTLS {
				listenErr = httpServer.ListenAndServeTLS(certFile, keyFile)
			} else {
				listenErr = httpServer.ListenAndServe()
			}
			if listenErr != nil && listenErr != http.ErrServerClosed {
				// Port race: another Trim already owns the port.
				if isAddrInUse(listenErr) {
					if os.Getenv("TRIM_AUTOSTART_ENFORCER") == "1" && healthURL != "" {
						// Become sidecar enforcer instead of exiting without pref polling.
						_ = runAutostartAttachEnforcer(cfg, chrome, healthURL)
					}
					os.Exit(0)
				}
				if chrome.ProxyFailedFmt != "" {
					log.Fatalf(chrome.ProxyFailedFmt, listenErr)
				}
				log.Fatal(listenErr)
			}
		}()

		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

		// Daemon-run only: re-check pref while proxy is up (dashboard uncheck → stop).
		// Manual `trim start` does not set TRIM_AUTOSTART_ENFORCER.
		if os.Getenv("TRIM_AUTOSTART_ENFORCER") == "1" {
			startAutostartEnforcer(cfg, chrome, httpServer, stop)
		}

		<-stop
		if chrome.StartStopped != "" {
			fmt.Println("\n" + chrome.StartStopped)
		}
		to, err := requireChromeTimeout(chrome.HTTPShutdownTimeoutMs, "CLI_HTTP_SHUTDOWN_TIMEOUT_MS", chrome)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), to)
		defer cancel()
		_ = httpServer.Shutdown(ctx)
		return nil
	},
}

// startAutostartEnforcer polls local (+ optional cloud) auto_start_with_ide and
// shuts down the proxy when the preference is off / managed-off (fail-closed).
func startAutostartEnforcer(cfg config.Local, chrome cliChrome, httpServer *http.Server, stop chan<- os.Signal) {
	ms, ok := autostartPrefPollIntervalMs(chrome)
	if !ok || ms == 0 {
		return // fail-closed: no invent poll interval; 0 = disabled
	}
	go func() {
		t := time.NewTicker(time.Duration(ms) * time.Millisecond)
		defer t.Stop()
		for range t.C {
			if !autostartEnforcerShouldRun(cfg, chrome) {
				if chrome.AutostartEnforcerStopped != "" {
					fmt.Println(chrome.AutostartEnforcerStopped)
				}
				to, err := requireChromeTimeout(chrome.HTTPShutdownTimeoutMs, "CLI_HTTP_SHUTDOWN_TIMEOUT_MS", chrome)
				if err == nil && to > 0 {
					ctx, cancel := context.WithTimeout(context.Background(), to)
					_ = httpServer.Shutdown(ctx)
					cancel()
				}
				releaseDaemonLoginItem(chrome)
				select {
				case stop <- syscall.SIGTERM:
				default:
				}
				return
			}
		}
	}()
}

// runAutostartAttachEnforcer is used when daemon run finds a healthy proxy already
// bound: do not double-bind; poll pref and stop that proxy when auto-start is off.
func runAutostartAttachEnforcer(cfg config.Local, chrome cliChrome, healthURL string) error {
	ms, ok := autostartPrefPollIntervalMs(chrome)
	if !ok || ms == 0 {
		return nil // attached; fail-closed no invent poll; 0 = disabled
	}
	healthTo, err := requireChromeTimeout(chrome.ProxyHealthTimeoutMs, "CLI_PROXY_HEALTH_TIMEOUT_MS", chrome)
	if err != nil {
		return err
	}
	t := time.NewTicker(time.Duration(ms) * time.Millisecond)
	defer t.Stop()
	for range t.C {
		if !proxyHealthOK(healthURL, healthTo) {
			return nil // proxy gone; sidecar done
		}
		if !autostartEnforcerShouldRun(cfg, chrome) {
			if chrome.AutostartEnforcerStopped != "" {
				fmt.Println(chrome.AutostartEnforcerStopped)
			}
			_ = stopCmd.RunE(stopCmd, nil)
			releaseDaemonLoginItem(chrome)
			return nil
		}
	}
	return nil
}

func autostartEnforcerShouldRun(cfg config.Local, chrome cliChrome) bool {
	if blocked, _ := deepopt.AutostartBlocked(); blocked {
		return false
	}
	_ = refreshLocalPreferencesFromCloud(cfg, chrome)
	p, err := deepopt.LoadPreferences()
	if err != nil {
		return false
	}
	if p.AutoStartSet() {
		return p.AutoStartEnabled()
	}
	def := strings.ToLower(strings.TrimSpace(chrome.DefaultAutoStartWithIDE))
	return def == "true"
}

// refreshLocalPreferencesFromCloud mirrors Dashboard → Settings into ~/.config/trim/preferences.json.
// Source of truth is GET /api/v1/me/preferences (profiles.*). No invent defaults.
func refreshLocalPreferencesFromCloud(cfg config.Local, chrome cliChrome) error {
	token, err := storage.GetToken()
	if err != nil || token == "" {
		return nil
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.APIBaseURL), "/")
	if base == "" {
		return nil
	}
	req, err := http.NewRequest(http.MethodGet, base+"/api/v1/me/preferences", nil)
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
		return nil
	}
	var remote remotePreferences
	if err := json.Unmarshal(body, &remote); err != nil {
		return err
	}
	cur, _ := deepopt.LoadPreferences()
	p := preferencesFromRemote(remote, cur)
	return deepopt.SavePreferences(p, prefsChromeFrom(chrome))
}

func isAddrInUse(err error) bool {
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		var sysErr *os.SyscallError
		if errors.As(opErr.Err, &sysErr) {
			return strings.Contains(strings.ToLower(sysErr.Error()), "address already in use") ||
				strings.Contains(strings.ToLower(sysErr.Error()), "only one usage of each socket address")
		}
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "address already in use") ||
		strings.Contains(msg, "only one usage of each socket address")
}

func init() {
	startCmd.Flags().IntVarP(&portFlag, "port", "p", 0, "")
}

// loadProviderAdapterRegistry returns nil when adapters are not synced (passthrough-only; no invent).
func loadProviderAdapterRegistry(chrome cliChrome) (*provideradapt.Registry, provideradapt.Chrome) {
	ach := provideradapt.Chrome{
		UnknownDialectFmt: chrome.ProxyAdapterUnknownDialectFmt,
		ModelRequired:     chrome.ProxyAdapterModelRequired,
		AliasRequiredFmt:  chrome.ProxyAdapterAliasRequiredFmt,
		TranslateFmt:      chrome.ProxyAdapterTranslateFmt,
		AuthFmt:           chrome.ProxyAdapterAuthFmt,
		ResponseFmt:       chrome.ProxyAdapterResponseFmt,
		ModelsNotSynced:   chrome.ProxyModelsNotSynced,
		ModelsMethod:      chrome.ProxyModelsMethod,
		BaseURLDoubleV1:   chrome.ProxyErrBaseURLDoubleV1,
		ModelNotFoundFmt:  chrome.ProxyErrModelNotFoundFmt,
		UpstreamAuthFmt:   chrome.ProxyErrUpstreamAuthFmt,
		UpstreamQuotaFmt:  chrome.ProxyErrUpstreamQuotaFmt,
	}
	prefs, err := deepopt.LoadPreferences()
	if err != nil || !prefs.ProviderAdaptersSynced {
		return nil, ach
	}
	reg, err := deepopt.AdapterRegistryFromPreferences(prefs, ach)
	if err != nil {
		return nil, ach
	}
	return reg, ach
}
