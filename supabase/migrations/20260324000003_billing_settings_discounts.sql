-- Billing settings and annual discount (all pricing driven from DB, not env)

create table if not exists public.billing_settings (
  id text primary key default 'default',
  -- Display / UX: annual toggle shows this percent off vs monthly*12
  annual_discount_percent int not null default 20
    check (annual_discount_percent >= 0 and annual_discount_percent <= 100),
  -- When true, checkout for interval=annual may attach paddle_discount_id
  -- Prefer separate yearly price IDs on plan_catalog; discount_id is optional overlay
  apply_paddle_discount_on_annual boolean not null default false,
  paddle_discount_id text,
  paddle_discount_code text,
  default_currency text not null default 'USD',
  updated_at timestamptz not null default now()
);

insert into public.billing_settings (
  id, annual_discount_percent, apply_paddle_discount_on_annual, default_currency
) values (
  'default', 20, false, 'USD'
) on conflict (id) do nothing;

-- Extend plan_catalog for top-up SKUs and richer paddle mapping
alter table public.plan_catalog
  add column if not exists plan_kind text not null default 'subscription'
    check (plan_kind in ('subscription', 'topup', 'enterprise'));

alter table public.plan_catalog
  add column if not exists paddle_price_id_topup text;

alter table public.plan_catalog
  add column if not exists currency_code text not null default 'USD';

alter table public.plan_catalog
  add column if not exists is_active boolean not null default true;

-- Team yearly option (20% off vs 12*monthly display)
update public.plan_catalog
set price_yearly_cents = 28800
where id = 'team' and price_yearly_cents is null;

-- Ensure pro yearly is 20% off of 12*20 = 240 -> 192
update public.plan_catalog
set price_yearly_cents = 19200
where id = 'pro';

insert into public.plan_catalog (
  id, display_name, description, price_monthly_cents, price_yearly_cents,
  credits_monthly, per_seat, plan_kind, features, is_public, sort_order, currency_code
) values (
  'topup_500',
  'Credit Pack 500',
  'One-time pack of 500 fast credits.',
  1000,
  null,
  500,
  false,
  'topup',
  '["500 fast credits","Never expires","Stacks on plan quota"]'::jsonb,
  true,
  10,
  'USD'
) on conflict (id) do nothing;

alter table public.billing_settings enable row level security;

create policy "billing_settings_public_read" on public.billing_settings
  for select using (true);

comment on table public.billing_settings is
  'Global billing UX and optional Paddle discount entity. Plan amounts and price IDs live in plan_catalog.';

comment on column public.billing_settings.paddle_discount_id is
  'Optional dsc_... from Paddle. Used only when apply_paddle_discount_on_annual is true.';

comment on column public.plan_catalog.paddle_price_id_monthly is
  'Paddle price ID pri_... for monthly billing. Set in Supabase after creating prices in Paddle.';

comment on column public.plan_catalog.paddle_price_id_yearly is
  'Paddle price ID pri_... for annual billing (create yearly price already at discount in Paddle).';
