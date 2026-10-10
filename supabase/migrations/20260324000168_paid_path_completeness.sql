-- Paid-path completeness: workspaces.updated_at (seat sync), chrome for credits/topup/seats/quota UX,
-- topup refund clawback, enterprise activate status.

alter table public.workspaces
  add column if not exists updated_at timestamptz not null default now();

comment on column public.workspaces.updated_at is
  'Updated when plan_tier / allocated_seats sync from Paddle or checkout.';

alter table public.billing_topup_ledger
  add column if not exists clawed_at timestamptz;

comment on column public.billing_topup_ledger.clawed_at is
  'Set when a refund/partial refund claws back credits_granted from purchased_topup_credits.';

alter table public.enterprise_inquiries
  drop constraint if exists enterprise_inquiries_status_check;

alter table public.enterprise_inquiries
  add constraint enterprise_inquiries_status_check
  check (status in ('new', 'contacted', 'closed', 'activated'));

insert into public.site_messages (code, body) values
  ('ADMIN_PLAN_CREDITS_DESC', 'Monthly cloud credits granted by this plan (or pack size for top-up plans).'),
  ('ADMIN_PLAN_PER_SEAT', 'Per-seat pricing'),
  ('ADMIN_PLAN_PER_SEAT_DESC', 'When on, checkout quantity multiplies credits and allocates workspace seats.'),
  ('DASHBOARD_METRIC_TOPUP', 'Top-up credits'),
  ('DASHBOARD_QUOTA_EXHAUSTED_TITLE', 'Cloud credits exhausted'),
  ('DASHBOARD_QUOTA_EXHAUSTED_BODY', 'Upgrade your plan or buy a top-up to keep using Trim cloud. Local Fast Mode still works on your machine.'),
  ('DASHBOARD_BUY_TOPUP_LABEL', 'Buy top-up'),
  ('WORKSPACE_META_SEATS_FMT', '%d/%d seats'),
  ('WORKSPACE_META_SEATS_UNIT', ' seats'),
  ('ADMIN_ENTERPRISE_ACTIVATE', 'Activate deal'),
  ('ADMIN_ENTERPRISE_ACTIVATE_DESC', 'Grant the offered seats and enterprise plan credits to the inquiring user. Does not charge Paddle.'),
  ('ADMIN_ENTERPRISE_ACTIVATE_PENDING', 'Activating...'),
  ('ADMIN_ENTERPRISE_ACTIVATED', 'Enterprise deal activated.'),
  ('ADMIN_ENTERPRISE_ACTIVATE_FAILED', 'Could not activate enterprise deal.'),
  ('ADMIN_ENTERPRISE_PLAN_MISSING', 'No active enterprise plan in the catalog.'),
  ('ADMIN_ENTERPRISE_USER_MISSING', 'Inquiry has no user to activate.'),
  ('ADMIN_ENTERPRISE_SEATS_REQUIRED', 'Set offered seats before activating.'),
  ('ADMIN_ENTERPRISE_STATUS_ACTIVATED', 'Activated')
on conflict (code) do update set body = excluded.body, updated_at = now();
