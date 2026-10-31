-- Microsoft LLMLingua coding safety: use_sentence_level_filter (default off).
-- Billing-driven only; synced to CLI with other Deep coding dials. Idempotent. No em dashes.

alter table public.billing_settings
  add column if not exists deep_use_sentence_level_filter boolean;

update public.billing_settings
set deep_use_sentence_level_filter = coalesce(deep_use_sentence_level_filter, false)
where id = 'default';

alter table public.billing_settings
  alter column deep_use_sentence_level_filter set default false,
  alter column deep_use_sentence_level_filter set not null;

comment on column public.billing_settings.deep_use_sentence_level_filter is
  'When true, pass use_sentence_level_filter into LLMLingua. Default false for coding safety (Microsoft). Synced to CLI; not inventable in optimizer.py.';

insert into public.site_messages (code, body, updated_at)
values
  (
    'ADMIN_BILLING_DEEP_USE_SENTENCE_LEVEL_FILTER',
    'Deep use_sentence_level_filter (LLMLingua)',
    now()
  ),
  (
    'ADMIN_BILLING_DEEP_USE_SENTENCE_LEVEL_FILTER_DESC',
    'When on, pass use_sentence_level_filter into LLMLingua. Keep off for coding agents (Microsoft coding safety). Independent of force_tokens. Synced to CLI from billing_settings; not inventable in the optimizer.',
    now()
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
