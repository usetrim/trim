-- Dashboard-issued unbound keys: chrome for "register a device before the key works".

insert into public.site_messages (code, body) values
  ('API_KEY_FRESH_DEVICE_REQUIRED', 'Copy this key now. It will not authenticate until you register a hardware UUID below (Settings → Register device).')
on conflict (code) do update set body = excluded.body;

comment on table public.api_key_devices is
  'Allowlisted hardware UUIDs for an API key. Empty allowlist (and null api_keys.hardware_uuid) rejects API-key auth until JWT device register.';
