-- Pluggable OpenAI-compat provider adapters + model aliases (DB-driven; no invent in proxy).
-- Anthropic is the first dialect plugin so Cursor OpenAI Base URL can reach Claude without OpenRouter.
-- Future dialects (Bedrock, Vertex, …) are additional rows - same registry, no marketplace.

create table if not exists public.provider_adapters (
  id text primary key,
  enabled boolean not null,
  sort_order int not null,
  dialect text not null,
  door_label text not null,
  match_model_prefixes jsonb not null,
  model_aliases jsonb not null,
  upstream_kind text not null,
  upstream_path text not null,
  auth_mode text not null,
  anthropic_version text,
  default_max_tokens int,
  require_alias boolean not null,
  updated_at timestamptz not null default now(),
  constraint provider_adapters_dialect_check
    check (dialect in ('anthropic_messages')),
  constraint provider_adapters_upstream_kind_check
    check (upstream_kind in ('anthropic', 'openai')),
  constraint provider_adapters_auth_mode_check
    check (auth_mode in ('bearer_to_x_api_key', 'passthrough')),
  constraint provider_adapters_prefixes_array_check
    check (jsonb_typeof(match_model_prefixes) = 'array'),
  constraint provider_adapters_aliases_object_check
    check (jsonb_typeof(model_aliases) = 'object'),
  constraint provider_adapters_sort_order_check
    check (sort_order >= 0 and sort_order <= 1000000),
  constraint provider_adapters_door_label_nonempty_check
    check (length(btrim(door_label)) > 0),
  constraint provider_adapters_upstream_path_nonempty_check
    check (length(btrim(upstream_path)) > 0),
  constraint provider_adapters_anthropic_required_check
    check (
      dialect <> 'anthropic_messages'
      or (
        anthropic_version is not null
        and length(btrim(anthropic_version)) > 0
        and default_max_tokens is not null
        and default_max_tokens >= 1
        and default_max_tokens <= 1000000
      )
    )
);

comment on table public.provider_adapters is
  'OpenAI /v1/chat/completions → provider dialect plugins. Synced to CLI via preferences; proxy fail-closed when row invalid.';
comment on column public.provider_adapters.dialect is
  'Translator plugin id. anthropic_messages = OpenAI chat ↔ Anthropic /v1/messages.';
comment on column public.provider_adapters.door_label is
  'Telemetry/stats door id (e.g. openai_to_anthropic). Never invent in code.';
comment on column public.provider_adapters.match_model_prefixes is
  'Lowercase prefixes; request model (lowercased) matching any prefix selects this adapter.';
comment on column public.provider_adapters.model_aliases is
  'JSON object client_model(lower) → upstream model id. Applied after prefix match.';
comment on column public.provider_adapters.require_alias is
  'When true, model must exist in model_aliases after prefix match; otherwise fail-closed.';
comment on column public.provider_adapters.default_max_tokens is
  'Used when OpenAI body omits max_tokens / max_completion_tokens (Anthropic requires max_tokens).';

create table if not exists public.openai_model_aliases (
  client_model text primary key,
  upstream_model text not null,
  upstream_host_contains text not null default '',
  enabled boolean not null,
  updated_at timestamptz not null default now(),
  constraint openai_model_aliases_client_nonempty_check
    check (length(btrim(client_model)) > 0),
  constraint openai_model_aliases_upstream_nonempty_check
    check (length(btrim(upstream_model)) > 0)
);

comment on table public.openai_model_aliases is
  'Rewrite OpenAI-compat request model ids before routing (e.g. trim-gemini-* → gemini-*). DB-only; no hardcode map in proxy.';
comment on column public.openai_model_aliases.upstream_host_contains is
  'When non-empty, alias applies only if UPSTREAM_OPENAI_URL host/url contains this substring (case-insensitive). Empty = always.';

create index if not exists provider_adapters_enabled_sort_idx
  on public.provider_adapters (enabled, sort_order);

create index if not exists openai_model_aliases_enabled_idx
  on public.openai_model_aliases (enabled);

