-- Drop invent column defaults so new rows require explicit operator values.
-- Existing row values are kept; only DEFAULT clauses are removed.

-- billing_settings: no invent 20% / USD / page_size / chart_top_n on new inserts
alter table public.billing_settings
  alter column annual_discount_percent drop default;

alter table public.billing_settings
  alter column default_currency drop default;

alter table public.billing_settings
  alter column default_page_size drop default;

alter table public.billing_settings
  alter column chart_top_n drop default;

-- plan_catalog: no invent USD on new rows
alter table public.plan_catalog
  alter column currency_code drop default;

-- subscriptions: no invent plan_tier = pro on new rows
alter table public.subscriptions
  alter column plan_tier drop default;

-- Clear invent display cents on paid catalog rows that still lack Paddle price IDs.
-- Free tier 0/0 stays. Operators set cents and/or bind pri_* via scripts/bind-paddle-prices.sql.
update public.plan_catalog
set
  price_monthly_cents = null,
  price_yearly_cents = null,
  updated_at = now()
where id in ('pro', 'team', 'topup_500')
  and nullif(btrim(coalesce(paddle_price_id_monthly, '')), '') is null
  and nullif(btrim(coalesce(paddle_price_id_yearly, '')), '') is null
  and nullif(btrim(coalesce(paddle_price_id_topup, '')), '') is null;

comment on column public.billing_settings.annual_discount_percent is
  'Operator-set annual savings percent for display toggle. No column default; seed row must set explicitly.';

comment on column public.billing_settings.default_currency is
  'Operator-set ISO currency for catalog display. No column default.';

comment on column public.billing_settings.default_page_size is
  'Operator-set skip/limit page size. No column default.';

comment on column public.billing_settings.chart_top_n is
  'Operator-set chart series top-N. No column default.';
