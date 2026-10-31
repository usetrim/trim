-- Min-token live Deep gate is removed: user Deep prefs alone decide when LLMLingua runs
-- (after chrome freeze / empty-prose). Keep skipped_min status key for old meter rows only.
-- Idempotent. No em dashes. No schema drops beyond chrome body sync.

update public.site_messages
set body = 'Deep skipped (legacy status; min-token gate removed)',
    updated_at = now()
where code = 'LOCAL_DEEP_STATUS_SKIPPED_MIN'
  and body is distinct from 'Deep skipped (legacy status; min-token gate removed)';

-- Ensure obsolete dial / skip-fmt chrome stays absent (fail-closed consumers treat missing as absent).
delete from public.site_messages
where code in (
  'PREFERENCES_LIVE_DEEP_MIN_HINT',
  'PREFERENCES_LIVE_DEEP_STREAM_HINT',
  'ADMIN_BILLING_LIVE_DEEP_MIN',
  'ADMIN_BILLING_LIVE_DEEP_MIN_DESC',
  'ADMIN_BILLING_LIVE_DEEP_STREAM',
  'ADMIN_BILLING_LIVE_DEEP_STREAM_DESC',
  'ADMIN_BILLING_LIVE_DEEP_SKIP_STREAM',
  'ADMIN_BILLING_LIVE_DEEP_SKIP_STREAM_DESC',
  'CLI_PROXY_DEEP_SKIPPED_MIN_FMT',
  'CLI_PROXY_DEEP_SKIPPED_STREAM'
);
