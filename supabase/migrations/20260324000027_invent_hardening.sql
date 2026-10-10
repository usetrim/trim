-- Invent hardening: Paddle discount XOR, dialog cancel chrome, deep question required.

-- Paddle Checkout.open accepts discountId XOR discountCode, never both.
alter table public.billing_settings
  drop constraint if exists billing_settings_paddle_discount_xor;

alter table public.billing_settings
  add constraint billing_settings_paddle_discount_xor
  check (
    nullif(btrim(coalesce(paddle_discount_id, '')), '') is null
    or nullif(btrim(coalesce(paddle_discount_code, '')), '') is null
  );

insert into public.site_messages (code, body) values
  ('DIALOG_CANCEL', 'Cancel'),
  ('CLI_DEEP_QUESTION_REQUIRED', 'engine=long requires --question (no invent default)')
on conflict (code) do nothing;
