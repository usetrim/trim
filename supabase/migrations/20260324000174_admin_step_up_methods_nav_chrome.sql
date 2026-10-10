-- Auth step-up: keep methods panel reachable after first factor enrolls.

insert into public.site_messages (code, body) values
  (
    'ADMIN_STEP_UP_METHODS_TITLE',
    'Authenticator and passkey'
  ),
  (
    'ADMIN_STEP_UP_METHODS_SUBTITLE',
    'Add either method, or both. You can return here anytime to finish setup.'
  ),
  (
    'ADMIN_STEP_UP_METHOD_ENROLLED',
    'Enrolled'
  ),
  (
    'ADMIN_STEP_UP_SHOW_METHODS',
    'Manage authenticator and passkey'
  ),
  (
    'ADMIN_STEP_UP_SHOW_VERIFY',
    'Back to verify'
  )
on conflict (code) do update set body = excluded.body, updated_at = now();
