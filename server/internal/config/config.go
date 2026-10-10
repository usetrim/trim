package config

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config is loaded only from environment variables.
// Missing required values cause Load() to return an error.
// There are no silent hardcoded production defaults.
type Config struct {
	Port        string
	DatabaseURL string
	// DatabaseReadURL is optional Postgres read replica / pooler URL.
	// Empty = all reads use DatabaseURL (single primary). Never invent a replica.
	DatabaseReadURL     string
	RedisURL            string
	JWTSecret           string
	SupabaseURL         string
	SupabaseAnonKey     string
	SupabaseServiceKey  string
	PaddleWebhookSecret string
	PaddleAPIKey        string
	PaddleEnv           string
	UpstreamOpenAI      string
	UpstreamAnthropic   string
	// Optional same-shape failover bases (Azure OpenAI, secondary Anthropic gateway, vLLM).
	UpstreamOpenAIFailover    string
	UpstreamAnthropicFailover string
	CORSOrigins               []string
	MinCLIVersion             string
	DeploymentMode            string
	AppPublicURL              string
	CompanyLegalName          string
	CompanySupportEmail       string
	CompanyLogoURL            string
	CompanyAddressLine1       string
	CompanyAddressLine2       string
	CompanyCity               string
	CompanyRegion             string
	CompanyPostalCode         string
	CompanyCountry            string
	CompanyVATID              string
	CompanyRegistration       string
	// CLIHMACSecret signs TrimCLI requests (X-Trim-Signature). Required in cloud mode.
	CLIHMACSecret string
	// Optional model routing (empty disables). Values come only from env, never hardcoded models.
	CheapModel     string
	RouteMaxTokens string
	// SavingsUsdPerMTok estimates USD saved from trimmed tokens (ops-configured, no hardcoded rate).
	SavingsUsdPerMTok float64
	// AllowedAuthProviders is required in cloud mode (comma-separated). No silent default.
	AllowedAuthProviders []string
	// CronSecret protects POST /internal/expire-subscriptions. Empty disables the route.
	CronSecret string
	// FreeTierPoWDifficulty is SHA-256 leading zero bits for free-tier LLM posts (0 disables).
	FreeTierPoWDifficulty int
	// CanaryAPIKeyPrefix bans hardware/IP when presented (honeypot). Empty disables.
	CanaryAPIKeyPrefix string
	// EnterpriseNotifyURL is an optional HTTPS webhook (Slack/Discord/ops) for new sales leads.
	EnterpriseNotifyURL string

	// Fraud / anti-abuse thresholds (env-driven; cloud defaults applied when unset).
	MaxAccountsPerHardware int
	MaxAccountsPerJA4      int
	CFThreatScoreMin       int
	CLITimestampSkewSec    int
	RateLimitPerIPPerMin   int
	RateLimitPerUserPerMin int
	IPRiskLookupTimeoutMs  int

	// Outbound HTTP timeouts (seconds). Required in cloud for Paddle; Cloudinary when configured.
	PaddleHTTPTimeoutSec     int
	PaddleListMaxPages       int
	PaddleListPerPage        int
	CloudinaryHTTPTimeoutSec int
	// ProxyHTTPTimeoutSec is outbound LLM upstream timeout (required in cloud and self; no invent 120 / infinite).
	ProxyHTTPTimeoutSec int
	// HTTPReadHeaderTimeoutSec is http.Server ReadHeaderTimeout (required; no invent 10).
	HTTPReadHeaderTimeoutSec int
	// HTTPShutdownTimeoutSec is graceful Shutdown context timeout (required; no invent 10).
	HTTPShutdownTimeoutSec int
	// WebhookProcessTimeoutSec is Paddle webhook worker DB/process budget (required in cloud; no invent 20).
	WebhookProcessTimeoutSec int
	// EnterpriseNotifyTimeoutSec is outbound enterprise sales webhook timeout (required when ENTERPRISE_NOTIFY_URL set; no invent 8).
	EnterpriseNotifyTimeoutSec int
	// QuotaSyncTimeoutSec is async Redis→Postgres quota sync budget (required in cloud; no invent 5).
	QuotaSyncTimeoutSec int
	// EventInsertTimeoutSec is async trim_events insert budget (required; no invent 5).
	EventInsertTimeoutSec int
	// EventInsertQueueSize is wake-channel depth for outbox workers (required; no invent 1000).
	EventInsertQueueSize int
	// EventInsertWorkers is outbox→trim_events worker count (required; no invent 3).
	EventInsertWorkers int
	// EventOutboxPollSec is periodic outbox drain interval so dropped wakes cannot starve (required; no invent).
	EventOutboxPollSec int
	// EventOutboxBatch is SKIP LOCKED claim batch size per drain (required; no invent 32).
	EventOutboxBatch int
	// WebhookQueueSize is Paddle Redis list max depth before 503 (required in cloud; no invent 1000).
	WebhookQueueSize int
	// WebhookWorkers is Paddle Redis BRPOP worker count (required in cloud; no invent 3).
	WebhookWorkers int
	// PGMaxConns / PGMinConns size pgx write (and default read) pools (required; no invent pgx defaults).
	PGMaxConns int32
	PGMinConns int32
	// PGReadMaxConns / PGReadMinConns size the read pool when DATABASE_READ_URL is set (required then; no invent).
	PGReadMaxConns int32
	PGReadMinConns int32
	// ReadyzTimeoutSec is the CLI/Docker -readyz probe budget (required; no invent 5).
	ReadyzTimeoutSec int
	// CORSMaxAgeSec is Access-Control-Max-Age for preflight cache (required; no invent 300).
	CORSMaxAgeSec int
	// SiteMessagesReloadSec is periodic chrome cache reload so missed pub/sub cannot starve (required; no invent).
	SiteMessagesReloadSec int

	// Redis TTLs (seconds). Required in cloud; no invent 70s / 1h / 24h / 30d.
	RedisVelocityBucketTTLSec int // rate-limit minute bucket key
	RedisAPIKeyCacheTTLSec    int // apikey:{hash} → user_id
	RedisAuthProvidersTTLSec  int // auth:allowed_providers
	RedisQuotaCacheTTLSec     int // user_quota / workspace_quota hashes
	RedisHardwareGraphTTLSec  int // hw_accounts / ja4_accounts sets
	RedisIPBlockTTLSec        int // ip_block:*
	RedisJA4SeenTTLSec        int // ja4_seen:*
	RedisIPRiskTTLSec         int // ip_risk:proxy:*
	RedisASNCheckedTTLSec     int // ip_risk:asn_checked:* success
	RedisASNMissTTLSec        int // asn_checked on lookup miss/error
	RedisTelemetryTTLSec      int // telemetry:counter:*
	// ExpireBatchDefaultLimit is default row cap for POST /internal/expire-subscriptions (required in cloud; no invent 200).
	ExpireBatchDefaultLimit int

	// CompressionMode for cloud/local proxy (mild|balanced|aggressive|custom).
	// Required in cloud; empty skips event mode telemetry (fail-closed, no invent).
	CompressionMode string
	// HistoryKeepTurns: last N non-system chat turns kept; 0 disables sliding window.
	HistoryKeepTurns int
	// Optional MaxMind GeoLite2 MMDB paths (ASN and/or Anonymous-IP). Empty skips offline IP risk.
	GeoLiteASNMMDBPath       string
	GeoLiteAnonymousMMDBPath string

	// Optional SMTP for workspace invite emails. All four (host, port, from; user/pass optional) required to send.
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string

	// Platform operator console (optional; cloud boot does not require).
	PlatformOwnerEmails []string
	AdminStepUpTTLSec   int
	AdminTOTPKey        string
	// AdminWebAuthnRPID is the WebAuthn relying party id (hostname). Empty derives from ADMIN_ALLOWED_ORIGINS.
	AdminWebAuthnRPID string
	GitHubToken       string
	GitHubRepo        string
	// Optional package-manager / marketplace distribution sync (same GitHub token for tap traffic).
	DistHomebrewTapRepo   string
	DistScoopBucketRepo   string
	DistWingetForkRepo    string
	DistVSCodeExtensionID string
	// AdminAllowedOrigins locks browser Origin for /api/v1/admin/* (required in cloud).
	AdminAllowedOrigins []string
	// AdminAllowedCIDRs optional client IP allow-list (CF-Connecting-IP / RemoteAddr).
	// Empty = no IP lock. Nonempty fails closed when client IP is outside all CIDRs.
	AdminAllowedCIDRs []string
	// CFAccessTeamDomain + CFAccessAUD enable Cloudflare Access JWT verification on admin routes.
	// Both empty = gate off. Both required together (validated at Load).
	CFAccessTeamDomain string
	CFAccessAUD        string
}

