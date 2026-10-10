-- OpenAI org/project env chrome + upstream /models?adapter= proxy-through chrome.
-- Default GET /v1/models remains DB catalog. Optional live discovery: ?adapter=<provider_adapters.id>.

insert into public.site_messages (code, body, updated_at)
values
  (
    'CLI_PROXY_MODELS_UPSTREAM_NOT_FOUND_FMT',
    'Unknown or disabled provider adapter %q for upstream model discovery. Use an enabled openai_compat adapter id from provider_adapters (e.g. openai, gemini, deepseek, mistral).',
    now()
  ),
  (
    'CLI_PROXY_MODELS_UPSTREAM_AUTH',
    'Authorization Bearer is required for GET /v1/models?adapter=… (upstream key-accurate discovery).',
    now()
  ),
  (
    'CLI_PROXY_MODELS_UPSTREAM_UNSUPPORTED_FMT',
    'Adapter %q cannot proxy upstream /models (openai_compat only). Default GET /v1/models without ?adapter= still returns the DB catalog.',
    now()
  ),
  (
    'CLI_PROXY_MODELS_UPSTREAM_ADAPTER_REQUIRED',
    'Upstream model discovery requires ?adapter=<id> (enabled openai_compat provider_adapters id).',
    now()
  ),
  (
    'PREFERENCES_CLI_HELP_9',
    'OpenAI org/project (optional): send OpenAI-Organization / OpenAI-Project on the request, or set TRIM_OPENAI_ORGANIZATION / TRIM_OPENAI_PROJECT once (client headers win). Live key-accurate model list: GET /v1/models?adapter=openai (or gemini|deepseek|mistral) with Authorization Bearer - default GET /v1/models stays the DB catalog.',
    now()
  )
on conflict (code) do update set
  body = excluded.body,
  updated_at = now();
