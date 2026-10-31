-- Tool-result clearing chrome (Claude/Cursor/Codex ladder) + Segmented Allowlist Deep docs.
-- Idempotent upsert. No em dashes. Clearing stays off until tool_result_clear_keep_recent > 0
-- and this chrome is synced (fail-closed: empty fmt disables clearing).

insert into public.site_messages (code, body, updated_at)
values
  (
    'CLI_PROXY_TOOL_RESULT_CLEARED_FMT',
    '[trim] prior tool output cleared (%d bytes; re-fetch if needed)',
    now()
  ),
  (
    'DOCS_FAST_VS_DEEP_LIVE',
    'Live proxy runs Fast Mode on every turn. When account preferences set compression_tier=deep, the local proxy also runs Segmented Allowlist Deep after Fast: freeze tools/system/paths/code/JSON, compress last-user prose islands only at your compression rate, validate, else fail-closed Fast. Models and OOM policy come from billing settings via trim config sync.',
    now()
  ),
  (
    'DOCS_FAST_VS_DEEP_SUMMARY',
    'Start with Fast Mode on the live proxy. Turn Deep Mode on in Preferences (then trim config sync) so trim start runs Segmented Allowlist Deep after Fast (prose islands only). Or use trim compress --deep for file/batch jobs.',
    now()
  ),
  (
    'LOCAL_SAVINGS_DETAIL_CHROME_TIP',
    '0% whole-request Saved on Claude Code / Cursor agent turns is often normal: system reminders, tools, and git status stay frozen on purpose. Eligible last-user prose still runs Deep; check Deep stage for real savings. Optional tool_result_clear_keep_recent clears older tool outputs (chrome from Trim cloud).',
    now()
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
