-- Disposable / throwaway email domain denylist (server + signup trigger).
-- Operators extend via INSERT; no invent allow-list on the client.

create table if not exists public.email_domain_denylist (
  domain text primary key,
  reason text not null default 'disposable',
  created_at timestamptz not null default now(),
  constraint email_domain_denylist_domain_lower check (domain = lower(domain)),
  constraint email_domain_denylist_domain_nonempty check (length(btrim(domain)) > 0)
);

comment on table public.email_domain_denylist is
  'Blocked email domains for signup and workspace invites. Queried by handle_new_user and Go API.';

alter table public.email_domain_denylist enable row level security;

create or replace function public.email_domain_is_denied(raw_email text)
returns boolean
language plpgsql
stable
security definer
set search_path = public
as $$
declare
  addr text;
  dom text;
  at_pos int;
begin
  addr := lower(btrim(coalesce(raw_email, '')));
  if addr = '' then
    return true;
  end if;
  at_pos := position('@' in addr);
  if at_pos < 2 or at_pos = length(addr) then
    return true;
  end if;
  dom := substring(addr from at_pos + 1);
  if dom = '' or position('@' in dom) > 0 then
    return true;
  end if;
  return exists (
    select 1 from public.email_domain_denylist d where d.domain = dom
  );
end;
$$;

revoke all on function public.email_domain_is_denied(text) from public;
grant execute on function public.email_domain_is_denied(text) to postgres, service_role, authenticated, anon;

comment on function public.email_domain_is_denied(text) is
  'True when the email domain is on email_domain_denylist or the address is malformed.';

insert into public.email_domain_denylist (domain, reason) values
  ('mailinator.com', 'disposable'),
  ('guerrillamail.com', 'disposable'),
  ('guerrillamail.net', 'disposable'),
  ('guerrillamail.org', 'disposable'),
  ('sharklasers.com', 'disposable'),
  ('grr.la', 'disposable'),
  ('yopmail.com', 'disposable'),
  ('yopmail.fr', 'disposable'),
  ('tempmail.com', 'disposable'),
  ('temp-mail.org', 'disposable'),
  ('throwaway.email', 'disposable'),
  ('10minutemail.com', 'disposable'),
  ('10minutemail.net', 'disposable'),
  ('trashmail.com', 'disposable'),
  ('trashmail.me', 'disposable'),
  ('discard.email', 'disposable'),
  ('mailnesia.com', 'disposable'),
  ('maildrop.cc', 'disposable'),
  ('getnada.com', 'disposable'),
  ('emailondeck.com', 'disposable'),
  ('fakeinbox.com', 'disposable'),
  ('tempail.com', 'disposable'),
  ('mohmal.com', 'disposable'),
  ('mailcatch.com', 'disposable'),
  ('inboxalias.com', 'disposable'),
  ('dispostable.com', 'disposable'),
  ('mailnull.com', 'disposable'),
  ('spamgourmet.com', 'disposable'),
  ('mailinator.net', 'disposable'),
  ('guerrillamailblock.com', 'disposable')
on conflict (domain) do nothing;

insert into public.site_messages (code, body) values
  ('LOGIN_OAUTH_EMAIL_DENIED', 'Sign-in with disposable or blocked email domains is not allowed. Use a lasting work or personal address.'),
  ('AUTH_EMAIL_DOMAIN_DENIED', 'This email domain is not allowed.'),
  ('WS_EMAIL_DOMAIN_DENIED', 'That invite email domain is blocked (disposable or denylisted).'),
  ('CLI_DEEP_REQUIREMENTS_MISSING', 'requirements-deep.txt (or requirements.txt) not found beside optimizer.py. Pack it with the CLI release or set TRIM_OPTIMIZER_PY to a tree that includes the requirements file.')
on conflict (code) do nothing;

-- Block disposable domains on first profile insert only (existing users keep access).
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
  already boolean;
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

  select exists(select 1 from public.profiles p where p.id = new.id) into already;
  if not already and public.email_domain_is_denied(new.email) then
    raise exception 'email domain denied for signup'
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
  'Profile + quota from site_messages. Blocks disposable domains on first insert. auth_provider stays the first signup provider when the row already exists.';
