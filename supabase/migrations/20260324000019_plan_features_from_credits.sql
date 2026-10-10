-- Derive catalog feature bullets from credits_monthly so copy cannot drift
-- from the real free/pro/team credit pools. No hardcoded "50 cloud credits".

update public.plan_catalog
set features = jsonb_build_array(
  'Local AST proxy',
  credits_monthly::text || ' cloud credits / mo',
  'Community support'
),
updated_at = now()
where id = 'free';

update public.plan_catalog
set features = jsonb_build_array(
  'Everything in Free',
  credits_monthly::text || ' cloud credits / mo',
  'Usage dashboard',
  'Receipt history',
  'Email support'
),
updated_at = now()
where id = 'pro';

update public.plan_catalog
set features = jsonb_build_array(
  'Everything in Pro',
  'Pooled team credits (' || credits_monthly::text || ' / mo base)',
  'Admin dashboard',
  'Shared .trimrc rules',
  'Invoice / Net-30 via Paddle'
),
updated_at = now()
where id = 'team';
