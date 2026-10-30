package deepopt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/usetrim/trim/cli/internal/clierr"
)

// Tier is the user-facing Fast vs Deep switch.
type Tier string

const (
	TierFast Tier = "fast"
	TierDeep Tier = "deep"
)

// Engine selects which Microsoft research path Deep Mode uses.
type Engine string

const (
	EngineV1   Engine = "v1"   // LLMLingua
	EngineLong Engine = "long" // LongLLMLingua
	EngineV2   Engine = "v2"   // LLMLingua-2
)

// OOMPolicy is billing_settings.live_deep_oom_policy (synced).
type OOMPolicy string

const (
	OOMFail OOMPolicy = "fail"
	OOMSkip OOMPolicy = "skip"
)

// Preferences live in ~/.config/trim/preferences.json (local source of truth for CLI).
// Empty DefaultTier / DeepEngine / CompressionRate<=0 means unset until sync or explicit config set.
// AutoStartWithIDE nil means unset (fail-closed: do not auto-start until sync/enable).
// Live Deep dials (OOM, models, force_tokens) come from billing_settings via API sync - no invent.
type Preferences struct {
	DefaultTier          Tier      `json:"default_tier"`
	DeepEngine            Engine    `json:"deep_engine"`
	CompressionRate          float64 `json:"compression_rate"`
	CompressionRateMin       float64 `json:"compression_rate_min"`
	CompressionRateMax       float64 `json:"compression_rate_max"`
	CompressionRateStep      float64 `json:"compression_rate_step"`
	DefaultCompressionRate   float64 `json:"default_compression_rate"`
	AutoStartWithIDE      *bool     `json:"auto_start_with_ide,omitempty"`
	LiveDeepOOMPolicy     OOMPolicy `json:"live_deep_oom_policy"`
	LiveDeepWarmupOnStart bool      `json:"live_deep_warmup_on_start"`
	DeepV1Model           string    `json:"deep_v1_model"`
	DeepV2Model           string    `json:"deep_v2_model"`
	DeepLongModel         string    `json:"deep_long_model"`
	DeepV2ForceTokens     []string  `json:"deep_v2_force_tokens"`
	// DeepV2Backbone is the Microsoft Space Base Model id (catalog row). DeepV2Model is its HF id.
	DeepV2Backbone          string             `json:"deep_v2_backbone"`
	DefaultDeepV2Backbone   string             `json:"default_deep_v2_backbone"`
	DeepV2BackboneOptions   []V2BackboneOption `json:"deep_v2_backbone_options"`
	LiveDeepPolicySynced  bool      `json:"live_deep_policy_synced"`
	// ProviderAdapters / OpenAIModelAliases / DiscoverableModels come from Trim cloud tables via preferences sync (no invent).
	ProviderAdapters       []ProviderAdapterPref     `json:"provider_adapters"`
	OpenAIModelAliases     []OpenAIModelAliasPref    `json:"openai_model_aliases"`
	DiscoverableModels     []DiscoverableModelPref   `json:"provider_discoverable_models"`
	ProviderAdaptersSynced bool                      `json:"provider_adapters_synced"`
}

// ProviderAdapterPref mirrors provider_adapters for local proxy (JSON-stable).
type ProviderAdapterPref struct {
	ID                 string            `json:"id"`
	Enabled            bool              `json:"enabled"`
	SortOrder          int               `json:"sort_order"`
	Dialect            string            `json:"dialect"`
	DoorLabel          string            `json:"door_label"`
	MatchModelPrefixes []string          `json:"match_model_prefixes"`
	ModelAliases       map[string]string `json:"model_aliases"`
	UpstreamKind       string            `json:"upstream_kind"`
	UpstreamPath       string            `json:"upstream_path"`
	UpstreamBaseURL    string            `json:"upstream_base_url,omitempty"`
	AuthMode           string            `json:"auth_mode"`
	AnthropicVersion     string            `json:"anthropic_version,omitempty"`
	AnthropicWorkspaceID string            `json:"anthropic_workspace_id,omitempty"`
	DefaultMaxTokens     int               `json:"default_max_tokens,omitempty"`
	RequireAlias         bool              `json:"require_alias"`
}

