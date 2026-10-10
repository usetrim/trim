package platformadmin

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/usetrim/trim/server/pkg/provideradapt"
)

func (h *Handler) listProviderAdapters(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), `
		select id, enabled, sort_order, dialect, door_label,
		       match_model_prefixes, model_aliases,
		       upstream_kind, upstream_path, coalesce(upstream_base_url, ''), auth_mode,
		       coalesce(anthropic_version, ''), coalesce(anthropic_workspace_id, ''), coalesce(default_max_tokens, 0), require_alias,
		       updated_at
		from public.provider_adapters
		order by sort_order asc, id asc
	`)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	type row struct {
		provideradapt.AdapterConfig
		UpdatedAt any `json:"updated_at"`
	}
	out := make([]row, 0)
	for rows.Next() {
		var item row
		var prefixesRaw, aliasesRaw []byte
		var updated any
		if err := rows.Scan(
			&item.ID, &item.Enabled, &item.SortOrder, &item.Dialect, &item.DoorLabel,
			&prefixesRaw, &aliasesRaw,
			&item.UpstreamKind, &item.UpstreamPath, &item.UpstreamBaseURL, &item.AuthMode,
			&item.AnthropicVersion, &item.AnthropicWorkspaceID, &item.DefaultMaxTokens, &item.RequireAlias,
			&updated,
		); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		if err := json.Unmarshal(prefixesRaw, &item.MatchModelPrefixes); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		if err := json.Unmarshal(aliasesRaw, &item.ModelAliases); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		item.UpdatedAt = updated
		out = append(out, item)
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"items":       out,
		"page_title":  strings.TrimSpace(h.msg("ADMIN_PROVIDER_ADAPTERS_TITLE")),
		"description": strings.TrimSpace(h.msg("ADMIN_PROVIDER_ADAPTERS_DESC")),
	})
}

func (h *Handler) putProviderAdapter(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	var body provideradapt.AdapterConfig
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	body.ID = strings.TrimSpace(body.ID)
	body.MatchModelPrefixes = provideradapt.NormalizePrefixes(body.MatchModelPrefixes)
	body.ModelAliases = provideradapt.NormalizeAliasMap(body.ModelAliases)
	if err := provideradapt.ValidateConfig(body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	prefixesJSON, err := json.Marshal(body.MatchModelPrefixes)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	aliasesJSON, err := json.Marshal(body.ModelAliases)
	if err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	_, err = h.DB.Exec(r.Context(), `
		insert into public.provider_adapters (
		  id, enabled, sort_order, dialect, door_label,
		  match_model_prefixes, model_aliases,
		  upstream_kind, upstream_path, auth_mode,
		  anthropic_version, anthropic_workspace_id, default_max_tokens, require_alias, upstream_base_url, updated_at
		) values (
		  $1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,$8,$9,$10,
		  nullif($11,''), nullif($12,''), nullif($13,0), $14, coalesce($15,''), now()
		)
		on conflict (id) do update set
		  enabled = excluded.enabled,
		  sort_order = excluded.sort_order,
		  dialect = excluded.dialect,
		  door_label = excluded.door_label,
		  match_model_prefixes = excluded.match_model_prefixes,
		  model_aliases = excluded.model_aliases,
		  upstream_kind = excluded.upstream_kind,
		  upstream_path = excluded.upstream_path,
		  auth_mode = excluded.auth_mode,
		  anthropic_version = excluded.anthropic_version,
		  anthropic_workspace_id = excluded.anthropic_workspace_id,
		  default_max_tokens = excluded.default_max_tokens,
		  require_alias = excluded.require_alias,
		  upstream_base_url = excluded.upstream_base_url,
		  updated_at = now()
	`, body.ID, body.Enabled, body.SortOrder, body.Dialect, body.DoorLabel,
		string(prefixesJSON), string(aliasesJSON),
		body.UpstreamKind, body.UpstreamPath, body.AuthMode,
		strings.TrimSpace(body.AnthropicVersion), strings.TrimSpace(body.AnthropicWorkspaceID),
		body.DefaultMaxTokens, body.RequireAlias,
		strings.TrimSpace(body.UpstreamBaseURL))
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	step := false
	if p != nil {
		step = h.stepUpUsed(r, p.UserID)
	}
	h.audit(r.Context(), r, "billing.provider_adapter_upsert", "provider_adapters", body.ID, nil, body, "", step)
	h.listProviderAdapters(w, r)
}

func (h *Handler) listOpenAIModelAliases(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), `
		select client_model, upstream_model, coalesce(upstream_host_contains, ''), enabled, updated_at
		from public.openai_model_aliases
		order by client_model asc
	`)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	defer rows.Close()
	type row struct {
		provideradapt.ModelAlias
		UpdatedAt any `json:"updated_at"`
	}
	out := make([]row, 0)
	for rows.Next() {
		var item row
		var updated any
		if err := rows.Scan(&item.ClientModel, &item.UpstreamModel, &item.UpstreamHostContains, &item.Enabled, &updated); err != nil {
			h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
			return
		}
		item.UpdatedAt = updated
		out = append(out, item)
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"items":       out,
		"page_title":  strings.TrimSpace(h.msg("ADMIN_OPENAI_MODEL_ALIASES_TITLE")),
		"description": strings.TrimSpace(h.msg("ADMIN_OPENAI_MODEL_ALIASES_DESC")),
	})
}

func (h *Handler) putOpenAIModelAlias(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFromContext(r.Context())
	var body provideradapt.ModelAlias
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	body.ClientModel = strings.ToLower(strings.TrimSpace(body.ClientModel))
	body.UpstreamModel = strings.TrimSpace(body.UpstreamModel)
	body.UpstreamHostContains = strings.TrimSpace(body.UpstreamHostContains)
	if body.ClientModel == "" || body.UpstreamModel == "" {
		h.writeErr(w, http.StatusBadRequest, "WS_INVALID_JSON")
		return
	}
	_, err := h.DB.Exec(r.Context(), `
		insert into public.openai_model_aliases (client_model, upstream_model, upstream_host_contains, enabled, updated_at)
		values ($1,$2,$3,$4, now())
		on conflict (client_model) do update set
		  upstream_model = excluded.upstream_model,
		  upstream_host_contains = excluded.upstream_host_contains,
		  enabled = excluded.enabled,
		  updated_at = now()
	`, body.ClientModel, body.UpstreamModel, body.UpstreamHostContains, body.Enabled)
	if err != nil {
		h.writeErr(w, http.StatusInternalServerError, "DATABASE_UNAVAILABLE")
		return
	}
	step := false
	if p != nil {
		step = h.stepUpUsed(r, p.UserID)
	}
	h.audit(r.Context(), r, "billing.openai_model_alias_upsert", "openai_model_aliases", body.ClientModel, nil, body, "", step)
	h.listOpenAIModelAliases(w, r)
}
