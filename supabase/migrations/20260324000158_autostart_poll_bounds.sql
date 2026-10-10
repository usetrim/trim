-- DB-driven bounds for autostart pref poll intervals (no invent min/max in binaries).

insert into public.site_messages (code, body) values
  ('AUTOSTART_PREF_POLL_MIN_MS', '15000'),
  ('AUTOSTART_PREF_POLL_MAX_MS', '600000'),
  ('AUTOSTART_PREF_POLL_BOUNDS_INVALID', 'AUTOSTART_PREF_POLL_MIN_MS/MAX_MS must be positive integers with min <= max')
on conflict (code) do nothing;
