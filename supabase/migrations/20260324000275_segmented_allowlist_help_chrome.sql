-- Align CLI help chrome with Segmented Allowlist Deep (prose islands).
-- Idempotent. No em dashes. Bodies are shown raw (cobra Long), not via Printf.

insert into public.site_messages (code, body, updated_at)
values
  (
    'CLI_HELP_ROOT_LONG',
    $body$Trim keeps a local Fast Mode proxy on your machine so IDE traffic uses less noisy context.

Everyday: start, stop, status, stats, upgrade, autostart, cursor-link, login, logout, uninstall, setup, version
Advanced: tui, compress, daemon, telemetry, config (get | set | sync | rates | models)

Fast vs Deep (new accounts default to Deep = Fast then Segmented Allowlist Deep on prose islands):
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
    now()
  ),
  (
    'CLI_HELP_CONFIG_LONG',
    $body$Manage local Fast/Deep preferences under ~/.config/trim/.

New accounts default to Deep Mode (live proxy runs Fast then Segmented Allowlist Deep on prose islands). Switch to Fast anytime.

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

Fast = live proxy Fast only. Deep = live proxy runs Fast then Segmented Allowlist Deep (LLMLingua-2 on prose islands; tools/paths frozen) at your compression rate and Base Model.
After Dashboard → Settings changes: trim config sync, then restart with trim start.$body$,
    now()
  ),
  (
    'CLI_HELP_COMPRESS_LONG',
    $body$Fast Mode is low-latency local compression (live proxy and files). Deep Mode is stronger on-machine LLMLingua: live proxy runs Segmented Allowlist Deep when Preferences Deep is on (prose islands only; tools/paths/schemas frozen), and trim compress --deep for file/batch. Engines are not loaded in Trim cloud.

Deep engines (pick one): v2 LLMLingua-2 (recommended; uses synced Base Model from trim config models), long LongLLMLingua (question-aware; requires --question), v1 LLMLingua (classic).

Keep-rate: --rate or synced compression-rate (Dashboard / trim config rates).
Base Model (v2 only): Dashboard picker or trim config models / set deep-v2-backbone (billing catalog; default mbert-base, optional xlm-roberta-large). No free-text Hugging Face box.

Licensed use still requires Trim login and network.$body$,
    now()
  ),
  (
    'CLI_HELP_STATS_LONG',
    $body$Reads the live local proxy (/v1/stats) and prints clear Deep status + Deep stage savings, then the raw JSON.

Useful when Claude Code / Cursor shows 0% whole-body Saved: check Deep status and Deep stage (Segmented Allowlist Deep = prose islands only; tools/paths/schemas stay frozen).

Everyday:
  trim start
  trim stats
  open http://127.0.0.1:<TRIM_PORT>/dashboard → Show savings detail

Use --tui for the Bubble Tea live dashboard (same as trim tui).$body$,
    now()
  ),
  (
    'CLI_HELP_COMPRESS_SHORT',
    'Compress a file (Fast or Deep). Live IDE proxy also runs Segmented Allowlist Deep when Preferences Deep is on (v2 uses synced Base Model).',
    now()
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
