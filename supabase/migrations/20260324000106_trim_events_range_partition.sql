-- Native monthly RANGE partitions on trim_events (pre-launch cutover).
-- PK includes created_at (Postgres partition-key rule). No DEFAULT partition:
-- ensure_trim_events_month_partitions must run so inserts always have a home.
-- Retention: DETACH+DROP whole months older than TTL, then DELETE leftovers.
-- Fail-closed: never invent months_ahead / ensure interval when settings row is missing.

alter table public.admin_retention_settings
  add column if not exists trim_events_partition_months_ahead int not null default 3
  check (trim_events_partition_months_ahead >= 1 and trim_events_partition_months_ahead <= 36);

alter table public.admin_retention_settings
  add column if not exists trim_events_partition_ensure_sec int not null default 86400
  check (trim_events_partition_ensure_sec >= 3600 and trim_events_partition_ensure_sec <= 604800);

comment on column public.admin_retention_settings.trim_events_partition_months_ahead is
  'How many future UTC months of trim_events partitions to pre-create (fail-closed; no invent).';

comment on column public.admin_retention_settings.trim_events_partition_ensure_sec is
  'API background interval (seconds) to re-ensure future trim_events month partitions (fail-closed; no invent).';

insert into public.site_messages (code, body) values
  ('ADMIN_COMPLIANCE_EVENTS_PARTITION_AHEAD', 'Trim events future partitions (months ahead)'),
  ('ADMIN_COMPLIANCE_EVENTS_PARTITION_ENSURE_SEC', 'Trim events partition ensure interval (seconds)'),
  ('ADMIN_EVENTS_PARTITION_AHEAD_MISSING', 'Set admin_retention_settings.trim_events_partition_months_ahead before event inserts.'),
  ('ADMIN_EVENTS_PARTITION_ENSURE_SEC_MISSING', 'Set admin_retention_settings.trim_events_partition_ensure_sec before partition ensure loop.'),
  ('ADMIN_EVENTS_PARTITION_ENSURE_FAILED', 'Failed to ensure trim_events month partitions.')
on conflict (code) do nothing;

create or replace function public.ensure_trim_events_month_partitions(
  months_ahead int,
  months_back int default 1
) returns int
language plpgsql
security definer
set search_path = public
as $$
declare
  i int;
  start_ts timestamptz;
  end_ts timestamptz;
  part_name text;
  created int := 0;
  base_month timestamp;
begin
  if months_ahead is null or months_ahead < 1 or months_ahead > 36 then
    raise exception 'months_ahead must be 1-36';
  end if;
  if months_back is null or months_back < 0 or months_back > 120 then
    raise exception 'months_back must be 0-120';
  end if;
  if not exists (
    select 1
    from pg_partitioned_table pt
    join pg_class c on c.oid = pt.partrelid
    join pg_namespace n on n.oid = c.relnamespace
    where n.nspname = 'public' and c.relname = 'trim_events'
  ) then
    raise exception 'public.trim_events is not a partitioned table';
  end if;

  base_month := date_trunc('month', timezone('utc', now()));
  for i in -months_back .. months_ahead loop
    start_ts := (base_month + make_interval(months => i)) at time zone 'utc';
    end_ts := (base_month + make_interval(months => i + 1)) at time zone 'utc';
    part_name := 'trim_events_' || to_char(base_month + make_interval(months => i), 'YYYY_MM');
    if to_regclass('public.' || part_name) is null then
      execute format(
        'create table public.%I partition of public.trim_events for values from (%L) to (%L)',
        part_name,
        start_ts,
        end_ts
      );
      created := created + 1;
    end if;
  end loop;
  return created;
end;
$$;

comment on function public.ensure_trim_events_month_partitions(int, int) is
  'Create missing UTC monthly partitions for trim_events from -months_back through +months_ahead.';

-- Drop by partition name trim_events_YYYY_MM (no fragile pg_get_expr regexp).
-- DETACH then DROP so the parent stays consistent under concurrent readers.
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
    -- Exclusive upper bound of the month partition (UTC).
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

do $$
declare
  ahead int;
begin
  if exists (
    select 1
    from pg_partitioned_table pt
    join pg_class c on c.oid = pt.partrelid
    join pg_namespace n on n.oid = c.relnamespace
    where n.nspname = 'public' and c.relname = 'trim_events'
  ) then
    raise notice 'trim_events already partitioned; skipping cutover';
    select trim_events_partition_months_ahead into ahead
    from public.admin_retention_settings where id = 'default';
    if ahead is null or ahead < 1 or ahead > 36 then
      raise exception 'admin_retention_settings.trim_events_partition_months_ahead missing or invalid';
    end if;
    perform public.ensure_trim_events_month_partitions(ahead, 24);
    return;
  end if;

  if to_regclass('public.trim_events') is null then
    raise exception 'public.trim_events missing';
  end if;

  select trim_events_partition_months_ahead into ahead
  from public.admin_retention_settings where id = 'default';
  if ahead is null or ahead < 1 or ahead > 36 then
    raise exception 'admin_retention_settings.trim_events_partition_months_ahead missing or invalid';
  end if;

  drop policy if exists "events_select_own" on public.trim_events;

  alter table public.trim_events rename to trim_events_legacy;

  alter index if exists public.idx_trim_events_user_created rename to idx_trim_events_legacy_user_created;
  alter index if exists public.idx_trim_events_created_at_brin rename to idx_trim_events_legacy_created_at_brin;
  alter index if exists public.idx_trim_events_error_code_created rename to idx_trim_events_legacy_error_code_created;

  create table public.trim_events (
    id uuid not null default gen_random_uuid(),
    user_id uuid references public.profiles(id) on delete set null,
    request_id text,
    model text,
    tokens_before int not null default 0,
    tokens_after int not null default 0,
    latency_ms int not null default 0,
    mode text not null default 'proxy',
    status text not null default 'success',
    created_at timestamptz not null default now(),
    tab_suggestions_shown int not null default 0,
    tab_suggestions_accepted int not null default 0,
    ai_lines_added int not null default 0,
    ai_lines_deleted int not null default 0,
    error_code text,
    primary key (id, created_at)
  ) partition by range (created_at);

  comment on table public.trim_events is
    'Proxy/IDE telemetry events; monthly RANGE partitions on created_at (UTC).';

  perform public.ensure_trim_events_month_partitions(ahead, 24);

  insert into public.trim_events (
    id, user_id, request_id, model, tokens_before, tokens_after, latency_ms,
    mode, status, created_at,
    tab_suggestions_shown, tab_suggestions_accepted, ai_lines_added, ai_lines_deleted,
    error_code
  )
  select
    id, user_id, request_id, model, tokens_before, tokens_after, latency_ms,
    mode, status, created_at,
    tab_suggestions_shown, tab_suggestions_accepted, ai_lines_added, ai_lines_deleted,
    error_code
  from public.trim_events_legacy;

  drop table public.trim_events_legacy;

  create index if not exists idx_trim_events_user_created
    on public.trim_events (user_id, created_at desc);

  create index if not exists idx_trim_events_created_at_brin
    on public.trim_events using brin (created_at);

  create index if not exists idx_trim_events_error_code_created
    on public.trim_events (error_code, created_at desc)
    where error_code is not null and error_code <> '';

  alter table public.trim_events enable row level security;

  create policy "events_select_own" on public.trim_events
    for select using (auth.uid() = user_id);
end;
$$;
