-- Success toasts for admin step-up verify + factor enroll.

insert into public.site_messages (code, body) values
  (
    'ADMIN_STEP_UP_SUCCESS',
    'Step-up verified. You can continue with protected admin actions.'
  ),
  (
    'ADMIN_TOTP_ENROLL_SUCCESS',
    'Authenticator app enrolled successfully.'
  ),
  (
    'ADMIN_WEBAUTHN_ENROLL_SUCCESS',
    'Passkey enrolled successfully.'
  )
on conflict (code) do update set body = excluded.body, updated_at = now();
