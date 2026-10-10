-- Fail-closed display interval, seat default, and pagination max from billing_settings
-- (no hardcoded annual / seat qty 1 / maxLimit 50|100 in app code).

alter table public.billing_settings
  add column if not exists default_plan_interval text;

alter table public.billing_settings
  add column if not exists default_seat_quantity int;

alter table public.billing_settings
  add column if not exists max_page_size int;

-- Seed ops values once; operators may change after. No column DEFAULT invent thereafter.
update public.billing_settings
set
  default_plan_interval = coalesce(nullif(trim(default_plan_interval), ''), 'annual'),
  default_seat_quantity = coalesce(default_seat_quantity, 1),
  max_page_size = coalesce(max_page_size, 100)
where id = 'default';

alter table public.billing_settings
  alter column default_plan_interval set not null,
  alter column default_seat_quantity set not null,
  alter column max_page_size set not null;

alter table public.billing_settings
  drop constraint if exists billing_settings_default_plan_interval_check;
alter table public.billing_settings
  add constraint billing_settings_default_plan_interval_check
  check (default_plan_interval in ('monthly', 'annual'));

alter table public.billing_settings
  drop constraint if exists billing_settings_default_seat_quantity_check;
alter table public.billing_settings
  add constraint billing_settings_default_seat_quantity_check
  check (default_seat_quantity >= 1 and default_seat_quantity <= 10000);

alter table public.billing_settings
  drop constraint if exists billing_settings_max_page_size_check;
alter table public.billing_settings
  add constraint billing_settings_max_page_size_check
  check (max_page_size >= 1 and max_page_size <= 500);

comment on column public.billing_settings.default_plan_interval is
  'Plan modal default billing interval (monthly|annual). Frontend must not invent.';
comment on column public.billing_settings.default_seat_quantity is
  'Initial seat quantity for per-seat plans in the plan modal. No client invent of 1.';
comment on column public.billing_settings.max_page_size is
  'Max skip/limit page size for list APIs. Handlers must not invent 50/100.';
