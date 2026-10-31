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
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/usetrim/trim/cli/internal/cliout"
	"github.com/usetrim/trim/cli/internal/config"
	"github.com/usetrim/trim/cli/internal/deepopt"
	"github.com/usetrim/trim/cli/internal/metrics"
	"github.com/usetrim/trim/cli/internal/projectconfig"
	"github.com/usetrim/trim/cli/internal/storage"
	"github.com/usetrim/trim/cli/internal/telemetry"
	"github.com/usetrim/trim/server/pkg/provideradapt"
	"github.com/usetrim/trim/server/pkg/proxy"
	"github.com/usetrim/trim/server/pkg/trimmer"
)

var (
	portFlag        int
	startForeground bool
)

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
		healthURL := healthURLForListenPort(chrome, listenPort)
		if healthURL != "" {
			to, err := requireChromeTimeout(chrome.ProxyHealthTimeoutMs, "CLI_PROXY_HEALTH_TIMEOUT_MS", chrome)
			if err != nil {
				return err
			}
			if proxyHealthOK(healthURL, to) {
				// Heal Deep sidecars even on attach (IDE / enforcer / second trim start).
				healWindowsDeepSidecarsForRunningExe()
				// Extension/daemon enforcer path: stay as sidecar so dashboard uncheck still stops.
				// Manual `trim start` exits 0 (user-owned process already up) with clear UX.
				if os.Getenv("TRIM_AUTOSTART_ENFORCER") == "1" {
					return runAutostartAttachEnforcer(cfg, chrome, healthURL)
				}
				printStartReadyUI(chrome, startListenScheme(), listenPort, true)
				return nil
			}
		}
		// Manual start: background worker owns the proxy; parent prints ready UI then
		// returns the shell (so the prompt never appears in the middle of banners).
		// daemon run / --foreground stay attached in this terminal.
		if startShouldDetach() {
			return startDetachedAndWaitHealthy(cmd, cfg, chrome, healthURL, listenPort)
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
		toolClearKeep := proj.ToolResultClearKeepRecent
		if toolClearKeep == 0 {
			toolClearKeep = cfg.ToolResultClearKeepRecent
		}
		activeFile := proj.ActiveFileProtection
		if !proj.ActiveFileProtectionSet {
			activeFile = chromeProxyActiveFileProtection(chrome)
		}

		// Pull account preferences (compression_tier / deep_* / adapters) before binding Deep.
		_ = refreshLocalPreferencesFromCloud(cfg, chrome)
		healWindowsDeepSidecarsForRunningExe()
		liveMode, deepOptimize, errDeepFmt, liveErr := prepareLiveDeep(cfg, chrome)
		if liveErr != nil {
			return liveErr
		}
		adapterReg, _ := loadProviderAdapterRegistry(chrome)
		// Detached worker: parent prints adapter/Deep banners after /health (keeps UI,
		// returns the shell). Foreground / daemon keep printing here.
		if adapterReg != nil && chrome.ProxyAdapterEnabledFmt != "" && !isTrimStartWorker() {
			fmt.Printf(chrome.ProxyAdapterEnabledFmt+"\n", adapterReg.EnabledCount())
		}

		httpServer := &http.Server{
			Addr:              ":" + listenPort,
			ReadHeaderTimeout: time.Duration(cfg.HTTPReadHeaderTimeoutSec) * time.Second,
		}
		cursorLinkTok := ""
		if st, err := loadCursorLinkState(); err == nil && st.Enabled {
			_ = ensureCursorLinkAccessToken(&st)
			_ = saveCursorLinkState(st)
			cursorLinkTok = strings.TrimSpace(st.AccessToken)
		}
		srv := proxy.NewServer(proxy.Options{
			UpstreamOpenAI:                cfg.UpstreamOpenAI,
			UpstreamAnthropic:             cfg.UpstreamAnthropic,
			UpstreamOpenAIFailover:        cfg.UpstreamOpenAIFailover,
			UpstreamAnthropicFailover:     cfg.UpstreamAnthropicFailover,
			NeverTrim:                     proj.MustNeverTrim,
			MaxLogBytes:                   proj.MaxLogBytes,
			LogCompactMinBytes:            proj.LogCompactMinBytes,
			LogCompactMaxLines:            proj.LogCompactMaxLines,
			LogNoiseSubstrings:            proj.LogNoiseSubstrings,
			ActiveFileProtection:          activeFile,
			CompressionMode:               mode,
			MinLines:                      resolved.MinLinesForMode(),
			DisableSkeletonize:            !resolved.SkeletonizeEnabled(),
			AlwaysCompactLogs:             resolved.AlwaysCompactLogs(),
			CustomQueries:                 proj.CustomQueries,
			FallbackUncompressed:          chromeProxyFallbackUncompressed(chrome),
			CheapModel:                    cfg.CheapModel,
			RouteMaxTokens:                cfg.RouteMaxTokens,
			SavingsUsdPerMTok:             cfg.SavingsUsdPerMTok,
			HistoryKeepTurns:              historyKeep,
			HistoryCompactedStub:          chrome.ProxyHistoryCompactedStub,
			ToolResultClearKeepRecent:     toolClearKeep,
			ToolResultClearedFmt:          chrome.ProxyToolResultClearedFmt,
			UpstreamHTTPTimeout:           time.Duration(cfg.ProxyHTTPTimeoutSec) * time.Second,
			LiveModeLabel:                 liveMode,
			DeepOptimize:                  deepOptimize,
			ErrDeepFmt:                    errDeepFmt,
			ProviderAdapters:              adapterReg,
			AnthropicWorkspaceID:          cfg.AnthropicWorkspaceID,
			OpenAIOrganization:            cfg.OpenAIOrganization,
			OpenAIProject:                 cfg.OpenAIProject,
			DoorOpenAI:                    chrome.ProxyDoorOpenAI,
			DoorAnthropic:                 chrome.ProxyDoorAnthropic,
			AdapterTranslateFmt:           chrome.ProxyAdapterTranslateFmt,
			AdapterAuthFmt:                chrome.ProxyAdapterAuthFmt,
			AdapterResponseFmt:            chrome.ProxyAdapterResponseFmt,
			ModelsNotSynced:               chrome.ProxyModelsNotSynced,
			ModelsMethod:                  chrome.ProxyModelsMethod,
			BaseURLDoubleV1:               chrome.ProxyErrBaseURLDoubleV1,
			ModelNotFoundFmt:              chrome.ProxyErrModelNotFoundFmt,
			UnknownModelFmt:               chrome.ProxyErrUnknownModelFmt,
			UpstreamAuthFmt:               chrome.ProxyErrUpstreamAuthFmt,
			UpstreamQuotaFmt:              chrome.ProxyErrUpstreamQuotaFmt,
			UpstreamUnavailableFmt:        chrome.ProxyErrUpstreamUnavailableFmt,
			ModelsUpstreamAdapterRequired: chrome.ProxyModelsUpstreamAdapterRequired,
			ModelsUpstreamNotFoundFmt:     chrome.ProxyModelsUpstreamNotFoundFmt,
			ModelsUpstreamAuthRequired:    chrome.ProxyModelsUpstreamAuthRequired,
			ModelsUpstreamUnsupportedFmt:  chrome.ProxyModelsUpstreamUnsupportedFmt,
			CursorLinkToken:               cursorLinkTok,
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

		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

		certFile := strings.TrimSpace(os.Getenv("TRIM_TLS_CERT"))
		keyFile := strings.TrimSpace(os.Getenv("TRIM_TLS_KEY"))
		useTLS := certFile != "" && keyFile != ""
		scheme := "http"
		if useTLS {
			scheme = "https"
		}

		go func() {
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
						os.Exit(0)
					}
					// Manual start: tell the user Trim is already up (same UX as health attach).
					printStartReadyUI(chrome, scheme, listenPort, true)
					os.Exit(0)
				}
				if chrome.ProxyFailedFmt != "" {
					log.Fatalf(chrome.ProxyFailedFmt, listenErr)
				}
				log.Fatal(listenErr)
			}
			// trim stop / Shutdown closed the listener: unblock main so the process exits
			// (important for detached `trim start` workers).
			select {
			case stop <- syscall.SIGTERM:
			default:
			}
		}()

		// Wait until /health is OK so banners (and Cursor Link) print after listen is real.
		// Parent detached start also waits on health; sync Cursor Link before the shell returns.
		if healthURL != "" {
			to, err := requireChromeTimeout(chrome.ProxyHealthTimeoutMs, "CLI_PROXY_HEALTH_TIMEOUT_MS", chrome)
			if err == nil && to > 0 {
				waitFor := to * 6
				if waitFor < 15*time.Second {
					waitFor = 15 * time.Second
				}
				if waitFor > 30*time.Second {
					waitFor = 30 * time.Second
				}
				deadline := time.Now().Add(waitFor)
				probe := 400 * time.Millisecond
				if to < probe {
					probe = to
				}
				for time.Now().Before(deadline) {
					if proxyHealthOK(healthURL, probe) {
						break
					}
					time.Sleep(probe)
				}
			}
		}
		isStartWorker := isTrimStartWorker()
		if isStartWorker {
			// Long-lived worker: bring Cursor Link up here (survives parent exit). Parent
			// prints the ready banners only after /health (and link, when enabled).
			if st, err := loadCursorLinkState(); err == nil && st.Enabled {
				_, _ = ensureCursorLink(listenPort)
			}
		} else {
			printStartReadyUI(chrome, scheme, listenPort, false)
			if chrome.StartModeFmt != "" {
				cliout.Muted(fmt.Sprintf(chrome.StartModeFmt, projectconfig.NormalizeMode(proj.Mode)))
			}
			if len(proj.NeverTrimPaths) > 0 || len(proj.IgnorePaths) > 0 {
				if chrome.StartRulesFmt != "" {
					cliout.Muted(fmt.Sprintf(chrome.StartRulesFmt,
						len(proj.NeverTrimPaths), len(proj.IgnorePaths)))
				}
			}
		}

		// Daemon-run only: re-check pref while proxy is up (dashboard uncheck → stop).
		// Manual `trim start` does not set TRIM_AUTOSTART_ENFORCER.
		if os.Getenv("TRIM_AUTOSTART_ENFORCER") == "1" {
			startAutostartEnforcer(cfg, chrome, httpServer, stop)
		}

		<-stop
		_ = stopCursorLink()
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
	startCmd.Flags().BoolVar(&startForeground, "foreground", false, "")
}

