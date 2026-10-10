-- Enforce Google-only signup for now while keeping auth_provider extensible.
-- Non-Google providers are rejected at the database trigger so email/phone signup
-- cannot create profiles even if Supabase Auth is misconfigured.

create or replace function public.handle_new_user()
returns trigger
language plpgsql
security definer
set search_path = public
as $$
declare
  provider text;
begin
  provider := lower(coalesce(new.raw_app_meta_data->>'provider', ''));
  if provider = '' then
    provider := lower(coalesce(new.raw_app_meta_data->'providers'->>0, 'google'));
  end if;

  -- Scalable allow-list: today Google only. Add providers here when product expands.
  if provider is distinct from 'google' then
    raise exception 'auth provider % is not allowed; Trim currently accepts Google only', provider
      using errcode = 'P0001';
  end if;

  insert into public.profiles (id, email, full_name, avatar_url, auth_provider)
  values (
    new.id,
    new.email,
    coalesce(new.raw_user_meta_data->>'full_name', new.raw_user_meta_data->>'name'),
    new.raw_user_meta_data->>'avatar_url',
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
      50
    ),
    0
  )
  on conflict (user_id) do nothing;

  return new;
end;
$$;
