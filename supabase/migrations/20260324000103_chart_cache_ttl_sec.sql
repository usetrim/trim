-- Chart cache TTL in billing_settings (DB-driven; no invent at request time).
-- Outbox rows share trim_events retention window on purge.

alter table public.billing_settings
  add column if not exists chart_cache_ttl_sec int not null default 60
  check (chart_cache_ttl_sec >= 1 and chart_cache_ttl_sec <= 86400);

comment on column public.billing_settings.chart_cache_ttl_sec is
  'Redis TTL seconds for dashboard usage/heatmap JSON cache; fail-closed if missing/invalid';

insert into public.site_messages (code, body) values
  ('CHART_CACHE_TTL_MISSING', 'billing_settings.chart_cache_ttl_sec is missing')
on conflict (code) do update set body = excluded.body;
