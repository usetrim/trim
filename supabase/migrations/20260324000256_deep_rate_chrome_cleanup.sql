-- Cleanup leftover target_token / Fast-default chrome after rate cutover.
-- No schema drops; site_messages body sync only + empty obsolete rows cleared.

update public.site_messages
set body = 'Deep Mode requires deep_engine and compression-rate (run: trim config sync, or trim config set deep-engine / compression-rate, or pass --engine and --rate)',
    updated_at = now()
where code = 'CLI_COMPRESS_DEEP_PREFS_REQUIRED'
  and body is distinct from 'Deep Mode requires deep_engine and compression-rate (run: trim config sync, or trim config set deep-engine / compression-rate, or pass --engine and --rate)';

update public.site_messages
set body = 'unknown key %q (default-tier|deep-engine|compression-rate)',
    updated_at = now()
where code = 'CLI_CONFIG_UNKNOWN_KEY_FMT'
  and body is distinct from 'unknown key %q (default-tier|deep-engine|compression-rate)';

update public.site_messages
set body = 'Live proxy runs Fast Mode on every turn. When account preferences set compression_tier=deep, the local proxy also runs Deep Mode (LLMLingua) after Fast using your compression rate. Models and OOM policy come from billing settings via trim config sync.',
    updated_at = now()
where code = 'DOCS_FAST_VS_DEEP_LIVE';

update public.site_messages
set body = 'Live proxy runs Fast Mode on every turn. When account preferences set compression_tier=deep, the local proxy also runs Deep Mode (LLMLingua) after Fast using your compression rate. Models and OOM policy come from billing settings via trim config sync.',
    updated_at = now()
where code = 'PREFERENCES_PAGE_DESCRIPTION';

-- Empty obsolete rows from removed dials (fail-closed chrome consumers treat blank as absent).
delete from public.site_messages
where code in (
  'PREFERENCES_LIVE_DEEP_MIN_HINT',
  'PREFERENCES_LIVE_DEEP_STREAM_HINT',
  'ADMIN_BILLING_LIVE_DEEP_MIN',
  'ADMIN_BILLING_LIVE_DEEP_MIN_DESC',
  'ADMIN_BILLING_LIVE_DEEP_STREAM',
  'ADMIN_BILLING_LIVE_DEEP_STREAM_DESC',
  'LOCAL_SAVINGS_DETAIL_SKIP_STREAM',
  'LOCAL_SAVINGS_DETAIL_SKIP_MIN'
)
and coalesce(btrim(body), '') = '';
