package config

import "strings"

// Official Trim Cloud product defaults for the published CLI (npm / Homebrew / installers).
// These are declared product constants for the hosted service - not silent invents.
// Self-hosters and operators override any field via environment variables or .env.
//
// Industry practice (Stripe CLI, Vercel CLI, Cloudflare Wrangler, etc.):
// official client binaries ship ready-to-run against the vendor cloud; env is for override.
const (
	HostedPort                      = "8888"
	HostedUpstreamOpenAI            = "https://api.openai.com"
	HostedUpstreamAnthropic         = "https://api.anthropic.com"
	HostedAPIBaseURL                = "https://api.use-trim.com"
	HostedAppPublicURL              = "https://use-trim.com"
	HostedSavingsUsdPerMTok         = 3.0
	HostedProxyHTTPTimeoutSec       = 120
	HostedCLIHTTPTimeoutSec         = 30
	HostedDeepTimeoutSec            = 600
	HostedHTTPReadHeaderTimeoutSec  = 10
	HostedTUIRefreshSec             = 2
	HostedStatsSeriesDays           = 14
	HostedTUISeriesDays             = 7
	HostedProxyPreviewMaxChars      = 1200
	HostedTUIErrorMaxChars          = 80
	HostedDeepDeviceMap             = "cpu"
)

// HostedCLIHMACSecret is the Trim Cloud request-integrity secret baked into official
// release binaries via GoReleaser / CI ldflags (never committed to source).
// Empty in local `go build` / `go test` unless set with:
//
//	-ldflags "-X github.com/usetrim/trim/cli/internal/config.HostedCLIHMACSecret=..."
//
// Env TRIM_CLI_HMAC_SECRET always wins when set (self-host / local override).
var HostedCLIHMACSecret string

// IsOfficialHostedAPI reports whether apiBase is the official Trim Cloud API origin.
func IsOfficialHostedAPI(apiBase string) bool {
	return normalizeOrigin(apiBase) == normalizeOrigin(HostedAPIBaseURL)
}

func normalizeOrigin(u string) string {
	s := strings.TrimSpace(u)
	s = strings.TrimRight(s, "/")
	return strings.ToLower(s)
}