const trimStartWorkerEnv = "TRIM_START_WORKER"

func isTrimStartWorker() bool {
	return strings.TrimSpace(os.Getenv(trimStartWorkerEnv)) == "1"
}

func startShouldDetach() bool {
	if strings.TrimSpace(os.Getenv("TRIM_AUTOSTART_ENFORCER")) == "1" {
		return false
	}
	if isTrimStartWorker() {
		return false
	}
	return !startForeground
}

// startDetachedAndWaitHealthy spawns the same CLI as a background worker, waits
// until /health is OK (and Cursor Link when enabled), prints ready UI in the parent,
// then exits so the shell returns with banners already complete.
func startDetachedAndWaitHealthy(cmd *cobra.Command, cfg config.Local, chrome cliChrome, healthURL, listenPort string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	args := []string{"start", "--foreground"}
	if cmd != nil && cmd.Flags().Changed("port") {
		args = append(args, "--port", fmt.Sprintf("%d", portFlag))
	}
	if flagJSON {
		args = append(args, "--json")
	}
	if flagVerbose {
		args = append(args, "--verbose")
	}
	child := exec.Command(exe, args...)
	child.Env = append(os.Environ(), trimStartWorkerEnv+"=1")
	// Do not inherit the parent console. On Windows a console-attached worker keeps
	// CMD/PowerShell looking "still running" after the parent exits. Parent prints UX.
	child.Stdout = nil
	child.Stderr = nil
	child.Stdin = nil
	configureDetachedStartProc(child)
	if err := child.Start(); err != nil {
		fmtStr := strings.TrimSpace(chrome.ProxyWorkerStartFmt)
		if fmtStr == "" {
			return chrome.fail("")
		}
		return fmt.Errorf(fmtStr, err)
	}
	waitErr := make(chan error, 1)
	go func() { waitErr <- child.Wait() }()

	to, err := requireChromeTimeout(chrome.ProxyHealthTimeoutMs, "CLI_PROXY_HEALTH_TIMEOUT_MS", chrome)
	if err != nil {
		_ = child.Process.Kill()
		return err
	}
	if to <= 0 {
		to = 2 * time.Second
	}
	// Cap parent wait so a cold-start failure cannot hold the shell for minutes.
	waitFor := to * 6
	if waitFor < 15*time.Second {
		waitFor = 15 * time.Second
	}
	if waitFor > 30*time.Second {
		waitFor = 30 * time.Second
	}
	deadline := time.Now().Add(waitFor)
	probe := 400 * time.Millisecond
	if to < probe {
		probe = to
	}
	healthy := false
	for time.Now().Before(deadline) {
		select {
		case err := <-waitErr:
			early := strings.TrimSpace(chrome.ProxyWorkerExitedEarlyFmt)
			if early == "" {
				return chrome.fail("")
			}
			detail := fmt.Sprintf(early, err)
			if chrome.ProxyFailedFmt != "" {
				return fmt.Errorf(chrome.ProxyFailedFmt, detail)
			}
			return fmt.Errorf("%s", detail)
		default:
		}
		if healthURL != "" && proxyHealthOK(healthURL, probe) {
			healthy = true
			break
		}
		time.Sleep(150 * time.Millisecond)
	}
	if !healthy {
		_ = child.Process.Kill()
		notHealthy := strings.TrimSpace(chrome.ProxyNotHealthy)
		if notHealthy == "" {
			return chrome.fail("")
		}
		if chrome.ProxyFailedFmt != "" {
			return fmt.Errorf(chrome.ProxyFailedFmt, notHealthy)
		}
		return fmt.Errorf("%s", notHealthy)
	}

	// Worker starts Cursor Link after health; wait so ready UI can include the URL when enabled.
	waitCursorLinkReadyAfterHealth(45 * time.Second)

	// Same order as before (Deep → adapters → ready → compression mode); parent owns stdout.
	printStartLiveStatusBanners(cfg, chrome)
	scheme := startListenScheme()
	printStartReadyUI(chrome, scheme, listenPort, false)
	if chrome.StartModeFmt != "" {
		mode := ""
		if proj, err := projectconfig.Load(""); err == nil {
			mode = projectconfig.NormalizeMode(proj.Mode)
		}
		cliout.Muted(fmt.Sprintf(chrome.StartModeFmt, mode))
	}
	return nil
}

