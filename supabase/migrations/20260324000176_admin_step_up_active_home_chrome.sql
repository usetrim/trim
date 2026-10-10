-- Auth step-up: enrolled home is methods; verify is opt-in; active session status.

insert into public.site_messages (code, body) values
  (
    'ADMIN_STEP_UP_ACTIVE',
    'Step-up is active for this session. Protected admin actions are unlocked.'
  ),
  (
    'ADMIN_STEP_UP_OPEN_VERIFY',
    'Verify step-up now'
  )
on conflict (code) do update set body = excluded.body, updated_at = now();

update public.site_messages
set body = 'Manage authenticator and passkey',
    updated_at = now()
where code = 'ADMIN_STEP_UP_SHOW_METHODS';

update public.site_messages
set body = 'Back to methods',
    updated_at = now()
where code = 'ADMIN_STEP_UP_SHOW_VERIFY';
