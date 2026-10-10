-- Admin → Paddle catalog sync: store product IDs; operators set amounts in admin only.
-- Paddle products/prices are created via API from plan_catalog cents (not pasted from vendor UI).

alter table public.plan_catalog
  add column if not exists paddle_product_id text;

comment on column public.plan_catalog.paddle_product_id is
  'Paddle product id pro_… created/updated by admin catalog sync from this plan row.';

comment on column public.plan_catalog.paddle_price_id_monthly is
  'Paddle price id pri_… for monthly billing. Written by admin → Paddle catalog sync from price_monthly_cents.';

comment on column public.plan_catalog.paddle_price_id_yearly is
  'Paddle price id pri_… for annual billing. Written by admin → Paddle catalog sync from price_yearly_cents.';

comment on column public.plan_catalog.paddle_price_id_topup is
  'Paddle one-time price id pri_… for top-up packs. Written by admin → Paddle catalog sync.';

comment on column public.billing_settings.pricing_bound is
  'True when admin catalog sync has bound live Paddle pri_* for all public sellable plans and currency is valid. Self-serve checkout fails closed while false.';

insert into public.site_messages (code, body) values
  ('ADMIN_PLAN_SYNC_PADDLE', 'Sync to Paddle'),
  ('ADMIN_PLAN_SYNC_PENDING', 'Syncing…'),
  ('ADMIN_PLAN_PADDLE_IDS', 'Paddle IDs (synced)'),
  ('ADMIN_PLAN_PRODUCT_ID', 'Product id'),
  ('ADMIN_PADDLE_CATALOG_SYNC_FAILED', 'Could not sync plan catalog to Paddle. Check API key permissions (product.write, price.write) and currency.'),
  ('ADMIN_PADDLE_CURRENCY_INVALID', 'Set a real ISO-4217 default currency in Billing settings before syncing to Paddle.'),
  ('ADMIN_PRICING_UNBOUND', 'Pricing is not bound. Save plan amounts, then Sync to Paddle. Self-serve checkout stays fail-closed until pricing_bound is true.'),
  ('BILLING_PRICING_UNBOUND', 'Billing pricing is not bound yet. An operator must set plan amounts and currency in the admin dashboard and sync the catalog to Paddle.'),
  ('PRICE_NOT_CONFIGURED', 'This plan is not synced to Paddle yet. An operator must sync the plan catalog from the admin dashboard.'),
  ('PRICE_NOT_CONFIGURED_REASON', 'This plan is not synced to Paddle yet. An operator must sync the plan catalog from the admin dashboard.'),
  ('ADMIN_PLAN_PRICE_MONTHLY', 'Monthly Paddle price (synced)'),
  ('ADMIN_PLAN_PRICE_YEARLY', 'Yearly Paddle price (synced)'),
  ('ADMIN_PLAN_PRICE_TOPUP', 'Top-up Paddle price (synced)')
on conflict (code) do update set body = excluded.body;
