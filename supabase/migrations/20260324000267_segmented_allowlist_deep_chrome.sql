-- Segmented Allowlist Deep UX chrome: freeze agent control / paths / fences / JSON;
-- compress prose islands only; meter Deep stage separately from whole-request %.
-- Idempotent upsert. No em dashes.

insert into public.site_messages (code, body, updated_at)
values
  (
    'LOCAL_DEEP_STATUS_SKIPPED_EMPTY',
    'Deep skipped (no prose islands / chrome frozen)',
    now()
  ),
  (
    'LOCAL_SAVINGS_DETAIL_LEAD',
    'Whole-request Saved % can look low when IDE agent chrome is frozen. Deep compresses prose islands only (paths, tools, and schemas stay verbatim). Deep stage shows savings on that prose.',
    now()
  ),
  (
    'LOCAL_SAVINGS_DETAIL_CHROME_TIP',
    '0% whole-request Saved on Claude Code / Cursor agent turns is often normal: system reminders, tools, and git status stay frozen on purpose. Eligible last-user prose still runs Deep; check Deep stage for real savings.',
    now()
  ),
  (
    'CLI_STATS_DEEP_STAGE_FMT',
    'Deep stage (prose islands): %s (%.1f%% saved)',
    now()
  ),
  (
    'CLI_HELP_STATS_LONG',
    E'Reads the live local proxy (/v1/stats) and prints clear Deep status + Deep stage savings, then the raw JSON.\n\nUseful when Claude Code / Cursor shows 0% whole-body Saved: check Deep status and Deep stage (prose islands only; tools/paths/schemas stay frozen).\n\nEveryday:\n  trim start\n  trim stats\n  open http://127.0.0.1:<TRIM_PORT>/dashboard → Show savings detail\n\nUse --tui for the Bubble Tea live dashboard (same as trim tui).',
    now()
  ),
  (
    'CLI_HELP_START_LONG',
    E'Starts the local proxy for IDE traffic. Fast Mode always runs. When account preferences set Deep Mode on, each request also runs Segmented Allowlist Deep after Fast: freeze tools/system/paths/code/JSON, compress last-user prose islands only, validate, else fail-closed Fast.\n\nAfter traffic flows, check savings clearly with:\n  trim stats\n  http://127.0.0.1:<port>/dashboard → Show savings detail\n\nWhole-body Saved %% can look low on agent chrome; Deep stage (prose) and Deep status explain why.',
    now()
  ),
  (
    'LOCAL_LABEL_DEEP_STAGE',
    'Deep stage (prose)',
    now()
  ),
  (
    'LOCAL_LABEL_DEEP_STAGE_SAVED',
    'Deep stage saved (prose)',
    now()
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
