-- Raise step-up TTL chrome + allow 12h elevation after one verify.
-- Operator UX: authenticate once (TOTP/passkey step-up), then writes work for the session.

update public.site_messages
set body = 'Set TRIM_ADMIN_STEP_UP_TTL_SEC (60-43200) before step-up can run.'
where code = 'ADMIN_STEP_UP_TTL_MISSING'
  and body is distinct from 'Set TRIM_ADMIN_STEP_UP_TTL_SEC (60-43200) before step-up can run.';
