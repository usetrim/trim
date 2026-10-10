-- DB-owned discoverable model catalog for GET /v1/models (OpenAI SDKs + Claude Code gateway discovery).
-- No hard-coded model lists in the proxy. Ops refresh rows when providers retire ids.
-- Claude Code (optional): CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY=1 → GET /v1/models?limit=1000
-- Cursor / OpenAI SDK: GET …/v1/models (Base URL includes /v1).

create table if not exists public.provider_discoverable_models (
  id text primary key,
  enabled boolean not null,
  sort_order int not null,
  display_name text not null default '',
  description text not null default '',
  owned_by text not null,
  adapter_id text references public.provider_adapters (id) on delete set null,
  updated_at timestamptz not null default now(),
  constraint provider_discoverable_models_id_nonempty_check
    check (length(btrim(id)) > 0 and length(btrim(id)) <= 256),
  constraint provider_discoverable_models_owned_by_nonempty_check
    check (length(btrim(owned_by)) > 0 and length(btrim(owned_by)) <= 128),
  constraint provider_discoverable_models_sort_order_check
    check (sort_order >= 0 and sort_order <= 1000000)
);

comment on table public.provider_discoverable_models is
  'Explicit model ids Trim advertises on GET /v1/models. Synced to CLI via preferences. Never invent ids from prefixes in code.';
comment on column public.provider_discoverable_models.owned_by is
  'OpenAI-shape owned_by (e.g. openai, google, anthropic, deepseek, mistral). DB-only.';
comment on column public.provider_discoverable_models.adapter_id is
  'Optional link to provider_adapters row that routes this id.';
comment on column public.provider_discoverable_models.display_name is
  'Claude Code gateway discovery display_name when non-empty.';
comment on column public.provider_discoverable_models.description is
  'Claude Code gateway discovery description; empty → client shows From gateway.';

create index if not exists provider_discoverable_models_enabled_sort_idx
  on public.provider_discoverable_models (enabled, sort_order, id);

-- Seed concrete ids (canonical + convenience). Update via later migrations when providers change.
insert into public.provider_discoverable_models
  (id, enabled, sort_order, display_name, description, owned_by, adapter_id, updated_at)
values
  -- Anthropic / Claude (Door A + Door C)
  ('claude-haiku-4-5-20251001', true, 10, 'Claude Haiku 4.5', 'Fast Claude via Trim', 'anthropic', 'anthropic', now()),
  ('claude-sonnet-5-5', true, 11, 'Claude Sonnet 5.5', 'Default coding Claude via Trim', 'anthropic', 'anthropic', now()),
  ('claude-opus-5-5', true, 12, 'Claude Opus 5.5', 'Highest capability Claude via Trim (key must allow)', 'anthropic', 'anthropic', now()),
  ('claude-sonnet-4-6', true, 13, 'Claude Sonnet 4.6', '', 'anthropic', 'anthropic', now()),
  ('claude-opus-4-8', true, 14, 'Claude Opus 4.8', '', 'anthropic', 'anthropic', now()),
  ('trim-claude-haiku', true, 20, 'Trim Claude Haiku', 'Convenience alias → Haiku 4.5', 'anthropic', 'anthropic', now()),
  ('trim-claude-sonnet', true, 21, 'Trim Claude Sonnet', 'Convenience alias → Sonnet 5.5', 'anthropic', 'anthropic', now()),
  ('trim-claude-opus', true, 22, 'Trim Claude Opus', 'Convenience alias → Opus 5.5', 'anthropic', 'anthropic', now()),
  -- OpenAI GPT
  ('gpt-4o', true, 30, 'GPT-4o', '', 'openai', 'openai', now()),
  ('gpt-4o-mini', true, 31, 'GPT-4o mini', '', 'openai', 'openai', now()),
  ('gpt-4.1', true, 32, 'GPT-4.1', '', 'openai', 'openai', now()),
  ('gpt-4.1-mini', true, 33, 'GPT-4.1 mini', '', 'openai', 'openai', now()),
  ('o1', true, 34, 'o1', '', 'openai', 'openai', now()),
  ('o3', true, 35, 'o3', '', 'openai', 'openai', now()),
  ('o4-mini', true, 36, 'o4-mini', '', 'openai', 'openai', now()),
  -- Gemini OpenAI-compat
  ('gemini-flash-latest', true, 40, 'Gemini Flash latest', 'Google OpenAI-compat via Trim', 'google', 'gemini', now()),
  ('gemini-2.5-flash', true, 41, 'Gemini 2.5 Flash', '', 'google', 'gemini', now()),
  ('trim-gemini-flash', true, 42, 'Trim Gemini Flash', 'Convenience alias → gemini-flash-latest', 'google', 'gemini', now()),
  ('trim-gemini-flash-latest', true, 43, 'Trim Gemini Flash latest', '', 'google', 'gemini', now()),
  -- DeepSeek
  ('deepseek-chat', true, 50, 'DeepSeek Chat', '', 'deepseek', 'deepseek', now()),
  ('deepseek-reasoner', true, 51, 'DeepSeek Reasoner', '', 'deepseek', 'deepseek', now()),
  -- Mistral
  ('mistral-large-latest', true, 60, 'Mistral Large', '', 'mistral', 'mistral', now()),
  ('mistral-small-latest', true, 61, 'Mistral Small', '', 'mistral', 'mistral', now())
