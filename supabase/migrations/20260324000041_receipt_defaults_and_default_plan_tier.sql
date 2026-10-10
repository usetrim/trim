-- Migration 41: remove receipt invent money defaults; signup reads DEFAULT_PLAN_TIER
-- from site_messages (no hardcoded 'free' plan_tier string).

-- Receipt money columns: require explicit webhook/API values (no invent 0 default).
alter table public.billing_receipts
  alter column subtotal_cents drop default;

alter table public.billing_receipts
  alter column tax_cents drop default;

alter table public.billing_receipts
  alter column total_cents drop default;

alter table public.billing_receipts
  alter column tax_rate_bps drop default;

alter table public.billing_receipt_line_items
  alter column unit_amount_cents drop default;

alter table public.billing_receipt_line_items
  alter column amount_cents drop default;

comment on column public.billing_receipts.subtotal_cents is
  'Paddle/sync-owned subtotal. No column default; inserts must set explicitly.';
comment on column public.billing_receipts.tax_cents is
  'Paddle/sync-owned tax. No column default; inserts must set explicitly.';
comment on column public.billing_receipts.total_cents is
  'Paddle/sync-owned total. No column default; inserts must set explicitly.';

-- Ensure DEFAULT_PLAN_TIER chrome exists (operator may override body later).
insert into public.site_messages (code, body) values
  ('DEFAULT_PLAN_TIER', 'free'),
  ('DEFAULT_PLAN_TIER_MISSING', 'DEFAULT_PLAN_TIER is not configured in site_messages'),
  ('ACCOUNT_DELETE_FAILED', 'Account deletion failed. Try again or contact support.')
on conflict (code) do nothing;

-- handle_new_user: plan_tier + credits from site_messages DEFAULT_PLAN_TIER + plan_catalog.
create or replace function public.handle_new_user()
returns trigger
language plpgsql
security definer
set search_path = public
as $$
declare
  provider text;
  allowed text[];
  default_tier text;
  tier_credits int;
  deep_target int;
begin
  provider := lower(coalesce(new.raw_app_meta_data->>'provider', ''));
  if provider = '' then
    provider := lower(coalesce(new.raw_app_meta_data->'providers'->>0, ''));
  end if;

  select s.allowed_providers into allowed
  from public.auth_settings s
  where s.id = 'default';

  if allowed is null or cardinality(allowed) = 0 then
    raise exception 'auth_settings.allowed_providers is empty; refuse signup'
      using errcode = 'P0001';
  end if;

  if provider = '' or not (provider = any (allowed)) then
    raise exception 'auth provider % is not allowed; enabled: %', provider, array_to_string(allowed, ', ')
      using errcode = 'P0001';
  end if;

  select lower(btrim(m.body)) into default_tier
  from public.site_messages m
  where m.code = 'DEFAULT_PLAN_TIER';

  if default_tier is null or default_tier = '' then
    raise exception 'site_messages DEFAULT_PLAN_TIER is missing or empty'
      using errcode = 'P0001';
  end if;

  select credits_monthly into tier_credits
  from public.plan_catalog
  where id = default_tier and is_active = true;

  if tier_credits is null then
    raise exception 'plan_catalog row for DEFAULT_PLAN_TIER=% is missing or inactive', default_tier
      using errcode = 'P0001';
  end if;

  select b.default_deep_target_token into deep_target
  from public.billing_settings b
  where b.id = 'default';

  if deep_target is null then
    raise exception 'billing_settings.default_deep_target_token is missing'
      using errcode = 'P0001';
  end if;

  insert into public.profiles (
    id, email, full_name, avatar_url, auth_provider,
    compression_tier, deep_engine, deep_target_token
  )
  values (
    new.id,
    new.email,
    coalesce(
      new.raw_user_meta_data->>'full_name',
      new.raw_user_meta_data->>'name',
      new.raw_user_meta_data->>'user_name'
    ),
    coalesce(
      new.raw_user_meta_data->>'avatar_url',
      new.raw_user_meta_data->>'picture'
    ),
    provider,
    'fast',
    'v2',
    deep_target
  )
  on conflict (id) do update set
    email = excluded.email,
    full_name = coalesce(excluded.full_name, public.profiles.full_name),
    avatar_url = coalesce(excluded.avatar_url, public.profiles.avatar_url),
    auth_provider = excluded.auth_provider,
    updated_at = now();

  insert into public.user_quotas (
    user_id,
    plan_tier,
    monthly_credit_limit,
    monthly_credit_used
  )
  values (
    new.id,
    default_tier,
    tier_credits,
    0
  )
  on conflict (user_id) do nothing;

  return new;
end;
$$;

comment on function public.handle_new_user() is
  'Profile + quota from site_messages DEFAULT_PLAN_TIER + plan_catalog; deep_target from billing_settings; providers from auth_settings.';

-- billing_settings invent seed (migration 03): operators must set annual_discount_percent
-- and default_currency to production values. Column defaults already dropped in migration 28.
-- Do not null existing rows here (LoadPolicy scans into int/string). Document in README.
comment on column public.billing_settings.annual_discount_percent is
  'Operator-set annual savings percent for display toggle. Historical seed may be 20 until overwritten; no column default on new inserts.';
comment on column public.billing_settings.default_currency is
  'Operator-set ISO currency. Historical seed may be USD until overwritten; no column default on new inserts.';
