-- Live Deep policy dials (DB-driven; no invent in CLI/proxy).
-- when-to-invoke, OOM policy, engine model ids, optional warm-up on trim start.

alter table public.billing_settings
  add column if not exists live_deep_min_input_tokens int,
  add column if not exists live_deep_oom_policy text,
  add column if not exists live_deep_warmup_on_start boolean,
  add column if not exists deep_v1_model text,
  add column if not exists deep_v2_model text,
  add column if not exists deep_long_model text,
  add column if not exists deep_v2_force_tokens jsonb;

-- Seed once for existing installs (ops may change in Admin → Billing settings).
update public.billing_settings
set
  live_deep_min_input_tokens = coalesce(live_deep_min_input_tokens, 256),
  live_deep_oom_policy = coalesce(nullif(btrim(live_deep_oom_policy), ''), 'fail'),
  live_deep_warmup_on_start = coalesce(live_deep_warmup_on_start, false),
  deep_v1_model = coalesce(nullif(btrim(deep_v1_model), ''), 'NousResearch/Llama-2-7b-hf'),
  deep_v2_model = coalesce(nullif(btrim(deep_v2_model), ''), 'microsoft/llmlingua-2-bert-base-multilingual-cased-meetingbank'),
  deep_long_model = coalesce(nullif(btrim(deep_long_model), ''), 'NousResearch/Llama-2-7b-hf'),
  deep_v2_force_tokens = coalesce(deep_v2_force_tokens, '["\n","?","!"]'::jsonb)
where id = 'default';

alter table public.billing_settings
  alter column live_deep_min_input_tokens set not null,
  alter column live_deep_oom_policy set not null,
  alter column live_deep_warmup_on_start set not null,
  alter column deep_v1_model set not null,
  alter column deep_v2_model set not null,
  alter column deep_long_model set not null,
  alter column deep_v2_force_tokens set not null;

alter table public.billing_settings
  drop constraint if exists billing_settings_live_deep_min_input_tokens_check;
alter table public.billing_settings
  add constraint billing_settings_live_deep_min_input_tokens_check
  check (live_deep_min_input_tokens >= 0 and live_deep_min_input_tokens <= 1000000);

alter table public.billing_settings
  drop constraint if exists billing_settings_live_deep_oom_policy_check;
alter table public.billing_settings
  add constraint billing_settings_live_deep_oom_policy_check
  check (live_deep_oom_policy in ('fail', 'skip'));

alter table public.billing_settings
  drop constraint if exists billing_settings_deep_models_nonempty_check;
alter table public.billing_settings
  add constraint billing_settings_deep_models_nonempty_check
  check (
    length(btrim(deep_v1_model)) > 0
    and length(btrim(deep_v2_model)) > 0
    and length(btrim(deep_long_model)) > 0
    and jsonb_typeof(deep_v2_force_tokens) = 'array'
  );

comment on column public.billing_settings.live_deep_min_input_tokens is
  'Live proxy skips Deep when estimated compressible input tokens are below this (0 = invoke on all turns). Synced to CLI via preferences.';
comment on column public.billing_settings.live_deep_oom_policy is
  'fail = fail-closed on Deep OOM/memory errors; skip = Fast-only for that request. Synced to CLI via preferences.';
comment on column public.billing_settings.live_deep_warmup_on_start is
  'When true, trim start warms the local Deep engine once after binding prefs (latency). Synced to CLI via preferences.';
comment on column public.billing_settings.deep_v1_model is
  'HuggingFace / local model id for Deep engine v1. Synced to CLI; not inventable in code.';
comment on column public.billing_settings.deep_v2_model is
  'HuggingFace / local model id for Deep engine v2. Synced to CLI; not inventable in code.';
comment on column public.billing_settings.deep_long_model is
  'HuggingFace / local model id for Deep engine long. Synced to CLI; not inventable in code.';
