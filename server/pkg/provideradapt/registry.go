package provideradapt

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// Registry holds enabled adapters + OpenAI model aliases + discoverable models from DB sync.
type Registry struct {
	Adapters     []AdapterConfig
	Aliases      []ModelAlias
	Discoverable []DiscoverableModel
	Chrome       Chrome
}

// NewRegistry keeps only enabled, validated adapters sorted by sort_order then id.
// discoverable must be a non-nil slice (empty allowed); nil fails closed.
func NewRegistry(adapters []AdapterConfig, aliases []ModelAlias, discoverable []DiscoverableModel, chrome Chrome) (*Registry, error) {
	if adapters == nil {
		return nil, fmt.Errorf("provider_adapters array required")
	}
	if aliases == nil {
		return nil, fmt.Errorf("openai_model_aliases array required")
	}
	if discoverable == nil {
		return nil, fmt.Errorf("provider_discoverable_models array required")
	}
	var enabled []AdapterConfig
	for _, c := range adapters {
		if !c.Enabled {
			continue
		}
		c.MatchModelPrefixes = NormalizePrefixes(c.MatchModelPrefixes)
		c.ModelAliases = NormalizeAliasMap(c.ModelAliases)
		if err := ValidateConfig(c); err != nil {
			return nil, err
		}
		enabled = append(enabled, c)
	}
	sort.SliceStable(enabled, func(i, j int) bool {
		if enabled[i].SortOrder != enabled[j].SortOrder {
			return enabled[i].SortOrder < enabled[j].SortOrder
		}
		return enabled[i].ID < enabled[j].ID
	})
	var enabledAliases []ModelAlias
	for _, a := range aliases {
		if !a.Enabled {
			continue
		}
		cm := strings.ToLower(strings.TrimSpace(a.ClientModel))
		um := strings.TrimSpace(a.UpstreamModel)
		if cm == "" || um == "" {
			return nil, fmt.Errorf("openai_model_aliases row missing client_model or upstream_model")
		}
		a.ClientModel = cm
		a.UpstreamModel = um
		a.UpstreamHostContains = strings.ToLower(strings.TrimSpace(a.UpstreamHostContains))
		enabledAliases = append(enabledAliases, a)
	}
	var enabledDisco []DiscoverableModel
	for _, m := range discoverable {
		if !m.Enabled {
			continue
		}
		if err := ValidateDiscoverableModel(m); err != nil {
			return nil, err
		}
		m.ID = strings.TrimSpace(m.ID)
		m.DisplayName = strings.TrimSpace(m.DisplayName)
		m.Description = strings.TrimSpace(m.Description)
		m.OwnedBy = strings.TrimSpace(m.OwnedBy)
		m.AdapterID = strings.TrimSpace(m.AdapterID)
		enabledDisco = append(enabledDisco, m)
	}
	sort.SliceStable(enabledDisco, func(i, j int) bool {
		if enabledDisco[i].SortOrder != enabledDisco[j].SortOrder {
			return enabledDisco[i].SortOrder < enabledDisco[j].SortOrder
		}
		return enabledDisco[i].ID < enabledDisco[j].ID
	})
	return &Registry{Adapters: enabled, Aliases: enabledAliases, Discoverable: enabledDisco, Chrome: chrome}, nil
}

// EnabledCount is for start-up logging.
func (r *Registry) EnabledCount() int {
	if r == nil {
		return 0
	}
	return len(r.Adapters)
}

// ByID returns an enabled adapter by id (exact).
func (r *Registry) ByID(id string) (AdapterConfig, bool) {
	if r == nil {
		return AdapterConfig{}, false
	}
	want := strings.TrimSpace(id)
	if want == "" {
		return AdapterConfig{}, false
	}
	for _, c := range r.Adapters {
		if c.ID == want {
			return c, true
		}
	}
	return AdapterConfig{}, false
}

