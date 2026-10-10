-- Auth allow-list in DB, kept in sync with API ALLOWED_AUTH_PROVIDERS on boot.
-- Signup trigger reads this table so DB and env stay aligned (no hardcoded provider list).

create table if not exists public.auth_settings (
  id text primary key default 'default',
  allowed_providers text[] not null default array['google', 'github', 'gitlab']::text[],
  updated_at timestamptz not null default now(),
  constraint auth_settings_providers_nonempty check (cardinality(allowed_providers) >= 1)
);

insert into public.auth_settings (id, allowed_providers)
values ('default', array['google', 'github', 'gitlab']::text[])
on conflict (id) do nothing;

comment on table public.auth_settings is
  'OAuth provider allow-list. Cloud API upserts from ALLOWED_AUTH_PROVIDERS on startup.';

create or replace function public.handle_new_user()
returns trigger
language plpgsql
security definer
set search_path = public
as $$
declare
  provider text;
  allowed text[];
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

  insert into public.profiles (id, email, full_name, avatar_url, auth_provider)
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
    provider
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
    coalesce(
      (select credits_monthly from public.plan_catalog where id = 'free' and is_active = true),
      (select credits_monthly from public.plan_catalog where id = 'free' limit 1)
    ),
    0
  )
  on conflict (user_id) do nothing;

  return new;
end;
$$;

comment on function public.handle_new_user() is
  'Creates profile + free quota on signup. Allowed providers come from auth_settings (synced from ALLOWED_AUTH_PROVIDERS).';
