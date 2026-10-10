-- Daemon-run enforcer: poll auto_start_with_ide while proxy is running (DB-driven).
-- Manual `trim start` does not poll. 0 = disabled. Same bounds as IDE poll.

insert into public.site_messages (code, body) values
  ('CLI_AUTOSTART_PREF_POLL_MS', '60000'),
  ('CLI_AUTOSTART_PREF_POLL_MS_INVALID', 'CLI_AUTOSTART_PREF_POLL_MS must be 0 or 15000-600000'),
  ('CLI_AUTOSTART_ENFORCER_STOPPED', 'Auto-start preference is off - daemon enforcer stopping local proxy')
on conflict (code) do nothing;
