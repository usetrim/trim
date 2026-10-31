-- Dashboard Preferences chrome: Segmented Allowlist Deep wording.
-- Idempotent. No em dashes.

insert into public.site_messages (code, body, updated_at)
values
  (
    'PREFERENCES_PAGE_DESCRIPTION',
    'Live proxy runs Fast Mode on every turn. When account preferences set compression_tier=deep, the local proxy also runs Segmented Allowlist Deep after Fast: freeze tools/system/paths/code/JSON, compress last-user prose islands only at your compression rate, validate, else fail-closed Fast. Models and OOM policy come from billing settings via trim config sync.',
    now()
  ),
  (
    'PREFERENCES_DEEP_HINT',
    'When Deep Mode is on, the live IDE proxy (trim start) runs Fast then Segmented Allowlist Deep (prose islands only; tools/paths frozen) using your engine and compression rate from this page. Off = Fast only. Trim cloud never loads Deep engines; auth, quotas, and billing still use Trim cloud.',
    now()
  ),
  (
    'PREFERENCES_NOTE',
    'Deep Mode runs on your machine for live IDE proxy (trim start: Segmented Allowlist Deep) and trim compress. Engines are not loaded in Trim cloud; auth and quotas still use Trim cloud when licensed.',
    now()
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
