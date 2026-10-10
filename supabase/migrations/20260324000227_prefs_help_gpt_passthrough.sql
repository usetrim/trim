-- Dashboard Preferences CLI help: GPT / OpenAI-compat passthrough (no adapter).

insert into public.site_messages (code, body, updated_at)
values
  (
    'PREFERENCES_CLI_HELP_9',
    'GPT / Gemini / DeepSeek / Groq / etc.: same OpenAI Base URL → Trim; set UPSTREAM_OPENAI_URL to that host’s OpenAI-compat base; put that host’s key in the OpenAI key field. No adapter unless the API dialect differs (Claude uses the Anthropic adapter).',
    now()
  ),
  (
    'ADMIN_PROVIDER_ADAPTERS_DESC',
    'OpenAI-compat /v1/chat/completions plugins. Anthropic dialect translates to /v1/messages. GPT, Gemini OpenAI-compat, DeepSeek, Groq, Azure OpenAI, vLLM/Ollama stay passthrough via UPSTREAM_OPENAI_URL (config only). Synced to CLI via preferences. No hard-coded model maps in the proxy.',
    now()
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
