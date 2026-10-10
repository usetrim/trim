-- Popular badge dial on plan_catalog (operator-controlled, like unlimited).
-- Default false. Operators may mark any combination of plans (including all).
-- Seed Pro as popular (industry conversion tier); admin can change anytime.
-- Badge label stays in site_messages LANDING_PRICING_POPULAR_BADGE.
-- See README “Landing pricing cards” and apps/admin/README.md.

alter table public.plan_catalog
  add column if not exists popular boolean not null default false;

comment on column public.plan_catalog.popular is
  'When true, public pricing cards show the Popular badge for this plan. Multiple plans may be popular. Default false.';

update public.plan_catalog
set popular = true
where id = 'pro'
  and popular = false;

insert into public.site_messages (code, body) values
  ('ADMIN_PLAN_POPULAR', 'Popular plan'),
  (
    'ADMIN_PLAN_POPULAR_DESC',
    'When on, the public pricing card shows the Popular badge for this plan. You can enable it on one, several, or all plans.'
  ),
  ('ADMIN_PLAN_POPULAR_COL', 'Popular')
on conflict (code) do update set body = excluded.body;
