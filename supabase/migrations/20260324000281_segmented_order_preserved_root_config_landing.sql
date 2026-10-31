-- Segmented Allowlist Deep: ROOT/CONFIG/LANDING/PREFERENCES chrome aligned to
-- per-island reassembly (order preserved beside paths). Idempotent. No em dashes.

update public.site_messages
set body = $body$Trim keeps a local Fast Mode proxy on your machine so IDE traffic uses less noisy context.

Everyday: start, stop, status, stats, upgrade, autostart, cursor-link, login, logout, uninstall, setup, version
Advanced: tui, compress, daemon, telemetry, config (get | set | sync | rates | models)

Fast vs Deep (new accounts default to Deep = Fast then Segmented Allowlist Deep on each last-user prose island in place, order preserved beside paths):
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

Run "trim help <command>" for details. Prefer trim start / autostart for daily use (new accounts default Deep = Fast then Segmented Allowlist Deep on the live proxy). Use trim compress --deep for file/batch.$body$,
    updated_at = now()
where code = 'CLI_HELP_ROOT_LONG';

update public.site_messages
set body = $body$Manage local Fast/Deep preferences under ~/.config/trim/.

New accounts default to Deep Mode (live proxy runs Fast then Segmented Allowlist Deep on each last-user prose island in place, order preserved beside paths). Switch to Fast anytime.

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

Fast = live proxy Fast only. Deep = live proxy runs Fast then Segmented Allowlist Deep (LLMLingua-2 on each last-user prose island in place, order preserved beside paths; tools/paths frozen) at your compression rate and Base Model.
After Dashboard → Settings changes: trim config sync, then restart with trim start.$body$,
    updated_at = now()
where code = 'CLI_HELP_CONFIG_LONG';

update public.site_messages
set body = 'Start with Fast Mode on the live proxy. Turn Deep Mode on in Preferences (then trim config sync) so trim start runs Segmented Allowlist Deep after Fast (each last-user prose island in place, order preserved beside paths). Or use trim compress --deep for file/batch jobs.',
    updated_at = now()
where code in ('LANDING_WHY_4_BODY', 'DOCS_FAST_VS_DEEP_SUMMARY');

update public.site_messages
set body = 'Deep Mode runs on your machine for live IDE proxy (trim start: Segmented Allowlist Deep, each last-user prose island in place, order preserved beside paths) and trim compress. Engines are not loaded in Trim cloud; auth and quotas still use Trim cloud when licensed.',
    updated_at = now()
where code = 'PREFERENCES_NOTE';
