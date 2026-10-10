-- Seed chrome for fail-closed pricing bind (migration 48 + Go LoadPolicy / loadSettings).

insert into public.site_messages (code, body) values
  (
    'BILLING_PRICING_UNBOUND',
    'Billing pricing is not bound yet. Run scripts/bind-paddle-prices.sql with live Paddle price IDs, annual_discount_percent, and default_currency.'
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
