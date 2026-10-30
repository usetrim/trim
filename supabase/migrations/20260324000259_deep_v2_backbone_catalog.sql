-- Microsoft LLMLingua-2 Base Model catalog (mbert-base default, xlm-roberta-large alternate).
-- Billing owns the option list + default; profiles store the user selection.
-- deep_v2_model remains the resolved Hugging Face id for the selected (or default) backbone.

-- ---------------------------------------------------------------------------
-- billing_settings: catalog + default backbone id
-- ---------------------------------------------------------------------------
alter table public.billing_settings
  add column if not exists deep_v2_backbone_options jsonb;

alter table public.billing_settings
  add column if not exists default_deep_v2_backbone text;

update public.billing_settings
set
  deep_v2_backbone_options = coalesce(
    deep_v2_backbone_options,
    '[
      {"id":"mbert-base","label":"mbert-base","model":"microsoft/llmlingua-2-bert-base-multilingual-cased-meetingbank"},
      {"id":"xlm-roberta-large","label":"xlm-roberta-large","model":"microsoft/llmlingua-2-xlm-roberta-large-meetingbank"}
    ]'::jsonb
  ),
  default_deep_v2_backbone = coalesce(nullif(btrim(default_deep_v2_backbone), ''), 'mbert-base'),
  deep_v2_model = coalesce(
    nullif(btrim(deep_v2_model), ''),
    'microsoft/llmlingua-2-bert-base-multilingual-cased-meetingbank'
  )
where id = 'default';

-- Keep deep_v2_model aligned with the default catalog entry's model id.
update public.billing_settings b
set deep_v2_model = (
  select nullif(btrim(opt->>'model'), '')
  from jsonb_array_elements(b.deep_v2_backbone_options) as opt
  where opt->>'id' = b.default_deep_v2_backbone
  limit 1
)
where b.id = 'default'
  and exists (
    select 1
    from jsonb_array_elements(b.deep_v2_backbone_options) as opt
    where opt->>'id' = b.default_deep_v2_backbone
      and nullif(btrim(opt->>'model'), '') is not null
  );

alter table public.billing_settings
  alter column deep_v2_backbone_options set not null,
  alter column default_deep_v2_backbone set not null;

alter table public.billing_settings
  drop constraint if exists billing_settings_deep_v2_backbone_default_check;

alter table public.billing_settings
  add constraint billing_settings_deep_v2_backbone_default_check
  check (
    length(btrim(default_deep_v2_backbone)) > 0
    and jsonb_typeof(deep_v2_backbone_options) = 'array'
    and jsonb_array_length(deep_v2_backbone_options) >= 1
  );

comment on column public.billing_settings.deep_v2_backbone_options is
  'Microsoft LLMLingua-2 Base Model catalog: [{id,label,model}, ...]. Admin-owned; no app invent.';
comment on column public.billing_settings.default_deep_v2_backbone is
  'Default catalog id seeded into profiles.deep_v2_backbone (Microsoft Space default: mbert-base).';

-- ---------------------------------------------------------------------------
-- profiles: user selection
-- ---------------------------------------------------------------------------
alter table public.profiles
  add column if not exists deep_v2_backbone text;

update public.profiles p
set deep_v2_backbone = b.default_deep_v2_backbone
from public.billing_settings b
where b.id = 'default'
  and (p.deep_v2_backbone is null or btrim(p.deep_v2_backbone) = '');

alter table public.profiles
  alter column deep_v2_backbone set not null;

alter table public.profiles
  drop constraint if exists profiles_deep_v2_backbone_check;

alter table public.profiles
  add constraint profiles_deep_v2_backbone_check
  check (length(btrim(deep_v2_backbone)) > 0);

comment on column public.profiles.deep_v2_backbone is
  'User-selected LLMLingua-2 Base Model id from billing deep_v2_backbone_options. Synced by trim config sync.';

