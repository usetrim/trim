-- Popular which-plan is plan_catalog.popular (migration 193), not a site_messages plan id.
-- Drop obsolete LANDING_PRICING_POPULAR_PLAN_ID chrome seeded in 191.
-- Badge label remains LANDING_PRICING_POPULAR_BADGE.
-- See README “Landing pricing cards”.

delete from public.site_messages
where code = 'LANDING_PRICING_POPULAR_PLAN_ID';
