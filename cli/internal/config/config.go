package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Local is the CLI / local-proxy configuration.
//
// Load order (industry-standard hosted CLI):
//  1. Official Trim Cloud product defaults (hosted.go) - zero-config for npm/Homebrew users
//  2. Optional project .env via godotenv (fills empty OS env only)
//  3. Explicit environment variables win over defaults
//
// Self-host: set TRIM_API_BASE_URL + APP_PUBLIC_URL + TRIM_CLI_HMAC_SECRET (and any
// other overrides). Official hosted HMAC is only applied when talking to Trim Cloud.
type Local struct {
	Port                      string
	UpstreamOpenAI            string
	UpstreamAnthropic         string
	UpstreamOpenAIFailover    string
	UpstreamAnthropicFailover string
	// AnthropicWorkspaceID is optional (TRIM_ANTHROPIC_WORKSPACE_ID) for org-scoped Anthropic keys.
	AnthropicWorkspaceID string
	// OpenAIOrganization / OpenAIProject are optional (TRIM_OPENAI_ORGANIZATION / TRIM_OPENAI_PROJECT).
	OpenAIOrganization string
	OpenAIProject      string
	APIBaseURL         string
	AppPublicURL       string
	// CLIHMACSecret must match server TRIM_CLI_HMAC_SECRET for cloud API calls.
	CLIHMACSecret string
	// SavingsUsdPerMTok estimates dollar savings from trimmed tokens (ops-configured).
	SavingsUsdPerMTok float64
	// Optional routing (empty disables).
	CheapModel     string
	RouteMaxTokens int
	// CompressionMode from TRIM_COMPRESSION_MODE when .trimrc has no mode= (empty = pass-through).
	CompressionMode string
	// HistoryKeepTurns from TRIM_HISTORY_KEEP_TURNS when .trimrc has no history_keep_turns (0 = off).
	HistoryKeepTurns int
	// ToolResultClearKeepRecent from TRIM_TOOL_RESULT_CLEAR_KEEP_RECENT when .trimrc omits it (0 = off).
	ToolResultClearKeepRecent int
	// ProxyHTTPTimeoutSec is outbound LLM upstream timeout for the local proxy.
	ProxyHTTPTimeoutSec int
	// CLIHTTPTimeoutSec is HTTP timeout for cloud API / chrome / telemetry / TUI.
	CLIHTTPTimeoutSec int
	// DeepTimeoutSec is wall-clock kill for local Deep Mode optimizer.py.
	DeepTimeoutSec int
	// HTTPReadHeaderTimeoutSec is local proxy http.Server ReadHeaderTimeout.
	HTTPReadHeaderTimeoutSec int
	// TUIRefreshSec is Bubble Tea live dashboard poll interval.
	TUIRefreshSec int
	// StatsSeriesDays is default SQLite series window for `trim stats` and proxy /series.
	StatsSeriesDays int
	// TUISeriesDays is SQLite series window for Bubble Tea TUI.
	TUISeriesDays int
	// ProxyPreviewMaxChars truncates local dashboard before/after previews.
	ProxyPreviewMaxChars int
	// TUIErrorMaxChars truncates TUI error strings.
	TUIErrorMaxChars int
	// DeepDeviceMap is machine-local (cpu/cuda). Deep model ids come from billing_settings via sync.
	DeepDeviceMap string
}

