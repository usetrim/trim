-- Force-logout Redis marker TTL (DB-driven; no invent 24h in Go).

alter table public.admin_retention_settings
  add column if not exists force_logout_ttl_sec int not null default 86400
  check (force_logout_ttl_sec >= 60 and force_logout_ttl_sec <= 2592000);

comment on column public.admin_retention_settings.force_logout_ttl_sec is
  'Redis TTL seconds for admin:force_logout:* markers; fail-closed if missing/invalid at write time';

insert into public.site_messages (code, body) values
  ('ADMIN_FORCE_LOGOUT_TTL_MISSING', 'Set admin_retention_settings.force_logout_ttl_sec before force-logout.'),
  ('ADMIN_COMPLIANCE_FORCE_LOGOUT_TTL', 'Force-logout marker TTL (seconds)')
on conflict (code) do update set body = excluded.body;
