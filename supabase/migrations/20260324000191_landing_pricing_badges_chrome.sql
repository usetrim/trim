-- Landing pricing: Popular badge label, annual-billing switch copy, save hint fmt.
-- Popular *which plans* is plan_catalog.popular (migration 193), not a chrome plan id.
-- See README “Landing pricing cards”.

insert into public.site_messages (code, body) values
  ('LANDING_PRICING_POPULAR_BADGE', 'Popular'),
  ('LANDING_PRICING_ANNUAL_BILLING', 'Annual billing'),
  (
    'LANDING_PRICING_ANNUAL_SAVE_HINT_FMT',
    'Save %d%% on yearly plans'
  )
on conflict (code) do update set body = excluded.body;