-- ---------------------------------------------------------------------------
-- Signup: seed backbone from billing_settings
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
  deep_backbone text;
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

  select b.default_deep_compression_rate, btrim(b.default_deep_v2_backbone)
    into deep_rate, deep_backbone
  from public.billing_settings b
  where b.id = 'default';

  if deep_rate is null then
    raise exception 'billing_settings.default_deep_compression_rate is missing'
      using errcode = 'P0001';
  end if;
  if deep_backbone is null or deep_backbone = '' then
    raise exception 'billing_settings.default_deep_v2_backbone is missing'
      using errcode = 'P0001';
  end if;

  insert into public.profiles (
    id, email, full_name, avatar_url, auth_provider,
    compression_tier, deep_engine, deep_compression_rate, deep_v2_backbone,
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
    deep_backbone,
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
  'Profile + quota from site_messages. deep_compression_rate and deep_v2_backbone from billing_settings.';

-- ---------------------------------------------------------------------------
-- Chrome: Base Model (backbone) UX for Dashboard, Admin, CLI
-- ---------------------------------------------------------------------------
insert into public.site_messages (code, body, updated_at) values
  ('PREFERENCES_BACKBONE_LABEL', 'Base model', now()),
  ('PREFERENCES_BACKBONE_HINT', 'Microsoft LLMLingua-2 backbone (Hugging Face Space Base Model). mbert-base is smaller and faster; xlm-roberta-large is larger and slower. Options come from Admin billing. Save, then run: trim config sync', now()),
  ('PREFERENCES_CLI_HELP_10', 'List Base Models: trim config models   |   Set: trim config set deep-v2-backbone mbert-base', now()),
  ('PREFERENCES_CLI_HELP_11', 'Base Model catalog and default come from Admin billing (Microsoft: mbert-base, xlm-roberta-large)', now()),
  ('ADMIN_BILLING_DEEP_V2_BACKBONE_DEFAULT', 'Default Deep v2 Base Model', now()),
  ('ADMIN_BILLING_DEEP_V2_BACKBONE_DEFAULT_DESC', 'Seeded into new profiles as deep_v2_backbone. Microsoft Space default is mbert-base.', now()),
  ('ADMIN_BILLING_DEEP_V2_BACKBONE_OPTIONS', 'Deep v2 Base Model catalog (JSON)', now()),
  ('ADMIN_BILLING_DEEP_V2_BACKBONE_OPTIONS_DESC', 'JSON array of {id,label,model}. model is the Hugging Face id (LLMLingua-2). Example includes mbert-base and xlm-roberta-large.', now()),
  ('ADMIN_BILLING_DEEP_V2_MODEL_DESC', 'Resolved Hugging Face id for the billing default Base Model (kept in sync when you save the catalog).', now()),
  ('CLI_CONFIG_MODELS_TITLE', 'Deep v2 Base Models (LLMLingua-2)', now()),
  ('CLI_CONFIG_MODELS_DEFAULT_LABEL', 'billing default', now()),
  ('CLI_CONFIG_MODELS_SELECTED_LABEL', 'selected', now()),
  ('CLI_CONFIG_MODELS_SELECTED_MARK', 'selected', now()),
  ('CLI_CONFIG_MODELS_SYNC_HINT', 'Catalog comes from Admin billing via trim config sync. Set with: trim config set deep-v2-backbone <id>', now()),
  ('CLI_CONFIG_MODELS_UNSYNCED', 'Base Model catalog not synced. Run: trim config sync', now()),
  ('CLI_HELP_CONFIG_MODELS_SHORT', 'List allowed Deep v2 Base Models; mark billing default and selected', now()),
  ('CLI_CONFIG_GET_BACKBONE_FMT', 'deep-v2-backbone: %s', now()),
  ('CLI_CONFIG_BACKBONE_INVALID', 'deep-v2-backbone must be an id from the synced Base Model catalog (trim config models)', now()),
  ('CLI_CONFIG_UNKNOWN_KEY_FMT', 'unknown key %q (default-tier|deep-engine|compression-rate|deep-v2-backbone)', now()),
  ('CLI_CONFIG_SYNCED_FMT', 'Synced preferences: tier=%s engine=%s backbone=%s compression-rate=%g', now()),
  ('CLI_PROXY_LIVE_DEEP_ENABLED_FMT', 'Live Deep Mode on (engine %s, backbone %s, compression_rate %g, oom=%s). Synced from account preferences / billing settings.', now()),
  ('CLI_HELP_CONFIG_GET_SHORT', 'Show preferences (default-tier, deep-engine, deep-v2-backbone, compression-rate)', now()),
  ('CLI_HELP_CONFIG_SET_SHORT', 'Set a preference (example: trim config set deep-v2-backbone mbert-base)', now()),
  ('CLI_HELP_CONFIG_SHORT', 'Local Fast/Deep preferences (get | set | sync | rates | models)', now()),
  ('DEEP_V2_BACKBONE_INVALID', 'deep_v2_backbone is not in the billing Base Model catalog', now()),
  ('CLI_UNINSTALL_HF_HUB_DIRS', 'models--microsoft--llmlingua-2-bert-base-multilingual-cased-meetingbank,models--microsoft--llmlingua-2-xlm-roberta-large-meetingbank', now())
on conflict (code) do update set body = excluded.body, updated_at = now();

update public.site_messages
set body = 'Trim keeps a local Fast Mode proxy on your machine so IDE traffic uses less noisy context.

Everyday: start, stop, status, stats, upgrade, autostart, cursor-link, login, logout, uninstall, setup, version
Advanced: tui, compress, daemon, telemetry, config (get | set | sync | rates | models)

Fast vs Deep (new accounts default to Deep = Fast then Deep):
  trim config get default-tier
  trim config set default-tier deep
  trim config set default-tier fast
  trim config rates
  trim config set compression-rate 0.7
  trim config models
  trim config set deep-v2-backbone mbert-base
  trim config sync

Deep keep-rate (after/before, Microsoft LLMLingua): Dashboard slider or trim config rates / set compression-rate (billing min/max/step).
Deep Base Model (Microsoft LLMLingua-2): Dashboard picker or trim config models / set deep-v2-backbone (billing catalog; default mbert-base, optional xlm-roberta-large).

Cursor Agent: trim cursor-link enable → trim start → trim cursor-link url (paste HTTPS …/v1; not 127.0.0.1).

Simple savings check after trim start:
  trim stats
  open http://127.0.0.1:<port>/dashboard → Show savings detail

trim status = cloud quota. trim stats = local Deep status / savings meter.

Full remove: trim uninstall (then brew/scoop/winget/npm uninstall if you installed that way).

Run "trim help <command>" for details. Prefer trim start / autostart for daily use (new accounts default Deep = Fast then Deep on the live proxy). Use trim compress --deep for file/batch.',
    updated_at = now()
where code = 'CLI_HELP_ROOT_LONG';

update public.site_messages
set body = 'Manage local Fast/Deep preferences under ~/.config/trim/.

New accounts default to Deep Mode (live proxy runs Fast then Deep). Switch to Fast anytime.

  trim config get
  trim config get default-tier
  trim config get compression-rate
  trim config get deep-v2-backbone
  trim config rates
  trim config models
  trim config set default-tier deep
  trim config set default-tier fast
  trim config set compression-rate 0.7
  trim config set deep-v2-backbone mbert-base
  trim config set deep-v2-backbone xlm-roberta-large
  trim config sync

get - show local preferences
rates - list allowed keep-rates; mark billing default and selected
models - list allowed LLMLingua-2 Base Models; mark billing default and selected
set - write a local preference key (when logged in, default-tier / deep-engine / compression-rate / deep-v2-backbone also update your Trim cloud profile)
sync - pull Fast/Deep preferences and provider adapters from your Trim cloud profile (requires trim login)

Fast = live proxy Fast only. Deep = live proxy runs Fast then Deep (LLMLingua-2) at your compression rate and Base Model.
After Dashboard → Settings changes: trim config sync, then restart with trim start.',
    updated_at = now()
where code = 'CLI_HELP_CONFIG_LONG';

-- Preferences chrome loads PREFERENCES_CLI_HELP_1..11 in account handler.
