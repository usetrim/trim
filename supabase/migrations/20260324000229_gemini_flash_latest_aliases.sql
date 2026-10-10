-- Retarget Gemini convenience aliases to a Google OpenAI-compat model id that
-- serves on the free Generative Language API today.
-- Live probe: gemini-flash-latest → HTTP 200; gemini-3.8-flash → 503;
-- gemini-2.0-flash / gemini-1.5-flash → retired 404 from Google.

update public.provider_adapters
set model_aliases = '{
  "trim-gemini-flash": "gemini-flash-latest",
  "trim-gemini-3.8-flash": "gemini-flash-latest",
  "trim-gemini-3.6-flash": "gemini-flash-latest",
  "trim-gemini-flash-latest": "gemini-flash-latest",
  "trim/gemini-flash": "gemini-flash-latest",
  "trim/gemini-3.8-flash": "gemini-flash-latest",
  "trim/gemini-3.6-flash": "gemini-flash-latest",
  "trim/gemini-flash-latest": "gemini-flash-latest"
}'::jsonb,
    updated_at = now()
where id = 'gemini';

update public.openai_model_aliases
set upstream_model = 'gemini-flash-latest',
    updated_at = now()
where client_model in (
  'trim-gemini-flash',
  'trim-gemini-3.6-flash',
  'trim-gemini-3.8-flash',
  'trim/gemini-3.6-flash',
  'trim/gemini-3.8-flash',
  'trim/gemini-flash'
)
and upstream_host_contains = 'generativelanguage.googleapis.com';

insert into public.openai_model_aliases (client_model, upstream_model, upstream_host_contains, enabled, updated_at)
values
  ('trim-gemini-flash-latest', 'gemini-flash-latest', 'generativelanguage.googleapis.com', true, now()),
  ('trim/gemini-flash-latest', 'gemini-flash-latest', 'generativelanguage.googleapis.com', true, now())
on conflict (client_model) do update set
  upstream_model = excluded.upstream_model,
  upstream_host_contains = excluded.upstream_host_contains,
  enabled = excluded.enabled,
  updated_at = now();
