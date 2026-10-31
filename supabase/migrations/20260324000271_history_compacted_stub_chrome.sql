-- History compaction stub from site_messages (Claude/Cursor ladder).
-- Fail-closed: empty chrome disables history stubbing even when history_keep_turns > 0.
-- Idempotent. No em dashes.

insert into public.site_messages (code, body, updated_at)
values
  (
    'CLI_PROXY_HISTORY_COMPACTED_STUB',
    '[trim] earlier conversation turns compacted; recent turns retained',
    now()
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
