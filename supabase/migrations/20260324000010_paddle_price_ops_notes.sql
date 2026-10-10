-- Ops template: bind Paddle Catalog price IDs into plan_catalog.
-- Pricing amounts stay in plan_catalog (price_*_cents) and billing_settings
-- (annual_discount_percent). Do not put dollar amounts in .env.
--
-- After creating prices in Paddle Billing (sandbox then live), run UPDATEs like:
--
--   update public.plan_catalog
--   set paddle_price_id_monthly = 'pri_xxx',
--       paddle_price_id_yearly  = 'pri_yyy',
--       updated_at = now()
--   where id = 'pro';
--
--   update public.plan_catalog
--   set paddle_price_id_monthly = 'pri_team_mo',
--       paddle_price_id_yearly  = 'pri_team_yr',
--       updated_at = now()
--   where id = 'team';
--
--   update public.plan_catalog
--   set paddle_price_id_topup = 'pri_topup',
--       updated_at = now()
--   where id = 'credits_500';  -- or your topup plan id
--
--   update public.billing_settings
--   set annual_discount_percent = 20,
--       apply_paddle_discount_on_annual = false,  -- prefer DB percent; set true only if using Paddle discount id
--       paddle_discount_id = null,
--       allow_downgrades = false,
--       updated_at = now()
--   where id = 'default';
--
-- This migration is intentionally a no-op so empty seeds stay empty until you
-- paste real pri_* values from your Paddle dashboard.

do $$ begin
  -- no-op placeholder for operator documentation
  null;
end $$;
