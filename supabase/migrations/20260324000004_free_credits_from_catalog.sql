-- Seed free-tier credit limit from plan_catalog (no hardcoded 50)

create or replace function public.handle_new_user()
returns trigger
language plpgsql
security definer
set search_path = public
as $$
declare
  free_credits int;
begin
  select credits_monthly into free_credits
  from public.plan_catalog
  where id = 'free' and is_active = true;

  if free_credits is null then
    raise exception 'plan_catalog free plan is missing or inactive';
  end if;

  insert into public.profiles (id, email, full_name, avatar_url, auth_provider)
  values (
    new.id,
    new.email,
    coalesce(new.raw_user_meta_data->>'full_name', new.raw_user_meta_data->>'name'),
    new.raw_user_meta_data->>'avatar_url',
    coalesce(new.raw_app_meta_data->>'provider', 'google')
  )
  on conflict (id) do update set
    email = excluded.email,
    full_name = coalesce(excluded.full_name, public.profiles.full_name),
    avatar_url = coalesce(excluded.avatar_url, public.profiles.avatar_url),
    updated_at = now();

  insert into public.user_quotas (user_id, plan_tier, monthly_credit_limit, monthly_credit_used)
  values (new.id, 'free', free_credits, 0)
  on conflict (user_id) do nothing;

  return new;
end;
$$;
