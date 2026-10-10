-- Linked OAuth identities (Google / GitHub / GitLab) from auth.identities.
-- Settings connect/disconnect chrome is backend-owned. No client invent.

create or replace function public.list_auth_identities(target_user_id uuid)
returns table (
  identity_id text,
  provider text,
  email text,
  last_sign_in_at text
)
language sql
security definer
set search_path = auth, public
as $$
  select
    i.id::text,
    lower(i.provider),
    coalesce(i.identity_data->>'email', ''),
    case
      when i.last_sign_in_at is null then null
      else i.last_sign_in_at::text
    end
  from auth.identities i
  where i.user_id = target_user_id
  order by i.last_sign_in_at desc nulls last, i.created_at desc;
$$;

revoke all on function public.list_auth_identities(uuid) from public;
grant execute on function public.list_auth_identities(uuid) to postgres, service_role;

comment on function public.list_auth_identities(uuid) is
  'Returns OAuth identities for one user from auth.identities. Called by the Go API only.';

insert into public.site_messages (code, body) values
  ('AUTH_OAUTH_LINK_INTENT', 'link'),
  ('LOGIN_OAUTH_LINK_FAILED', 'Could not connect that sign-in provider. Try again.'),
  ('ACCOUNT_IDENTITIES_UNAVAILABLE', 'Signed-in account identities could not be loaded from the database.'),
  ('PREFERENCES_LINK_HINT', 'Connect another provider so you can sign in with it later.'),
  ('PREFERENCES_UNLINK_CONFIRM', 'Disconnect this sign-in provider from your account?'),
  ('PREFERENCES_UNLINK_LAST_BLOCKED', 'Keep at least one sign-in provider connected.'),
  ('PREFERENCES_UNLINK_FAILED', 'Could not disconnect that sign-in provider. Try again.'),
  ('PREFERENCES_IDENTITY_PRIMARY_LABEL', 'Primary'),
  ('PREFERENCES_IDENTITY_LAST_USED_LABEL', 'Last used')
on conflict (code) do nothing;

-- Keep the first signup provider on profiles when the trigger upserts again.
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
  fast_tier text;
  deep_tier text;
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

  select lower(btrim(m.body)) into fast_tier
  from public.site_messages m
  where m.code = 'COMPRESSION_TIER_FAST';

  select lower(btrim(m.body)) into deep_tier
  from public.site_messages m
  where m.code = 'COMPRESSION_TIER_DEEP';

  if fast_tier is null or fast_tier = '' or deep_tier is null or deep_tier = '' then
    raise exception 'site_messages COMPRESSION_TIER_FAST / COMPRESSION_TIER_DEEP missing'
      using errcode = 'P0001';
  end if;

  if default_compression is null or default_compression not in (fast_tier, deep_tier) then
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
      new.raw_user_meta_data->>'user_name',
      new.raw_user_meta_data->>'preferred_username',
      new.raw_user_meta_data->>'nickname'
    ),
    coalesce(
      new.raw_user_meta_data->>'avatar_url',
      new.raw_user_meta_data->>'picture',
      new.raw_user_meta_data->>'avatar'
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
    auth_provider = public.profiles.auth_provider,
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
  'Profile + quota from site_messages. auth_provider stays the first signup provider when the row already exists.';
