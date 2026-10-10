-- Catalog sync audit follow-ups: amount field chrome, create-plan chrome, pricing_bound ops copy.

insert into public.site_messages (code, body) values
  ('ADMIN_PLAN_AMOUNT_MONTHLY', 'Monthly amount (cents)'),
  ('ADMIN_PLAN_AMOUNT_YEARLY', 'Yearly amount (cents)'),
  ('ADMIN_PLAN_CREDITS', 'Credits / month'),
  ('ADMIN_PLAN_CREATE', 'Create plan'),
  ('ADMIN_PLAN_CREATE_PENDING', 'Creating…'),
  ('ADMIN_PLAN_ID', 'Plan id'),
  ('ADMIN_BILLING_PRICING_BOUND_HINT', 'Pricing bound is set automatically after Sync to Paddle succeeds for all public sellable plans.'),
  ('ADMIN_BILLING_ANNUAL_DISCOUNT_HINT', 'Saving annual discount % updates yearly cents on subscription plans (monthly×12×(100−%)/100) and re-syncs Paddle yearly prices.'),
  ('PLAN_CONFLICT', 'A plan with this id already exists.')
on conflict (code) do update set body = excluded.body;

comment on column public.billing_settings.annual_discount_percent is
  'Operator annual savings percent (0-90). Saving in admin rebakes plan_catalog.price_yearly_cents then syncs yearly pri_* to Paddle. Web badges use monthly vs yearly cents math.';

comment on column public.billing_settings.apply_paddle_discount_on_annual is
  'Deprecated for self-serve checkout. Annual charge uses yearly pri_* only; attaching a Paddle % discount would double-discount. Keep false.';