// OpenAICompatModelsURL builds GET …/models for an openai_compat host.
// Uses upstream_path to choose /v1/models vs /models (OpenAI/Mistral vs Gemini/DeepSeek shapes).
func OpenAICompatModelsURL(upstreamBase, upstreamPath string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(upstreamBase), "/")
	path := strings.TrimSpace(upstreamPath)
	if base == "" {
		return "", fmt.Errorf("upstream_base_url required")
	}
	if _, err := url.Parse(base); err != nil {
		return "", fmt.Errorf("invalid upstream_base_url")
	}
	lowerBase := strings.ToLower(base)
	// Google OpenAI-compat: …/openai + /chat/completions → …/openai/models
	if strings.HasSuffix(lowerBase, "/openai") {
		return base + "/models", nil
	}
	// Path already under /v1 (OpenAI, Mistral): {base}/v1/models unless base already ends with /v1.
	if strings.Contains(path, "/v1/") || strings.HasPrefix(path, "/v1") {
		if strings.HasSuffix(lowerBase, "/v1") {
			return base + "/models", nil
		}
		return base + "/v1/models", nil
	}
	// DeepSeek-style: host + /chat/completions → host/models (also accepts /v1/models).
	return base + "/models", nil
}

// RewriteModel applies exact openai_model_aliases rows only (no invent prefix maps).
func (r *Registry) RewriteModel(model, upstreamOpenAIBase string) string {
	if r == nil {
		return model
	}
	lower := strings.ToLower(strings.TrimSpace(model))
	if lower == "" {
		return model
	}
	host := ""
	if u, err := url.Parse(upstreamOpenAIBase); err == nil {
		host = strings.ToLower(u.Host)
	}
	baseLower := strings.ToLower(upstreamOpenAIBase)
	for _, a := range r.Aliases {
		if a.ClientModel != lower {
			continue
		}
		if a.UpstreamHostContains != "" {
			if !strings.Contains(host, a.UpstreamHostContains) && !strings.Contains(baseLower, a.UpstreamHostContains) {
				continue
			}
		}
		return a.UpstreamModel
	}
	return model
}

// Match returns the first enabled adapter whose DB prefix matches model.
func (r *Registry) Match(model string) (AdapterConfig, bool) {
	if r == nil {
		return AdapterConfig{}, false
	}
	lower := strings.ToLower(strings.TrimSpace(model))
	if lower == "" {
		return AdapterConfig{}, false
	}
	for _, a := range r.Adapters {
		for _, p := range a.MatchModelPrefixes {
			if p != "" && strings.HasPrefix(lower, p) {
				return a, true
			}
		}
	}
	return AdapterConfig{}, false
}

// ResolveUpstreamModel applies adapter.model_aliases; enforces require_alias; otherwise
// uses the client model as-is, except namespace prefixes ending in "/" strip the namespace
// (prefix list is DB-driven, e.g. anthropic/claude-x → claude-x).
func (r *Registry) ResolveUpstreamModel(cfg AdapterConfig, model string) (string, error) {
	lower := strings.ToLower(strings.TrimSpace(model))
	if lower == "" {
		if r != nil && r.Chrome.ModelRequired != "" {
			return "", fmt.Errorf("%s", r.Chrome.ModelRequired)
		}
		return "", fmt.Errorf("model required")
	}
	if mapped, ok := cfg.ModelAliases[lower]; ok && strings.TrimSpace(mapped) != "" {
		return strings.TrimSpace(mapped), nil
	}
	if cfg.RequireAlias {
		if r != nil && r.Chrome.AliasRequiredFmt != "" {
			return "", fmt.Errorf(r.Chrome.AliasRequiredFmt, model, cfg.ID)
		}
		return "", fmt.Errorf("model %q requires an alias on adapter %q", model, cfg.ID)
	}
	for _, p := range cfg.MatchModelPrefixes {
		if strings.HasSuffix(p, "/") && strings.HasPrefix(lower, p) {
			rest := strings.TrimPrefix(lower, p)
			if rest != "" {
				return rest, nil
			}
		}
	}
	return lower, nil
}

// ModelsListResponse is OpenAI-shaped + Claude Code gateway fields on each item.
type ModelsListResponse struct {
	Object  string           `json:"object"`
	Data    []ModelsListItem `json:"data"`
	HasMore bool             `json:"has_more"`
	FirstID string           `json:"first_id,omitempty"`
	LastID  string           `json:"last_id,omitempty"`
}

// ModelsListItem satisfies OpenAI SDK clients and Claude Code gateway discovery.
// Official Claude Code reads id + optional display_name; also emits Anthropic-native
// type alongside OpenAI object so both client families parse the same payload.
type ModelsListItem struct {
	ID          string `json:"id"`
	Object      string `json:"object"`
	Type        string `json:"type,omitempty"`
	Created     int64  `json:"created"`
	OwnedBy     string `json:"owned_by"`
	DisplayName string `json:"display_name,omitempty"`
	Description string `json:"description,omitempty"`
}

