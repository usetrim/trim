-- Admin billing chrome for Deep coding dials (force_reserve_digit, drop_consecutive).
-- Idempotent. No em dashes. Labels only; values live on billing_settings.

insert into public.site_messages (code, body, updated_at)
values
  (
    'ADMIN_BILLING_DEEP_FORCE_RESERVE_DIGIT',
    'Deep force_reserve_digit (LLMLingua coding safety)',
    now()
  ),
  (
    'ADMIN_BILLING_DEEP_FORCE_RESERVE_DIGIT_DESC',
    'When on with force_tokens, pass force_reserve_digit into LLMLingua. Synced to CLI from billing_settings; not inventable in the optimizer.',
    now()
  ),
  (
    'ADMIN_BILLING_DEEP_DROP_CONSECUTIVE',
    'Deep drop_consecutive (LLMLingua)',
    now()
  ),
  (
    'ADMIN_BILLING_DEEP_DROP_CONSECUTIVE_DESC',
    'When on with force_tokens, pass drop_consecutive into LLMLingua. Synced to CLI from billing_settings; not inventable in the optimizer.',
    now()
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
