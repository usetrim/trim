package config

import (
	"os"
	"strings"
	"testing"
)

func clearTrimEnv(t *testing.T) {
	t.Helper()
	// Avoid loading a developer cli/.env via godotenv during tests.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	keys := []string{
		"TRIM_PORT", "UPSTREAM_OPENAI_URL", "UPSTREAM_ANTHROPIC_URL",
		"TRIM_API_BASE_URL", "APP_PUBLIC_URL", "TRIM_CLI_HMAC_SECRET",
		"TRIM_SAVINGS_USD_PER_MTOK", "TRIM_PROXY_HTTP_TIMEOUT_SEC",
		"TRIM_CLI_HTTP_TIMEOUT_SEC", "TRIM_DEEP_TIMEOUT_SEC",
		"TRIM_HTTP_READ_HEADER_TIMEOUT_SEC", "TRIM_TUI_REFRESH_SEC",
		"TRIM_STATS_SERIES_DAYS", "TRIM_TUI_SERIES_DAYS",
		"TRIM_PROXY_PREVIEW_MAX_CHARS", "TRIM_TUI_ERROR_MAX_CHARS",
		"TRIM_DEEP_DEVICE_MAP", "TRIM_ROUTE_MAX_TOKENS", "TRIM_HISTORY_KEEP_TURNS",
		"TRIM_TOOL_RESULT_CLEAR_KEEP_RECENT",
		"TRIM_CHEAP_MODEL", "TRIM_COMPRESSION_MODE",
		"UPSTREAM_OPENAI_FAILOVER_URL", "UPSTREAM_ANTHROPIC_FAILOVER_URL",
		"TRIM_ANTHROPIC_WORKSPACE_ID", "TRIM_OPENAI_ORGANIZATION", "TRIM_OPENAI_PROJECT",
	}
	for _, k := range keys {
		_ = os.Unsetenv(k)
	}
}

func TestLoadLocal_HostedZeroConfigWithBakedHMAC(t *testing.T) {
	clearTrimEnv(t)
	prev := HostedCLIHMACSecret
	HostedCLIHMACSecret = "test-hosted-hmac-secret-hex"
	t.Cleanup(func() { HostedCLIHMACSecret = prev })

	cfg, err := LoadLocal()
	if err != nil {
		t.Fatalf("hosted zero-config LoadLocal: %v", err)
	}
	if cfg.Port != HostedPort {
		t.Fatalf("port: got %q want %q", cfg.Port, HostedPort)
	}
	if cfg.APIBaseURL != HostedAPIBaseURL {
		t.Fatalf("api: got %q want %q", cfg.APIBaseURL, HostedAPIBaseURL)
	}
	if cfg.AppPublicURL != HostedAppPublicURL {
		t.Fatalf("app: got %q want %q", cfg.AppPublicURL, HostedAppPublicURL)
	}
	if cfg.CLIHMACSecret != "test-hosted-hmac-secret-hex" {
		t.Fatalf("hmac: got %q", cfg.CLIHMACSecret)
	}
	if cfg.SavingsUsdPerMTok != HostedSavingsUsdPerMTok {
		t.Fatalf("savings: got %v want %v", cfg.SavingsUsdPerMTok, HostedSavingsUsdPerMTok)
	}
	if cfg.ProxyHTTPTimeoutSec != HostedProxyHTTPTimeoutSec {
		t.Fatalf("proxy timeout: got %d", cfg.ProxyHTTPTimeoutSec)
	}
	if cfg.DeepDeviceMap != HostedDeepDeviceMap {
		t.Fatalf("deep device: got %q", cfg.DeepDeviceMap)
	}
}

func TestLoadLocal_HostedMissingBakeFailsClosed(t *testing.T) {
	clearTrimEnv(t)
	prev := HostedCLIHMACSecret
	HostedCLIHMACSecret = ""
	t.Cleanup(func() { HostedCLIHMACSecret = prev })

	_, err := LoadLocal()
	if err == nil {
		t.Fatal("expected CONFIG_ENV_MISSING for HMAC without bake")
	}
	if !strings.Contains(err.Error(), "TRIM_CLI_HMAC_SECRET") {
		t.Fatalf("want HMAC missing, got %v", err)
	}
}