func Load() (Config, error) {
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

	mode := get("DEPLOYMENT_MODE")
	cfg := Config{
		Port:                      get("PORT"),
		DatabaseURL:               get("DATABASE_URL"),
		DatabaseReadURL:           optional("DATABASE_READ_URL"),
		RedisURL:                  get("REDIS_URL"),
		JWTSecret:                 get("JWT_SECRET"),
		UpstreamOpenAI:            get("UPSTREAM_OPENAI_URL"),
		UpstreamAnthropic:         get("UPSTREAM_ANTHROPIC_URL"),
		UpstreamOpenAIFailover:    optional("UPSTREAM_OPENAI_FAILOVER_URL"),
		UpstreamAnthropicFailover: optional("UPSTREAM_ANTHROPIC_FAILOVER_URL"),
		CORSOrigins:               splitCSV(get("CORS_ORIGINS")),
		MinCLIVersion:             get("MIN_CLI_VERSION"),
		DeploymentMode:            mode,
		AppPublicURL:              get("APP_PUBLIC_URL"),
		CompanyLegalName:          get("COMPANY_LEGAL_NAME"),
		CompanySupportEmail:       get("COMPANY_SUPPORT_EMAIL"),
		CompanyLogoURL:            optional("COMPANY_LOGO_URL"),
		CompanyAddressLine1:       optional("COMPANY_ADDRESS_LINE1"),
		CompanyAddressLine2:       optional("COMPANY_ADDRESS_LINE2"),
		CompanyCity:               optional("COMPANY_CITY"),
		CompanyRegion:             optional("COMPANY_REGION"),
		CompanyPostalCode:         optional("COMPANY_POSTAL_CODE"),
		CompanyCountry:            optional("COMPANY_COUNTRY"),
		CompanyVATID:              optional("COMPANY_VAT_ID"),
		CompanyRegistration:       optional("COMPANY_REGISTRATION"),
		CheapModel:                optional("TRIM_CHEAP_MODEL"),
		RouteMaxTokens:            optional("TRIM_ROUTE_MAX_TOKENS"),
		CronSecret:                optional("CRON_SECRET"),
		CanaryAPIKeyPrefix:        optional("TRIM_CANARY_API_KEY_PREFIX"),
		EnterpriseNotifyURL:       optional("ENTERPRISE_NOTIFY_WEBHOOK_URL"),
		CompressionMode:           optional("TRIM_COMPRESSION_MODE"),
		GeoLiteASNMMDBPath:        optional("TRIM_GEOLITE_ASN_MMDB_PATH"),
		GeoLiteAnonymousMMDBPath:  optional("TRIM_GEOLITE_ANONYMOUS_MMDB_PATH"),
		SMTPHost:                  optional("SMTP_HOST"),
		SMTPPort:                  optional("SMTP_PORT"),
		SMTPUsername:              optional("SMTP_USERNAME"),
		SMTPPassword:              optional("SMTP_PASSWORD"),
		SMTPFrom:                  optional("SMTP_FROM"),
		PlatformOwnerEmails:       splitCSV(optional("TRIM_PLATFORM_OWNER_EMAILS")),
		GitHubToken:               optional("TRIM_GITHUB_TOKEN"),
		GitHubRepo:                optional("TRIM_GITHUB_REPO"),
		DistHomebrewTapRepo:       optional("TRIM_DIST_HOMEBREW_TAP_REPO"),
		DistScoopBucketRepo:       optional("TRIM_DIST_SCOOP_BUCKET_REPO"),
		DistWingetForkRepo:        optional("TRIM_DIST_WINGET_FORK_REPO"),
		DistVSCodeExtensionID:     optional("TRIM_DIST_VSCODE_EXTENSION_ID"),
		AdminAllowedOrigins:       splitCSV(optional("ADMIN_ALLOWED_ORIGINS")),
		AdminAllowedCIDRs:         splitCSV(optional("ADMIN_ALLOWED_CIDRS")),
		AdminTOTPKey:              optional("TRIM_ADMIN_TOTP_KEY"),
		AdminWebAuthnRPID:         optional("ADMIN_WEBAUTHN_RP_ID"),
		CFAccessTeamDomain:        optional("CF_ACCESS_TEAM_DOMAIN"),
		CFAccessAUD:               optional("CF_ACCESS_AUD"),
	}

	// Fail closed: step-up TTL must be explicit when platform admin is used.
	// Empty is allowed at boot; Gate/step-up handlers refuse if unset or invalid.
	if stepRaw := optional("TRIM_ADMIN_STEP_UP_TTL_SEC"); stepRaw != "" {
		n, err := strconv.Atoi(stepRaw)
		// 60s min; 12h max so one verify covers a full operator session.
		if err != nil || n < 60 || n > 43200 {
			return Config{}, fmt.Errorf("TRIM_ADMIN_STEP_UP_TTL_SEC must be 60-43200")
		}
		cfg.AdminStepUpTTLSec = n
	}
	if cfg.AdminTOTPKey != "" {
		if _, err := hex.DecodeString(cfg.AdminTOTPKey); err != nil || len(cfg.AdminTOTPKey) != 64 {
			return Config{}, fmt.Errorf("TRIM_ADMIN_TOTP_KEY must be 64 hex characters (32 bytes)")
		}
	}
	if (cfg.CFAccessTeamDomain == "") != (cfg.CFAccessAUD == "") {
		return Config{}, fmt.Errorf("CF_ACCESS_TEAM_DOMAIN and CF_ACCESS_AUD must both be set or both empty")
	}

	if histRaw := optional("TRIM_HISTORY_KEEP_TURNS"); histRaw != "" {
		n, err := strconv.Atoi(histRaw)
		if err != nil || n < 0 {
			return Config{}, fmt.Errorf("TRIM_HISTORY_KEEP_TURNS must be a non-negative integer")
		}
		cfg.HistoryKeepTurns = n
	}

	if powRaw := optional("TRIM_POW_DIFFICULTY"); powRaw != "" {
		n, err := strconv.Atoi(powRaw)
		if err != nil || n < 0 || n > 24 {
			return Config{}, fmt.Errorf("TRIM_POW_DIFFICULTY must be 0-24")
		}
		cfg.FreeTierPoWDifficulty = n
	} else if mode == "cloud" {
		// Fail closed: no silent invent of 16.
		missing = append(missing, "TRIM_POW_DIFFICULTY")
	}

	savingsRaw := get("TRIM_SAVINGS_USD_PER_MTOK")
	if savingsRaw != "" {
		n, err := strconv.ParseFloat(savingsRaw, 64)
		if err != nil || n <= 0 {
			return Config{}, fmt.Errorf("TRIM_SAVINGS_USD_PER_MTOK must be a positive number")
		}
		cfg.SavingsUsdPerMTok = n
	}

	// Fail closed: cloud must set ALLOWED_AUTH_PROVIDERS explicitly (no silent default).
	if mode == "cloud" {
		providers := get("ALLOWED_AUTH_PROVIDERS")
		cfg.AllowedAuthProviders = splitCSV(providers)
	} else if p := optional("ALLOWED_AUTH_PROVIDERS"); p != "" {
		cfg.AllowedAuthProviders = splitCSV(p)
	}

	cfg.MaxAccountsPerHardware = intFromEnv(optional("TRIM_MAX_ACCOUNTS_PER_HARDWARE"), 0)
	cfg.MaxAccountsPerJA4 = intFromEnv(optional("TRIM_MAX_ACCOUNTS_PER_JA4"), 0)

	if mode == "cloud" {
		// No invent floors for skew / rate limits / IP lookup timeout / CF threat.
		skewRaw := get("TRIM_CLI_TIMESTAMP_SKEW_SEC")
		ipRL := get("TRIM_RATE_LIMIT_IP_PER_MIN")
		userRL := get("TRIM_RATE_LIMIT_USER_PER_MIN")
		ipTO := get("TRIM_IP_RISK_LOOKUP_TIMEOUT_MS")
		cfThreat := get("TRIM_CF_THREAT_SCORE_MIN")
		paddleTO := get("PADDLE_HTTP_TIMEOUT_SEC")
		paddleMaxPages := get("PADDLE_LIST_MAX_PAGES")
		paddlePerPage := get("PADDLE_LIST_PER_PAGE")
		proxyTO := get("TRIM_PROXY_HTTP_TIMEOUT_SEC")
		readHdr := get("TRIM_HTTP_READ_HEADER_TIMEOUT_SEC")
		shutdownTO := get("TRIM_HTTP_SHUTDOWN_TIMEOUT_SEC")
		webhookTO := get("TRIM_WEBHOOK_PROCESS_TIMEOUT_SEC")
		quotaTO := get("TRIM_QUOTA_SYNC_TIMEOUT_SEC")
		eventTO := get("TRIM_EVENT_INSERT_TIMEOUT_SEC")
		cfg.CLITimestampSkewSec = intFromEnv(skewRaw, 0)
		cfg.RateLimitPerIPPerMin = intFromEnv(ipRL, 0)
		cfg.RateLimitPerUserPerMin = intFromEnv(userRL, 0)
		cfg.IPRiskLookupTimeoutMs = intFromEnv(ipTO, 0)
		cfg.CFThreatScoreMin = intFromEnv(cfThreat, 0)
		cfg.PaddleHTTPTimeoutSec = intFromEnv(paddleTO, 0)
		cfg.PaddleListMaxPages = intFromEnv(paddleMaxPages, 0)
		cfg.PaddleListPerPage = intFromEnv(paddlePerPage, 0)
		cfg.ProxyHTTPTimeoutSec = intFromEnv(proxyTO, 0)
		cfg.HTTPReadHeaderTimeoutSec = intFromEnv(readHdr, 0)
		cfg.HTTPShutdownTimeoutSec = intFromEnv(shutdownTO, 0)
		cfg.WebhookProcessTimeoutSec = intFromEnv(webhookTO, 0)
		cfg.QuotaSyncTimeoutSec = intFromEnv(quotaTO, 0)
		cfg.EventInsertTimeoutSec = intFromEnv(eventTO, 0)
		cfg.EventInsertQueueSize = intFromEnv(get("TRIM_EVENT_INSERT_QUEUE_SIZE"), 0)
		cfg.EventInsertWorkers = intFromEnv(get("TRIM_EVENT_INSERT_WORKERS"), 0)
		cfg.EventOutboxPollSec = intFromEnv(get("TRIM_EVENT_OUTBOX_POLL_SEC"), 0)
		cfg.EventOutboxBatch = intFromEnv(get("TRIM_EVENT_OUTBOX_BATCH"), 0)
		cfg.WebhookQueueSize = intFromEnv(get("TRIM_WEBHOOK_QUEUE_SIZE"), 0)
		cfg.WebhookWorkers = intFromEnv(get("TRIM_WEBHOOK_WORKERS"), 0)
		cfg.PGMaxConns = int32(intFromEnv(get("TRIM_PG_MAX_CONNS"), 0))
		cfg.PGMinConns = int32(intFromEnv(get("TRIM_PG_MIN_CONNS"), 0))
		cfg.PGReadMaxConns = int32(intFromEnv(optional("TRIM_PG_READ_MAX_CONNS"), 0))
		cfg.PGReadMinConns = int32(intFromEnv(optional("TRIM_PG_READ_MIN_CONNS"), 0))
		cfg.ReadyzTimeoutSec = intFromEnv(get("TRIM_READYZ_TIMEOUT_SEC"), 0)
		cfg.CORSMaxAgeSec = intFromEnv(get("TRIM_CORS_MAX_AGE_SEC"), 0)
		cfg.SiteMessagesReloadSec = intFromEnv(get("TRIM_SITE_MESSAGES_RELOAD_SEC"), 0)
		hw := get("TRIM_MAX_ACCOUNTS_PER_HARDWARE")
		ja4 := get("TRIM_MAX_ACCOUNTS_PER_JA4")
		cfg.MaxAccountsPerHardware = intFromEnv(hw, 0)
		cfg.MaxAccountsPerJA4 = intFromEnv(ja4, 0)
		cfg.RedisVelocityBucketTTLSec = intFromEnv(get("TRIM_REDIS_TTL_VELOCITY_SEC"), 0)
		cfg.RedisAPIKeyCacheTTLSec = intFromEnv(get("TRIM_REDIS_TTL_API_KEY_SEC"), 0)
		cfg.RedisAuthProvidersTTLSec = intFromEnv(get("TRIM_REDIS_TTL_AUTH_PROVIDERS_SEC"), 0)
		cfg.RedisQuotaCacheTTLSec = intFromEnv(get("TRIM_REDIS_TTL_QUOTA_SEC"), 0)
		cfg.RedisHardwareGraphTTLSec = intFromEnv(get("TRIM_REDIS_TTL_HARDWARE_GRAPH_SEC"), 0)
		cfg.RedisIPBlockTTLSec = intFromEnv(get("TRIM_REDIS_TTL_IP_BLOCK_SEC"), 0)
		cfg.RedisJA4SeenTTLSec = intFromEnv(get("TRIM_REDIS_TTL_JA4_SEEN_SEC"), 0)
		cfg.RedisIPRiskTTLSec = intFromEnv(get("TRIM_REDIS_TTL_IP_RISK_SEC"), 0)
		cfg.RedisASNCheckedTTLSec = intFromEnv(get("TRIM_REDIS_TTL_ASN_CHECKED_SEC"), 0)
		cfg.RedisASNMissTTLSec = intFromEnv(get("TRIM_REDIS_TTL_ASN_MISS_SEC"), 0)
		cfg.RedisTelemetryTTLSec = intFromEnv(get("TRIM_REDIS_TTL_TELEMETRY_SEC"), 0)
		cfg.ExpireBatchDefaultLimit = intFromEnv(get("TRIM_EXPIRE_BATCH_DEFAULT_LIMIT"), 0)
	} else {
		// Self-host: optional fraud knobs; proxy + listen timeouts still required (no invent / infinite).
		cfg.CFThreatScoreMin = intFromEnv(optional("TRIM_CF_THREAT_SCORE_MIN"), 0)
		cfg.CLITimestampSkewSec = intFromEnv(optional("TRIM_CLI_TIMESTAMP_SKEW_SEC"), 0)
		cfg.RateLimitPerIPPerMin = intFromEnv(optional("TRIM_RATE_LIMIT_IP_PER_MIN"), 0)
		cfg.RateLimitPerUserPerMin = intFromEnv(optional("TRIM_RATE_LIMIT_USER_PER_MIN"), 0)
		cfg.IPRiskLookupTimeoutMs = intFromEnv(optional("TRIM_IP_RISK_LOOKUP_TIMEOUT_MS"), 0)
		cfg.PaddleHTTPTimeoutSec = intFromEnv(optional("PADDLE_HTTP_TIMEOUT_SEC"), 0)
		cfg.PaddleListMaxPages = intFromEnv(optional("PADDLE_LIST_MAX_PAGES"), 0)
		cfg.PaddleListPerPage = intFromEnv(optional("PADDLE_LIST_PER_PAGE"), 0)
		proxyTO := get("TRIM_PROXY_HTTP_TIMEOUT_SEC")
		readHdr := get("TRIM_HTTP_READ_HEADER_TIMEOUT_SEC")
		shutdownTO := get("TRIM_HTTP_SHUTDOWN_TIMEOUT_SEC")
		cfg.ProxyHTTPTimeoutSec = intFromEnv(proxyTO, 0)
		cfg.HTTPReadHeaderTimeoutSec = intFromEnv(readHdr, 0)
		cfg.HTTPShutdownTimeoutSec = intFromEnv(shutdownTO, 0)
		cfg.WebhookProcessTimeoutSec = intFromEnv(optional("TRIM_WEBHOOK_PROCESS_TIMEOUT_SEC"), 0)
		quotaTO := get("TRIM_QUOTA_SYNC_TIMEOUT_SEC")
		eventTO := get("TRIM_EVENT_INSERT_TIMEOUT_SEC")
		cfg.QuotaSyncTimeoutSec = intFromEnv(quotaTO, 0)
		cfg.EventInsertTimeoutSec = intFromEnv(eventTO, 0)
		cfg.EventInsertQueueSize = intFromEnv(get("TRIM_EVENT_INSERT_QUEUE_SIZE"), 0)
		cfg.EventInsertWorkers = intFromEnv(get("TRIM_EVENT_INSERT_WORKERS"), 0)
		cfg.EventOutboxPollSec = intFromEnv(get("TRIM_EVENT_OUTBOX_POLL_SEC"), 0)
		cfg.EventOutboxBatch = intFromEnv(get("TRIM_EVENT_OUTBOX_BATCH"), 0)
		cfg.WebhookQueueSize = intFromEnv(get("TRIM_WEBHOOK_QUEUE_SIZE"), 0)
		cfg.WebhookWorkers = intFromEnv(get("TRIM_WEBHOOK_WORKERS"), 0)
		cfg.PGMaxConns = int32(intFromEnv(get("TRIM_PG_MAX_CONNS"), 0))
		cfg.PGMinConns = int32(intFromEnv(get("TRIM_PG_MIN_CONNS"), 0))
		cfg.PGReadMaxConns = int32(intFromEnv(optional("TRIM_PG_READ_MAX_CONNS"), 0))
		cfg.PGReadMinConns = int32(intFromEnv(optional("TRIM_PG_READ_MIN_CONNS"), 0))
		cfg.ReadyzTimeoutSec = intFromEnv(get("TRIM_READYZ_TIMEOUT_SEC"), 0)
		cfg.CORSMaxAgeSec = intFromEnv(get("TRIM_CORS_MAX_AGE_SEC"), 0)
		cfg.SiteMessagesReloadSec = intFromEnv(get("TRIM_SITE_MESSAGES_RELOAD_SEC"), 0)
		cfg.RedisVelocityBucketTTLSec = intFromEnv(optional("TRIM_REDIS_TTL_VELOCITY_SEC"), 0)
		cfg.RedisAPIKeyCacheTTLSec = intFromEnv(optional("TRIM_REDIS_TTL_API_KEY_SEC"), 0)
		cfg.RedisAuthProvidersTTLSec = intFromEnv(optional("TRIM_REDIS_TTL_AUTH_PROVIDERS_SEC"), 0)
		cfg.RedisQuotaCacheTTLSec = intFromEnv(optional("TRIM_REDIS_TTL_QUOTA_SEC"), 0)
		cfg.RedisHardwareGraphTTLSec = intFromEnv(optional("TRIM_REDIS_TTL_HARDWARE_GRAPH_SEC"), 0)
		cfg.RedisIPBlockTTLSec = intFromEnv(optional("TRIM_REDIS_TTL_IP_BLOCK_SEC"), 0)
		cfg.RedisJA4SeenTTLSec = intFromEnv(optional("TRIM_REDIS_TTL_JA4_SEEN_SEC"), 0)
		cfg.RedisIPRiskTTLSec = intFromEnv(optional("TRIM_REDIS_TTL_IP_RISK_SEC"), 0)
		cfg.RedisASNCheckedTTLSec = intFromEnv(optional("TRIM_REDIS_TTL_ASN_CHECKED_SEC"), 0)
		cfg.RedisASNMissTTLSec = intFromEnv(optional("TRIM_REDIS_TTL_ASN_MISS_SEC"), 0)
		cfg.RedisTelemetryTTLSec = intFromEnv(optional("TRIM_REDIS_TTL_TELEMETRY_SEC"), 0)
		cfg.ExpireBatchDefaultLimit = intFromEnv(optional("TRIM_EXPIRE_BATCH_DEFAULT_LIMIT"), 0)
	}

	// Enterprise notify timeout required whenever notify URL is set (no invent 8s).
	if strings.TrimSpace(cfg.EnterpriseNotifyURL) != "" {
		entTO := get("TRIM_ENTERPRISE_NOTIFY_TIMEOUT_SEC")
		cfg.EnterpriseNotifyTimeoutSec = intFromEnv(entTO, 0)
	} else if v := optional("TRIM_ENTERPRISE_NOTIFY_TIMEOUT_SEC"); v != "" {
		cfg.EnterpriseNotifyTimeoutSec = intFromEnv(v, 0)
	}

	// Cloudinary timeout: required only when Cloudinary is configured (no invent 30s).
	cloudName := optional("CLOUDINARY_CLOUD_NAME")
	cloudTO := optional("CLOUDINARY_HTTP_TIMEOUT_SEC")
	if cloudName != "" {
		if cloudTO == "" {
			missing = append(missing, "CLOUDINARY_HTTP_TIMEOUT_SEC")
		} else {
			cfg.CloudinaryHTTPTimeoutSec = intFromEnv(cloudTO, 0)
		}
	} else if cloudTO != "" {
		cfg.CloudinaryHTTPTimeoutSec = intFromEnv(cloudTO, 0)
	}

	if mode == "cloud" {
		if cfg.CompressionMode == "" {
			missing = append(missing, "TRIM_COMPRESSION_MODE")
		}
	}
	if cfg.CompressionMode != "" {
		switch strings.ToLower(cfg.CompressionMode) {
		case "mild", "balanced", "aggressive", "custom":
			cfg.CompressionMode = strings.ToLower(cfg.CompressionMode)
		default:
			return Config{}, fmt.Errorf("TRIM_COMPRESSION_MODE must be mild, balanced, aggressive, or custom")
		}
	}

	if mode == "cloud" {
		cfg.SupabaseURL = get("SUPABASE_URL")
		cfg.SupabaseAnonKey = get("SUPABASE_ANON_KEY")
		cfg.SupabaseServiceKey = get("SUPABASE_SERVICE_ROLE_KEY")
		cfg.PaddleWebhookSecret = get("PADDLE_WEBHOOK_SECRET")
		cfg.PaddleAPIKey = get("PADDLE_API_KEY")
		cfg.PaddleEnv = get("PADDLE_ENV")
		cfg.CLIHMACSecret = get("TRIM_CLI_HMAC_SECRET")
	}

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	if mode != "cloud" && mode != "self" {
		return Config{}, fmt.Errorf("DEPLOYMENT_MODE must be cloud or self, got %q", mode)
	}
	if mode == "cloud" && cfg.PaddleEnv != "sandbox" && cfg.PaddleEnv != "production" {
		return Config{}, fmt.Errorf("PADDLE_ENV must be sandbox or production, got %q", cfg.PaddleEnv)
	}
	if mode == "cloud" && len(cfg.AllowedAuthProviders) == 0 {
		return Config{}, fmt.Errorf("ALLOWED_AUTH_PROVIDERS must list at least one provider (e.g. google,github,gitlab)")
	}
	if mode == "cloud" && len(cfg.AdminAllowedOrigins) == 0 {
		return Config{}, fmt.Errorf("ADMIN_ALLOWED_ORIGINS must list at least one admin app origin in cloud (e.g. https://admin.use-trim.com)")
	}
	if mode == "cloud" {
		if cfg.MaxAccountsPerHardware < 1 || cfg.MaxAccountsPerJA4 < 1 {
			return Config{}, fmt.Errorf("TRIM_MAX_ACCOUNTS_PER_HARDWARE and TRIM_MAX_ACCOUNTS_PER_JA4 must be >= 1 in cloud")
		}
		if cfg.CLITimestampSkewSec < 5 || cfg.CLITimestampSkewSec > 300 {
			return Config{}, fmt.Errorf("TRIM_CLI_TIMESTAMP_SKEW_SEC must be 5-300")
		}
		if cfg.RateLimitPerIPPerMin < 1 || cfg.RateLimitPerUserPerMin < 1 {
			return Config{}, fmt.Errorf("TRIM_RATE_LIMIT_IP_PER_MIN and TRIM_RATE_LIMIT_USER_PER_MIN must be >= 1 in cloud")
		}
		if cfg.IPRiskLookupTimeoutMs < 100 || cfg.IPRiskLookupTimeoutMs > 30000 {
			return Config{}, fmt.Errorf("TRIM_IP_RISK_LOOKUP_TIMEOUT_MS must be 100-30000 in cloud")
		}
		if cfg.PaddleHTTPTimeoutSec < 1 || cfg.PaddleHTTPTimeoutSec > 300 {
			return Config{}, fmt.Errorf("PADDLE_HTTP_TIMEOUT_SEC must be 1-300 in cloud")
		}
		if cfg.PaddleListMaxPages < 1 || cfg.PaddleListMaxPages > 1000 {
			return Config{}, fmt.Errorf("PADDLE_LIST_MAX_PAGES must be 1-1000 in cloud")
		}
		if cfg.PaddleListPerPage < 1 || cfg.PaddleListPerPage > 200 {
			return Config{}, fmt.Errorf("PADDLE_LIST_PER_PAGE must be 1-200 in cloud")
		}
		if cfg.CFThreatScoreMin < 1 || cfg.CFThreatScoreMin > 100 {
			return Config{}, fmt.Errorf("TRIM_CF_THREAT_SCORE_MIN must be 1-100 in cloud (Cloudflare CF-Threat-Score floor)")
		}
		if cfg.WebhookProcessTimeoutSec < 1 || cfg.WebhookProcessTimeoutSec > 300 {
			return Config{}, fmt.Errorf("TRIM_WEBHOOK_PROCESS_TIMEOUT_SEC must be 1-300 in cloud")
		}
		if err := validateRedisTTLs(cfg, true); err != nil {
			return Config{}, err
		}
		if cfg.ExpireBatchDefaultLimit < 1 || cfg.ExpireBatchDefaultLimit > 2000 {
			return Config{}, fmt.Errorf("TRIM_EXPIRE_BATCH_DEFAULT_LIMIT must be 1-2000 in cloud")
		}
	} else if cfg.CLITimestampSkewSec != 0 && (cfg.CLITimestampSkewSec < 5 || cfg.CLITimestampSkewSec > 300) {
		return Config{}, fmt.Errorf("TRIM_CLI_TIMESTAMP_SKEW_SEC must be 5-300 when set")
	}
	if cfg.ProxyHTTPTimeoutSec < 1 || cfg.ProxyHTTPTimeoutSec > 600 {
		return Config{}, fmt.Errorf("TRIM_PROXY_HTTP_TIMEOUT_SEC must be 1-600")
	}
	if cfg.HTTPReadHeaderTimeoutSec < 1 || cfg.HTTPReadHeaderTimeoutSec > 120 {
		return Config{}, fmt.Errorf("TRIM_HTTP_READ_HEADER_TIMEOUT_SEC must be 1-120")
	}
	if cfg.HTTPShutdownTimeoutSec < 1 || cfg.HTTPShutdownTimeoutSec > 120 {
		return Config{}, fmt.Errorf("TRIM_HTTP_SHUTDOWN_TIMEOUT_SEC must be 1-120")
	}
	if cfg.QuotaSyncTimeoutSec < 1 || cfg.QuotaSyncTimeoutSec > 60 {
		return Config{}, fmt.Errorf("TRIM_QUOTA_SYNC_TIMEOUT_SEC must be 1-60")
	}
	if cfg.EventInsertTimeoutSec < 1 || cfg.EventInsertTimeoutSec > 60 {
		return Config{}, fmt.Errorf("TRIM_EVENT_INSERT_TIMEOUT_SEC must be 1-60")
	}
	if cfg.EventInsertQueueSize < 1 || cfg.EventInsertQueueSize > 100000 {
		return Config{}, fmt.Errorf("TRIM_EVENT_INSERT_QUEUE_SIZE must be 1-100000")
	}
	if cfg.EventInsertWorkers < 1 || cfg.EventInsertWorkers > 64 {
		return Config{}, fmt.Errorf("TRIM_EVENT_INSERT_WORKERS must be 1-64")
	}
	if cfg.EventOutboxPollSec < 1 || cfg.EventOutboxPollSec > 300 {
		return Config{}, fmt.Errorf("TRIM_EVENT_OUTBOX_POLL_SEC must be 1-300")
	}
	if cfg.EventOutboxBatch < 1 || cfg.EventOutboxBatch > 500 {
		return Config{}, fmt.Errorf("TRIM_EVENT_OUTBOX_BATCH must be 1-500")
	}
	if cfg.PGMaxConns < 1 || cfg.PGMaxConns > 500 {
		return Config{}, fmt.Errorf("TRIM_PG_MAX_CONNS must be 1-500")
	}
	if cfg.PGMinConns < 0 || cfg.PGMinConns > cfg.PGMaxConns {
		return Config{}, fmt.Errorf("TRIM_PG_MIN_CONNS must be 0..TRIM_PG_MAX_CONNS")
	}
	if cfg.ReadyzTimeoutSec < 1 || cfg.ReadyzTimeoutSec > 60 {
		return Config{}, fmt.Errorf("TRIM_READYZ_TIMEOUT_SEC must be 1-60")
	}
	if cfg.CORSMaxAgeSec < 1 || cfg.CORSMaxAgeSec > 86400 {
		return Config{}, fmt.Errorf("TRIM_CORS_MAX_AGE_SEC must be 1-86400")
	}
	if cfg.SiteMessagesReloadSec < 1 || cfg.SiteMessagesReloadSec > 3600 {
		return Config{}, fmt.Errorf("TRIM_SITE_MESSAGES_RELOAD_SEC must be 1-3600")
	}
	if strings.TrimSpace(cfg.DatabaseReadURL) != "" {
		if cfg.PGReadMaxConns < 1 || cfg.PGReadMaxConns > 500 {
			return Config{}, fmt.Errorf("TRIM_PG_READ_MAX_CONNS must be 1-500 when DATABASE_READ_URL is set")
		}
		if cfg.PGReadMinConns < 0 || cfg.PGReadMinConns > cfg.PGReadMaxConns {
			return Config{}, fmt.Errorf("TRIM_PG_READ_MIN_CONNS must be 0..TRIM_PG_READ_MAX_CONNS when DATABASE_READ_URL is set")
		}
	} else if cfg.PGReadMaxConns != 0 || cfg.PGReadMinConns != 0 {
		return Config{}, fmt.Errorf("TRIM_PG_READ_MAX_CONNS/TRIM_PG_READ_MIN_CONNS require DATABASE_READ_URL")
	}
	if mode == "cloud" {
		if cfg.WebhookQueueSize < 1 || cfg.WebhookQueueSize > 100000 {
			return Config{}, fmt.Errorf("TRIM_WEBHOOK_QUEUE_SIZE must be 1-100000 in cloud")
		}
		if cfg.WebhookWorkers < 1 || cfg.WebhookWorkers > 64 {
			return Config{}, fmt.Errorf("TRIM_WEBHOOK_WORKERS must be 1-64 in cloud")
		}
	} else {
		if cfg.WebhookQueueSize < 1 || cfg.WebhookQueueSize > 100000 {
			return Config{}, fmt.Errorf("TRIM_WEBHOOK_QUEUE_SIZE must be 1-100000")
		}
		if cfg.WebhookWorkers < 1 || cfg.WebhookWorkers > 64 {
			return Config{}, fmt.Errorf("TRIM_WEBHOOK_WORKERS must be 1-64")
		}
	}
	if cfg.PaddleHTTPTimeoutSec != 0 && (cfg.PaddleHTTPTimeoutSec < 1 || cfg.PaddleHTTPTimeoutSec > 300) {
		return Config{}, fmt.Errorf("PADDLE_HTTP_TIMEOUT_SEC must be 1-300 when set")
	}
	if cfg.CloudinaryHTTPTimeoutSec != 0 && (cfg.CloudinaryHTTPTimeoutSec < 1 || cfg.CloudinaryHTTPTimeoutSec > 300) {
		return Config{}, fmt.Errorf("CLOUDINARY_HTTP_TIMEOUT_SEC must be 1-300 when set")
	}
	if cfg.EnterpriseNotifyTimeoutSec != 0 && (cfg.EnterpriseNotifyTimeoutSec < 1 || cfg.EnterpriseNotifyTimeoutSec > 60) {
		return Config{}, fmt.Errorf("TRIM_ENTERPRISE_NOTIFY_TIMEOUT_SEC must be 1-60 when set")
	}
	if cfg.WebhookProcessTimeoutSec != 0 && (cfg.WebhookProcessTimeoutSec < 1 || cfg.WebhookProcessTimeoutSec > 300) {
		return Config{}, fmt.Errorf("TRIM_WEBHOOK_PROCESS_TIMEOUT_SEC must be 1-300 when set")
	}
	if mode != "cloud" {
		if err := validateRedisTTLs(cfg, false); err != nil {
			return Config{}, err
		}
		if cfg.ExpireBatchDefaultLimit != 0 && (cfg.ExpireBatchDefaultLimit < 1 || cfg.ExpireBatchDefaultLimit > 2000) {
			return Config{}, fmt.Errorf("TRIM_EXPIRE_BATCH_DEFAULT_LIMIT must be 1-2000 when set")
		}
	}
	if len(cfg.CORSOrigins) == 0 {
		return Config{}, fmt.Errorf("CORS_ORIGINS must include at least one origin")
	}
	return cfg, nil
}

