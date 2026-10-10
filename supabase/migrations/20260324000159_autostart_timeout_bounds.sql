-- DB-driven bounds for autostart duration timeouts (settle, shutdown, health).
-- Binaries must not invent min/max; fail-closed if missing/invalid.

insert into public.site_messages (code, body) values
  ('AUTOSTART_TIMEOUT_MIN_MS', '1'),
  ('AUTOSTART_TIMEOUT_MAX_MS', '120000'),
  ('AUTOSTART_TIMEOUT_BOUNDS_INVALID', 'AUTOSTART_TIMEOUT_MIN_MS/MAX_MS must be positive integers with min <= max')
on conflict (code) do nothing;

-- Poll INVALID copy references bound codes (no invent numeric range in prose as sole source).
update public.site_messages
set body = 'IDE_AUTOSTART_PREF_POLL_MS must be 0 or within AUTOSTART_PREF_POLL_MIN_MS..AUTOSTART_PREF_POLL_MAX_MS'
where code = 'IDE_AUTOSTART_PREF_POLL_MS_INVALID';

update public.site_messages
set body = 'CLI_AUTOSTART_PREF_POLL_MS must be 0 or within AUTOSTART_PREF_POLL_MIN_MS..AUTOSTART_PREF_POLL_MAX_MS'
where code = 'CLI_AUTOSTART_PREF_POLL_MS_INVALID';
