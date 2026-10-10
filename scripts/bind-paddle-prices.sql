-- DEPRECATED: Prefer Admin → Plans → Save / Sync to Paddle.
-- The API creates Paddle products + prices from plan_catalog cents and writes pri_*/pro_* back.
-- This script remains only as an emergency manual override if the API cannot reach Paddle.
--
-- Pricing amounts stay in plan_catalog (price_*_cents) and billing_settings
-- (annual_discount_percent, default_currency). Do not put dollar amounts in .env.
--
-- Prefer: set amounts + currency in admin dashboard, then Sync to Paddle.
-- Emergency only:
-- 1. Create products + prices in Paddle (sandbox, then live).
-- 2. Copy each pri_* ID below.
-- 3. Ensure billing_settings.annual_discount_percent and default_currency are set.
-- 4. Run against your Supabase SQL editor or: psql "$DATABASE_URL" -f scripts/bind-paddle-prices.sql
--
-- REPLACE every pri_REPLACE_* placeholder before running.

begin;

update public.plan_catalog
set paddle_price_id_monthly = 'pri_REPLACE_PRO_MONTHLY',
    paddle_price_id_yearly  = 'pri_REPLACE_PRO_YEARLY',
    updated_at = now()
where id = 'pro';

update public.plan_catalog
set paddle_price_id_monthly = 'pri_REPLACE_TEAM_MONTHLY',
    paddle_price_id_yearly  = 'pri_REPLACE_TEAM_YEARLY',
    updated_at = now()
where id = 'team';

-- Optional credit top-up catalog row (id may differ in your seed).
update public.plan_catalog
set paddle_price_id_topup = 'pri_REPLACE_TOPUP_500',
    updated_at = now()
where id in ('credits_500', 'topup_500')
  and plan_kind = 'topup';

-- Fail closed: do not invent annual_discount_percent or default_currency here.
-- Set those in billing_settings before binding (or uncomment and set YOUR values):
--   update public.billing_settings
--   set annual_discount_percent = <YOUR_PERCENT>,
--       default_currency = '<YOUR_ISO4217>',
--       updated_at = now()
--   where id = 'default';

update public.billing_settings
set apply_paddle_discount_on_annual = false,
    paddle_discount_id = null,
    allow_downgrades = false,
    -- Flip fail-closed gate after you replace every pri_REPLACE_* above with live IDs.
    pricing_bound = true,
    updated_at = now()
where id = 'default'
  and annual_discount_percent is not null
  and annual_discount_percent between 1 and 90
  and default_currency is not null
  and length(btrim(default_currency)) = 3;

-- Abort if operator forgot to set discount / currency (no invent).
do $$
begin
  if not exists (
    select 1 from public.billing_settings
    where id = 'default'
      and annual_discount_percent between 1 and 90
      and default_currency is not null
      and length(btrim(default_currency)) = 3
      and pricing_bound = true
  ) then
    raise exception 'bind-paddle-prices: set billing_settings.annual_discount_percent (1-90) and default_currency (ISO 4217) before pricing_bound=true; script does not invent 20 or USD';
  end if;
end $$;

-- Derive yearly display cents from monthly × (100 - annual_discount_percent) / 100.
-- Uses the live DB discount percent (not a hardcoded 20).
-- Override any row after this if your Paddle yearly price is not exactly that math.
update public.plan_catalog pc
set price_yearly_cents = round(
      pc.price_monthly_cents::numeric
      * 12
      * (100 - bs.annual_discount_percent)
      / 100
    )::int,
    updated_at = now()
from public.billing_settings bs
where bs.id = 'default'
  and pc.is_active = true
  and pc.price_monthly_cents is not null
  and pc.price_monthly_cents > 0
  and bs.annual_discount_percent between 1 and 90
  and pc.plan_kind = 'subscription'
  and lower(pc.id) not in (
    select lower(btrim(body)) from public.site_messages where code = 'DEFAULT_PLAN_TIER'
  );

select id,
       paddle_product_id,
       paddle_price_id_monthly,
       paddle_price_id_yearly,
       paddle_price_id_topup,
       price_monthly_cents,
       price_yearly_cents
from public.plan_catalog
where is_active = true
order by sort_order;

select annual_discount_percent, default_currency, pricing_bound, allow_downgrades
from public.billing_settings
where id = 'default';

commit;
