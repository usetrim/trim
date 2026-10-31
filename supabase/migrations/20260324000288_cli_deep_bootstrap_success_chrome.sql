-- Deep bootstrap success chrome + help: local llmlingua one-time install UX.
-- Aligns site_messages with CLI chrome_defaults / MessageForCode. Idempotent. No em dashes.

insert into public.site_messages (code, body, updated_at) values
  ('CLI_DEEP_BOOTSTRAP_REQS', 'Installing Deep Mode dependencies on this machine (not in Trim cloud)…', now()),
  ('CLI_DEEP_BOOTSTRAP_PIP', 'Installing Deep Mode dependencies on this machine (not in Trim cloud)…', now()),
  ('CLI_DEEP_BOOTSTRAP_OK', 'Deep Mode dependencies installed on this machine', now()),
  ('CLI_DEEP_BOOTSTRAP_OK_DETAIL', 'Microsoft LLMLingua (llmlingua) and requirements are ready for trim compress --deep and live Deep. Not installed in Trim cloud.', now()),
  ('CLI_DEEP_AUTO_INSTALL', 'Deep Mode runtime not found. Installing locally once…', now()),
  ('CLI_HELP_COMPRESS_FLAG_BOOTSTRAP', 'One-time local install of Deep Mode deps (Microsoft LLMLingua / llmlingua via pip). Prints a green success banner when complete. Same step after npm, brew, curl/irm, Scoop, or winget.', now())
on conflict (code) do update
set body = excluded.body,
    updated_at = now();

update public.site_messages
set body = $body$Fast Mode is low-latency local compression (live proxy and files). Deep Mode is stronger on-machine LLMLingua: live proxy runs Segmented Allowlist Deep when Preferences Deep is on (each last-user prose island in place, order preserved beside paths; tools/paths/schemas frozen), and trim compress --deep for file/batch. Engines are not loaded in Trim cloud.

One-time local Deep deps (every install channel - npm, Homebrew, curl/irm, deb/rpm, Scoop, winget):
  trim compress --bootstrap
That runs Python pip on this PC only (llmlingua + requirements-deep.txt beside optimizer.py). Success prints a green banner like trim start. Package managers do not pip-install LLMLingua.

Deep engines (pick one): v2 LLMLingua-2 (recommended; uses synced Base Model from trim config models), long LongLLMLingua (question-aware; requires --question), v1 LLMLingua (classic).

Keep-rate: --rate or synced compression-rate (Dashboard / trim config rates).
Base Model (v2 only): Dashboard picker or trim config models / set deep-v2-backbone (billing catalog; default mbert-base, optional xlm-roberta-large). No free-text Hugging Face box.

Licensed use still requires Trim login and network.
Docs: https://use-trim.com/docs/installation#deep-runtime$body$,
    updated_at = now()
where code = 'CLI_HELP_COMPRESS_LONG';

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

One-time Deep deps on this PC (not Trim cloud):
  trim compress --bootstrap

Cursor Agent: trim cursor-link enable → trim start → trim cursor-link url (paste HTTPS …/v1; not 127.0.0.1).

Simple savings check after trim start:
  trim stats
  open http://127.0.0.1:<port>/dashboard → Show savings detail

trim status = cloud quota. trim stats = local Deep status / savings meter.

Full remove: trim uninstall (then brew/scoop/winget/npm uninstall if you installed that way).

Run "trim help <command>" for details. Prefer trim start / autostart for daily use (new accounts default Deep = Fast then Segmented Allowlist Deep on the live proxy). Use trim compress --bootstrap once, then trim compress --deep for file/batch.$body$,
    updated_at = now()
where code = 'CLI_HELP_ROOT_LONG';

update public.site_messages
set body = 'trim compress --bootstrap installs Deep Mode deps on this machine (local pip / llmlingua). Success prints a green banner; not Trim cloud.',
    updated_at = now()
where code = 'PREFERENCES_CLI_HELP_3';
