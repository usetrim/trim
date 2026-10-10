-- Trim core schema: identity, quotas, api keys, devices, workspaces, billing

create extension if not exists "pgcrypto";

-- Profiles (Google OAuth only for now; auth_provider kept extensible)
create table if not exists public.profiles (
  id uuid primary key references auth.users(id) on delete cascade,
  email text unique not null,
  full_name text,
  avatar_url text,
  auth_provider text not null default 'google',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists public.device_fingerprints (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references public.profiles(id) on delete cascade,
  hardware_uuid text not null,
  last_seen_at timestamptz not null default now(),
  created_at timestamptz not null default now(),
  unique (user_id, hardware_uuid)
);

create index if not exists idx_device_fingerprint_hash
  on public.device_fingerprints(hardware_uuid);

create table if not exists public.user_quotas (
  user_id uuid primary key references public.profiles(id) on delete cascade,
  plan_tier text not null default 'free',
  monthly_credit_limit int not null default 50,
  monthly_credit_used int not null default 0,
  purchased_topup_credits int not null default 0,
  last_reset_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists public.api_keys (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references public.profiles(id) on delete cascade,
  key_hash text unique not null,
  key_prefix text not null,
  hardware_uuid text,
  revoked boolean not null default false,
  expires_at timestamptz,
  created_at timestamptz not null default now()
);

create index if not exists idx_api_keys_hash
  on public.api_keys(key_hash) where revoked = false;

create table if not exists public.subscriptions (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references public.profiles(id) on delete cascade,
  paddle_customer_id text not null,
  paddle_subscription_id text unique not null,
  status text not null,
  price_id text not null,
  plan_tier text not null default 'pro',
  current_period_start timestamptz not null,
  current_period_end timestamptz not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists idx_subscriptions_user_status
  on public.subscriptions(user_id, status);

create table if not exists public.workspaces (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  owner_id uuid not null references public.profiles(id) on delete cascade,
  paddle_customer_id text unique,
  paddle_subscription_id text unique,
  plan_tier text not null default 'free',
  allocated_seats int not null default 1,
  created_at timestamptz not null default now()
);

create table if not exists public.workspace_members (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid not null references public.workspaces(id) on delete cascade,
  user_id uuid not null references public.profiles(id) on delete cascade,
  role text not null default 'member',
  joined_at timestamptz not null default now(),
  unique (workspace_id, user_id)
);

create table if not exists public.workspace_quotas (
  workspace_id uuid primary key references public.workspaces(id) on delete cascade,
  monthly_shared_credits int not null default 5000,
  credits_consumed int not null default 0,
  updated_at timestamptz not null default now()
);

-- Proxy request metrics (local + cloud rollup)
create table if not exists public.trim_events (
  id uuid primary key default gen_random_uuid(),
  user_id uuid references public.profiles(id) on delete set null,
  request_id text,
  model text,
  tokens_before int not null default 0,
  tokens_after int not null default 0,
  latency_ms int not null default 0,
  mode text not null default 'proxy',
  status text not null default 'success',
  created_at timestamptz not null default now()
);

create index if not exists idx_trim_events_user_created
  on public.trim_events(user_id, created_at desc);

-- Auto-create profile + free quota on signup
create or replace function public.handle_new_user()
returns trigger
language plpgsql
security definer
set search_path = public
as $$
begin
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
  values (new.id, 'free', 50, 0)
  on conflict (user_id) do nothing;

  return new;
end;
$$;

drop trigger if exists on_auth_user_created on auth.users;
create trigger on_auth_user_created
  after insert on auth.users
  for each row execute function public.handle_new_user();

-- GDPR hard delete helper
create or replace function public.execute_gdpr_deletion(target_user_id uuid)
returns void
language plpgsql
security definer
set search_path = public
as $$
begin
  delete from public.api_keys where user_id = target_user_id;
  delete from public.device_fingerprints where user_id = target_user_id;
  delete from public.user_quotas where user_id = target_user_id;
  delete from public.workspace_members where user_id = target_user_id;
  update public.trim_events set user_id = null where user_id = target_user_id;
  delete from public.profiles where id = target_user_id;
end;
$$;

alter table public.profiles enable row level security;
alter table public.user_quotas enable row level security;
alter table public.api_keys enable row level security;
alter table public.trim_events enable row level security;
alter table public.subscriptions enable row level security;
alter table public.workspaces enable row level security;
alter table public.workspace_members enable row level security;

create policy "profiles_select_own" on public.profiles
  for select using (auth.uid() = id);

create policy "quotas_select_own" on public.user_quotas
  for select using (auth.uid() = user_id);

create policy "api_keys_own" on public.api_keys
  for all using (auth.uid() = user_id);

create policy "events_select_own" on public.trim_events
  for select using (auth.uid() = user_id);

create policy "subscriptions_select_own" on public.subscriptions
  for select using (auth.uid() = user_id);
