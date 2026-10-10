-- Same-shape OpenAI-compat host routing by model prefix (DB-driven).
-- Users must not flip UPSTREAM_OPENAI_URL daily for GPT vs Gemini vs DeepSeek.
-- dialect openai_compat = no body translate; upstream_base_url selects the host.
-- UPSTREAM_OPENAI_URL remains the default only when no adapter prefix matches.

alter table public.provider_adapters
  add column if not exists upstream_base_url text not null default '';

comment on column public.provider_adapters.upstream_base_url is
  'Optional absolute OpenAI-compat or provider base URL. When non-empty, proxy uses it instead of UPSTREAM_* env for this adapter. Required for dialect openai_compat.';

alter table public.provider_adapters
  drop constraint if exists provider_adapters_dialect_check;

alter table public.provider_adapters
  add constraint provider_adapters_dialect_check
  check (dialect in ('anthropic_messages', 'openai_compat'));

alter table public.provider_adapters
  drop constraint if exists provider_adapters_openai_compat_url_check;

alter table public.provider_adapters
  add constraint provider_adapters_openai_compat_url_check
  check (
    dialect <> 'openai_compat'
    or (
      length(btrim(upstream_base_url)) > 0
      and left(lower(btrim(upstream_base_url)), 8) = 'https://'
    )
  );

-- GPT / OpenAI (official Chat Completions base).
insert into public.provider_adapters (
  id, enabled, sort_order, dialect, door_label,
  match_model_prefixes, model_aliases,
  upstream_kind, upstream_path, auth_mode,
  anthropic_version, default_max_tokens, require_alias,
  upstream_base_url, updated_at
) values (
  'openai',
  true,
  20,
  'openai_compat',
  'openai_compat_openai',
  '["gpt-","o1-","o3-","o4-","chatgpt-"]'::jsonb,
  '{}'::jsonb,
  'openai',
  '/v1/chat/completions',
  'passthrough',
  null,
  null,
  false,
  'https://api.openai.com',
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
  require_alias = excluded.require_alias,
  upstream_base_url = excluded.upstream_base_url,
  updated_at = now();

-- Gemini OpenAI-compat (official Google OpenAI dialect base).
insert into public.provider_adapters (
  id, enabled, sort_order, dialect, door_label,
  match_model_prefixes, model_aliases,
  upstream_kind, upstream_path, auth_mode,
  anthropic_version, default_max_tokens, require_alias,
  upstream_base_url, updated_at
) values (
  'gemini',
  true,
  30,
  'openai_compat',
  'openai_compat_gemini',
  '["gemini-","trim-gemini-","trim/gemini-"]'::jsonb,
  '{
    "trim-gemini-flash": "gemini-3.8-flash",
    "trim-gemini-3.8-flash": "gemini-3.8-flash",
    "trim-gemini-3.6-flash": "gemini-3.8-flash",
    "trim/gemini-flash": "gemini-3.8-flash",
    "trim/gemini-3.8-flash": "gemini-3.8-flash",
    "trim/gemini-3.6-flash": "gemini-3.8-flash"
  }'::jsonb,
  'openai',
  '/chat/completions',
  'passthrough',
  null,
  null,
  false,
  'https://generativelanguage.googleapis.com/v1beta/openai',
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
  require_alias = excluded.require_alias,
  upstream_base_url = excluded.upstream_base_url,
  updated_at = now();

-- DeepSeek OpenAI-compat (official docs: https://api.deepseek.com + /chat/completions).
insert into public.provider_adapters (
  id, enabled, sort_order, dialect, door_label,
  match_model_prefixes, model_aliases,
  upstream_kind, upstream_path, auth_mode,
  anthropic_version, default_max_tokens, require_alias,
  upstream_base_url, updated_at
) values (
  'deepseek',
  true,
  40,
  'openai_compat',
  'openai_compat_deepseek',
  '["deepseek-"]'::jsonb,
  '{}'::jsonb,
  'openai',
  '/chat/completions',
  'passthrough',
  null,
  null,
  false,
  'https://api.deepseek.com',
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
  require_alias = excluded.require_alias,
  upstream_base_url = excluded.upstream_base_url,
  updated_at = now();

-- Mistral OpenAI-compat (official API host).
insert into public.provider_adapters (
  id, enabled, sort_order, dialect, door_label,
  match_model_prefixes, model_aliases,
  upstream_kind, upstream_path, auth_mode,
  anthropic_version, default_max_tokens, require_alias,
  upstream_base_url, updated_at
) values (
  'mistral',
  true,
  50,
  'openai_compat',
  'openai_compat_mistral',
  '["mistral-","codestral-","open-mistral-","open-codestral-","pixtral-"]'::jsonb,
  '{}'::jsonb,
  'openai',
  '/v1/chat/completions',
  'passthrough',
  null,
  null,
  false,
  'https://api.mistral.ai',
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
  require_alias = excluded.require_alias,
  upstream_base_url = excluded.upstream_base_url,
  updated_at = now();

insert into public.site_messages (code, body, updated_at) values
  (
    'CLI_PROXY_ADAPTER_UNKNOWN_DIALECT_FMT',
    'Provider adapter %q has unsupported dialect %q. Update provider_adapters in the database (supported: anthropic_messages, openai_compat).',
    now()
  ),
  (
    'CLI_PROXY_ADAPTER_ENABLED_FMT',
    'OpenAI-compat provider adapters loaded (%d enabled). Model prefixes route via DB (Claude Messages and same-shape hosts with upstream_base_url). Unmatched models use UPSTREAM_OPENAI_URL.',
    now()
  ),
  (
    'ADMIN_PROVIDER_ADAPTERS_DESC',
    'OpenAI /v1/chat/completions plugins. anthropic_messages translates to /v1/messages. openai_compat keeps OpenAI shape and routes by match_model_prefixes to upstream_base_url (GPT, Gemini, DeepSeek, Mistral, …). No daily UPSTREAM_OPENAI_URL flipping. Synced to CLI via preferences.',
    now()
  ),
  (
    'PREFERENCES_CLI_HELP_9',
    'GPT / Gemini / DeepSeek / Mistral: same OpenAI Base URL → Trim. DB provider_adapters (openai_compat) pick the host from the model prefix. Put that host’s key in the OpenAI key field for the model you pick. UPSTREAM_OPENAI_URL is only the default for unmatched models. Do not edit .env every day.',
    now()
  ),
  (
    'DOCS_PROVIDER_ADAPTERS_SUMMARY',
    'Trim exposes one Base URL with two native doors plus DB-driven adapters: Claude via anthropic_messages, and same-shape hosts (GPT, Gemini, DeepSeek, Mistral) via openai_compat + upstream_base_url. No OpenRouter. Keep Cursor Base URL = Trim.',
    now()
  )
on conflict (code) do update set
  body = excluded.body,
  updated_at = now();
