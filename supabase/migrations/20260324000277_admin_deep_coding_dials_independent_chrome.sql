-- Coding dials apply independently of force_tokens (optimizer.py + deepopt Runtime).
-- Fix Admin DESC chrome + column comments that still said "with force_tokens".
-- Idempotent. No em dashes.

update public.site_messages
set body = 'When on, pass force_reserve_digit into LLMLingua (Microsoft coding safety). Independent of force_tokens. Synced to CLI from billing_settings; not inventable in the optimizer.',
    updated_at = now()
where code = 'ADMIN_BILLING_DEEP_FORCE_RESERVE_DIGIT_DESC';

update public.site_messages
set body = 'When on, pass drop_consecutive into LLMLingua. Independent of force_tokens. Synced to CLI from billing_settings; not inventable in the optimizer.',
    updated_at = now()
where code = 'ADMIN_BILLING_DEEP_DROP_CONSECUTIVE_DESC';

comment on column public.billing_settings.deep_force_reserve_digit is
  'When true, pass force_reserve_digit into LLMLingua (Microsoft coding safety). Independent of force_tokens. Synced to CLI; not inventable in optimizer.py.';

comment on column public.billing_settings.deep_drop_consecutive is
  'When true, pass drop_consecutive into LLMLingua. Independent of force_tokens. Synced to CLI; not inventable in optimizer.py.';
