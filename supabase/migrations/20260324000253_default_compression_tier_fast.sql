-- Product default: Fast Mode for new accounts (DEFAULT_COMPRESSION_TIER = fast).
-- Signup (handle_new_user) and preferences UI read this from site_messages; no invent.
-- Existing accounts that already chose Deep stay on Deep (no mass flip).

do $$
declare
  fast_tier text;
begin
  select lower(btrim(m.body)) into fast_tier
  from public.site_messages m
  where m.code = 'COMPRESSION_TIER_FAST';

  if fast_tier is null or fast_tier = '' then
    raise exception 'site_messages COMPRESSION_TIER_FAST missing';
  end if;

  update public.site_messages
  set body = fast_tier, updated_at = now()
  where code = 'DEFAULT_COMPRESSION_TIER'
    and lower(btrim(body)) is distinct from fast_tier;

  if not exists (
    select 1 from public.site_messages where code = 'DEFAULT_COMPRESSION_TIER'
  ) then
    insert into public.site_messages (code, body) values ('DEFAULT_COMPRESSION_TIER', fast_tier);
  end if;
end;
$$;

update public.site_messages
set body = 'Default compression tier for new accounts when unset in preferences (fast = Fast Mode only; deep = Deep Mode on).',
    updated_at = now()
where code = 'ADMIN_PRODUCT_MODE_DESC'
  and body is distinct from 'Default compression tier for new accounts when unset in preferences (fast = Fast Mode only; deep = Deep Mode on).';

update public.admin_product_settings
set
  default_compression_mode = 'fast',
  updated_at = now()
where id = 'default'
  and lower(btrim(coalesce(default_compression_mode, ''))) is distinct from 'fast';

-- CLI help: how to configure Fast vs Deep (default Fast for new accounts).
insert into public.site_messages (code, body) values
  (
    'CLI_HELP_CONFIG_LONG',
    E'Manage local Fast/Deep preferences under ~/.config/trim/.\n\nNew accounts default to Fast Mode. Deep Mode is optional.\n\n  trim config get\n  trim config get default-tier\n  trim config set default-tier fast\n  trim config set default-tier deep\n  trim config sync\n\nget - show local preferences\nset - write a local preference key (when logged in, default-tier also updates your Trim cloud profile)\nsync - pull Fast/Deep preferences and provider adapters from your Trim cloud profile (requires trim login)\n\nFast = live proxy Fast only. Deep = live proxy runs Fast then Deep (LLMLingua) when prefs/billing allow.\nAfter Dashboard → Settings changes: trim config sync, then restart with trim start.'
  ),
  (
    'CLI_HELP_CONFIG_SET_SHORT',
    'Set a preference (example: trim config set default-tier fast)'
  ),
  (
    'PREFERENCES_CLI_HELP_4',
    'trim config set default-tier fast'
  ),
  (
    'CLI_HELP_ROOT_LONG',
    E'Trim keeps a local Fast Mode proxy on your machine so IDE traffic uses less noisy context.\n\nEveryday: start, stop, status, stats, upgrade, autostart, cursor-link, login, logout, uninstall, setup, version\nAdvanced: tui, compress, daemon, telemetry, config (get | set | sync)\n\nFast vs Deep (new accounts default to Fast):\n  trim config get default-tier\n  trim config set default-tier fast\n  trim config set default-tier deep\n  trim config sync\n\nCursor Agent: trim cursor-link enable → trim start → trim cursor-link url (paste HTTPS …/v1; not 127.0.0.1).\n\nSimple savings check after trim start:\n  trim stats\n  open http://127.0.0.1:<port>/dashboard → Show savings detail\n\ntrim status = cloud quota. trim stats = local Deep status / savings meter.\n\nFull remove: trim uninstall (then brew/scoop/winget/npm uninstall if you installed that way).\n\nRun "trim help <command>" for details. Prefer trim start / autostart for daily use; compress and Deep Mode are file/batch tools, not the everyday IDE path.'
  )
on conflict (code) do update
set body = excluded.body, updated_at = now();
