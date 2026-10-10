-- Honest gateway chrome for upstream auth (401) and quota/rate-limit (429) failures.
-- Fail-closed: empty site_messages leave upstream bodies unchanged.

insert into public.site_messages (code, body, updated_at)
values
  (
    'CLI_PROXY_ERR_UPSTREAM_AUTH_FMT',
    'Upstream rejected the API key for model %q (authentication failed). Use a key that matches that model''s provider (Anthropic for Claude, OpenAI for GPT, Gemini/DeepSeek/Mistral keys for those hosts). For Claude via Cursor put the Anthropic key in the OpenAI API key field.',
    now()
  ),
  (
    'CLI_PROXY_ERR_UPSTREAM_QUOTA_FMT',
    'Upstream rate limit or quota exhausted for model %q. Wait and retry, check the provider billing/plan, or pick another model from GET /v1/models.',
    now()
  )
on conflict (code) do update set
  body = excluded.body,
  updated_at = now();
