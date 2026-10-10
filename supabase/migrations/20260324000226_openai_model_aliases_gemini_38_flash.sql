-- Refresh Gemini OpenAI-compat convenience aliases to current Google OpenAI-compat docs
-- (base remains UPSTREAM_OPENAI_URL …/v1beta/openai; model ids are DB-owned).

update public.openai_model_aliases
set upstream_model = 'gemini-3.8-flash',
    updated_at = now()
where client_model in (
  'trim-gemini-flash',
  'trim-gemini-3.6-flash',
  'trim/gemini-3.6-flash',
  'trim/gemini-flash'
)
and upstream_host_contains = 'generativelanguage.googleapis.com';

insert into public.openai_model_aliases (client_model, upstream_model, upstream_host_contains, enabled, updated_at)
values
  ('trim-gemini-3.8-flash', 'gemini-3.8-flash', 'generativelanguage.googleapis.com', true, now()),
  ('trim/gemini-3.8-flash', 'gemini-3.8-flash', 'generativelanguage.googleapis.com', true, now())
on conflict (client_model) do update set
  upstream_model = excluded.upstream_model,
  upstream_host_contains = excluded.upstream_host_contains,
  enabled = excluded.enabled,
  updated_at = now();
