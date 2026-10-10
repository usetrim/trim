-- Install beacon path dimension on distribution aggregates (e.g. /install.sh vs /install.ps1).
-- Non-install sources keep path = ''.

alter table public.distribution_daily_stats
  add column if not exists path text not null default '';

comment on column public.distribution_daily_stats.path is
  'Installer path for source=install (e.g. /install.sh); empty for other sources.';

alter table public.distribution_daily_stats
  drop constraint if exists distribution_daily_stats_pkey;

alter table public.distribution_daily_stats
  add primary key (day, source, metric, country, path);

-- Stale install aggregates lacked path; Sync rebuilds today's rows with path.
delete from public.distribution_daily_stats where source = 'install';

comment on table public.distribution_daily_stats is
  'Aggregated GitHub traffic, release downloads, install hits (by country + path). Country is ISO code or empty.';

insert into public.site_messages (code, body) values
  ('ADMIN_DIST_COL_PATH', 'Path')
on conflict (code) do update set body = excluded.body, updated_at = now();
