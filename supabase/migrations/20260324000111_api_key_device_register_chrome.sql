-- JWT-only API key device registration chrome (no auto-enroll on API-key auth).

insert into public.site_messages (code, body) values
  ('API_KEY_DEVICE_REGISTER', 'Register device'),
  ('API_KEY_DEVICE_REGISTER_PENDING', 'Registering...'),
  ('API_KEY_DEVICE_REMOVE', 'Remove device'),
  ('API_KEY_DEVICE_REMOVE_PENDING', 'Removing...'),
  ('API_KEY_DEVICE_HW_LABEL', 'Hardware UUID'),
  ('API_KEY_DEVICE_AGENT_LABEL', 'Agent ID (cli, ide, or ci)'),
  ('API_KEY_DEVICE_BOUND_LABEL', 'Device-bound'),
  ('API_KEY_DEVICE_UNBOUND_LABEL', 'Not device-bound'),
  ('API_KEY_DEVICE_COUNT_FMT', '%d device(s)'),
  ('API_KEY_DEVICE_HINT', 'CLI-auth keys bind the issuing machine. Register IDE/CI hardware from Settings (signed-in), not via the API key alone.'),
  ('API_KEY_DEVICE_HW_REQUIRED', 'Hardware UUID is required.'),
  ('API_KEY_DEVICE_NOT_FOUND', 'Device registration not found.'),
  ('IDE_CMD_COPY_HARDWARE_ID', 'Trim: Copy Hardware ID'),
  ('IDE_HARDWARE_ID_COPIED', 'Trim hardware ID copied. Register it under Dashboard → Settings → API keys.'),
  ('CI_AGENT_HINT', 'CI jobs must send X-Trim-Agent-Id: ci and X-Hardware-UUID for device-bound keys.')
on conflict (code) do nothing;
