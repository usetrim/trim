package cli

import (
	"testing"

	"github.com/usetrim/trim/cli/internal/deepopt"
)

func TestPreferencesFromRemotePreservesAdaptersWhenAPIOmits(t *testing.T) {
	cur := deepopt.Preferences{
		ProviderAdaptersSynced: true,
		ProviderAdapters: []deepopt.ProviderAdapterPref{{
			ID: "anthropic", Enabled: true, DoorLabel: "openai_to_anthropic",
		}},
		OpenAIModelAliases: []deepopt.OpenAIModelAliasPref{{
			ClientModel: "trim-gemini-flash", UpstreamModel: "gemini-3.6-flash", Enabled: true,
		}},
		DiscoverableModels: []deepopt.DiscoverableModelPref{{
			ID: "claude-haiku-4-5-20251001", Enabled: true, OwnedBy: "anthropic",
		}},
	}
	remote := remotePreferences{
		CompressionTier: "deep",
		DeepEngine:      "v2",
		DeepTargetToken: 300,
		// ProviderAdaptersSynced false / nil arrays = older API; must not wipe.
	}
	out := preferencesFromRemote(remote, cur)
	if !out.ProviderAdaptersSynced {
		t.Fatal("expected local adapters preserved")
	}
	if len(out.ProviderAdapters) != 1 || out.ProviderAdapters[0].ID != "anthropic" {
		t.Fatalf("adapters wiped: %+v", out.ProviderAdapters)
	}
	if len(out.OpenAIModelAliases) != 1 {
		t.Fatalf("aliases wiped: %+v", out.OpenAIModelAliases)
	}
	if len(out.DiscoverableModels) != 1 {
		t.Fatalf("discoverable wiped: %+v", out.DiscoverableModels)
	}
}

func TestPreferencesFromRemoteAppliesSyncedAdapters(t *testing.T) {
	cur := deepopt.Preferences{ProviderAdaptersSynced: false}
	remote := remotePreferences{
		CompressionTier:       "fast",
		ProviderAdaptersSynced: true,
		ProviderAdapters:       []deepopt.ProviderAdapterPref{{ID: "anthropic", Enabled: true, DoorLabel: "openai_to_anthropic"}},
		OpenAIModelAliases:     []deepopt.OpenAIModelAliasPref{},
		DiscoverableModels:     []deepopt.DiscoverableModelPref{{ID: "gpt-4o", Enabled: true, OwnedBy: "openai"}},
	}
	out := preferencesFromRemote(remote, cur)
	if !out.ProviderAdaptersSynced || len(out.ProviderAdapters) != 1 {
		t.Fatalf("expected remote adapters applied: synced=%v n=%d", out.ProviderAdaptersSynced, len(out.ProviderAdapters))
	}
	if len(out.DiscoverableModels) != 1 || out.DiscoverableModels[0].ID != "gpt-4o" {
		t.Fatalf("discoverable not applied: %+v", out.DiscoverableModels)
	}
}
