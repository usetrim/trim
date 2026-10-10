// Package provideradapt implements DB-driven OpenAI-compat → provider dialect plugins.
// Anthropic Messages is the first dialect; more dialects register the same way (no marketplace).
package provideradapt

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Dialect identifiers (must match provider_adapters.dialect check constraint).
const (
	DialectAnthropicMessages = "anthropic_messages"
	DialectOpenAICompat      = "openai_compat"
)

// UpstreamKind matches provider_adapters.upstream_kind.
const (
	UpstreamAnthropic = "anthropic"
	UpstreamOpenAI    = "openai"
)

// AuthMode matches provider_adapters.auth_mode.
const (
	AuthBearerToXAPIKey = "bearer_to_x_api_key"
	AuthPassthrough     = "passthrough"
)

// HeaderAnthropicWorkspaceID is Anthropic's required request header for multi-workspace
// (organization-scoped) API keys. Workspace-scoped keys omit it. See Anthropic Authentication docs.
const HeaderAnthropicWorkspaceID = "anthropic-workspace-id"

// OpenAI organization / project headers (optional). Client headers win; else process env.
// See OpenAI Auth docs: OpenAI-Organization / OpenAI-Project.
const (
	HeaderOpenAIOrganization = "OpenAI-Organization"
	HeaderOpenAIProject      = "OpenAI-Project"
)

// AdapterConfig is one row from public.provider_adapters (synced to CLI; never invent).
type AdapterConfig struct {
	ID                   string            `json:"id"`
	Enabled              bool              `json:"enabled"`
	SortOrder            int               `json:"sort_order"`
	Dialect              string            `json:"dialect"`
	DoorLabel            string            `json:"door_label"`
	MatchModelPrefixes   []string          `json:"match_model_prefixes"`
	ModelAliases         map[string]string `json:"model_aliases"`
	UpstreamKind         string            `json:"upstream_kind"`
	UpstreamPath         string            `json:"upstream_path"`
	UpstreamBaseURL      string            `json:"upstream_base_url,omitempty"`
	AuthMode             string            `json:"auth_mode"`
	AnthropicVersion string `json:"anthropic_version,omitempty"`
	// AnthropicWorkspaceID may appear in synced prefs from an older Admin column; Trim does not
	// inject it. Official product path: workspace-scoped key (omit), or client header, or
	// TRIM_ANTHROPIC_WORKSPACE_ID. Non-empty values are still validated fail-closed.
	AnthropicWorkspaceID string `json:"anthropic_workspace_id,omitempty"`
	DefaultMaxTokens     int    `json:"default_max_tokens,omitempty"`
	RequireAlias         bool   `json:"require_alias"`
}

