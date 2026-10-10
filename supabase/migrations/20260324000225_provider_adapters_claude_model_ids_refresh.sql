-- Refresh Anthropic adapter convenience aliases to current Claude API model IDs
-- (official Models overview: Sonnet 5.5 / Opus 5.5 / Haiku 4.5).
-- Retired seeds (claude-*-4-20250514) are replaced. Prefix match still accepts any claude-* id.

update public.provider_adapters
set model_aliases = '{
  "trim-claude-sonnet": "claude-sonnet-5-5",
  "trim-claude-opus": "claude-opus-5-5",
  "trim-claude-haiku": "claude-haiku-4-5-20251001",
  "trim/claude-sonnet": "claude-sonnet-5-5",
  "trim/claude-opus": "claude-opus-5-5",
  "trim/claude-haiku": "claude-haiku-4-5-20251001",
  "anthropic/claude-sonnet": "claude-sonnet-5-5",
  "anthropic/claude-opus": "claude-opus-5-5",
  "anthropic/claude-haiku": "claude-haiku-4-5-20251001",
  "anthropic/claude-sonnet-5": "claude-sonnet-5-5",
  "anthropic/claude-opus-5": "claude-opus-5-5",
  "anthropic/claude-sonnet-4": "claude-sonnet-4-6",
  "anthropic/claude-opus-4": "claude-opus-4-8",
  "anthropic/claude-haiku-4": "claude-haiku-4-5-20251001"
}'::jsonb,
    updated_at = now()
where id = 'anthropic';
