-- Align Base URL troubleshooting chrome with Anthropic official OpenAI-compat guidance
-- (their layer is limited / not long-term production; Trim keeps Base URL = Trim + Messages adapter).

insert into public.site_messages (code, body, updated_at)
values
  (
    'DOCS_PROVIDER_ADAPTERS_SUMMARY',
    'Trim exposes one Base URL with two native doors: OpenAI /v1/chat/completions and Anthropic /v1/messages. A DB-driven OpenAI→provider adapter (Anthropic Messages first) lets Cursor send Claude on the OpenAI door without OpenRouter. Anthropic’s own OpenAI-compat URL is limited and not Trim’s path - keep Base URL = Trim.',
    now()
  ),
  (
    'DOCS_TROUBLESHOOT_BASE_URL',
    'Do not set Cursor Override OpenAI Base URL to https://api.anthropic.com (bypasses Trim; Anthropic’s OpenAI-compat layer is limited / not their long-term production path). Do not mix Cursor Anthropic BYOK with Trim OpenAI override. Claude in Cursor: Trim Base URL + Claude model + Anthropic key in the OpenAI key field + trim config sync. Claude Code: ANTHROPIC_BASE_URL=Trim /v1 only.',
    now()
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
