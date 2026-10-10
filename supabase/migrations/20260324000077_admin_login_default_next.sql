-- Admin login home (fail-closed; never reuse web LOGIN_DEFAULT_NEXT=/dashboard).
-- Ops checklist coverage for notifications + webauthn migrations.

insert into public.site_messages (code, body) values
  ('ADMIN_LOGIN_TITLE', 'Operator sign in'),
  ('ADMIN_LOGIN_DEFAULT_NEXT', '/'),
  ('ADMIN_LOGIN_DEFAULT_NEXT_MISSING', 'Admin sign-in destination is not configured. Set ADMIN_LOGIN_DEFAULT_NEXT in site_messages.'),
  ('ADMIN_CHECKLIST_MIGRATIONS_NOTIF', 'In-app notifications migration applied'),
  ('ADMIN_CHECKLIST_MIGRATIONS_WEBAUTHN', 'WebAuthn credentials migration applied'),
  ('ADMIN_CHECKLIST_MIGRATIONS_CREDITS_NAV', 'Credits nav migration applied'),
  ('ADMIN_ACCESS_REVIEW_FILENAME_FMT', 'access-review-{generated_at}.json')
on conflict (code) do update set body = excluded.body;
