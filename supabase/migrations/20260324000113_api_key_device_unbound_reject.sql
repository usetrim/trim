-- Reject unbound API keys at auth until at least one device is JWT-registered (or set at CLI issue).

insert into public.site_messages (code, body) values
  ('API_KEY_DEVICE_UNBOUND', 'This API key has no registered devices. Sign in to the dashboard, open Settings → API keys, and register this machine''s hardware UUID before using the key.'),
  ('API_KEY_DEVICE_HINT', 'Every API key must be device-bound. CLI login binds the issuing machine. For IDE/CI or dashboard-issued keys, register hardware under Settings (signed-in) before the key will authenticate.')
on conflict (code) do update set body = excluded.body;
