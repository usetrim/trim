-- Enroll-once admin MFA: enrollment unlocks CRUD; no per-action confirm chrome.

update public.site_messages
set body = '',
    updated_at = now()
where code = 'ADMIN_STEP_UP_REQUIRED'
  and body is distinct from '';

update public.site_messages
set body = 'Authenticator or passkey enrolled. You can use protected admin actions.',
    updated_at = now()
where code = 'ADMIN_STEP_UP_SUCCESS';

update public.site_messages
set body = 'Authenticator / passkey is enrolled. Protected admin writes are unlocked.',
    updated_at = now()
where code = 'ADMIN_STEP_UP_ACTIVE';
