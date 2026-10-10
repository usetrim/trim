-- Refresh discoverable catalog from current official provider ids (DB-only; no proxy invent).
-- DeepSeek docs: deepseek-flash / deepseek-v4-pro (legacy deepseek-chat still accepted).
-- Gemini OpenAI-compat docs: gemini-3.8-flash current flash id.
-- Honest chrome when upstream returns HTML/5xx (Cloudflare 522, etc.).

insert into public.provider_discoverable_models
  (id, enabled, sort_order, display_name, description, owned_by, adapter_id, updated_at)
values
  ('deepseek-flash', true, 48, 'DeepSeek Flash', 'Current DeepSeek OpenAI-compat flash id', 'deepseek', 'deepseek', now()),
  ('deepseek-v4-pro', true, 49, 'DeepSeek V4 Pro', 'Current DeepSeek OpenAI-compat pro id', 'deepseek', 'deepseek', now()),
  ('gemini-3.8-flash', true, 39, 'Gemini 3.8 Flash', 'Current Google OpenAI-compat flash id', 'google', 'gemini', now())
on conflict (id) do update set
  enabled = excluded.enabled,
  sort_order = excluded.sort_order,
  display_name = excluded.display_name,
  description = excluded.description,
  owned_by = excluded.owned_by,
  adapter_id = excluded.adapter_id,
  updated_at = now();

-- Keep legacy DeepSeek chat id advertised; rewrite to current official flash id upstream.
update public.provider_adapters
set
  model_aliases = coalesce(model_aliases, '{}'::jsonb) || jsonb_build_object(
    'deepseek-chat', 'deepseek-flash'
  ),
  updated_at = now()
where id = 'deepseek';

insert into public.site_messages (code, body, updated_at)
values
  (
    'CLI_PROXY_ERR_UPSTREAM_UNAVAILABLE_FMT',
    'Upstream provider is temporarily unavailable for model %q (gateway/host error). Retry later, or pick another model/provider from GET /v1/models. This is not a Trim Base URL mistake.',
    now()
  )
on conflict (code) do update set
  body = excluded.body,
  updated_at = now();
