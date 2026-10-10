-- Live Deep streaming policy dial (DB-driven; no invent).
-- When true, live proxy skips Deep for requests with stream=true (Fast only); response SSE still works.

alter table public.billing_settings
  add column if not exists live_deep_skip_on_stream boolean;

update public.billing_settings
set live_deep_skip_on_stream = coalesce(live_deep_skip_on_stream, false)
where id = 'default';

alter table public.billing_settings
  alter column live_deep_skip_on_stream set not null;

comment on column public.billing_settings.live_deep_skip_on_stream is
  'When true, trim start skips Deep on stream=true requests (Fast only). When false, Deep still runs on the full request body before upstream SSE. Synced to CLI via preferences.';

insert into public.site_messages (code, body, updated_at)
values
  (
    'CLI_PROXY_DEEP_SKIPPED_STREAM',
    'Live Deep skipped (stream=true and billing live_deep_skip_on_stream). Fast Mode only for this request.',
    now()
  ),
  (
    'ADMIN_BILLING_LIVE_DEEP_SKIP_STREAM',
    'Skip Deep on stream requests',
    now()
  ),
  (
    'ADMIN_BILLING_LIVE_DEEP_SKIP_STREAM_DESC',
    'When on, live proxy skips Deep for stream=true chat (lower latency). When off, Deep still runs on the full body before upstream streaming.',
    now()
  ),
  (
    'PREFERENCES_LIVE_DEEP_STREAM_HINT',
    'Billing may skip live Deep when the client sets stream=true (live_deep_skip_on_stream). Synced by trim config sync.',
    now()
  ),
  (
    'DOCS_FAST_VS_DEEP_LIVE',
    'Live proxy runs Fast Mode on every turn. When account preferences set compression_tier=deep, the local proxy also runs Deep Mode (LLMLingua) after Fast when input tokens meet billing live_deep_min_input_tokens (0 = all turns), unless live_deep_skip_on_stream skips Deep on stream=true. Models and OOM policy come from billing settings via trim config sync.',
    now()
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();

update public.site_messages
set body = 'Live Deep Mode on (engine %s, target_token %d, min_input_tokens %d, oom=%s, skip_stream=%t). Synced from account preferences / billing settings.',
    updated_at = now()
where code = 'CLI_PROXY_LIVE_DEEP_ENABLED_FMT';
