-- Cloud proxy options from site_messages (no invent true / 5s in binary).

insert into public.site_messages (code, body) values
  ('PROXY_ACTIVE_FILE_PROTECTION', 'true'),
  ('PROXY_ACTIVE_FILE_PROTECTION_INVALID', 'PROXY_ACTIVE_FILE_PROTECTION must be true or false'),
  ('PROXY_OPTIONS_REFRESH_MS', '5000'),
  ('PROXY_OPTIONS_REFRESH_MS_INVALID', 'PROXY_OPTIONS_REFRESH_MS must be a positive integer')
on conflict (code) do nothing;