comment on column public.billing_settings.deep_v2_force_tokens is
  'JSON string array of force_tokens for LLMLingua-2. Synced to CLI; not inventable in code.';

insert into public.site_messages (code, body, updated_at)
values
  (
    'PREFERENCES_LIVE_DEEP_MIN_HINT',
    'Live Deep runs only when estimated compressible input tokens are at least live_deep_min_input_tokens from billing settings (0 = every turn). Synced by trim config sync.',
    now()
  ),
  (
    'CLI_PROXY_DEEP_SKIPPED_MIN_FMT',
    'Live Deep skipped (input ~%d tokens < min %d). Fast Mode only for this request.',
    now()
  ),
  (
    'CLI_PROXY_DEEP_OOM_SKIPPED',
    'Live Deep skipped after OOM/memory error (billing live_deep_oom_policy=skip). Fast Mode only for this request.',
    now()
  ),
  (
    'CLI_PROXY_DEEP_RUNTIME_REQUIRED',
    'Live Deep requires synced billing Deep models and oom policy (trim config sync). Device map stays local: TRIM_DEEP_DEVICE_MAP.',
    now()
  ),
  (
    'CLI_PROXY_LIVE_DEEP_ENABLED_FMT',
    'Live Deep Mode on (engine %s, target_token %d, min_input_tokens %d, oom=%s). Synced from account preferences / billing settings.',
    now()
  ),
  (
    'DOCS_FAST_VS_DEEP_LIVE',
    'Live proxy runs Fast Mode on every turn. When account preferences set compression_tier=deep, the local proxy also runs Deep Mode (LLMLingua) after Fast when input tokens meet billing live_deep_min_input_tokens (0 = all turns). Models and OOM policy come from billing settings via trim config sync.',
    now()
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();

update public.site_messages
set body = 'When Deep Mode is on, the live IDE proxy (trim start) runs Fast then Deep (LLMLingua) using your engine and target_token from this page, subject to billing live_deep_min_input_tokens. Off = Fast only. Trim cloud never loads Deep engines; auth, quotas, and billing still use Trim cloud.',
    updated_at = now()
where code = 'PREFERENCES_PAGE_DESCRIPTION';

update public.site_messages
set body = 'On = Deep Mode for live IDE proxy (trim start) and trim compress. Live Deep also respects billing min input tokens and OOM policy (synced). Off = Fast Mode only on the live proxy and for compress defaults. Engine and target apply whenever Deep is on. Run trim config sync after saving.',
    updated_at = now()
where code = 'PREFERENCES_DEEP_HINT';

insert into public.site_messages (code, body, updated_at)
values
  ('ADMIN_BILLING_LIVE_DEEP_MIN', 'Live Deep min input tokens', now()),
  ('ADMIN_BILLING_LIVE_DEEP_MIN_DESC', 'Skip Deep on the live proxy when estimated input tokens are below this (0 = every turn). Synced to CLI.', now()),
  ('ADMIN_BILLING_LIVE_DEEP_OOM', 'Live Deep OOM policy', now()),
  ('ADMIN_BILLING_LIVE_DEEP_OOM_DESC', 'fail = fail-closed on memory errors; skip = Fast-only for that request.', now()),
  ('ADMIN_BILLING_LIVE_DEEP_WARMUP', 'Warm Deep on trim start', now()),
  ('ADMIN_BILLING_LIVE_DEEP_WARMUP_DESC', 'When on, trim start warms the local Deep engine once after binding prefs.', now()),
  ('ADMIN_BILLING_DEEP_V1_MODEL', 'Deep v1 model id', now()),
  ('ADMIN_BILLING_DEEP_V2_MODEL', 'Deep v2 model id', now()),
  ('ADMIN_BILLING_DEEP_LONG_MODEL', 'Deep long model id', now()),
  ('ADMIN_BILLING_DEEP_V2_FORCE_TOKENS', 'Deep v2 force_tokens JSON', now())
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
