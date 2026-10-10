package deepopt

import (
	"fmt"

	"github.com/usetrim/trim/server/pkg/provideradapt"
)

// AdapterRegistryFromPreferences builds the proxy registry from synced preferences.
// Fail-closed when ProviderAdaptersSynced is false or arrays are nil.
func AdapterRegistryFromPreferences(p Preferences, chrome provideradapt.Chrome) (*provideradapt.Registry, error) {
	if !p.ProviderAdaptersSynced {
		return nil, fmt.Errorf("provider adapters not synced")
	}
	if p.ProviderAdapters == nil || p.OpenAIModelAliases == nil || p.DiscoverableModels == nil {
		return nil, fmt.Errorf("provider adapters arrays required")
	}
	adapters := make([]provideradapt.AdapterConfig, 0, len(p.ProviderAdapters))
	for _, a := range p.ProviderAdapters {
		adapters = append(adapters, provideradapt.AdapterConfig{
			ID:                   a.ID,
			Enabled:              a.Enabled,
			SortOrder:            a.SortOrder,
			Dialect:              a.Dialect,
			DoorLabel:            a.DoorLabel,
			MatchModelPrefixes:   append([]string(nil), a.MatchModelPrefixes...),
			ModelAliases:         copyStringMap(a.ModelAliases),
			UpstreamKind:         a.UpstreamKind,
			UpstreamPath:         a.UpstreamPath,
			UpstreamBaseURL:      a.UpstreamBaseURL,
			AuthMode:             a.AuthMode,
			AnthropicVersion:     a.AnthropicVersion,
			AnthropicWorkspaceID: a.AnthropicWorkspaceID,
			DefaultMaxTokens:     a.DefaultMaxTokens,
			RequireAlias:         a.RequireAlias,
		})
	}
	aliases := make([]provideradapt.ModelAlias, 0, len(p.OpenAIModelAliases))
	for _, a := range p.OpenAIModelAliases {
		aliases = append(aliases, provideradapt.ModelAlias{
			ClientModel:          a.ClientModel,
			UpstreamModel:        a.UpstreamModel,
			UpstreamHostContains: a.UpstreamHostContains,
			Enabled:              a.Enabled,
		})
	}
	disco := make([]provideradapt.DiscoverableModel, 0, len(p.DiscoverableModels))
	for _, m := range p.DiscoverableModels {
		disco = append(disco, provideradapt.DiscoverableModel{
			ID:          m.ID,
			Enabled:     m.Enabled,
			SortOrder:   m.SortOrder,
			DisplayName: m.DisplayName,
			Description: m.Description,
			OwnedBy:     m.OwnedBy,
			AdapterID:   m.AdapterID,
			Created:     m.Created,
		})
	}
	return provideradapt.NewRegistry(adapters, aliases, disco, chrome)
}

func copyStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