func LoadLocal() (Local, error) {
	_ = godotenv.Load()
	var missing []string

	envOr := func(key, def string) string {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
		return def
	}
	optional := func(key string) string {
		return strings.TrimSpace(os.Getenv(key))
	}

	cfg := Local{
		Port:                      envOr("TRIM_PORT", HostedPort),
		UpstreamOpenAI:            envOr("UPSTREAM_OPENAI_URL", HostedUpstreamOpenAI),
		UpstreamAnthropic:         envOr("UPSTREAM_ANTHROPIC_URL", HostedUpstreamAnthropic),
		UpstreamOpenAIFailover:    optional("UPSTREAM_OPENAI_FAILOVER_URL"),
		UpstreamAnthropicFailover: optional("UPSTREAM_ANTHROPIC_FAILOVER_URL"),
		AnthropicWorkspaceID:      optional("TRIM_ANTHROPIC_WORKSPACE_ID"),
		OpenAIOrganization:        optional("TRIM_OPENAI_ORGANIZATION"),
		OpenAIProject:             optional("TRIM_OPENAI_PROJECT"),
		APIBaseURL:                envOr("TRIM_API_BASE_URL", HostedAPIBaseURL),
		AppPublicURL:              envOr("APP_PUBLIC_URL", HostedAppPublicURL),
		CheapModel:                optional("TRIM_CHEAP_MODEL"),
		CompressionMode:           optional("TRIM_COMPRESSION_MODE"),
		DeepDeviceMap:             envOr("TRIM_DEEP_DEVICE_MAP", HostedDeepDeviceMap),
	}

	cfg.CLIHMACSecret = resolveCLIHMACSecret(cfg.APIBaseURL)
	if cfg.CLIHMACSecret == "" {
		missing = append(missing, "TRIM_CLI_HMAC_SECRET")
	}

	savingsRaw := strings.TrimSpace(os.Getenv("TRIM_SAVINGS_USD_PER_MTOK"))
	if savingsRaw == "" {
		cfg.SavingsUsdPerMTok = HostedSavingsUsdPerMTok
	} else {
		n, err := strconv.ParseFloat(savingsRaw, 64)
		if err != nil || n <= 0 {
			return Local{}, fmt.Errorf("CONFIG_ENV_TRIM_SAVINGS_USD_PER_MTOK_INVALID")
		}
		cfg.SavingsUsdPerMTok = n
	}

	if v := optional("TRIM_ROUTE_MAX_TOKENS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Local{}, fmt.Errorf("CONFIG_ENV_TRIM_ROUTE_MAX_TOKENS_INVALID")
		}
		cfg.RouteMaxTokens = n
	}
	if v := optional("TRIM_HISTORY_KEEP_TURNS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return Local{}, fmt.Errorf("CONFIG_ENV_TRIM_HISTORY_KEEP_TURNS_INVALID")
		}
		cfg.HistoryKeepTurns = n
	}
	if v := optional("TRIM_TOOL_RESULT_CLEAR_KEEP_RECENT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return Local{}, fmt.Errorf("CONFIG_ENV_TRIM_TOOL_RESULT_CLEAR_KEEP_RECENT_INVALID")
		}
		cfg.ToolResultClearKeepRecent = n
	}

	var err error
	if cfg.ProxyHTTPTimeoutSec, err = intEnvOr("TRIM_PROXY_HTTP_TIMEOUT_SEC", HostedProxyHTTPTimeoutSec, 1, 600); err != nil {
		return Local{}, err
	}
	if cfg.CLIHTTPTimeoutSec, err = intEnvOr("TRIM_CLI_HTTP_TIMEOUT_SEC", HostedCLIHTTPTimeoutSec, 1, 300); err != nil {
		return Local{}, err
	}
	if cfg.DeepTimeoutSec, err = intEnvOr("TRIM_DEEP_TIMEOUT_SEC", HostedDeepTimeoutSec, 1, 3600); err != nil {
		return Local{}, err
	}
	if cfg.HTTPReadHeaderTimeoutSec, err = intEnvOr("TRIM_HTTP_READ_HEADER_TIMEOUT_SEC", HostedHTTPReadHeaderTimeoutSec, 1, 120); err != nil {
		return Local{}, err
	}
	if cfg.TUIRefreshSec, err = intEnvOr("TRIM_TUI_REFRESH_SEC", HostedTUIRefreshSec, 1, 60); err != nil {
		return Local{}, err
	}
	if cfg.StatsSeriesDays, err = intEnvOr("TRIM_STATS_SERIES_DAYS", HostedStatsSeriesDays, 1, 365); err != nil {
		return Local{}, err
	}
	if cfg.TUISeriesDays, err = intEnvOr("TRIM_TUI_SERIES_DAYS", HostedTUISeriesDays, 1, 365); err != nil {
		return Local{}, err
	}
	if cfg.ProxyPreviewMaxChars, err = intEnvOr("TRIM_PROXY_PREVIEW_MAX_CHARS", HostedProxyPreviewMaxChars, 32, 100000); err != nil {
		return Local{}, err
	}
	if cfg.TUIErrorMaxChars, err = intEnvOr("TRIM_TUI_ERROR_MAX_CHARS", HostedTUIErrorMaxChars, 16, 2000); err != nil {
		return Local{}, err
	}

	// Structural required strings (always filled by hosted defaults unless cleared somehow).
	for _, pair := range []struct{ key, val string }{
		{"TRIM_PORT", cfg.Port},
		{"UPSTREAM_OPENAI_URL", cfg.UpstreamOpenAI},
		{"UPSTREAM_ANTHROPIC_URL", cfg.UpstreamAnthropic},
		{"TRIM_API_BASE_URL", cfg.APIBaseURL},
		{"APP_PUBLIC_URL", cfg.AppPublicURL},
		{"TRIM_DEEP_DEVICE_MAP", cfg.DeepDeviceMap},
	} {
		if strings.TrimSpace(pair.val) == "" {
			missing = append(missing, pair.key)
		}
	}

	if len(missing) > 0 {
		return Local{}, fmt.Errorf("CONFIG_ENV_MISSING:%s", strings.Join(missing, ","))
	}
	return cfg, nil
}

// resolveCLIHMACSecret: explicit env always wins. Else official release bake when
// targeting Trim Cloud. Self-host custom API never inherits the hosted bake.
func resolveCLIHMACSecret(apiBase string) string {
	if v := strings.TrimSpace(os.Getenv("TRIM_CLI_HMAC_SECRET")); v != "" {
		return v
	}
	if IsOfficialHostedAPI(apiBase) {
		return strings.TrimSpace(HostedCLIHMACSecret)
	}
	return ""
}

func intEnvOr(key string, def, min, max int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < min || n > max {
		return 0, fmt.Errorf("CONFIG_ENV_%s_INVALID", key)
	}
	return n, nil
}
