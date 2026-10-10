-- DB-driven runtime override process-cache TTL + local active-file default (no invent).

insert into public.site_messages (code, body) values
  ('RUNTIME_OVERRIDES_CACHE_MS', '5000'),
  ('RUNTIME_OVERRIDES_CACHE_MS_INVALID', 'RUNTIME_OVERRIDES_CACHE_MS must be a positive integer'),
  ('PROXY_ACTIVE_FILE_PROTECTION', 'true'),
  ('PROXY_ACTIVE_FILE_PROTECTION_INVALID', 'PROXY_ACTIVE_FILE_PROTECTION must be true or false')
on conflict (code) do nothing;
