-- Plan-level Unlimited metering flag (operator-controlled).
-- Default false: capability exists on every plan row; not enabled until admin checks it.
-- When true, AuthQuota skips credit debit. credits_monthly remains for display / quota init.

alter table public.plan_catalog
  add column if not exists unlimited boolean not null default false;

comment on column public.plan_catalog.unlimited is
  'When true, cloud metering skips debit for users/workspaces on this plan_tier. credits_monthly still stored for display and signup init. Default false.';

insert into public.site_messages (code, body) values
  ('ADMIN_PLAN_UNLIMITED', 'Unlimited metering'),
  (
    'ADMIN_PLAN_UNLIMITED_DESC',
    'When on, members on this plan are not debited cloud credits. credits_monthly stays for display and signup init. Default off - use as a growth dial, not forever on every plan.'
  ),
  ('ADMIN_PLAN_UNLIMITED_COL', 'Unlimited'),
  ('PLAN_FEATURE_UNLIMITED_METERING', 'Unlimited cloud metering'),
  ('DASHBOARD_QUOTA_UNLIMITED_LABEL', 'Unlimited')
on conflict (code) do update set body = excluded.body;
