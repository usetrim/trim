-- Migration 42: GitHub/GitLab OAuth scopes from site_messages; signup prefs
-- no longer invent compression_tier='fast' or deep_engine='v2'.

insert into public.site_messages (code, body) values
  ('AUTH_GITHUB_OAUTH_SCOPES', 'user:email'),
  ('AUTH_GITHUB_OAUTH_SCOPES_MISSING', 'GitHub OAuth scopes are not configured. Set AUTH_GITHUB_OAUTH_SCOPES in site_messages.'),
  ('AUTH_GITLAB_OAUTH_SCOPES', 'read_user'),
  ('AUTH_GITLAB_OAUTH_SCOPES_MISSING', 'GitLab OAuth scopes are not configured. Set AUTH_GITLAB_OAUTH_SCOPES in site_messages.'),
  ('DEFAULT_COMPRESSION_TIER', 'fast'),
  ('DEFAULT_COMPRESSION_TIER_MISSING', 'DEFAULT_COMPRESSION_TIER is not configured in site_messages'),
  ('DEFAULT_DEEP_ENGINE', 'v2'),
  ('DEFAULT_DEEP_ENGINE_MISSING', 'DEFAULT_DEEP_ENGINE is not configured in site_messages')
on conflict (code) do nothing;

comment on column public.site_messages.body is
  'Operator-owned chrome. AUTH_GITHUB_OAUTH_SCOPES / AUTH_GITLAB_OAUTH_SCOPES drive login scopes.';

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
  default_compression text;
  default_engine text;
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

  select lower(btrim(m.body)) into default_compression
  from public.site_messages m
  where m.code = 'DEFAULT_COMPRESSION_TIER';

  if default_compression is null or default_compression not in ('fast', 'deep') then
    raise exception 'site_messages DEFAULT_COMPRESSION_TIER is missing or invalid'
      using errcode = 'P0001';
  end if;

  select lower(btrim(m.body)) into default_engine
  from public.site_messages m
  where m.code = 'DEFAULT_DEEP_ENGINE';

  if default_engine is null or default_engine not in ('v1', 'long', 'v2') then
    raise exception 'site_messages DEFAULT_DEEP_ENGINE is missing or invalid'
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
    default_compression,
    default_engine,
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
  'Profile + quota from site_messages DEFAULT_PLAN_TIER / DEFAULT_COMPRESSION_TIER / DEFAULT_DEEP_ENGINE + plan_catalog; deep_target from billing_settings; providers from auth_settings.';
