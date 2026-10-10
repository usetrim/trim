package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Local is the CLI / local-proxy configuration. Every required field must come from env.
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
	APIBaseURL                string
	AppPublicURL              string
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
	// ProxyHTTPTimeoutSec is outbound LLM upstream timeout for the local proxy (required; no invent 120).
	ProxyHTTPTimeoutSec int
	// CLIHTTPTimeoutSec is HTTP timeout for cloud API / chrome / telemetry / TUI (required; no invent 30/8/3).
	CLIHTTPTimeoutSec int
	// DeepTimeoutSec is wall-clock kill for local Deep Mode optimizer.py (required; no invent 600).
	DeepTimeoutSec int
	// HTTPReadHeaderTimeoutSec is local proxy http.Server ReadHeaderTimeout (required; no invent 10).
	HTTPReadHeaderTimeoutSec int
	// TUIRefreshSec is Bubble Tea live dashboard poll interval (required; no invent 2).
	TUIRefreshSec int
	// StatsSeriesDays is default SQLite series window for `trim stats` and proxy /series (required; no invent 14).
	StatsSeriesDays int
	// TUISeriesDays is SQLite series window for Bubble Tea TUI (required; no invent 7).
	TUISeriesDays int
	// ProxyPreviewMaxChars truncates local dashboard before/after previews (required; no invent 1200).
	ProxyPreviewMaxChars int
	// TUIErrorMaxChars truncates TUI error strings (required; no invent 80).
	TUIErrorMaxChars int
	// DeepDeviceMap is machine-local (cpu/cuda). Deep model ids come from billing_settings via sync.
	DeepDeviceMap string
}

func LoadLocal() (Local, error) {
	_ = godotenv.Load()
	var missing []string
	get := func(key string) string {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}
	optional := func(key string) string {
		return strings.TrimSpace(os.Getenv(key))
	}

	cfg := Local{
		Port:                      get("TRIM_PORT"),
		UpstreamOpenAI:            get("UPSTREAM_OPENAI_URL"),
		UpstreamAnthropic:         get("UPSTREAM_ANTHROPIC_URL"),
		UpstreamOpenAIFailover:    optional("UPSTREAM_OPENAI_FAILOVER_URL"),
		UpstreamAnthropicFailover: optional("UPSTREAM_ANTHROPIC_FAILOVER_URL"),
		AnthropicWorkspaceID:      optional("TRIM_ANTHROPIC_WORKSPACE_ID"),
		OpenAIOrganization:        optional("TRIM_OPENAI_ORGANIZATION"),
		OpenAIProject:             optional("TRIM_OPENAI_PROJECT"),
		APIBaseURL:                get("TRIM_API_BASE_URL"),
		AppPublicURL:              get("APP_PUBLIC_URL"),
		CLIHMACSecret:             get("TRIM_CLI_HMAC_SECRET"),
		CheapModel:                optional("TRIM_CHEAP_MODEL"),
		CompressionMode:           optional("TRIM_COMPRESSION_MODE"),
	}
	savingsRaw := get("TRIM_SAVINGS_USD_PER_MTOK")
	if savingsRaw != "" {
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

	proxyTO := get("TRIM_PROXY_HTTP_TIMEOUT_SEC")
	cliTO := get("TRIM_CLI_HTTP_TIMEOUT_SEC")
	deepTO := get("TRIM_DEEP_TIMEOUT_SEC")
	readHdr := get("TRIM_HTTP_READ_HEADER_TIMEOUT_SEC")
	tuiRefresh := get("TRIM_TUI_REFRESH_SEC")
	statsDays := get("TRIM_STATS_SERIES_DAYS")
	tuiDays := get("TRIM_TUI_SERIES_DAYS")
	previewMax := get("TRIM_PROXY_PREVIEW_MAX_CHARS")
	tuiErrMax := get("TRIM_TUI_ERROR_MAX_CHARS")
	cfg.DeepDeviceMap = get("TRIM_DEEP_DEVICE_MAP")
	// Deep model ids / force_tokens / min tokens / OOM: billing_settings via trim config sync only.
	if proxyTO != "" {
		n, err := strconv.Atoi(proxyTO)
		if err != nil || n < 1 || n > 600 {
			return Local{}, fmt.Errorf("CONFIG_ENV_TRIM_PROXY_HTTP_TIMEOUT_SEC_INVALID")
		}
		cfg.ProxyHTTPTimeoutSec = n
	}
	if cliTO != "" {
		n, err := strconv.Atoi(cliTO)
		if err != nil || n < 1 || n > 300 {
			return Local{}, fmt.Errorf("CONFIG_ENV_TRIM_CLI_HTTP_TIMEOUT_SEC_INVALID")
		}
		cfg.CLIHTTPTimeoutSec = n
	}
	if deepTO != "" {
		n, err := strconv.Atoi(deepTO)
		if err != nil || n < 1 || n > 3600 {
			return Local{}, fmt.Errorf("CONFIG_ENV_TRIM_DEEP_TIMEOUT_SEC_INVALID")
		}
		cfg.DeepTimeoutSec = n
	}
	if readHdr != "" {
		n, err := strconv.Atoi(readHdr)
		if err != nil || n < 1 || n > 120 {
			return Local{}, fmt.Errorf("CONFIG_ENV_TRIM_HTTP_READ_HEADER_TIMEOUT_SEC_INVALID")
		}
		cfg.HTTPReadHeaderTimeoutSec = n
	}
	if tuiRefresh != "" {
		n, err := strconv.Atoi(tuiRefresh)
		if err != nil || n < 1 || n > 60 {
			return Local{}, fmt.Errorf("CONFIG_ENV_TRIM_TUI_REFRESH_SEC_INVALID")
		}
		cfg.TUIRefreshSec = n
	}
	if statsDays != "" {
		n, err := strconv.Atoi(statsDays)
		if err != nil || n < 1 || n > 365 {
			return Local{}, fmt.Errorf("CONFIG_ENV_TRIM_STATS_SERIES_DAYS_INVALID")
		}
		cfg.StatsSeriesDays = n
	}
	if tuiDays != "" {
		n, err := strconv.Atoi(tuiDays)
		if err != nil || n < 1 || n > 365 {
			return Local{}, fmt.Errorf("CONFIG_ENV_TRIM_TUI_SERIES_DAYS_INVALID")
		}
		cfg.TUISeriesDays = n
	}
	if previewMax != "" {
		n, err := strconv.Atoi(previewMax)
		if err != nil || n < 32 || n > 100000 {
			return Local{}, fmt.Errorf("CONFIG_ENV_TRIM_PROXY_PREVIEW_MAX_CHARS_INVALID")
		}
		cfg.ProxyPreviewMaxChars = n
	}
	if tuiErrMax != "" {
		n, err := strconv.Atoi(tuiErrMax)
		if err != nil || n < 16 || n > 2000 {
			return Local{}, fmt.Errorf("CONFIG_ENV_TRIM_TUI_ERROR_MAX_CHARS_INVALID")
		}
		cfg.TUIErrorMaxChars = n
	}

	if len(missing) > 0 {
		return Local{}, fmt.Errorf("CONFIG_ENV_MISSING:%s", strings.Join(missing, ","))
	}
	return cfg, nil
}
