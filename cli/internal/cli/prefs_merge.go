package cli

import (
	"strings"

	"github.com/usetrim/trim/cli/internal/deepopt"
)

// remotePreferences is the JSON shape of GET /api/v1/me/preferences used by sync paths.
type remotePreferences struct {
	CompressionTier        string                            `json:"compression_tier"`
	DeepEngine              string                            `json:"deep_engine"`
	DeepTargetToken         int                               `json:"deep_target_token"`
	AutoStartWithIDE        bool                              `json:"auto_start_with_ide"`
	LiveDeepMinInputTokens  int                               `json:"live_deep_min_input_tokens"`
	LiveDeepOOMPolicy       string                            `json:"live_deep_oom_policy"`
	LiveDeepWarmupOnStart   bool                              `json:"live_deep_warmup_on_start"`
	LiveDeepSkipOnStream    bool                              `json:"live_deep_skip_on_stream"`
	DeepV1Model             string                            `json:"deep_v1_model"`
	DeepV2Model             string                            `json:"deep_v2_model"`
	DeepLongModel           string                            `json:"deep_long_model"`
	DeepV2ForceTokens       []string                          `json:"deep_v2_force_tokens"`
	ProviderAdapters        []deepopt.ProviderAdapterPref     `json:"provider_adapters"`
	OpenAIModelAliases      []deepopt.OpenAIModelAliasPref    `json:"openai_model_aliases"`
	DiscoverableModels      []deepopt.DiscoverableModelPref   `json:"provider_discoverable_models"`
	ProviderAdaptersSynced  bool                              `json:"provider_adapters_synced"`
}

// preferencesFromRemote maps API preferences into local Preferences.
// Provider adapters are applied only when the API marks them synced with non-nil arrays.
// Otherwise local adapter rows are preserved (fail-closed: never wipe DB-synced adapters
// when an older API build omits the fields).
func preferencesFromRemote(remote remotePreferences, cur deepopt.Preferences) deepopt.Preferences {
	oom := deepopt.NormalizeOOMPolicy(remote.LiveDeepOOMPolicy)
	policySynced := oom != "" &&
		remote.LiveDeepMinInputTokens >= 0 &&
		strings.TrimSpace(remote.DeepV1Model) != "" &&
		strings.TrimSpace(remote.DeepV2Model) != "" &&
		strings.TrimSpace(remote.DeepLongModel) != "" &&
		remote.DeepV2ForceTokens != nil

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
		DefaultTier:            deepopt.NormalizeTier(remote.CompressionTier),
		DeepEngine:             deepopt.NormalizeEngine(remote.DeepEngine),
		TargetToken:            remote.DeepTargetToken,
		AutoStartWithIDE:       deepopt.BoolPtr(remote.AutoStartWithIDE),
		LiveDeepMinInputTokens: remote.LiveDeepMinInputTokens,
		LiveDeepOOMPolicy:      oom,
		LiveDeepWarmupOnStart:  remote.LiveDeepWarmupOnStart,
		LiveDeepSkipOnStream:   remote.LiveDeepSkipOnStream,
		DeepV1Model:            strings.TrimSpace(remote.DeepV1Model),
		DeepV2Model:            strings.TrimSpace(remote.DeepV2Model),
		DeepLongModel:          strings.TrimSpace(remote.DeepLongModel),
		DeepV2ForceTokens:      append([]string(nil), remote.DeepV2ForceTokens...),
		LiveDeepPolicySynced:   policySynced,
		ProviderAdapters:       adapters,
		OpenAIModelAliases:     aliases,
		DiscoverableModels:     disco,
		ProviderAdaptersSynced: adaptersSynced,
	}
}
