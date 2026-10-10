-- Product runtime row labels + segment credits join + UI join seps (no client invent).

insert into public.site_messages (code, body) values
  ('ADMIN_RUNTIME_DEPLOYMENT_MODE', 'Deployment mode'),
  ('ADMIN_RUNTIME_COMPRESSION_MODE_ENV', 'Compression mode (env)'),
  ('ADMIN_RUNTIME_MIN_CLI_VERSION', 'Effective min CLI'),
  ('ADMIN_RUNTIME_MIN_CLI_VERSION_ENV', 'Min CLI (env)'),
  ('ADMIN_RUNTIME_POW_DIFFICULTY', 'PoW difficulty (runtime)'),
  ('ADMIN_RUNTIME_RATE_LIMIT_IP', 'Rate limit IP/min (runtime)'),
  ('ADMIN_RUNTIME_RATE_LIMIT_USER', 'Rate limit user/min (runtime)'),
  ('ADMIN_RUNTIME_MAX_ACCOUNTS_HW', 'Max accounts per hardware (runtime)'),
  ('ADMIN_RUNTIME_MAX_ACCOUNTS_JA4', 'Max accounts per JA4 (runtime)'),
  ('ADMIN_RUNTIME_CF_THREAT_SCORE_MIN', 'CF threat score min (runtime)'),
  ('ADMIN_RUNTIME_GEOLITE_ASN_PATH', 'GeoLite ASN MMDB'),
  ('ADMIN_RUNTIME_GEOLITE_ANON_PATH', 'GeoLite anonymous MMDB'),
  ('ADMIN_RUNTIME_CHEAP_MODEL', 'Cheap model'),
  ('ADMIN_RUNTIME_ROUTE_MAX_TOKENS', 'Route max tokens'),
  ('ADMIN_RUNTIME_SMTP_CONFIGURED', 'SMTP configured'),
  ('ADMIN_SEGMENT_CREDITS_FMT', '{used}/{limit}'),
  ('ADMIN_UI_SEP_DOT', ' · '),
  ('ADMIN_UI_SEP_COMMA', ', '),
  ('ADMIN_META_FIELD_FMT', '{label}: {value}')
on conflict (code) do update set body = excluded.body;
