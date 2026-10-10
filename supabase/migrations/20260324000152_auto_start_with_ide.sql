-- Auto-start Trim with IDE: profiles column + site_messages default (DB-driven, no invent).
-- Fail-closed: no column DEFAULT; signup and backfill read DEFAULT_AUTO_START_WITH_IDE.

insert into public.site_messages (code, body) values
  ('DEFAULT_AUTO_START_WITH_IDE', 'true'),
  ('DEFAULT_AUTO_START_WITH_IDE_MISSING', 'DEFAULT_AUTO_START_WITH_IDE is not configured in site_messages'),
  ('DEFAULT_AUTO_START_WITH_IDE_INVALID', 'DEFAULT_AUTO_START_WITH_IDE must be true or false'),
  ('PREFERENCES_AUTO_START_TITLE', 'Start Trim with your IDE'),
  ('PREFERENCES_AUTO_START_LABEL', 'Auto-start local Trim when the IDE opens'),
  ('PREFERENCES_AUTO_START_HINT', 'Keeps the local Trim proxy ready while you work. Compression stays Fast (structural) on the proxy path. Uncheck anytime. The browser cannot start Trim by itself - CLI, daemon, or IDE extension apply this setting on your machine.'),
  ('PREFERENCES_AUTO_START_NOTE', 'This preference syncs to your device via trim config sync. Local enforcers: trim autostart, trim daemon, and the Trim IDE extension.'),
  ('PREFERENCES_CLI_HELP_6', 'trim autostart enable|disable|status - everyday proxy auto-start with IDE (syncs preference).'),
  ('CLI_AUTOSTART_ENABLED', 'Auto-start with IDE: on'),
  ('CLI_AUTOSTART_DISABLED', 'Auto-start with IDE: off'),
  ('CLI_AUTOSTART_UNSET', 'Auto-start with IDE: unset (fail-closed; run trim config sync or trim autostart enable)'),
  ('CLI_AUTOSTART_ENABLE_HINT', 'Enable with: trim autostart enable'),
  ('CLI_AUTOSTART_DISABLE_HINT', 'Disable with: trim autostart disable'),
  ('CLI_AUTOSTART_ENABLED_OK', 'Auto-start with IDE enabled. Daemon install attempted when supported.'),
  ('CLI_AUTOSTART_DISABLED_OK', 'Auto-start with IDE disabled. Daemon uninstall attempted when supported.'),
  ('CLI_AUTOSTART_DAEMON_HINT', 'OS login start uses trim daemon install/uninstall. Extension attaches to a healthy local proxy.'),
  ('CLI_CONFIG_GET_AUTOSTART_FMT', 'auto-start-with-ide=%v'),
  ('IDE_AUTOSTART_ENSURING', 'Ensuring local Trim proxy is running…'),
  ('IDE_AUTOSTART_PROXY_OK', 'Local Trim proxy is healthy'),
  ('IDE_AUTOSTART_PROXY_STARTED', 'Started local Trim proxy'),
  ('IDE_AUTOSTART_PROXY_FAILED_FMT', 'Could not start local Trim proxy (%s)'),
  ('IDE_AUTOSTART_SKIPPED_OFF', 'Auto-start with IDE is off - proxy not started by extension'),
  ('IDE_AUTOSTART_SKIPPED_UNSET', 'Auto-start preference unset - fail-closed, proxy not started'),
  ('IDE_AUTOSTART_CLI_MISSING', 'trim CLI not found on PATH - install Trim CLI for auto-start'),
  ('IDE_PROXY_HEALTH_URL', 'http://127.0.0.1:8000/health'),
  ('IDE_AUTOSTART_SETTLE_MS', '1500')
on conflict (code) do nothing;

alter table public.profiles
  add column if not exists auto_start_with_ide boolean;

comment on column public.profiles.auto_start_with_ide is
  'When true, CLI daemon / IDE extension should keep local Trim proxy available with the IDE. Seeded from site_messages DEFAULT_AUTO_START_WITH_IDE; no column default.';

-- Backfill existing rows from site_messages (fail if seed missing/invalid).
do $$
declare
  v text;
  b boolean;
begin
  select lower(btrim(m.body)) into v
  from public.site_messages m
  where m.code = 'DEFAULT_AUTO_START_WITH_IDE';

  if v is null or v = '' then
    raise exception 'site_messages DEFAULT_AUTO_START_WITH_IDE is missing or empty';
  end if;
  if v = 'true' then
    b := true;
  elsif v = 'false' then
    b := false;
  else
    raise exception 'site_messages DEFAULT_AUTO_START_WITH_IDE must be true or false, got %', v;
  end if;

  update public.profiles
  set auto_start_with_ide = b
  where auto_start_with_ide is null;
end;
$$;

alter table public.profiles
  alter column auto_start_with_ide set not null;

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
  default_autostart_raw text;
  default_autostart boolean;
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

  select lower(btrim(m.body)) into default_autostart_raw
  from public.site_messages m
  where m.code = 'DEFAULT_AUTO_START_WITH_IDE';

  if default_autostart_raw is null or default_autostart_raw = '' then
    raise exception 'site_messages DEFAULT_AUTO_START_WITH_IDE is missing or empty'
      using errcode = 'P0001';
  end if;
  if default_autostart_raw = 'true' then
    default_autostart := true;
  elsif default_autostart_raw = 'false' then
    default_autostart := false;
  else
    raise exception 'site_messages DEFAULT_AUTO_START_WITH_IDE must be true or false'
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
    compression_tier, deep_engine, deep_target_token,
    auto_start_with_ide,
    account_status
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
    deep_target,
    default_autostart,
    'active'
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
  'Profile + quota from site_messages. Sets account_status=active and auto_start_with_ide from DEFAULT_AUTO_START_WITH_IDE. Blocks disposable domains on first insert.';