-- Seed Anthropic adapter (Cursor OpenAI door → Claude).
insert into public.provider_adapters (
  id, enabled, sort_order, dialect, door_label,
  match_model_prefixes, model_aliases,
  upstream_kind, upstream_path, auth_mode,
  anthropic_version, default_max_tokens, require_alias, updated_at
) values (
  'anthropic',
  true,
  10,
  'anthropic_messages',
  'openai_to_anthropic',
  '["claude-","anthropic/","trim-claude-","trim/claude-"]'::jsonb,
  '{
    "trim-claude-sonnet": "claude-sonnet-4-20250514",
    "trim-claude-opus": "claude-opus-4-20250514",
    "trim-claude-haiku": "claude-haiku-4-20250414",
    "trim/claude-sonnet": "claude-sonnet-4-20250514",
    "trim/claude-opus": "claude-opus-4-20250514",
    "trim/claude-haiku": "claude-haiku-4-20250414",
    "anthropic/claude-sonnet-4": "claude-sonnet-4-20250514",
    "anthropic/claude-opus-4": "claude-opus-4-20250514",
    "anthropic/claude-haiku-4": "claude-haiku-4-20250414"
  }'::jsonb,
  'anthropic',
  '/v1/messages',
  'bearer_to_x_api_key',
  '2023-06-01',
  4096,
  false,
  now()
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
  default_max_tokens = excluded.default_max_tokens,
  require_alias = excluded.require_alias,
  updated_at = now();

-- Former hard-coded Gemini Cursor aliases → DB.
insert into public.openai_model_aliases (client_model, upstream_model, upstream_host_contains, enabled, updated_at)
values
  ('trim-gemini-flash', 'gemini-3.6-flash', 'generativelanguage.googleapis.com', true, now()),
  ('trim-gemini-3.6-flash', 'gemini-3.6-flash', 'generativelanguage.googleapis.com', true, now()),
  ('trim-gemini-2.5-flash', 'gemini-2.5-flash', 'generativelanguage.googleapis.com', true, now()),
  ('trim-gemini-2.5-pro', 'gemini-2.5-pro', 'generativelanguage.googleapis.com', true, now()),
  ('trim/gemini-3.6-flash', 'gemini-3.6-flash', 'generativelanguage.googleapis.com', true, now()),
  ('trim/gemini-2.5-flash', 'gemini-2.5-flash', 'generativelanguage.googleapis.com', true, now()),
  ('trim/gemini-flash', 'gemini-3.6-flash', 'generativelanguage.googleapis.com', true, now())
on conflict (client_model) do update set
  upstream_model = excluded.upstream_model,
  upstream_host_contains = excluded.upstream_host_contains,
  enabled = excluded.enabled,
  updated_at = now();

