-- Product default: Deep Mode for new accounts (savings-first).
-- Existing accounts keep their chosen compression_tier (no mass flip).
-- Adds billing deep_compression_rate_step for CLI/Dashboard rate lists (Microsoft 0.1 steps).

-- ---------------------------------------------------------------------------
-- Rate step (DB-driven; no app invent)
-- ---------------------------------------------------------------------------
alter table public.billing_settings
  add column if not exists deep_compression_rate_step numeric(4,2);

update public.billing_settings
set deep_compression_rate_step = coalesce(deep_compression_rate_step, 0.10)
where id = 'default';

alter table public.billing_settings
  alter column deep_compression_rate_step set not null;

alter table public.billing_settings
  drop constraint if exists billing_settings_deep_compression_rate_step_check;

alter table public.billing_settings
  add constraint billing_settings_deep_compression_rate_step_check
  check (deep_compression_rate_step > 0 and deep_compression_rate_step <= 1);

comment on column public.billing_settings.deep_compression_rate_step is
  'Step size for Dashboard slider and CLI rate list (Microsoft HF Space uses 0.1).';

-- ---------------------------------------------------------------------------
-- New-account default: Deep
-- ---------------------------------------------------------------------------
do $$
declare
  deep_tier text;
begin
  select lower(btrim(m.body)) into deep_tier
  from public.site_messages m
  where m.code = 'COMPRESSION_TIER_DEEP';

  if deep_tier is null or deep_tier = '' then
    raise exception 'site_messages COMPRESSION_TIER_DEEP missing';
  end if;

  update public.site_messages
  set body = deep_tier, updated_at = now()
  where code = 'DEFAULT_COMPRESSION_TIER'
    and lower(btrim(body)) is distinct from deep_tier;

  if not exists (
    select 1 from public.site_messages where code = 'DEFAULT_COMPRESSION_TIER'
  ) then
    insert into public.site_messages (code, body, updated_at)
    values ('DEFAULT_COMPRESSION_TIER', deep_tier, now());
  end if;
end;
$$;

update public.site_messages
set body = 'Default compression tier for new accounts (deep = Fast then Deep on live proxy; fast = Fast only).',
    updated_at = now()
where code = 'ADMIN_PRODUCT_MODE_DESC';

update public.admin_product_settings
set
  default_compression_mode = 'deep',
  updated_at = now()
where id = 'default'
  and lower(btrim(coalesce(default_compression_mode, ''))) is distinct from 'deep';

-- ---------------------------------------------------------------------------
-- CLI / Dashboard chrome for rate list + Deep default copy
-- ---------------------------------------------------------------------------
insert into public.site_messages (code, body, updated_at) values
  ('CLI_CONFIG_RATES_TITLE', 'Compression rates (after / before)', now()),
  ('CLI_CONFIG_RATES_DEFAULT_LABEL', 'billing default', now()),
  ('CLI_CONFIG_RATES_SELECTED_LABEL', 'selected', now()),
  ('CLI_CONFIG_RATES_SELECTED_MARK', 'selected', now()),
  ('CLI_CONFIG_RATES_SYNC_HINT', 'Bounds and step come from Admin billing via trim config sync. Set with: trim config set compression-rate <rate>', now()),
  ('CLI_CONFIG_RATES_UNSYNCED', 'Rate bounds not synced. Run: trim config sync', now()),
  ('CLI_HELP_CONFIG_RATES_SHORT', 'List allowed Deep compression rates; mark billing default and selected', now()),
  ('ADMIN_BILLING_DEEP_RATE_STEP', 'Deep compression rate step', now()),
  ('ADMIN_BILLING_DEEP_RATE_STEP_DESC', 'Slider and CLI list step (Microsoft HF Space uses 0.1).', now()),
  ('PREFERENCES_CLI_HELP_5', 'List rates: trim config rates   |   Set: trim config set compression-rate 0.7', now()),
  ('PREFERENCES_CLI_HELP_6', 'Rate bounds and step come from Admin billing (typically min 0.3, max 1.0, step 0.1)', now()),
  ('PREFERENCES_RATE_HINT', 'Keep-rate after / before (Microsoft LLMLingua). Lower = more compression. Drag within billing min/max by billing step (usually 0.1). Default from billing is usually 0.7. Save, then run: trim config sync', now()),
  ('PREFERENCES_TARGET_HINT', 'Keep-rate after / before (Microsoft LLMLingua). Lower = more compression. Drag within billing min/max by billing step (usually 0.1). Default from billing is usually 0.7. Save, then run: trim config sync', now())
on conflict (code) do update set body = excluded.body, updated_at = now();

update public.site_messages
set body = 'Trim keeps a local Fast Mode proxy on your machine so IDE traffic uses less noisy context.

Everyday: start, stop, status, stats, upgrade, autostart, cursor-link, login, logout, uninstall, setup, version
Advanced: tui, compress, daemon, telemetry, config (get | set | sync | rates)

Fast vs Deep (new accounts default to Deep = Fast then Deep):
  trim config get default-tier
  trim config set default-tier deep
  trim config set default-tier fast
  trim config rates
  trim config set compression-rate 0.7
  trim config sync

Deep keep-rate (after/before, Microsoft LLMLingua): Dashboard slider or trim config rates / set compression-rate (billing min/max/step).

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

New accounts default to Deep Mode (live proxy runs Fast then Deep). Switch to Fast anytime.

  trim config get
  trim config get default-tier
  trim config get compression-rate
  trim config rates
  trim config set default-tier deep
  trim config set default-tier fast
  trim config set compression-rate 0.7
  trim config sync

get - show local preferences
rates - list allowed keep-rates; mark billing default and selected
set - write a local preference key (when logged in, default-tier / deep-engine / compression-rate also update your Trim cloud profile)
sync - pull Fast/Deep preferences and provider adapters from your Trim cloud profile (requires trim login)

Fast = live proxy Fast only. Deep = live proxy runs Fast then Deep (LLMLingua) at your compression rate.
After Dashboard → Settings changes: trim config sync, then restart with trim start.',
    updated_at = now()
where code = 'CLI_HELP_CONFIG_LONG';