// OpenAIModelAliasPref mirrors openai_model_aliases.
type OpenAIModelAliasPref struct {
	ClientModel          string `json:"client_model"`
	UpstreamModel        string `json:"upstream_model"`
	UpstreamHostContains string `json:"upstream_host_contains"`
	Enabled              bool   `json:"enabled"`
}

// DiscoverableModelPref mirrors provider_discoverable_models.
type DiscoverableModelPref struct {
	ID          string `json:"id"`
	Enabled     bool   `json:"enabled"`
	SortOrder   int    `json:"sort_order"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	OwnedBy     string `json:"owned_by"`
	AdapterID   string `json:"adapter_id,omitempty"`
	Created     int64  `json:"created"`
}

// AutoStartEnabled is true only when the preference is explicitly set on.
func (p Preferences) AutoStartEnabled() bool {
	return p.AutoStartWithIDE != nil && *p.AutoStartWithIDE
}

// AutoStartSet reports whether the preference has been explicitly set (sync or CLI).
func (p Preferences) AutoStartSet() bool {
	return p.AutoStartWithIDE != nil
}

func BoolPtr(v bool) *bool {
	return &v
}

// PrefsChrome holds API-driven validation copy for preferences. Empty = fail-closed.
type PrefsChrome struct {
	Unavailable    string
	EngineInvalid  string
	RateNonNeg     string
	TierRequired   string
	EngineRequired string
	RateRequired   string
}

func (c PrefsChrome) err(msg string) error {
	if msg != "" {
		return fmt.Errorf("%s", msg)
	}
	if c.Unavailable != "" {
		return fmt.Errorf("%s", c.Unavailable)
	}
	return clierr.ErrChromeUnavailable
}

func DefaultPreferences() Preferences {
	return Preferences{
		DefaultTier:     "",
		DeepEngine:      "",
		CompressionRate: 0,
	}
}

func prefsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "trim")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "preferences.json"), nil
}

func LoadPreferences() (Preferences, error) {
	p := DefaultPreferences()
	path, err := prefsPath()
	if err != nil {
		return p, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return p, nil
		}
		return p, err
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return DefaultPreferences(), fmt.Errorf("preferences.json: %w", err)
	}
	if strings.TrimSpace(string(p.DefaultTier)) != "" {
		p.DefaultTier = NormalizeTier(string(p.DefaultTier))
	} else {
		p.DefaultTier = ""
	}
	if strings.TrimSpace(string(p.DeepEngine)) != "" {
		p.DeepEngine = NormalizeEngine(string(p.DeepEngine))
	} else {
		p.DeepEngine = ""
	}
	p.LiveDeepOOMPolicy = NormalizeOOMPolicy(string(p.LiveDeepOOMPolicy))
	return p, nil
}

func SavePreferences(p Preferences, chrome PrefsChrome) error {
	if strings.TrimSpace(string(p.DefaultTier)) != "" {
		p.DefaultTier = NormalizeTier(string(p.DefaultTier))
	} else {
		p.DefaultTier = ""
	}
	if strings.TrimSpace(string(p.DeepEngine)) != "" {
		p.DeepEngine = NormalizeEngine(string(p.DeepEngine))
		if p.DeepEngine == "" {
			return chrome.err(chrome.EngineInvalid)
		}
	} else {
		p.DeepEngine = ""
	}
	if p.CompressionRate < 0 {
		return chrome.err(chrome.RateNonNeg)
	}
	p.LiveDeepOOMPolicy = NormalizeOOMPolicy(string(p.LiveDeepOOMPolicy))
	path, err := prefsPath()
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

// RequireCompressPrefs fail-closes when Fast/Deep defaults are incomplete.
func RequireCompressPrefs(p Preferences, chrome PrefsChrome) error {
	if p.DefaultTier == "" {
		return chrome.err(chrome.TierRequired)
	}
	if p.DeepEngine == "" {
		return chrome.err(chrome.EngineRequired)
	}
	if p.CompressionRate <= 0 || p.CompressionRate > 1 {
		return chrome.err(chrome.RateRequired)
	}
	return nil
}

// RateInBillingBounds reports whether rate is inside synced billing min/max.
func RateInBillingBounds(rate, min, max float64) bool {
	if min <= 0 || max < min || max > 1 {
		return false
	}
	return rate >= min && rate <= max
}

// RequireLiveDeepPolicy fail-closes when billing live-Deep dials were not synced.
func RequireLiveDeepPolicy(p Preferences) bool {
	if !p.LiveDeepPolicySynced {
		return false
	}
	if NormalizeOOMPolicy(string(p.LiveDeepOOMPolicy)) == "" {
		return false
	}
	if strings.TrimSpace(p.DeepV1Model) == "" || strings.TrimSpace(p.DeepV2Model) == "" || strings.TrimSpace(p.DeepLongModel) == "" {
		return false
	}
	if p.DeepV2ForceTokens == nil {
		return false
	}
	if ValidateBackboneCatalog(p.DeepV2BackboneOptions, p.DefaultDeepV2Backbone) != nil {
		return false
	}
	if !BackboneInCatalog(p.DeepV2BackboneOptions, p.DeepV2Backbone) {
		return false
	}
	if resolved, ok := ResolveV2Model(p.DeepV2BackboneOptions, p.DeepV2Backbone); !ok || resolved != strings.TrimSpace(p.DeepV2Model) {
		return false
	}
	if p.CompressionRateMin <= 0 || p.CompressionRateMax < p.CompressionRateMin || p.CompressionRateMax > 1 {
		return false
	}
	if p.CompressionRateStep <= 0 || p.CompressionRateStep > 1 {
		return false
	}
	if p.DefaultCompressionRate <= 0 || p.DefaultCompressionRate > 1 {
		return false
	}
	return true
}

// ListCompressionRates builds the Microsoft-style rate ladder from synced billing bounds.
func ListCompressionRates(min, max, step float64) []float64 {
	if min <= 0 || max < min || max > 1 || step <= 0 {
		return nil
	}
	out := make([]float64, 0, 16)
	for v := min; v <= max+step/1000; v += step {
		r := float64(int(v*100+0.5)) / 100
		if r < min-1e-9 || r > max+1e-9 {
			continue
		}
		if len(out) > 0 && r <= out[len(out)-1]+1e-9 {
			continue
		}
		out = append(out, r)
	}
	if len(out) == 0 || out[len(out)-1] < max-1e-9 {
		last := float64(int(max*100+0.5)) / 100
		if len(out) == 0 || last > out[len(out)-1]+1e-9 {
			out = append(out, last)
		}
	}
	return out
}

// RuntimeFromPreferences builds Deep Runtime from synced billing models (device_map stays local).
func RuntimeFromPreferences(p Preferences, deviceMap string) Runtime {
	v2 := strings.TrimSpace(p.DeepV2Model)
	if resolved, ok := ResolveV2Model(p.DeepV2BackboneOptions, p.DeepV2Backbone); ok {
		v2 = resolved
	}
	return Runtime{
		DeviceMap:     deviceMap,
		V1Model:       strings.TrimSpace(p.DeepV1Model),
		V2Model:       v2,
		LongModel:     strings.TrimSpace(p.DeepLongModel),
		V2ForceTokens: append([]string(nil), p.DeepV2ForceTokens...),
	}
}

func NormalizeTier(raw string) Tier {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return ""
	case "deep", "llmlingua", "ai":
		return TierDeep
	case "fast":
		return TierFast
	default:
		// Fail closed: never invent "fast" for unknown values.
		return ""
	}
}

func NormalizeEngine(raw string) Engine {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return ""
	case "v1", "llmlingua", "1":
		return EngineV1
	case "long", "longllmlingua":
		return EngineLong
	case "v2", "llmlingua2", "2":
		return EngineV2
	default:
		return ""
	}
}

func NormalizeOOMPolicy(raw string) OOMPolicy {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "fail":
		return OOMFail
	case "skip":
		return OOMSkip
	default:
		return ""
	}
}

// RequestWantsStream reports OpenAI/Anthropic stream:true on the JSON body.
func RequestWantsStream(body []byte) bool {
	if !gjson.ValidBytes(body) {
		return false
	}
	return gjson.GetBytes(body, "stream").Bool()
}

// IsDeepOOMError reports memory/OOM failures from local Deep (torch / CUDA / allocator).
func IsDeepOOMError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	needles := []string{
		"out of memory",
		"cuda out of memory",
		"torch.cuda.outofmemoryerror",
		"cuda_error_out_of_memory",
		"hip out of memory",
		"cannot allocate memory",
		"std::bad_alloc",
		"memoryerror",
		"resource exhausted: oom",
	}
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}
