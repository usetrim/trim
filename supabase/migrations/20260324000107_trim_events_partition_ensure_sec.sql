-- Harden partition helpers after 106: ensure interval (DB), DETACH+DROP by name, chrome.
-- Idempotent when 106 already included these pieces.

alter table public.admin_retention_settings
  add column if not exists trim_events_partition_ensure_sec int not null default 86400
  check (trim_events_partition_ensure_sec >= 3600 and trim_events_partition_ensure_sec <= 604800);

comment on column public.admin_retention_settings.trim_events_partition_ensure_sec is
  'API background interval (seconds) to re-ensure future trim_events month partitions (fail-closed; no invent).';

insert into public.site_messages (code, body) values
  ('ADMIN_COMPLIANCE_EVENTS_PARTITION_ENSURE_SEC', 'Trim events partition ensure interval (seconds)'),
  ('ADMIN_EVENTS_PARTITION_ENSURE_SEC_MISSING', 'Set admin_retention_settings.trim_events_partition_ensure_sec before partition ensure loop.')
on conflict (code) do nothing;

create or replace function public.drop_trim_events_partitions_older_than(ttl_days int)
returns int
language plpgsql
security definer
set search_path = public
as $$
declare
  cutoff timestamptz;
  r record;
  dropped int := 0;
  y int;
  m int;
  range_to timestamptz;
begin
  if ttl_days is null or ttl_days < 1 then
    raise exception 'ttl_days must be >= 1';
  end if;
  cutoff := now() - make_interval(days => ttl_days);

  for r in
    select c.relname as part_name
    from pg_inherits i
    join pg_class parent on parent.oid = i.inhparent
    join pg_namespace n on n.oid = parent.relnamespace
    join pg_class c on c.oid = i.inhrelid
    where n.nspname = 'public'
      and parent.relname = 'trim_events'
      and c.relkind = 'r'
      and c.relname ~ '^trim_events_[0-9]{4}_[0-9]{2}$'
  loop
    y := substring(r.part_name from 'trim_events_([0-9]{4})_')::int;
    m := substring(r.part_name from 'trim_events_[0-9]{4}_([0-9]{2})$')::int;
    if m < 1 or m > 12 then
      continue;
    end if;
    range_to := make_timestamptz(y, m, 1, 0, 0, 0, 'UTC') + interval '1 month';
    if range_to <= cutoff then
      execute format('alter table public.trim_events detach partition public.%I', r.part_name);
      execute format('drop table public.%I', r.part_name);
      dropped := dropped + 1;
    end if;
  end loop;
  return dropped;
end;
$$;

comment on function public.drop_trim_events_partitions_older_than(int) is
  'DETACH+DROP monthly trim_events partitions whose exclusive upper bound is at or before now()-ttl_days.';