func TestLoadLocal_EnvOverridesHostedDefaults(t *testing.T) {
	clearTrimEnv(t)
	prev := HostedCLIHMACSecret
	HostedCLIHMACSecret = "baked-should-lose"
	t.Cleanup(func() { HostedCLIHMACSecret = prev })

	t.Setenv("TRIM_PORT", "9999")
	t.Setenv("TRIM_API_BASE_URL", "https://api.example.com")
	t.Setenv("APP_PUBLIC_URL", "https://app.example.com")
	t.Setenv("TRIM_CLI_HMAC_SECRET", "selfhost-hmac")
	t.Setenv("TRIM_SAVINGS_USD_PER_MTOK", "1.5")
	t.Setenv("TRIM_PROXY_HTTP_TIMEOUT_SEC", "90")

	cfg, err := LoadLocal()
	if err != nil {
		t.Fatalf("LoadLocal: %v", err)
	}
	if cfg.Port != "9999" {
		t.Fatalf("port override: %q", cfg.Port)
	}
	if cfg.APIBaseURL != "https://api.example.com" {
		t.Fatalf("api override: %q", cfg.APIBaseURL)
	}
	if cfg.CLIHMACSecret != "selfhost-hmac" {
		t.Fatalf("env hmac must win over bake: %q", cfg.CLIHMACSecret)
	}
	if cfg.SavingsUsdPerMTok != 1.5 {
		t.Fatalf("savings override: %v", cfg.SavingsUsdPerMTok)
	}
	if cfg.ProxyHTTPTimeoutSec != 90 {
		t.Fatalf("timeout override: %d", cfg.ProxyHTTPTimeoutSec)
	}
	// Untouched fields still use hosted product defaults.
	if cfg.UpstreamOpenAI != HostedUpstreamOpenAI {
		t.Fatalf("openai default: %q", cfg.UpstreamOpenAI)
	}
}

func TestLoadLocal_SelfHostCustomAPIDoesNotUseHostedBake(t *testing.T) {
	clearTrimEnv(t)
	prev := HostedCLIHMACSecret
	HostedCLIHMACSecret = "hosted-bake-must-not-apply"
	t.Cleanup(func() { HostedCLIHMACSecret = prev })

	t.Setenv("TRIM_API_BASE_URL", "https://trim.internal.corp")
	t.Setenv("APP_PUBLIC_URL", "https://trim.internal.corp")
	// Intentionally no TRIM_CLI_HMAC_SECRET

	_, err := LoadLocal()
	if err == nil {
		t.Fatal("self-host custom API must fail closed without HMAC")
	}
	if !strings.Contains(err.Error(), "TRIM_CLI_HMAC_SECRET") {
		t.Fatalf("want HMAC missing, got %v", err)
	}
}

func TestIsOfficialHostedAPI(t *testing.T) {
	if !IsOfficialHostedAPI("https://api.use-trim.com") {
		t.Fatal("exact hosted")
	}
	if !IsOfficialHostedAPI("https://api.use-trim.com/") {
		t.Fatal("trailing slash")
	}
	if !IsOfficialHostedAPI("HTTPS://API.USE-TRIM.COM") {
		t.Fatal("case")
	}
	if IsOfficialHostedAPI("https://api.example.com") {
		t.Fatal("custom must be false")
	}
}

func TestLoadLocal_InvalidTimeoutFailsClosed(t *testing.T) {
	clearTrimEnv(t)
	prev := HostedCLIHMACSecret
	HostedCLIHMACSecret = "test-hmac"
	t.Cleanup(func() { HostedCLIHMACSecret = prev })

	t.Setenv("TRIM_PROXY_HTTP_TIMEOUT_SEC", "0")
	_, err := LoadLocal()
	if err == nil || !strings.Contains(err.Error(), "TRIM_PROXY_HTTP_TIMEOUT_SEC_INVALID") {
		t.Fatalf("want invalid proxy timeout, got %v", err)
	}
}

func TestLoadLocal_HostedZeroConfigFillsAllRequiredInts(t *testing.T) {
	clearTrimEnv(t)
	prev := HostedCLIHMACSecret
	HostedCLIHMACSecret = "test-hmac"
	t.Cleanup(func() { HostedCLIHMACSecret = prev })

	cfg, err := LoadLocal()
	if err != nil {
		t.Fatal(err)
	}
	checks := map[string]int{
		"CLIHTTP":    cfg.CLIHTTPTimeoutSec,
		"Deep":       cfg.DeepTimeoutSec,
		"ReadHeader": cfg.HTTPReadHeaderTimeoutSec,
		"TUIRefresh": cfg.TUIRefreshSec,
		"StatsDays":  cfg.StatsSeriesDays,
		"TUIDays":    cfg.TUISeriesDays,
		"Preview":    cfg.ProxyPreviewMaxChars,
		"TUIErr":     cfg.TUIErrorMaxChars,
	}
	for name, v := range checks {
		if v <= 0 {
			t.Fatalf("%s must be filled from hosted defaults, got %d", name, v)
		}
	}
	if cfg.UpstreamOpenAI != HostedUpstreamOpenAI || cfg.UpstreamAnthropic != HostedUpstreamAnthropic {
		t.Fatalf("upstream defaults missing: openai=%q anthropic=%q", cfg.UpstreamOpenAI, cfg.UpstreamAnthropic)
	}
}
