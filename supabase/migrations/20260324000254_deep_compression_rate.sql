-- Deep Mode: Microsoft-style compression rate (after/before) replaces absolute target_token.
-- Product: default 0.7, user range 0.3..1 (billing_settings driven; no app invent).
-- Removes silent gates: live_deep_min_input_tokens, live_deep_skip_on_stream.

-- ---------------------------------------------------------------------------
-- billing_settings: rate dials
-- ---------------------------------------------------------------------------
alter table public.billing_settings
  add column if not exists default_deep_compression_rate numeric(4,2);

alter table public.billing_settings
  add column if not exists deep_compression_rate_min numeric(4,2);

alter table public.billing_settings
  add column if not exists deep_compression_rate_max numeric(4,2);

update public.billing_settings
set
  default_deep_compression_rate = coalesce(default_deep_compression_rate, 0.70),
  deep_compression_rate_min = coalesce(deep_compression_rate_min, 0.30),
  deep_compression_rate_max = coalesce(deep_compression_rate_max, 1.00)
where id = 'default';

alter table public.billing_settings
  alter column default_deep_compression_rate set not null,
  alter column deep_compression_rate_min set not null,
  alter column deep_compression_rate_max set not null;

alter table public.billing_settings
  drop constraint if exists billing_settings_deep_compression_rate_bounds_check;

alter table public.billing_settings
  add constraint billing_settings_deep_compression_rate_bounds_check
  check (
    deep_compression_rate_min >= 0.10
    and deep_compression_rate_max <= 1.00
    and deep_compression_rate_max >= deep_compression_rate_min
    and default_deep_compression_rate >= deep_compression_rate_min
    and default_deep_compression_rate <= deep_compression_rate_max
  );

comment on column public.billing_settings.default_deep_compression_rate is
  'Seeded into profiles.deep_compression_rate on signup. Keep-rate after/before (Microsoft LLMLingua). Change in Admin billing; no app invent.';
comment on column public.billing_settings.deep_compression_rate_min is
  'Minimum deep_compression_rate users may select (Dashboard / CLI).';
comment on column public.billing_settings.deep_compression_rate_max is
  'Maximum deep_compression_rate users may select (Dashboard / CLI).';

-- ---------------------------------------------------------------------------
-- profiles: rate column; migrate then drop target_token
-- ---------------------------------------------------------------------------
alter table public.profiles
  add column if not exists deep_compression_rate numeric(4,2);

update public.profiles p
set deep_compression_rate = b.default_deep_compression_rate
from public.billing_settings b
where b.id = 'default'
  and (p.deep_compression_rate is null);

alter table public.profiles
  alter column deep_compression_rate set not null;

alter table public.profiles
  drop constraint if exists profiles_deep_compression_rate_check;

alter table public.profiles
  add constraint profiles_deep_compression_rate_check
  check (deep_compression_rate > 0 and deep_compression_rate <= 1);

comment on column public.profiles.deep_compression_rate is
  'User Deep keep-rate (after/before). Bounds from billing_settings. Synced by trim config sync.';

alter table public.profiles
  drop constraint if exists profiles_deep_target_token_check;

alter table public.profiles
  drop column if exists deep_target_token;

-- ---------------------------------------------------------------------------
-- Drop obsolete billing dials
-- ---------------------------------------------------------------------------
alter table public.billing_settings
  drop constraint if exists billing_settings_default_deep_target_token_check;

alter table public.billing_settings
  drop constraint if exists billing_settings_deep_target_token_bounds_check;

alter table public.billing_settings
  drop constraint if exists billing_settings_live_deep_min_input_tokens_check;

alter table public.billing_settings
  drop column if exists default_deep_target_token;

alter table public.billing_settings
  drop column if exists deep_target_token_min;

alter table public.billing_settings
  drop column if exists deep_target_token_max;

alter table public.billing_settings
  drop column if exists live_deep_min_input_tokens;

