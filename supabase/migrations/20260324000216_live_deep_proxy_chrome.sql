-- Live Deep Mode on local proxy (trim start): chrome + preference copy.
-- Source of truth remains profiles.compression_tier / deep_engine / deep_target_token (API → trim config sync).

insert into public.site_messages (code, body, updated_at)
values
  (
    'CLI_PROXY_DEEP_FAILED_FMT',
    'Live Deep Mode failed: %v',
    now()
  ),
  (
    'CLI_PROXY_LIVE_DEEP_MODE_FMT',
    'deep/%s',
    now()
  ),
  (
    'CLI_PROXY_LIVE_DEEP_ENABLED_FMT',
    'Live Deep Mode on (engine %s, target_token %d). Synced from account preferences.',
    now()
  ),
  (
    'CLI_PROXY_DEEP_COMPACT_STUB',
    '(prior turns compacted by Deep Mode)',
    now()
  ),
  (
    'CLI_PROXY_DEEP_CHROME_REQUIRED',
    'Live Deep Mode chrome missing from Trim cloud (CLI_PROXY_LIVE_DEEP_MODE_FMT / CLI_PROXY_DEEP_COMPACT_STUB). Sync site_messages and retry.',
    now()
  ),
  (
    'CLI_PROXY_DEEP_PREFS_REQUIRED',
    'Live Deep Mode requires compression_tier=deep plus deep_engine and deep_target_token. Save Dashboard → Settings, then run: trim config sync',
    now()
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();

update public.site_messages
set body = 'On = Deep Mode for live IDE proxy (trim start) and trim compress. Off = Fast Mode only on the live proxy and for compress defaults. Engine and target apply whenever Deep is on. Run trim config sync after saving.',
    updated_at = now()
where code = 'PREFERENCES_DEEP_HINT';

update public.site_messages
set body = 'Default selected: LLMLingua-2 (v2). Pick long (needs a question) or v1 if you prefer. Applies to live proxy Deep and trim compress when Deep Mode is on.',
    updated_at = now()
where code = 'PREFERENCES_ENGINE_HINT';

update public.site_messages
set body = 'Target token budget for Deep Mode (live proxy and trim compress). Only applies when Deep Mode is on.',
    updated_at = now()
where code = 'PREFERENCES_TARGET_HINT';

update public.site_messages
set body = 'When Deep Mode is on, the live IDE proxy (trim start) runs Fast then Deep (LLMLingua) using your engine and target_token from this page. Off = Fast only. Trim cloud never loads Deep engines; auth, quotas, and billing still use Trim cloud.',
    updated_at = now()
where code = 'PREFERENCES_PAGE_DESCRIPTION';

update public.site_messages
set body = 'Deep Mode runs on your machine for live IDE proxy (trim start) and trim compress. Engines are not loaded in Trim cloud; auth and quotas still use Trim cloud when licensed.',
    updated_at = now()
where code = 'PREFERENCES_NOTE';
