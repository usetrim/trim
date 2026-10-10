-- Align prefs/docs chrome with fail-closed unknown-model behavior when adapters are synced.
-- Code path: synced adapters + no prefix match → CLI_PROXY_ERR_UNKNOWN_MODEL_FMT (not silent UPSTREAM_OPENAI_URL).

insert into public.site_messages (code, body, updated_at)
values
  (
    'PREFERENCES_CLI_HELP_9',
    'GPT / Gemini / DeepSeek / Mistral: same OpenAI Base URL → Trim. DB provider_adapters (openai_compat) pick the host from the model prefix. Put that host’s key in the OpenAI key field for the model you pick. After trim config sync, unmatched prefixes return a clear Trim error (no silent invent). UPSTREAM_OPENAI_URL is only the OpenAI-door default when adapters are not synced. Do not edit .env every day.',
    now()
  ),
  (
    'CLI_PROXY_ADAPTER_ENABLED_FMT',
    'OpenAI-compat provider adapters loaded (%d enabled). Model prefixes route via DB (Claude Messages and same-shape hosts with upstream_base_url). GET /v1/models lists DB provider_discoverable_models. Unmatched chat models return a clear Trim error (no silent host invent).',
    now()
  )
on conflict (code) do update set
  body = excluded.body,
  updated_at = now();
