-- Seed ACTION:/PENDING: chrome for JWT device register/remove (ActionLabelForCode / PendingLabelForCode).
-- Plain API_KEY_DEVICE_* messages already exist from 00111; handlers also MessageForCode-fallback.

insert into public.site_messages (code, body) values
  ('ACTION:API_KEY_DEVICE_REGISTER', 'Register device'),
  ('PENDING:API_KEY_DEVICE_REGISTER', 'Registering...'),
  ('ACTION:API_KEY_DEVICE_REMOVE', 'Remove device'),
  ('PENDING:API_KEY_DEVICE_REMOVE', 'Removing...')
on conflict (code) do nothing;