func validateRedisTTLs(cfg Config, required bool) error {
	type field struct {
		name string
		val  int
		min  int
		max  int
	}
	fields := []field{
		{"TRIM_REDIS_TTL_VELOCITY_SEC", cfg.RedisVelocityBucketTTLSec, 1, 600},
		{"TRIM_REDIS_TTL_API_KEY_SEC", cfg.RedisAPIKeyCacheTTLSec, 1, 86400 * 7},
		{"TRIM_REDIS_TTL_AUTH_PROVIDERS_SEC", cfg.RedisAuthProvidersTTLSec, 1, 3600},
		{"TRIM_REDIS_TTL_QUOTA_SEC", cfg.RedisQuotaCacheTTLSec, 1, 86400 * 7},
		{"TRIM_REDIS_TTL_HARDWARE_GRAPH_SEC", cfg.RedisHardwareGraphTTLSec, 1, 86400 * 90},
		{"TRIM_REDIS_TTL_IP_BLOCK_SEC", cfg.RedisIPBlockTTLSec, 1, 86400 * 90},
		{"TRIM_REDIS_TTL_JA4_SEEN_SEC", cfg.RedisJA4SeenTTLSec, 1, 86400 * 90},
		{"TRIM_REDIS_TTL_IP_RISK_SEC", cfg.RedisIPRiskTTLSec, 1, 86400 * 7},
		{"TRIM_REDIS_TTL_ASN_CHECKED_SEC", cfg.RedisASNCheckedTTLSec, 1, 86400 * 7},
		{"TRIM_REDIS_TTL_ASN_MISS_SEC", cfg.RedisASNMissTTLSec, 1, 86400},
		{"TRIM_REDIS_TTL_TELEMETRY_SEC", cfg.RedisTelemetryTTLSec, 1, 86400 * 90},
	}
	for _, f := range fields {
		if f.val == 0 {
			if required {
				return fmt.Errorf("%s must be set in cloud (no invent Redis TTL)", f.name)
			}
			continue
		}
		if f.val < f.min || f.val > f.max {
			return fmt.Errorf("%s must be %d-%d when set", f.name, f.min, f.max)
		}
	}
	return nil
}

func intFromEnv(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
