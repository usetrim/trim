-- App path chrome + unpaid default plan tier (no client /dashboard invent).
-- Column is `body` (public.site_messages).

insert into public.site_messages (code, body, updated_at)
values
  ('APP_PATH_DASHBOARD', '/dashboard', now()),
  ('APP_PATH_TEAM', '/dashboard/team', now()),
  ('APP_PATH_SETTINGS', '/dashboard/settings', now()),
  ('APP_PATH_RECEIPTS_PREFIX', '/dashboard/receipts/', now()),
  ('APP_PATH_LOGIN', '/login', now()),
  ('DEFAULT_PLAN_TIER', 'free', now()),
  ('LOGIN_DEFAULT_NEXT_MISSING', 'Sign-in destination is not configured. Set LOGIN_DEFAULT_NEXT in site_messages.', now())
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