// ListModels returns DB discoverable models only (no invent from prefixes).
// limit <= 0 means no truncation. When truncated, has_more is true.
func (r *Registry) ListModels(limit int) ModelsListResponse {
	out := ModelsListResponse{Object: "list", Data: []ModelsListItem{}}
	if r == nil {
		return out
	}
	items := r.Discoverable
	if limit > 0 && len(items) > limit {
		items = items[:limit]
		out.HasMore = true
	}
	out.Data = make([]ModelsListItem, 0, len(items))
	for _, m := range items {
		out.Data = append(out.Data, ModelsListItem{
			ID:          m.ID,
			Object:      "model",
			Type:        "model",
			Created:     m.Created,
			OwnedBy:     m.OwnedBy,
			DisplayName: m.DisplayName,
			Description: m.Description,
		})
	}
	if len(out.Data) > 0 {
		out.FirstID = out.Data[0].ID
		out.LastID = out.Data[len(out.Data)-1].ID
	}
	return out
}

// ParseModelsLimit parses ?limit= for GET /v1/models. Empty → 0 (no truncate).
// Invalid values return error (fail-closed; no silent default invent).
func ParseModelsLimit(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	n := 0
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid limit")
		}
		n = n*10 + int(ch-'0')
		if n > 10000 {
			return 0, fmt.Errorf("limit too large")
		}
	}
	return n, nil
}

// LooksLikeModelNotFound reports Anthropic/OpenAI-style model missing errors in upstream bodies.
func LooksLikeModelNotFound(body []byte) bool {
	s := strings.ToLower(string(body))
	if strings.Contains(s, "model_not_found") {
		return true
	}
	if strings.Contains(s, "not_found_error") && strings.Contains(s, "model") {
		return true
	}
	if strings.Contains(s, `"code":"model_not_found"`) || strings.Contains(s, `"code": "model_not_found"`) {
		return true
	}
	return false
}

// LooksLikeAuthError reports upstream auth failures (invalid/missing API key).
func LooksLikeAuthError(body []byte, status int) bool {
	if status == http.StatusUnauthorized {
		return true
	}
	s := strings.ToLower(string(body))
	switch {
	case strings.Contains(s, "authentication_error"):
		return true
	case strings.Contains(s, "invalid_api_key"):
		return true
	case strings.Contains(s, "incorrect api key"):
		return true
	case strings.Contains(s, "invalid api key"):
		return true
	case strings.Contains(s, "api key is invalid"):
		return true
	case strings.Contains(s, "authentication fails"):
		return true
	case strings.Contains(s, "please pass a valid api key"):
		return true
	default:
		return false
	}
}

// LooksLikeQuotaError reports upstream quota / rate-limit / billing exhaustion.
func LooksLikeQuotaError(body []byte, status int) bool {
	if status == http.StatusTooManyRequests {
		return true
	}
	s := strings.ToLower(string(body))
	switch {
	case strings.Contains(s, "rate_limit"):
		return true
	case strings.Contains(s, "rate limit"):
		return true
	case strings.Contains(s, "insufficient_quota"):
		return true
	case strings.Contains(s, "quota exceeded"):
		return true
	case strings.Contains(s, "billing"):
		return strings.Contains(s, "quota") || strings.Contains(s, "limit") || strings.Contains(s, "credit")
	case strings.Contains(s, "resource_exhausted"):
		return true
	default:
		return false
	}
}

// LooksLikeUpstreamUnavailable reports non-JSON upstream outages (Cloudflare/HTML 5xx, bad gateway).
// Used so Trim can show honest chrome instead of raw HTML when a provider host is down.
func LooksLikeUpstreamUnavailable(body []byte, status int) bool {
	if status == http.StatusBadGateway || status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout {
		return true
	}
	// Cloudflare origin timeout / host error pages often use 522/523/524.
	if status == 522 || status == 523 || status == 524 {
		return true
	}
	if status < 500 {
		return false
	}
	s := strings.ToLower(string(body))
	if strings.Contains(s, "<html") || strings.Contains(s, "<!doctype") {
		return true
	}
	if strings.Contains(s, "cloudflare") && (strings.Contains(s, "error") || strings.Contains(s, "timed out")) {
		return true
	}
	return false
}

