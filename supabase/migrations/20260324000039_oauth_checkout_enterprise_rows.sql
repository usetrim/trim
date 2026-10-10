-- Google OAuth query params, non-seat checkout qty, enterprise textarea rows (no invent).
alter table public.billing_settings
  add column if not exists default_checkout_quantity int;

alter table public.billing_settings
  add column if not exists enterprise_message_rows int;

update public.billing_settings
set
  default_checkout_quantity = coalesce(default_checkout_quantity, 1),
  enterprise_message_rows = coalesce(enterprise_message_rows, 4)
where id = 'default';

alter table public.billing_settings
  alter column default_checkout_quantity set not null,
  alter column enterprise_message_rows set not null;

alter table public.billing_settings
  drop constraint if exists billing_settings_default_checkout_quantity_check;
alter table public.billing_settings
  add constraint billing_settings_default_checkout_quantity_check
  check (default_checkout_quantity >= 1 and default_checkout_quantity <= 10000);

alter table public.billing_settings
  drop constraint if exists billing_settings_enterprise_message_rows_check;
alter table public.billing_settings
  add constraint billing_settings_enterprise_message_rows_check
  check (enterprise_message_rows >= 2 and enterprise_message_rows <= 40);

comment on column public.billing_settings.default_checkout_quantity is
  'Paddle quantity for non-seat catalog checkouts (usually 1).';
comment on column public.billing_settings.enterprise_message_rows is
  'Visible rows for the enterprise inquiry textarea in the plan modal.';

insert into public.site_messages (code, body) values
  ('AUTH_GOOGLE_OAUTH_ACCESS_TYPE', 'offline'),
  ('AUTH_GOOGLE_OAUTH_PROMPT', 'consent'),
  ('AUTH_GOOGLE_OAUTH_PARAMS_MISSING', 'Google OAuth query params are not configured. Set AUTH_GOOGLE_OAUTH_ACCESS_TYPE and AUTH_GOOGLE_OAUTH_PROMPT in site_messages.')
on conflict (code) do update set body = excluded.body;
