-- Free-tier credits come only from plan_catalog. No coalesce(..., 50) fallback.
-- Inserts into user_quotas must always supply monthly_credit_limit explicitly.

alter table public.user_quotas
  alter column monthly_credit_limit drop default;

create or replace function public.handle_new_user()
returns trigger
language plpgsql
security definer
set search_path = public
as $$
declare
  provider text;
  free_credits int;
begin
  provider := lower(coalesce(new.raw_app_meta_data->>'provider', ''));
  if provider = '' then
    provider := lower(coalesce(new.raw_app_meta_data->'providers'->>0, ''));
  end if;

  if provider not in ('google', 'github', 'gitlab') then
    raise exception 'auth provider % is not allowed; Trim accepts Google, GitHub, or GitLab', provider
      using errcode = 'P0001';
  end if;

  select credits_monthly into free_credits
  from public.plan_catalog
  where id = 'free' and is_active = true;

  if free_credits is null then
    raise exception 'plan_catalog free plan is missing or inactive'
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
    free_credits,
    0
  )
  on conflict (user_id) do nothing;

  return new;
end;
$$;

comment on function public.handle_new_user() is
  'Creates profile + free quota from plan_catalog only. Providers: google, github, gitlab.';