on conflict (id) do update set
  enabled = excluded.enabled,
  sort_order = excluded.sort_order,
  display_name = excluded.display_name,
  description = excluded.description,
  owned_by = excluded.owned_by,
  adapter_id = excluded.adapter_id,
  updated_at = now();

-- Gateway / proxy chrome (fail-closed empty in code = generic error only).
insert into public.site_messages (code, body, updated_at)
values
  (
    'CLI_PROXY_MODELS_NOT_SYNCED',
    'Discoverable models are not synced. Run trim config sync while logged in, then restart trim start.',
    now()
  ),
  (
    'CLI_PROXY_MODELS_METHOD',
    'GET only on /v1/models',
    now()
  ),
  (
    'CLI_PROXY_ERR_BASE_URL_DOUBLE_V1',
    'Invalid path /v1/v1/…. For Claude Code set ANTHROPIC_BASE_URL to the Trim listen origin only (e.g. http://127.0.0.1:8888) with no /v1 suffix. For Cursor / OpenAI SDKs use Base URL …/v1 (single /v1).',
    now()
  ),
  (
    'CLI_PROXY_ERR_MODEL_NOT_FOUND_FMT',
    'Upstream rejected model %q (not found or not allowed for this API key). Pick another model from GET /v1/models, use a Trim alias from Admin → Provider adapters, or check your provider plan entitlements.',
    now()
  ),
  (
    'CLI_PROXY_ADAPTER_ENABLED_FMT',
    'OpenAI-compat provider adapters loaded (%d enabled). Model prefixes route via DB (Claude Messages and same-shape hosts with upstream_base_url). GET /v1/models lists DB provider_discoverable_models. Unmatched chat models use UPSTREAM_OPENAI_URL.',
    now()
  ),
  (
    'DOCS_PROVIDER_ADAPTERS_SUMMARY',
    'Trim exposes one Base URL with two native doors plus DB-driven adapters: Claude via anthropic_messages, and same-shape hosts (GPT, Gemini, DeepSeek, Mistral) via openai_compat + upstream_base_url. GET /v1/models advertises provider_discoverable_models from the database. No OpenRouter. Keep Cursor Base URL = Trim …/v1; Claude Code ANTHROPIC_BASE_URL = Trim origin without /v1.',
    now()
  ),
  (
    'PREFERENCES_CLI_HELP_8',
    'Claude Code: export ANTHROPIC_BASE_URL to the Trim listen origin only (e.g. http://127.0.0.1:8888) with no /v1 suffix - Claude Code appends /v1/messages. Optional: CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY=1 to load models from Trim GET /v1/models. Prefer a workspace-scoped Anthropic key; org keys need TRIM_ANTHROPIC_WORKSPACE_ID or anthropic-workspace-id.',
    now()
  )
on conflict (code) do update set
  body = excluded.body,
  updated_at = now();
