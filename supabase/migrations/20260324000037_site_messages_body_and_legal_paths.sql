-- Invent fix: site_messages column is `body` (not `message`).
-- Re-seed rows from migrations 35/36 that used the wrong column name.

insert into public.site_messages (code, body, updated_at)
values
  ('LOGIN_DEFAULT_NEXT', '/dashboard', now()),
  ('API_KEY_PREFIX_ELLIPSIS', '...', now()),
  ('LOCAL_PREVIEW_TRUNC_SUFFIX', E'\n… (truncated)', now()),
  ('LOCAL_TUI_TRUNC_SUFFIX', '…', now()),
  ('APP_PATH_DASHBOARD', '/dashboard', now()),
  ('APP_PATH_TEAM', '/dashboard/team', now()),
  ('APP_PATH_SETTINGS', '/dashboard/settings', now()),
  ('APP_PATH_RECEIPTS_PREFIX', '/dashboard/receipts/', now()),
  ('APP_PATH_LOGIN', '/login', now()),
  ('APP_PATH_PRIVACY', '/privacy', now()),
  ('APP_PATH_TERMS', '/terms', now()),
  ('APP_PATH_HOME', '/', now()),
  ('APP_PATH_UPGRADE', '/dashboard', now()),
  ('APP_PATH_AUTH_CALLBACK', '/auth/callback', now()),
  ('APP_PATH_CLI_AUTH', '/cli/auth', now()),
  ('APP_PATH_INVITE_PREFIX', '/invite/', now()),
  ('DEFAULT_PLAN_TIER', 'free', now()),
  ('LOGIN_DEFAULT_NEXT_MISSING', 'Sign-in destination is not configured. Set LOGIN_DEFAULT_NEXT in site_messages.', now())
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
