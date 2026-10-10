-- Align invite / default-tier / paddle payment-failure chrome with fail-closed codes.
-- Column is `body` (not `message`).

insert into public.site_messages (code, body, updated_at)
values
  ('WS_INVITE_PATH_MISSING', 'APP_PATH_INVITE_PREFIX is not configured in site_messages', now()),
  ('WS_INVITE_PATH_INVALID', 'APP_PATH_INVITE_PREFIX must be a safe absolute path', now()),
  ('DEFAULT_PLAN_TIER_MISSING', 'DEFAULT_PLAN_TIER is not configured in site_messages', now()),
  ('PADDLE_ON_PAYMENT_FAILURE', 'prevent_change', now()),
  ('PADDLE_ON_PAYMENT_FAILURE_MISSING', 'PADDLE_ON_PAYMENT_FAILURE is not configured in site_messages', now())
on conflict (code) do update set
  body = excluded.body,
  updated_at = now();
