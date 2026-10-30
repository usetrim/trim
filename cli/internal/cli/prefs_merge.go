package cli

import (
	"strings"

	"github.com/usetrim/trim/cli/internal/deepopt"
)

// remotePreferences is the JSON shape of GET /api/v1/me/preferences used by sync paths.
type remotePreferences struct {
	CompressionTier       string                          `json:"compression_tier"`
	DeepEngine             string                          `json:"deep_engine"`
	DeepCompressionRate        float64 `json:"deep_compression_rate"`
	DeepCompressionRateMin     float64 `json:"deep_compression_rate_min"`
	DeepCompressionRateMax     float64 `json:"deep_compression_rate_max"`
	DeepCompressionRateStep    float64 `json:"deep_compression_rate_step"`
	DefaultDeepCompressionRate float64 `json:"default_deep_compression_rate"`
	AutoStartWithIDE       bool                            `json:"auto_start_with_ide"`
	LiveDeepOOMPolicy      string                          `json:"live_deep_oom_policy"`
	LiveDeepWarmupOnStart  bool                            `json:"live_deep_warmup_on_start"`
	DeepV1Model            string                          `json:"deep_v1_model"`
	DeepV2Model            string                          `json:"deep_v2_model"`
	DeepLongModel          string                          `json:"deep_long_model"`
	DeepV2ForceTokens      []string                        `json:"deep_v2_force_tokens"`
	DeepV2Backbone         string                          `json:"deep_v2_backbone"`
	DefaultDeepV2Backbone  string                          `json:"default_deep_v2_backbone"`
	DeepV2BackboneOptions  []deepopt.V2BackboneOption      `json:"deep_v2_backbone_options"`
	ProviderAdapters       []deepopt.ProviderAdapterPref   `json:"provider_adapters"`
	OpenAIModelAliases     []deepopt.OpenAIModelAliasPref  `json:"openai_model_aliases"`
	DiscoverableModels     []deepopt.DiscoverableModelPref `json:"provider_discoverable_models"`
	ProviderAdaptersSynced bool                            `json:"provider_adapters_synced"`
}

// preferencesFromRemote maps API preferences into local Preferences.
// Provider adapters are applied only when the API marks them synced with non-nil arrays.
// Otherwise local adapter rows are preserved (fail-closed: never wipe DB-synced adapters
// when an older API build omits the fields).
func preferencesFromRemote(remote remotePreferences, cur deepopt.Preferences) deepopt.Preferences {
	oom := deepopt.NormalizeOOMPolicy(remote.LiveDeepOOMPolicy)
	rateMin := remote.DeepCompressionRateMin
	rateMax := remote.DeepCompressionRateMax
	rateStep := remote.DeepCompressionRateStep
	rateDefault := remote.DefaultDeepCompressionRate
	backboneOpts := append([]deepopt.V2BackboneOption{}, remote.DeepV2BackboneOptions...)
	backboneID := strings.TrimSpace(remote.DeepV2Backbone)
	defaultBackbone := strings.TrimSpace(remote.DefaultDeepV2Backbone)
	v2Model := strings.TrimSpace(remote.DeepV2Model)
	if resolved, ok := deepopt.ResolveV2Model(backboneOpts, backboneID); ok {
		v2Model = resolved
	}
	policySynced := oom != "" &&
		rateMin > 0 && rateMax >= rateMin && rateMax <= 1 &&
		rateStep > 0 && rateStep <= 1 &&
		rateDefault > 0 && rateDefault <= 1 &&
		strings.TrimSpace(remote.DeepV1Model) != "" &&
		v2Model != "" &&
		strings.TrimSpace(remote.DeepLongModel) != "" &&
		remote.DeepV2ForceTokens != nil &&
		deepopt.ValidateBackboneCatalog(backboneOpts, defaultBackbone) == nil &&
		deepopt.BackboneInCatalog(backboneOpts, backboneID)

	adapters := cur.ProviderAdapters
	aliases := cur.OpenAIModelAliases
	disco := cur.DiscoverableModels
	adaptersSynced := cur.ProviderAdaptersSynced
	if remote.ProviderAdaptersSynced &&
		remote.ProviderAdapters != nil &&
		remote.OpenAIModelAliases != nil &&
		remote.DiscoverableModels != nil {
		adapters = append([]deepopt.ProviderAdapterPref{}, remote.ProviderAdapters...)
		aliases = append([]deepopt.OpenAIModelAliasPref{}, remote.OpenAIModelAliases...)
		disco = append([]deepopt.DiscoverableModelPref{}, remote.DiscoverableModels...)
		adaptersSynced = true
	}

	return deepopt.Preferences{
		DefaultTier:           deepopt.NormalizeTier(remote.CompressionTier),
		DeepEngine:            deepopt.NormalizeEngine(remote.DeepEngine),
		CompressionRate:        remote.DeepCompressionRate,
		CompressionRateMin:     rateMin,
		CompressionRateMax:     rateMax,
		CompressionRateStep:    rateStep,
		DefaultCompressionRate: rateDefault,
		AutoStartWithIDE:      deepopt.BoolPtr(remote.AutoStartWithIDE),
		LiveDeepOOMPolicy:     oom,
		LiveDeepWarmupOnStart: remote.LiveDeepWarmupOnStart,
		DeepV1Model:           strings.TrimSpace(remote.DeepV1Model),
		DeepV2Model:           v2Model,
		DeepLongModel:         strings.TrimSpace(remote.DeepLongModel),
		DeepV2ForceTokens:     append([]string(nil), remote.DeepV2ForceTokens...),
		DeepV2Backbone:        backboneID,
		DefaultDeepV2Backbone: defaultBackbone,
		DeepV2BackboneOptions: backboneOpts,
		LiveDeepPolicySynced:  policySynced,
		ProviderAdapters:      adapters,
		OpenAIModelAliases:    aliases,
		DiscoverableModels:    disco,
		ProviderAdaptersSynced: adaptersSynced,
	}
}
