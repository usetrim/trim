-- Neutralize historical seed invent prices from migration 02.
-- Checkout and plan display fail closed until operators set real cents + pri_* in plan_catalog.
-- Only touches rows that still have no Paddle price binding (safe for already-configured installs).

update public.plan_catalog
set
  price_monthly_cents = null,
  price_yearly_cents = null,
  updated_at = now()
where id in ('pro', 'team')
  and (paddle_price_id_monthly is null or btrim(paddle_price_id_monthly) = '')
  and (paddle_price_id_yearly is null or btrim(paddle_price_id_yearly) = '');

comment on column public.plan_catalog.price_monthly_cents is
  'Backend-owned monthly price in cents. Null until operator sets; no client invent.';
comment on column public.plan_catalog.price_yearly_cents is
  'Backend-owned yearly price in cents (annual discount applied in catalog). Null until operator sets; no client invent.';
