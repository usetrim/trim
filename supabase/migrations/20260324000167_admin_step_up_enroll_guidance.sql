-- Clear step-by-step guidance for admin TOTP + passkey enrollment (worldwide operators).
-- Updates existing chrome bodies and inserts new step/hint codes.

insert into public.site_messages (code, body) values
  ('ADMIN_STEP_UP_ENROLL_CHOICE', 'You only need one method. An authenticator app works on any phone. A passkey uses this device fingerprint, face unlock, PIN, or a security key.'),
  ('ADMIN_TOTP_ENROLL_INTRO', 'Uses a free authenticator app such as Google Authenticator, Microsoft Authenticator, Authy, or 1Password.'),
  ('ADMIN_TOTP_STEP_1', 'Generate a QR code for this admin account.'),
  ('ADMIN_TOTP_STEP_2', 'Open your authenticator app, add a new account, and scan this QR code. If you cannot scan, enter the manual setup key instead.'),
  ('ADMIN_TOTP_STEP_3', 'Enter the 6-digit code currently shown in the app, then confirm to finish setup.'),
  ('ADMIN_TOTP_QR_CAPTION', 'Scan with your authenticator app'),
  ('ADMIN_TOTP_SECRET_HINT', 'Use this only if your app cannot scan the QR code. Do not share it.'),
  ('ADMIN_TOTP_CODE_PLACEHOLDER', '000000'),
  ('ADMIN_WEBAUTHN_ENROLL_INTRO', 'Uses this browser and device. After setup, you confirm sensitive actions with fingerprint, face unlock, PIN, or a security key.'),
  ('ADMIN_WEBAUTHN_STEP_1', 'Optional: give this passkey a short name so you recognize it later (for example Work laptop).'),
  ('ADMIN_WEBAUTHN_STEP_2', 'Register the passkey, then approve the browser prompt with your fingerprint, face unlock, PIN, or security key.'),
  ('ADMIN_WEBAUTHN_NAME_PLACEHOLDER', 'Work laptop'),
  ('ADMIN_WEBAUTHN_NAME_HINT', 'Optional. Helps you tell devices apart if you add more than one.'),
  ('ADMIN_STEP_UP_VERIFY_HINT', 'Open your authenticator app for the current 6-digit code, or use your passkey on this device.')
on conflict (code) do update set body = excluded.body, updated_at = now();

update public.site_messages
set body = 'Set up an authenticator app or a passkey before you can approve sensitive admin actions.',
    updated_at = now()
where code = 'ADMIN_STEP_UP_FACTOR_REQUIRED';

update public.site_messages
set body = 'Authenticator app',
    updated_at = now()
where code = 'ADMIN_TOTP_ENROLL_TITLE';

update public.site_messages
set body = 'Confirm with authenticator',
    updated_at = now()
where code = 'ADMIN_TOTP_VERIFY_TITLE';

update public.site_messages
set body = '6-digit code from your app',
    updated_at = now()
where code = 'ADMIN_TOTP_CODE_LABEL';

update public.site_messages
set body = 'Manual setup key',
    updated_at = now()
where code = 'ADMIN_TOTP_SECRET_LABEL';

update public.site_messages
set body = 'Generate QR code',
    updated_at = now()
where code = 'ADMIN_TOTP_BEGIN';

update public.site_messages
set body = 'Confirm and finish',
    updated_at = now()
where code = 'ADMIN_TOTP_CONFIRM';

update public.site_messages
set body = 'Enroll TOTP before you can approve sensitive admin actions.',
    updated_at = now()
where code = 'ADMIN_TOTP_REQUIRED';

update public.site_messages
set body = 'That code is incorrect or expired. Wait for a new code in your authenticator app and try again.',
    updated_at = now()
where code = 'ADMIN_TOTP_CODE_INVALID';

update public.site_messages
set body = 'Enter the 6-digit code from your authenticator app.',
    updated_at = now()
where code = 'ADMIN_TOTP_CODE_REQUIRED';

update public.site_messages
set body = 'Passkey',
    updated_at = now()
where code = 'ADMIN_WEBAUTHN_ENROLL_TITLE';

update public.site_messages
set body = 'Confirm with passkey',
    updated_at = now()
where code = 'ADMIN_WEBAUTHN_VERIFY_TITLE';

update public.site_messages
set body = 'Register passkey',
    updated_at = now()
where code = 'ADMIN_WEBAUTHN_BEGIN_REGISTER';

update public.site_messages
set body = 'Use passkey',
    updated_at = now()
where code = 'ADMIN_WEBAUTHN_BEGIN_ASSERT';

update public.site_messages
set body = 'Passkey name',
    updated_at = now()
where code = 'ADMIN_WEBAUTHN_NAME_LABEL';

update public.site_messages
set body = 'This browser does not support passkeys. Use an authenticator app instead, or try a current Chrome, Edge, Safari, or Firefox release.',
    updated_at = now()
where code = 'ADMIN_WEBAUTHN_BROWSER_UNSUPPORTED';

update public.site_messages
set body = 'Passkeys are not configured on this instance. Ask an operator to set ADMIN_WEBAUTHN_RP_ID or ADMIN_ALLOWED_ORIGINS, or use an authenticator app instead.',
    updated_at = now()
where code = 'ADMIN_WEBAUTHN_RP_MISSING';

update public.site_messages
set body = 'Confirm this action with your authenticator app or passkey, then try again.',
    updated_at = now()
where code = 'ADMIN_STEP_UP_REQUIRED';
