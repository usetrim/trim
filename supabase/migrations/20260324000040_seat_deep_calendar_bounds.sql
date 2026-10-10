-- Backend-driven UI/API bounds (no invent min=1 / months=2 / max=100000).
alter table public.billing_settings
  add column if not exists min_seat_quantity int;

alter table public.billing_settings
  add column if not exists date_range_months int;

alter table public.billing_settings
  add column if not exists deep_target_token_min int;

alter table public.billing_settings
  add column if not exists deep_target_token_max int;

update public.billing_settings
set
  min_seat_quantity = coalesce(min_seat_quantity, 1),
  date_range_months = coalesce(date_range_months, 2),
  deep_target_token_min = coalesce(deep_target_token_min, 1),
  deep_target_token_max = coalesce(deep_target_token_max, 100000)
where id = 'default';

alter table public.billing_settings
  alter column min_seat_quantity set not null,
  alter column date_range_months set not null,
  alter column deep_target_token_min set not null,
  alter column deep_target_token_max set not null;

alter table public.billing_settings
  drop constraint if exists billing_settings_min_seat_quantity_check;
alter table public.billing_settings
  add constraint billing_settings_min_seat_quantity_check
  check (min_seat_quantity >= 1 and min_seat_quantity <= 10000);

alter table public.billing_settings
  drop constraint if exists billing_settings_date_range_months_check;
alter table public.billing_settings
  add constraint billing_settings_date_range_months_check
  check (date_range_months >= 1 and date_range_months <= 12);

alter table public.billing_settings
  drop constraint if exists billing_settings_deep_target_token_bounds_check;
alter table public.billing_settings
  add constraint billing_settings_deep_target_token_bounds_check
  check (
    deep_target_token_min >= 1
    and deep_target_token_max >= deep_target_token_min
    and deep_target_token_max <= 1000000
  );

comment on column public.billing_settings.min_seat_quantity is
  'Floor for seat quantity inputs (plan modal and enterprise inquiry).';
comment on column public.billing_settings.date_range_months is
  'Number of months shown in dashboard event date-range calendar.';
comment on column public.billing_settings.deep_target_token_min is
  'Minimum deep_target_token for preferences and CLI deep mode.';
comment on column public.billing_settings.deep_target_token_max is
  'Maximum deep_target_token for preferences and CLI deep mode.';

insert into public.site_messages (code, body) values
  ('CLI_STATS_URL_FMT', 'http://127.0.0.1:{port}/v1/stats'),
  ('CLI_STATS_URL_MISSING', 'CLI_STATS_URL_FMT must include {port} in site_messages'),
  ('LOCAL_STATS_URL_FMT', 'http://127.0.0.1:{port}/v1/stats'),
  ('BILLING_SETTINGS_UNAVAILABLE', 'Billing settings are unavailable'),
  ('CONFIG_ENV_NEXT_PUBLIC_API_URL_MISSING', 'NEXT_PUBLIC_API_URL is not set')
on conflict (code) do update set body = excluded.body;