insert into public.site_messages (code, body, updated_at)
values
  (
    'CLI_PROXY_ADAPTER_REQUIRED',
    'Provider adapters are not synced. Run trim config sync while logged in, then trim start. Adapters come from Trim cloud (provider_adapters); the proxy does not invent routing.',
    now()
  ),
  (
    'CLI_PROXY_ADAPTER_UNKNOWN_DIALECT_FMT',
    'Provider adapter %q has unsupported dialect %q. Update provider_adapters in the database (supported: anthropic_messages).',
    now()
  ),
  (
    'CLI_PROXY_ADAPTER_MODEL_REQUIRED',
    'OpenAI-compat adapter routing requires a non-empty model field on /v1/chat/completions.',
    now()
  ),
  (
    'CLI_PROXY_ADAPTER_ALIAS_REQUIRED_FMT',
    'Model %q matched adapter %q but require_alias is on and no model_aliases entry exists. Add an alias in Admin → Provider adapters or use a real upstream model id.',
    now()
  ),
  (
    'CLI_PROXY_ADAPTER_TRANSLATE_FMT',
    'Provider adapter %q failed to translate the request: %v',
    now()
  ),
  (
    'CLI_PROXY_ADAPTER_AUTH_FMT',
    'Provider adapter %q auth mapping failed: %v. For Claude via Cursor, put your Anthropic API key in the OpenAI API key field (Bearer → x-api-key).',
    now()
  ),
  (
    'CLI_PROXY_ADAPTER_RESPONSE_FMT',
    'Provider adapter %q failed to translate the upstream response: %v',
    now()
  ),
  (
    'CLI_PROXY_ADAPTER_ENABLED_FMT',
    'OpenAI-compat provider adapters loaded (%d enabled). Claude-family models on /v1/chat/completions route through the Anthropic adapter; other models use UPSTREAM_OPENAI_URL.',
    now()
  ),
  (
    'CLI_SETUP_SHELL_HEADER',
    'Shell / Claude Code / Aider (same Trim Base URL; different dialect env):',
    now()
  ),
  (
    'CLI_SETUP_SHELL_OPENAI_FMT',
    '  export OPENAI_BASE_URL=%q   # Cursor / OpenAI-compat clients (Gemini, GPT, Claude-via-adapter)',
    now()
  ),
  (
    'CLI_SETUP_SHELL_ANTHROPIC_FMT',
    '  export ANTHROPIC_BASE_URL=%q   # Claude Code / Anthropic SDK (native /v1/messages; not Cursor OpenAI Base URL)',
    now()
  ),
  (
    'DOCS_PROVIDER_ADAPTERS_SUMMARY',
    'Trim exposes one Base URL with two native doors: OpenAI /v1/chat/completions and Anthropic /v1/messages. A DB-driven OpenAI→provider adapter layer (Anthropic first) lets Cursor send Claude models on the OpenAI door without OpenRouter. Never paste https://api.anthropic.com into Cursor Override OpenAI Base URL.',
    now()
  ),
  (
    'DOCS_CONNECT_IDE_SUMMARY',
    'Point Cursor/VS Code OpenAI Base URL at Trim. Point Claude Code at the same Trim URL via ANTHROPIC_BASE_URL. Run trim config sync so provider adapters and model aliases load from the database.',
    now()
  ),
  (
    'DOCS_TROUBLESHOOT_BASE_URL',
    'Do not set Cursor Override OpenAI Base URL to https://api.anthropic.com (wrong protocol). Do not mix Cursor Anthropic BYOK with Trim OpenAI override. For Claude in Cursor: Trim Base URL + Claude model id + Anthropic key in the OpenAI key field. For Claude Code: ANTHROPIC_BASE_URL=Trim /v1 only.',
    now()
  ),
  (
    'ADMIN_PROVIDER_ADAPTERS_TITLE',
    'Provider adapters',
    now()
  ),
  (
    'ADMIN_PROVIDER_ADAPTERS_DESC',
    'OpenAI-compat /v1/chat/completions plugins. Anthropic dialect translates to /v1/messages. Synced to CLI via preferences. No hard-coded model maps in the proxy.',
    now()
  ),
  (
    'ADMIN_OPENAI_MODEL_ALIASES_TITLE',
    'OpenAI model aliases',
    now()
  ),
  (
    'ADMIN_OPENAI_MODEL_ALIASES_DESC',
    'Rewrite client model ids before routing (e.g. trim-gemini-* → gemini-*). Optional upstream_host_contains scopes an alias to a specific OpenAI upstream host.',
    now()
  ),
  (
    'PREFERENCES_CLI_HELP_7',
    'Claude in Cursor: same OpenAI Base URL → Trim; pick a Claude / trim-claude-* model; put your Anthropic key in the OpenAI API key field. Adapters sync via trim config sync (no OpenRouter).',
    now()
  ),
  (
    'PREFERENCES_CLI_HELP_8',
    'Claude Code: export ANTHROPIC_BASE_URL to the same Trim …/v1 URL. Do not paste api.anthropic.com into Cursor OpenAI Base URL.',
    now()
  ),
  (
    'LOCAL_STATS_LAST_DOOR',
    'Last door',
    now()
  ),
  (
    'CLI_PROXY_DOOR_OPENAI',
    'openai',
    now()
  ),
  (
    'CLI_PROXY_DOOR_ANTHROPIC',
    'anthropic',
    now()
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();

update public.site_messages
set body = 'Shell / Claude Code / Aider (same Trim Base URL; different dialect env):',
    updated_at = now()
where code = 'CLI_SETUP_SHELL_HEADER';

insert into public.site_messages (code, body, updated_at)
values ('ADMIN_NAV_PROVIDER_ADAPTERS', 'Provider adapters', now())
on conflict (code) do update
set body = excluded.body,
    updated_at = now();

insert into public.admin_nav_items (id, href, chrome_code, permission_code, sort_order, is_active)
values ('provider_adapters', '/billing/provider-adapters', 'ADMIN_NAV_PROVIDER_ADAPTERS', 'billing.settings', 61, true)
on conflict (id) do update set
  href = excluded.href,
  chrome_code = excluded.chrome_code,
  permission_code = excluded.permission_code,
  sort_order = excluded.sort_order,
  is_active = excluded.is_active;