// printStartLiveStatusBanners mirrors the worker's Deep + adapter startup lines so
// detached start keeps the same console UI without inheriting the worker's console.
func printStartLiveStatusBanners(cfg config.Local, chrome cliChrome) {
	prefs, err := deepopt.LoadPreferences()
	if err == nil && prefs.DefaultTier == deepopt.TierDeep &&
		chrome.ProxyLiveDeepEnabledFmt != "" &&
		strings.TrimSpace(chrome.ProxyLiveDeepModeFmt) != "" &&
		strings.TrimSpace(chrome.ProxyDeepCompactStub) != "" &&
		strings.TrimSpace(cfg.DeepDeviceMap) != "" &&
		deepopt.RequireCompressPrefs(prefs, prefsChromeFrom(chrome)) == nil &&
		deepopt.RequireLiveDeepPolicy(prefs) {
		fmt.Printf(chrome.ProxyLiveDeepEnabledFmt+"\n",
			prefs.DeepEngine, prefs.DeepV2Backbone, prefs.CompressionRate, prefs.LiveDeepOOMPolicy)
	}
	adapterReg, _ := loadProviderAdapterRegistry(chrome)
	if adapterReg != nil && chrome.ProxyAdapterEnabledFmt != "" {
		fmt.Printf(chrome.ProxyAdapterEnabledFmt+"\n", adapterReg.EnabledCount())
	}
}

