-- DB-driven bounds for CLI_DAEMON_RESTART_SEC (no invent 1..3600 in binary).
-- Plus IDE chrome for post-start health failure detail (no invent English in extension).

insert into public.site_messages (code, body) values
  ('DAEMON_RESTART_MIN_SEC', '1'),
  ('DAEMON_RESTART_MAX_SEC', '3600'),
  ('DAEMON_RESTART_BOUNDS_INVALID', 'DAEMON_RESTART_MIN_SEC/MAX_SEC must be positive integers with min <= max'),
  ('IDE_AUTOSTART_HEALTH_AFTER_START_FAILED', 'health check failed after start')
on conflict (code) do nothing;