// DiscoverableModel is one row from public.provider_discoverable_models (synced to CLI).
type DiscoverableModel struct {
	ID          string `json:"id"`
	Enabled     bool   `json:"enabled"`
	SortOrder   int    `json:"sort_order"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	OwnedBy     string `json:"owned_by"`
	AdapterID   string `json:"adapter_id,omitempty"`
	Created     int64  `json:"created"` // unix seconds from updated_at
}

// ModelAlias is one row from public.openai_model_aliases.
type ModelAlias struct {
	ClientModel          string `json:"client_model"`
	UpstreamModel        string `json:"upstream_model"`
	UpstreamHostContains string `json:"upstream_host_contains"`
	Enabled              bool   `json:"enabled"`
}

// Chrome is fail-closed copy from site_messages (empty = generic Go errors only).
type Chrome struct {
	UnknownDialectFmt  string
	ModelRequired      string
	AliasRequiredFmt   string
	TranslateFmt       string
	AuthFmt            string
	ResponseFmt        string
	ModelsNotSynced    string
	ModelsMethod       string
	BaseURLDoubleV1    string
	ModelNotFoundFmt   string
	UpstreamAuthFmt    string
	UpstreamQuotaFmt   string
}

// ValidateConfig fail-closes invalid DB rows (no invent).
func ValidateConfig(c AdapterConfig) error {
	id := strings.TrimSpace(c.ID)
	if id == "" {
		return fmt.Errorf("provider_adapters.id required")
	}
	if strings.TrimSpace(c.DoorLabel) == "" {
		return fmt.Errorf("provider_adapters.door_label required for %s", id)
	}
	if strings.TrimSpace(c.UpstreamPath) == "" {
		return fmt.Errorf("provider_adapters.upstream_path required for %s", id)
	}
	baseURL := strings.TrimSpace(c.UpstreamBaseURL)
	switch strings.TrimSpace(c.Dialect) {
	case DialectAnthropicMessages:
		if strings.TrimSpace(c.AnthropicVersion) == "" {
			return fmt.Errorf("provider_adapters.anthropic_version required for %s", id)
		}
		if c.DefaultMaxTokens < 1 {
			return fmt.Errorf("provider_adapters.default_max_tokens required for %s", id)
		}
		if err := validateAnthropicWorkspaceID(c.AnthropicWorkspaceID, id); err != nil {
			return err
		}
		if baseURL != "" {
			if err := validateUpstreamBaseURL(baseURL, id); err != nil {
				return err
			}
		}
	case DialectOpenAICompat:
		if baseURL == "" {
			return fmt.Errorf("provider_adapters.upstream_base_url required for openai_compat adapter %s", id)
		}
		if err := validateUpstreamBaseURL(baseURL, id); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported dialect %q for adapter %s", c.Dialect, id)
	}
	switch strings.TrimSpace(c.UpstreamKind) {
	case UpstreamAnthropic, UpstreamOpenAI:
	default:
		return fmt.Errorf("unsupported upstream_kind %q for adapter %s", c.UpstreamKind, id)
	}
	switch strings.TrimSpace(c.AuthMode) {
	case AuthBearerToXAPIKey, AuthPassthrough:
	default:
		return fmt.Errorf("unsupported auth_mode %q for adapter %s", c.AuthMode, id)
	}
	if c.MatchModelPrefixes == nil {
		return fmt.Errorf("provider_adapters.match_model_prefixes required for %s", id)
	}
	if c.ModelAliases == nil {
		return fmt.Errorf("provider_adapters.model_aliases required for %s", id)
	}
	return nil
}

// NormalizeAliasMap lowercases keys for stable matching.
func NormalizeAliasMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		lk := strings.ToLower(strings.TrimSpace(k))
		uv := strings.TrimSpace(v)
		if lk == "" || uv == "" {
			continue
		}
		out[lk] = uv
	}
	return out
}

// NormalizePrefixes lowercases + trims prefixes; drops empties.
func NormalizePrefixes(in []string) []string {
	var out []string
	for _, p := range in {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ParseAdaptersJSON unmarshals preferences / admin payload; nil slice is invalid (fail-closed).
func ParseAdaptersJSON(raw json.RawMessage) ([]AdapterConfig, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, fmt.Errorf("provider_adapters json required")
	}
	var list []AdapterConfig
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, err
	}
	if list == nil {
		return nil, fmt.Errorf("provider_adapters must be a JSON array")
	}
	for i := range list {
		list[i].MatchModelPrefixes = NormalizePrefixes(list[i].MatchModelPrefixes)
		list[i].ModelAliases = NormalizeAliasMap(list[i].ModelAliases)
		if err := ValidateConfig(list[i]); err != nil {
			return nil, err
		}
	}
	return list, nil
}

// ParseAliasesJSON unmarshals openai_model_aliases; nil slice is invalid.
func ParseAliasesJSON(raw json.RawMessage) ([]ModelAlias, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, fmt.Errorf("openai_model_aliases json required")
	}
	var list []ModelAlias
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, err
	}
	if list == nil {
		return nil, fmt.Errorf("openai_model_aliases must be a JSON array")
	}
	return list, nil
}

// ValidateDiscoverableModel fail-closes invalid catalog rows (no invent).
func ValidateDiscoverableModel(m DiscoverableModel) error {
	id := strings.TrimSpace(m.ID)
	if id == "" || len(id) > 256 {
		return fmt.Errorf("provider_discoverable_models.id invalid")
	}
	if strings.TrimSpace(m.OwnedBy) == "" || len(strings.TrimSpace(m.OwnedBy)) > 128 {
		return fmt.Errorf("provider_discoverable_models.owned_by required for %s", id)
	}
	if m.SortOrder < 0 || m.SortOrder > 1000000 {
		return fmt.Errorf("provider_discoverable_models.sort_order invalid for %s", id)
	}
	return nil
}

// validateAnthropicWorkspaceID fail-closes non-empty ids that are not Anthropic workspace ids.
// Empty is allowed (workspace-scoped keys omit the header). Official ids use the wrkspc_ prefix.
func validateAnthropicWorkspaceID(raw, adapterID string) error {
	w := strings.TrimSpace(raw)
	if w == "" {
		return nil
	}
	if len(w) < 8 || len(w) > 128 {
		return fmt.Errorf("provider_adapters.anthropic_workspace_id length invalid for %s", adapterID)
	}
	if !strings.HasPrefix(strings.ToLower(w), "wrkspc_") {
		return fmt.Errorf("provider_adapters.anthropic_workspace_id must start with wrkspc_ for %s", adapterID)
	}
	return nil
}

func validateUpstreamBaseURL(raw, id string) error {
	u, err := url.Parse(raw)
	if err != nil || u == nil {
		return fmt.Errorf("provider_adapters.upstream_base_url invalid for %s", id)
	}
	if strings.TrimSpace(u.Host) == "" {
		return fmt.Errorf("provider_adapters.upstream_base_url host required for %s", id)
	}
	scheme := strings.ToLower(strings.TrimSpace(u.Scheme))
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	switch scheme {
	case "https":
		return nil
	case "http":
		// Local mock / loopback only (httptest). Production DB seeds must stay https.
		if host == "127.0.0.1" || host == "localhost" || host == "::1" {
			return nil
		}
		return fmt.Errorf("provider_adapters.upstream_base_url http only allowed on loopback for %s", id)
	default:
		return fmt.Errorf("provider_adapters.upstream_base_url must be https for %s", id)
	}
}

// ResolveUpstreamBase returns cfg.UpstreamBaseURL when set; otherwise envBase for that upstream_kind.
// Fail-closed when neither is usable.
func ResolveUpstreamBase(cfg AdapterConfig, envOpenAI, envAnthropic string) (string, error) {
	if base := strings.TrimSpace(cfg.UpstreamBaseURL); base != "" {
		if err := validateUpstreamBaseURL(base, cfg.ID); err != nil {
			return "", err
		}
		return strings.TrimRight(base, "/"), nil
	}
	switch strings.TrimSpace(cfg.UpstreamKind) {
	case UpstreamAnthropic:
		base := strings.TrimSpace(envAnthropic)
		if base == "" {
			return "", fmt.Errorf("adapter %s needs UPSTREAM_ANTHROPIC_URL or upstream_base_url", cfg.ID)
		}
		return strings.TrimRight(base, "/"), nil
	case UpstreamOpenAI:
		base := strings.TrimSpace(envOpenAI)
		if base == "" {
			return "", fmt.Errorf("adapter %s needs UPSTREAM_OPENAI_URL or upstream_base_url", cfg.ID)
		}
		return strings.TrimRight(base, "/"), nil
	default:
		return "", fmt.Errorf("unsupported upstream_kind %q for adapter %s", cfg.UpstreamKind, cfg.ID)
	}
}

// ApplyAnthropicWorkspaceID sets anthropic-workspace-id when missing.
// Client header always wins. fallback is TRIM_ANTHROPIC_WORKSPACE_ID (process env).
// Empty fallback leaves the header unset (workspace-scoped keys omit it per Anthropic docs).
func ApplyAnthropicWorkspaceID(h http.Header, fallback string) {
	if h == nil {
		return
	}
	if strings.TrimSpace(h.Get(HeaderAnthropicWorkspaceID)) != "" {
		return
	}
	if w := strings.TrimSpace(fallback); w != "" {
		h.Set(HeaderAnthropicWorkspaceID, w)
	}
}

// ApplyOpenAIOrgProject sets OpenAI-Organization / OpenAI-Project when missing.
// Client headers always win. Fallbacks are TRIM_OPENAI_ORGANIZATION / TRIM_OPENAI_PROJECT.
func ApplyOpenAIOrgProject(h http.Header, orgFallback, projectFallback string) {
	if h == nil {
		return
	}
	if strings.TrimSpace(h.Get(HeaderOpenAIOrganization)) == "" {
		if o := strings.TrimSpace(orgFallback); o != "" {
			h.Set(HeaderOpenAIOrganization, o)
		}
	}
	if strings.TrimSpace(h.Get(HeaderOpenAIProject)) == "" {
		if p := strings.TrimSpace(projectFallback); p != "" {
			h.Set(HeaderOpenAIProject, p)
		}
	}
}

// FirstAnthropicWorkspaceFallback returns the first non-empty workspace id candidate.
func FirstAnthropicWorkspaceFallback(values ...string) string {
	for _, v := range values {
		if w := strings.TrimSpace(v); w != "" {
			return w
		}
	}
	return ""
}

// MapRequestAuth rewrites client headers for the selected adapter auth_mode.
// Does not inject anthropic-workspace-id from DB; proxy applies env after auth mapping
// (client header is preserved on the cloned header when present).
func MapRequestAuth(src http.Header, cfg AdapterConfig) (http.Header, error) {
	out := src.Clone()
	if out == nil {
		out = make(http.Header)
	}
	switch strings.TrimSpace(cfg.AuthMode) {
	case AuthPassthrough:
		return out, nil
	case AuthBearerToXAPIKey:
		key := bearerToken(out)
		if key == "" {
			// Also accept already-set x-api-key from some clients.
			if strings.TrimSpace(out.Get("x-api-key")) != "" {
				if strings.TrimSpace(cfg.AnthropicVersion) != "" {
					out.Set("anthropic-version", strings.TrimSpace(cfg.AnthropicVersion))
				}
				out.Del("Authorization")
				return out, nil
			}
			return nil, fmt.Errorf("missing Authorization Bearer API key")
		}
		out.Set("x-api-key", key)
		out.Del("Authorization")
		if strings.TrimSpace(cfg.AnthropicVersion) == "" {
			return nil, fmt.Errorf("anthropic_version missing on adapter %s", cfg.ID)
		}
		out.Set("anthropic-version", strings.TrimSpace(cfg.AnthropicVersion))
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported auth_mode %q", cfg.AuthMode)
	}
}

func bearerToken(h http.Header) string {
	raw := strings.TrimSpace(h.Get("Authorization"))
	if raw == "" {
		return ""
	}
	const p = "bearer "
	if len(raw) > len(p) && strings.EqualFold(raw[:len(p)], p) {
		return strings.TrimSpace(raw[len(p):])
	}
	return ""
}
