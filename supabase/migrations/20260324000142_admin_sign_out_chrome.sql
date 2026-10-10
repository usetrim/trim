-- Operator console sign-out chrome (header action + pending).
insert into public.site_messages (code, body) values
  ('ADMIN_SIGN_OUT', 'Sign out'),
  ('ADMIN_SIGN_OUT_PENDING', 'Signing out...')
on conflict (code) do nothing;
