-- deep_target_token: no invent default on profiles.
-- Ops sets billing_settings.default_deep_target_token; signup copies it into profiles.

alter table public.billing_settings
  add column if not exists default_deep_target_token int;

update public.billing_settings
set default_deep_target_token = 300
where id = 'default' and default_deep_target_token is null;

alter table public.billing_settings
  alter column default_deep_target_token set not null;

do $$
begin
  if not exists (
    select 1 from pg_constraint where conname = 'billing_settings_default_deep_target_token_check'
  ) then
    alter table public.billing_settings
      add constraint billing_settings_default_deep_target_token_check
      check (default_deep_target_token > 0 and default_deep_target_token <= 100000);
  end if;
end $$;

comment on column public.billing_settings.default_deep_target_token is
  'Seeded into profiles.deep_target_token on signup. Change in DB only; no app invent.';

-- Drop invent column default; existing rows keep their values.
alter table public.profiles
  alter column deep_target_token drop default;

alter table public.profiles
  alter column compression_tier drop default;

alter table public.profiles
  alter column deep_engine drop default;

create or replace function public.handle_new_user()
returns trigger
language plpgsql
security definer
set search_path = public
as $$
declare
  provider text;
  allowed text[];
  free_credits int;
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

  select credits_monthly into free_credits
  from public.plan_catalog
  where id = 'free' and is_active = true;

  if free_credits is null then
    raise exception 'plan_catalog free plan is missing or inactive'
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
    'free',
    free_credits,
    0
  )
  on conflict (user_id) do nothing;

  return new;
end;
$$;

comment on function public.handle_new_user() is
  'Profile + free quota from plan_catalog; deep_target_token from billing_settings; providers from auth_settings.';
