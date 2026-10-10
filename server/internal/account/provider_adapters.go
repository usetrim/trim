package account

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/usetrim/trim/server/pkg/provideradapt"
)

// loadProviderAdaptersFromDB returns all adapter rows (enabled and disabled) for preferences sync.
// Fail-closed: query error or invalid row shape returns error (caller maps to 503).
func loadProviderAdaptersFromDB(ctx context.Context, db *pgxpool.Pool) ([]provideradapt.AdapterConfig, error) {
	rows, err := db.Query(ctx, `
		select id, enabled, sort_order, dialect, door_label,
		       match_model_prefixes, model_aliases,
		       upstream_kind, upstream_path, coalesce(upstream_base_url, ''), auth_mode,
		       coalesce(anthropic_version, ''), coalesce(anthropic_workspace_id, ''), coalesce(default_max_tokens, 0), require_alias
		from public.provider_adapters
		order by sort_order asc, id asc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]provideradapt.AdapterConfig, 0)
	for rows.Next() {
		var c provideradapt.AdapterConfig
		var prefixesRaw, aliasesRaw []byte
		if err := rows.Scan(
			&c.ID, &c.Enabled, &c.SortOrder, &c.Dialect, &c.DoorLabel,
			&prefixesRaw, &aliasesRaw,
			&c.UpstreamKind, &c.UpstreamPath, &c.UpstreamBaseURL, &c.AuthMode,
			&c.AnthropicVersion, &c.AnthropicWorkspaceID, &c.DefaultMaxTokens, &c.RequireAlias,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(prefixesRaw, &c.MatchModelPrefixes); err != nil || c.MatchModelPrefixes == nil {
			return nil, err
		}
		if err := json.Unmarshal(aliasesRaw, &c.ModelAliases); err != nil || c.ModelAliases == nil {
			return nil, err
		}
		c.MatchModelPrefixes = provideradapt.NormalizePrefixes(c.MatchModelPrefixes)
		c.ModelAliases = provideradapt.NormalizeAliasMap(c.ModelAliases)
		if err := provideradapt.ValidateConfig(c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func loadOpenAIModelAliasesFromDB(ctx context.Context, db *pgxpool.Pool) ([]provideradapt.ModelAlias, error) {
	rows, err := db.Query(ctx, `
		select client_model, upstream_model, coalesce(upstream_host_contains, ''), enabled
		from public.openai_model_aliases
		order by client_model asc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]provideradapt.ModelAlias, 0)
	for rows.Next() {
		var a provideradapt.ModelAlias
		if err := rows.Scan(&a.ClientModel, &a.UpstreamModel, &a.UpstreamHostContains, &a.Enabled); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func loadDiscoverableModelsFromDB(ctx context.Context, db *pgxpool.Pool) ([]provideradapt.DiscoverableModel, error) {
	rows, err := db.Query(ctx, `
		select id, enabled, sort_order,
		       coalesce(display_name, ''), coalesce(description, ''),
		       owned_by, coalesce(adapter_id, ''),
		       extract(epoch from updated_at)::bigint
		from public.provider_discoverable_models
		order by sort_order asc, id asc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]provideradapt.DiscoverableModel, 0)
	for rows.Next() {
		var m provideradapt.DiscoverableModel
		if err := rows.Scan(
			&m.ID, &m.Enabled, &m.SortOrder,
			&m.DisplayName, &m.Description,
			&m.OwnedBy, &m.AdapterID, &m.Created,
		); err != nil {
			return nil, err
		}
		if err := provideradapt.ValidateDiscoverableModel(m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
