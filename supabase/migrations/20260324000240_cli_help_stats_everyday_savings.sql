-- Everyday help: put clear local savings commands next to status.
-- Idempotent upsert so older CLI_HELP_STATS_SHORT ("Show local proxy stats") is replaced.

insert into public.site_messages (code, body, updated_at)
values
  ('CLI_HELP_ROOT_LONG', E'Trim keeps a local Fast Mode proxy on your machine so IDE traffic uses less noisy context.\n\nEveryday: start, stop, status, stats, autostart, login, setup\nAdvanced: tui, compress, daemon, telemetry, config\n\nSimple savings check after trim start:\n  trim stats\n  open http://127.0.0.1:<port>/dashboard → Show savings detail\n\ntrim status = cloud quota. trim stats = local Deep status / savings meter.\n\nRun "trim help <command>" for details. Prefer trim start / autostart for daily use; compress and Deep Mode are file/batch tools, not the everyday IDE path.', now()),
  ('CLI_HELP_STATUS_LONG', E'Shows cloud login/quota for your Trim account.\n\nFor local savings / Deep status after trim start, use:\n  trim stats\n  open http://127.0.0.1:<port>/dashboard → Show savings detail\n\ntrim status does not replace the local meter.', now()),
  ('CLI_HELP_STATS_SHORT', 'Show local proxy savings and Deep status', now()),
  ('CLI_HELP_STATS_LONG', E'Reads the live local proxy (/v1/stats) and prints clear Deep status + Deep stage savings, then the raw JSON.\n\nUseful when Claude Code / Cursor shows 0% whole-body Saved: check Deep status (skipped below min, chrome frozen, fail-closed) and Deep stage tokens.\n\nEveryday:\n  trim start\n  trim stats\n  open http://127.0.0.1:<TRIM_PORT>/dashboard → Show savings detail\n\nUse --tui for the Bubble Tea live dashboard (same as trim tui).', now()),
  ('CLI_HELP_START_LONG', E'Starts the local proxy for IDE traffic. Fast Mode always runs. When account preferences set Deep Mode on, each request also runs Deep (LLMLingua) after Fast before forwarding upstream.\n\nAfter traffic flows, check savings clearly with:\n  trim stats\n  http://127.0.0.1:<port>/dashboard → Show savings detail\n\nWhole-body Saved %% can look low on agent chrome; Deep stage and Deep status explain why.', now()),
  ('CLI_STATS_DEEP_STATUS_FMT', 'Deep: %s', now()),
  ('CLI_STATS_DEEP_STAGE_FMT', 'Deep stage: %s (%.1f%% saved)', now()),
  ('CLI_STATS_LAST_REQUEST_FMT', 'Last request (whole body): %s (%.1f%% saved)', now()),
  ('CLI_STATS_DOOR_FMT', 'Last door: %s', now()),
  ('CLI_STATS_DASHBOARD_TIP_FMT', 'Local meter: http://127.0.0.1:{port}/dashboard → Show savings detail (0% whole-body Saved on Claude Code chrome is often normal).', now()),
  ('CLI_STATS_TUI_TIP', 'Tip: trim stats --tui   (or: trim tui)', now())
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
