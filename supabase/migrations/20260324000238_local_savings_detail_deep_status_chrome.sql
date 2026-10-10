-- Savings clarity chrome for local dashboard /v1/stats (Deep stage + status).
-- Fail-closed: empty bodies until SeedAndRefresh; builtins cover LOCAL_* bootstrap.

insert into public.site_messages (code, body, updated_at)
values
  ('LOCAL_LABEL_DEEP_STATUS', 'Deep status', now()),
  ('LOCAL_LABEL_DEEP_STAGE', 'Deep stage', now()),
  ('LOCAL_LABEL_DEEP_STAGE_SAVED', 'Deep stage saved', now()),
  ('LOCAL_DEEP_STATUS_APPLIED', 'Deep applied', now()),
  ('LOCAL_DEEP_STATUS_SKIPPED_MIN', 'Deep skipped (below min input tokens)', now()),
  ('LOCAL_DEEP_STATUS_SKIPPED_STREAM', 'Deep skipped (stream request)', now()),
  ('LOCAL_DEEP_STATUS_SKIPPED_EMPTY', 'Deep skipped (nothing compressible / chrome frozen)', now()),
  ('LOCAL_DEEP_STATUS_SKIPPED_OFF', 'Deep off for this request', now()),
  ('LOCAL_DEEP_STATUS_PATH_SKIP', 'Deep skipped (this path is Fast-only)', now()),
  ('LOCAL_DEEP_STATUS_FAIL_CLOSED_EXPAND', 'Deep kept Fast (expansion fail-closed)', now()),
  ('LOCAL_DEEP_STATUS_FAIL_CLOSED_ERROR', 'Deep kept Fast (engine error)', now()),
  ('LOCAL_DEEP_STATUS_OOM_SKIP', 'Deep skipped after OOM (kept Fast)', now()),
  ('LOCAL_DEEP_STATUS_REJECTED', 'Deep result rejected (kept Fast)', now()),
  ('LOCAL_DEEP_STATUS_FAST_ONLY', 'Fast only (Deep not enabled)', now()),
  ('LOCAL_SAVINGS_DETAIL_TITLE', 'Savings detail', now()),
  ('LOCAL_SAVINGS_DETAIL_LEAD', 'Whole-request Saved % can look low when IDE agent chrome is frozen. Deep stage shows savings on the compressible text Trim actually rewrote.', now()),
  ('LOCAL_SAVINGS_DETAIL_SHOW', 'Show savings detail', now()),
  ('LOCAL_SAVINGS_DETAIL_HIDE', 'Hide savings detail', now()),
  ('LOCAL_SAVINGS_DETAIL_WIRE_FMT', 'Whole request: {before} → {after} ({pct}% saved)', now()),
  ('LOCAL_SAVINGS_DETAIL_STAGE_FMT', 'Deep stage: {before} → {after} ({pct}% saved)', now()),
  ('LOCAL_SAVINGS_DETAIL_STATUS_FMT', 'Status: {status}', now()),
  ('LOCAL_SAVINGS_DETAIL_CHROME_TIP', '0% whole-request Saved on Claude Code / Cursor agent turns is often normal: system reminders, tools, and git status stay frozen on purpose so the agent loop stays safe.', now()),
  ('PREFERENCES_LIVE_DEEP_MIN_HINT', 'Live Deep runs only when estimated compressible input tokens are at least live_deep_min_input_tokens from billing settings (0 = every turn). Lower values run Deep on more chats (more CPU/latency). Synced by trim config sync.', now()),
  ('ADMIN_BILLING_LIVE_DEEP_MIN_DESC', 'Skip Deep on the live proxy when estimated input tokens are below this (0 = every turn). Lowering increases Deep coverage and local CPU/latency. Synced to CLI.', now())
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
