-- Segmented Allowlist Deep coding dials: force_reserve_digit + drop_consecutive
-- from billing_settings only (no invent True in optimizer.py). Idempotent.

alter table public.billing_settings
  add column if not exists deep_force_reserve_digit boolean,
  add column if not exists deep_drop_consecutive boolean;

update public.billing_settings
set deep_force_reserve_digit = coalesce(deep_force_reserve_digit, true),
    deep_drop_consecutive = coalesce(deep_drop_consecutive, true),
    updated_at = now()
where true;

alter table public.billing_settings
  alter column deep_force_reserve_digit set not null,
  alter column deep_drop_consecutive set not null;

comment on column public.billing_settings.deep_force_reserve_digit is
  'When true with force_tokens, pass force_reserve_digit into LLMLingua (Microsoft coding safety). Synced to CLI; not inventable in optimizer.py.';
comment on column public.billing_settings.deep_drop_consecutive is
  'When true with force_tokens, pass drop_consecutive into LLMLingua. Synced to CLI; not inventable in optimizer.py.';