alter table public.billing_settings
  drop column if exists live_deep_skip_on_stream;

-- ---------------------------------------------------------------------------
-- Signup: seed compression rate from billing_settings
-- ---------------------------------------------------------------------------
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
  deep_rate numeric(4,2);
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

  select b.default_deep_compression_rate into deep_rate
  from public.billing_settings b
  where b.id = 'default';

  if deep_rate is null then
    raise exception 'billing_settings.default_deep_compression_rate is missing'
      using errcode = 'P0001';
  end if;

  insert into public.profiles (
    id, email, full_name, avatar_url, auth_provider,
    compression_tier, deep_engine, deep_compression_rate,
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
    deep_rate,
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
  'Profile + quota from site_messages. deep_compression_rate from billing_settings. Blocks disposable domains on first insert.';

-- ---------------------------------------------------------------------------
-- Chrome: rate UX; retire target / min-input / skip-stream copy
-- ---------------------------------------------------------------------------
insert into public.site_messages (code, body, updated_at) values
  ('PREFERENCES_RATE_LABEL', 'Compression rate', now()),
  ('PREFERENCES_RATE_HINT', 'Keep-rate after / before (Microsoft LLMLingua). Lower = more compression. Drag 0.3 to 1.0 (step 0.1). Default from billing is usually 0.7. Save, then run: trim config sync', now()),
  ('PREFERENCES_TARGET_LABEL', 'Compression rate', now()),
  ('PREFERENCES_TARGET_HINT', 'Keep-rate after / before (Microsoft LLMLingua). Lower = more compression. Drag 0.3 to 1.0 (step 0.1). Default from billing is usually 0.7. Save, then run: trim config sync', now()),
  ('PREFERENCES_DEEP_HINT', 'When Deep Mode is on, the live IDE proxy (trim start) runs Fast then Deep (LLMLingua) using your engine and compression rate from this page. Off = Fast only. Trim cloud never loads Deep engines; auth, quotas, and billing still use Trim cloud.', now()),
  ('PREFERENCES_CLI_HELP_1', 'Sync cloud preferences to this machine: trim config sync', now()),
  ('PREFERENCES_CLI_HELP_2', 'Show local prefs: trim config get', now()),
  ('PREFERENCES_CLI_HELP_3', 'Set mode: trim config set default-tier deep|fast', now()),
  ('PREFERENCES_CLI_HELP_4', 'Set engine: trim config set deep-engine v2|v1|long', now()),
  ('PREFERENCES_CLI_HELP_5', 'Set keep-rate (after/before): trim config set compression-rate 0.7', now()),
  ('PREFERENCES_CLI_HELP_6', 'Allowed rate range comes from Admin billing (typically 0.3 to 1.0, step 0.1)', now()),
  ('PREFERENCES_CLI_HELP_7', 'After Dashboard Save or CLI set, run trim config sync on each machine', now()),
  ('PREFERENCES_CLI_HELP_8', 'Live proxy: trim start (Deep runs whenever Deep Mode is on; no silent min-token or stream skip)', now()),
  ('PREFERENCES_CLI_HELP_9', 'Batch: trim compress --deep --rate 0.7 path/to/file', now()),
  ('ADMIN_BILLING_DEEP_RATE_DEFAULT', 'Default deep compression rate', now()),
  ('ADMIN_BILLING_DEEP_RATE_DEFAULT_DESC', 'Seeded into new subscriptions as profiles.deep_compression_rate. Microsoft keep-rate after/before (typical 0.7).', now()),
  ('ADMIN_BILLING_DEEP_RATE_MIN', 'Deep compression rate min', now()),
  ('ADMIN_BILLING_DEEP_RATE_MIN_DESC', 'Lower bound users may select on Dashboard and CLI (professional floor is 0.3; Microsoft demo allows 0.1).', now()),
  ('ADMIN_BILLING_DEEP_RATE_MAX', 'Deep compression rate max', now()),
  ('ADMIN_BILLING_DEEP_RATE_MAX_DESC', 'Upper bound users may select (1.0 = keep almost everything).', now()),
  ('ADMIN_BILLING_DEEP_TARGET_DEFAULT', 'Default deep compression rate', now()),
  ('ADMIN_BILLING_DEEP_TARGET_DEFAULT_DESC', 'Seeded into new subscriptions as profiles.deep_compression_rate. Microsoft keep-rate after/before (typical 0.7).', now()),
  ('ADMIN_BILLING_DEEP_TARGET_MIN', 'Deep compression rate min', now()),
  ('ADMIN_BILLING_DEEP_TARGET_MIN_DESC', 'Lower bound users may select on Dashboard and CLI (professional floor is 0.3; Microsoft demo allows 0.1).', now()),
  ('ADMIN_BILLING_DEEP_TARGET_MAX', 'Deep compression rate max', now()),
  ('ADMIN_BILLING_DEEP_TARGET_MAX_DESC', 'Upper bound users may select (1.0 = keep almost everything).', now()),
  ('CLI_CONFIG_GET_RATE_FMT', 'compression-rate: %g', now()),
  ('CLI_CONFIG_GET_TARGET_FMT', 'compression-rate: %g', now()),
  ('CLI_CONFIG_RATE_RANGE', 'compression-rate must be within billing deep_compression_rate_min..max (typically 0.3 to 1.0)', now()),
  ('CLI_CONFIG_TARGET_POSITIVE', 'compression-rate must be within billing deep_compression_rate_min..max (typically 0.3 to 1.0)', now()),
  ('CLI_CONFIG_SYNCED_FMT', 'Synced preferences: tier=%s engine=%s compression-rate=%g', now()),
  ('CLI_DEEP_RATE_REQUIRED', 'Deep Mode requires deep_engine and compression-rate (run: trim config sync, or trim config set deep-engine / compression-rate, or pass --engine and --rate)', now()),
  ('CLI_DEEP_TARGET_REQUIRED', 'Deep Mode requires deep_engine and compression-rate (run: trim config sync, or trim config set deep-engine / compression-rate, or pass --engine and --rate)', now()),
  ('CLI_PREFS_RATE_REQUIRED', 'compression-rate must be a positive keep-rate (after/before); set via trim config or preferences', now()),
  ('CLI_PREFS_TARGET_REQUIRED', 'compression-rate must be a positive keep-rate (after/before); set via trim config or preferences', now()),
  ('CLI_PREFS_RATE_NONNEG', 'compression-rate must be greater than 0', now()),
  ('CLI_PREFS_TARGET_NONNEG', 'compression-rate must be greater than 0', now()),
  ('CLI_PROXY_LIVE_DEEP_ENABLED_FMT', 'Live Deep Mode on (engine %s, compression_rate %g, oom=%s). Synced from account preferences / billing settings.', now()),
  ('CLI_PROXY_DEEP_PREFS_REQUIRED', 'Live Deep Mode requires compression_tier=deep plus deep_engine and deep_compression_rate. Save Dashboard → Settings, then run: trim config sync', now()),
  ('CLI_HELP_CONFIG_GET_SHORT', 'Show preferences (default-tier, deep-engine, compression-rate)', now()),
  ('CLI_HELP_COMPRESS_FLAG_RATE', 'Deep keep-rate after/before (Microsoft LLMLingua); omit to use synced preferences', now()),
  ('CLI_HELP_COMPRESS_FLAG_TARGET', 'Deep keep-rate after/before (Microsoft LLMLingua); omit to use synced preferences', now()),
  ('DEEP_COMPRESSION_RATE_RANGE', 'deep_compression_rate out of range', now()),
  ('DEEP_TARGET_TOKEN_RANGE', 'deep_compression_rate out of range', now()),
  ('LOCAL_DEEP_STATUS_SKIPPED_MIN', 'Deep skipped (below min) (legacy)', now()),
  ('LOCAL_DEEP_STATUS_SKIPPED_STREAM', 'Deep skipped (stream request) (legacy)', now()),
  ('PREFERENCES_LIVE_DEEP_MIN_HINT', '', now()),
  ('PREFERENCES_LIVE_DEEP_STREAM_HINT', '', now()),
  ('ADMIN_BILLING_LIVE_DEEP_MIN', '', now()),
  ('ADMIN_BILLING_LIVE_DEEP_MIN_DESC', '', now()),
  ('CLI_PROXY_DEEP_SKIPPED_MIN_FMT', '', now()),
  ('CLI_PROXY_DEEP_SKIPPED_STREAM', '', now())
on conflict (code) do update set body = excluded.body, updated_at = now();

update public.site_messages
set body = 'Live proxy runs Fast Mode on every turn. When account preferences set compression_tier=deep, the local proxy also runs Deep Mode (LLMLingua) after Fast using your compression rate. Models and OOM policy come from billing settings via trim config sync.',
    updated_at = now()
where code = 'PREFERENCES_PAGE_DESCRIPTION';

update public.site_messages
set body = 'Trim keeps a local Fast Mode proxy on your machine so IDE traffic uses less noisy context.

Everyday: start, stop, status, stats, upgrade, autostart, cursor-link, login, logout, uninstall, setup, version
Advanced: tui, compress, daemon, telemetry, config (get | set | sync)

Fast vs Deep (new accounts default to Fast):
  trim config get default-tier
  trim config set default-tier fast
  trim config set default-tier deep
  trim config set compression-rate 0.7
  trim config sync

Deep keep-rate (after/before, Microsoft LLMLingua): Dashboard slider or trim config set compression-rate (billing bounds, typically 0.3 to 1.0 step 0.1).

Cursor Agent: trim cursor-link enable → trim start → trim cursor-link url (paste HTTPS …/v1; not 127.0.0.1).

Simple savings check after trim start:
  trim stats
  open http://127.0.0.1:<port>/dashboard → Show savings detail

trim status = cloud quota. trim stats = local Deep status / savings meter.

Full remove: trim uninstall (then brew/scoop/winget/npm uninstall if you installed that way).

Run "trim help <command>" for details. Prefer trim start / autostart for daily use; compress and Deep Mode are file/batch tools, not the everyday IDE path.',
    updated_at = now()
where code = 'CLI_HELP_ROOT_LONG';

update public.site_messages
set body = 'Manage local Fast/Deep preferences under ~/.config/trim/.

New accounts default to Fast Mode. Deep Mode is optional.

  trim config get
  trim config get default-tier
  trim config get compression-rate
  trim config set default-tier fast
  trim config set default-tier deep
  trim config set compression-rate 0.7
  trim config sync

get - show local preferences
set - write a local preference key (when logged in, default-tier / deep-engine / compression-rate also update your Trim cloud profile)
sync - pull Fast/Deep preferences and provider adapters from your Trim cloud profile (requires trim login)

Fast = live proxy Fast only. Deep = live proxy runs Fast then Deep (LLMLingua) at your compression rate (keep after/before; typically 0.3 to 1.0 step 0.1).
After Dashboard → Settings changes: trim config sync, then restart with trim start.',
    updated_at = now()
where code = 'CLI_HELP_CONFIG_LONG';

update public.site_messages
set body = 'Show preferences (default-tier, deep-engine, compression-rate)',
    updated_at = now()
where code = 'CLI_HELP_CONFIG_GET_SHORT';

update public.site_messages
set body = 'Deep keep-rate after/before (Microsoft LLMLingua); omit to use synced preferences',
    updated_at = now()
where code in ('CLI_HELP_COMPRESS_FLAG_RATE', 'CLI_HELP_COMPRESS_FLAG_TARGET');
