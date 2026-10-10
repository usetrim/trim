-- DB-driven local proxy failover flag (no invent FallbackUncompressed in binary).

insert into public.site_messages (code, body) values
  ('CLI_PROXY_FALLBACK_UNCOMPRESSED', 'true'),
  ('CLI_PROXY_FALLBACK_UNCOMPRESSED_INVALID', 'CLI_PROXY_FALLBACK_UNCOMPRESSED must be true or false')
on conflict (code) do nothing;