// waitCursorLinkReadyAfterHealth returns when Cursor Link is off, already has a
// public URL, or timeout elapses (worker may still be bringing cloudflared up).
func waitCursorLinkReadyAfterHealth(timeout time.Duration) {
	if timeout <= 0 {
		return
	}
	st, err := loadCursorLinkState()
	if err != nil || !st.Enabled {
		return
	}
	if strings.TrimSpace(st.PublicBaseURL) != "" || strings.TrimSpace(st.PublicOrigin) != "" {
		return
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		st, err := loadCursorLinkState()
		if err != nil || !st.Enabled {
			return
		}
		if strings.TrimSpace(st.PublicBaseURL) != "" || strings.TrimSpace(st.PublicOrigin) != "" {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func healthURLForListenPort(chrome cliChrome, listenPort string) string {
	listenPort = strings.TrimSpace(listenPort)
	base := strings.TrimSpace(chrome.ProxyHealthURL)
	if listenPort == "" {
		return base
	}
	if base == "" {
		return fmt.Sprintf("http://127.0.0.1:%s/health", listenPort)
	}
	u, err := url.Parse(base)
	if err != nil || u.Host == "" {
		return base
	}
	host := u.Hostname()
	if host == "" {
		host = "127.0.0.1"
	}
	u.Host = net.JoinHostPort(host, listenPort)
	return u.String()
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

func startListenScheme() string {
	certFile := strings.TrimSpace(os.Getenv("TRIM_TLS_CERT"))
	keyFile := strings.TrimSpace(os.Getenv("TRIM_TLS_KEY"))
	if certFile != "" && keyFile != "" {
		return "https"
	}
	return "http"
}

// printStartReadyUI matches status / autostart / daemon coloring:
// ● Success (green) → Table (cyan labels, accent/ok values) → Muted tips (gray).
// Worldwide users must never see a blank `trim start` when the proxy is already healthy.
// Local Base URL is for on-device clients. Cursor Agent needs Trim Cursor Link (HTTPS), not 127.0.0.1.
func printStartReadyUI(chrome cliChrome, scheme, port string, attached bool) {
	scheme = strings.TrimSpace(scheme)
	port = strings.TrimSpace(port)
	if scheme == "" {
		scheme = "http"
	}
	if port == "" {
		return
	}
	if attached {
		if chrome.StartAttachedOkFmt != "" {
			cliout.Success(fmt.Sprintf(chrome.StartAttachedOkFmt, scheme, port))
		} else if chrome.StartRunningFmt != "" {
			cliout.Success(fmt.Sprintf(chrome.StartRunningFmt, scheme, port))
		}
	} else if chrome.StartRunningFmt != "" {
		cliout.Success(fmt.Sprintf(chrome.StartRunningFmt, scheme, port))
	}

	rows := make([]cliout.Row, 0, 3)
	if lab := strings.TrimSpace(chrome.StartBaseURLLabel); lab != "" {
		rows = append(rows, cliout.Row{
			Label: lab,
			Value: fmt.Sprintf("%s://127.0.0.1:%s/v1", scheme, port),
			Kind:  "accent",
		})
	}
	if lab := strings.TrimSpace(chrome.StartDashboardLabel); lab != "" {
		rows = append(rows, cliout.Row{
			Label: lab,
			Value: fmt.Sprintf("%s://127.0.0.1:%s/dashboard", scheme, port),
			Kind:  "ok",
		})
	}

	cursorURL := ""
	if st, err := loadCursorLinkState(); err == nil && st.Enabled {
		url, err := ensureCursorLink(port)
		if err != nil {
			// Fail-closed: only print synced tip chrome (no invent English prefix).
			if tip := strings.TrimSpace(chrome.StartCursorAgentCloudflaredTip); tip != "" {
				cliout.Muted(tip)
			}
		} else {
			cursorURL = strings.TrimSpace(url)
		}
	}
	if cursorURL != "" {
		lab := strings.TrimSpace(chrome.StartCursorAgentLabel)
		if lab != "" {
			rows = append(rows, cliout.Row{Label: lab, Value: cursorURL, Kind: "accent"})
		}
	}
	if len(rows) > 0 {
		cliout.Table(rows)
	}

	if attached {
		if tip := strings.TrimSpace(chrome.StartAttachedHint); tip != "" {
			cliout.Muted(tip)
		}
	}
	if chrome.StartPointIDEFmt != "" {
		cliout.Muted(fmt.Sprintf(chrome.StartPointIDEFmt, scheme, port))
	}
	if cursorURL != "" {
		if tip := strings.TrimSpace(chrome.StartCursorAgentTipOn); tip != "" {
			cliout.Muted(tip)
		}
		if tip := strings.TrimSpace(chrome.StartCursorAgentLocalTip); tip != "" {
			cliout.Muted(fmt.Sprintf(tip, port))
		}
	} else if tip := strings.TrimSpace(chrome.StartCursorAgentHintOff); tip != "" {
		cliout.Muted(tip)
	}
}
