-- Admin login chrome: fail closed on login (toast), not a dead-end dashboard page.
insert into public.site_messages (code, body) values
  (
    'ADMIN_FORBIDDEN_PAGE',
    'This account is not a platform admin on this instance.'
  ),
  (
    'ADMIN_BOOTSTRAP_REQUIRED',
    'Platform admin bootstrap is required before sign-in can complete.'
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
