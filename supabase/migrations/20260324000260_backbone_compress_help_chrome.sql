-- Align compress help + models hf-model chrome with Microsoft Base Model catalog (parity with rates).
-- Idempotent site_messages upserts only.

insert into public.site_messages (code, body, updated_at) values
  ('CLI_HELP_COMPRESS_SHORT', 'Compress a file (Fast or Deep). Live IDE proxy also runs Deep when Preferences Deep is on (v2 uses synced Base Model).', now()),
  ('CLI_HELP_COMPRESS_LONG', $body$Fast Mode is low-latency local compression (live proxy and files). Deep Mode is stronger on-machine LLMLingua: live proxy when Preferences Deep is on, and trim compress --deep for file/batch. Engines are not loaded in Trim cloud.

Deep engines (pick one): v2 LLMLingua-2 (recommended; uses synced Base Model from trim config models), long LongLLMLingua (question-aware; requires --question), v1 LLMLingua (classic).

Keep-rate: --rate or synced compression-rate (Dashboard / trim config rates).
Base Model (v2 only): Dashboard picker or trim config models / set deep-v2-backbone (billing catalog; default mbert-base, optional xlm-roberta-large). No free-text Hugging Face box.

Licensed use still requires Trim login and network.$body$, now()),
  ('CLI_CONFIG_MODELS_HF_LABEL', 'hf-model', now()),
  ('PREFERENCES_CLI_HELP_2', 'trim compress docs.txt --deep --engine v2   |   Base Models: trim config models', now()),
  ('CLI_HELP_CONFIG_SHORT', 'Local Fast/Deep preferences (get | set | sync | rates | models)', now()),
  ('CLI_HELP_CONFIG_GET_SHORT', 'Show preferences (default-tier, deep-engine, deep-v2-backbone, compression-rate)', now()),
  ('CLI_HELP_CONFIG_SET_SHORT', 'Set a preference (example: trim config set deep-v2-backbone mbert-base)', now()),
  ('CLI_HELP_COMPRESS_FLAG_ENGINE', 'Deep engine: v2 (recommended; uses synced Base Model) | long (needs --question) | v1 (classic). Else prefs deep-engine.', now())
on conflict (code) do update set body = excluded.body, updated_at = now();

update public.site_messages
set body = $body$Manage local Fast/Deep preferences under ~/.config/trim/.

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
After Dashboard → Settings changes: trim config sync, then restart with trim start.$body$,
    updated_at = now()
where code = 'CLI_HELP_CONFIG_LONG';
