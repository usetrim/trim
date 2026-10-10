-- Fail closed on invent billing seed (migration 03: annual_discount_percent=20, default_currency=USD).
-- Operators must run scripts/bind-paddle-prices.sql (or equivalent) which sets pricing_bound=true
-- together with real annual_discount_percent, default_currency, plan cents, and pri_* IDs.
-- Already-bound installs (any live paddle_price_id_* on plan_catalog) are marked bound.

alter table public.billing_settings
  add column if not exists pricing_bound boolean not null default false;

comment on column public.billing_settings.pricing_bound is
  'True only after operator binds catalog prices and billing_settings via bind-paddle-prices.sql (or equivalent). Plans API fails closed while false.';

-- Mark installs that already have at least one real Paddle price ID as bound.
update public.billing_settings bs
set pricing_bound = true,
    updated_at = now()
where bs.id = 'default'
  and exists (
    select 1
    from public.plan_catalog pc
    where nullif(btrim(coalesce(pc.paddle_price_id_monthly, '')), '') is not null
       or nullif(btrim(coalesce(pc.paddle_price_id_yearly, '')), '') is not null
       or nullif(btrim(coalesce(pc.paddle_price_id_topup, '')), '') is not null
  );

-- Fresh / unbound installs: clear invent 20% and USD so operators must set real values.
-- Keep numeric/currency columns valid for Scan (not null); pricing_bound gates the API.
update public.billing_settings
set annual_discount_percent = 0,
    default_currency = 'XXX',
    pricing_bound = false,
    updated_at = now()
where id = 'default'
  and pricing_bound = false
  and annual_discount_percent = 20
  and upper(btrim(default_currency)) = 'USD';

comment on column public.billing_settings.annual_discount_percent is
  'Operator-set annual savings percent (0-100). Historical invent seed cleared when pricing_bound=false; set via bind-paddle-prices.sql.';

comment on column public.billing_settings.default_currency is
  'Operator-set ISO-4217 currency. Unbound invent USD replaced with XXX until bind; plans API requires pricing_bound=true.';
