-- Autostart: DB-driven timeouts + managed-off / DO_NOT_TRACK-style gates.

insert into public.site_messages (code, body) values
  ('CLI_PROXY_HEALTH_TIMEOUT_MS', '2000'),
  ('CLI_PROXY_SHUTDOWN_TIMEOUT_MS', '5000'),
  ('CLI_HTTP_SHUTDOWN_TIMEOUT_MS', '5000'),
  ('IDE_AUTOSTART_SHUTDOWN_TIMEOUT_MS', '5000'),
  ('CLI_AUTOSTART_SKIPPED_MANAGED', 'Auto-start skipped (TRIM_AUTOSTART_DISABLED=1 or local autostart.off)'),
  ('CLI_AUTOSTART_SKIPPED_DNT', 'Auto-start skipped (DO_NOT_TRACK=1)'),
  ('IDE_AUTOSTART_SKIPPED_MANAGED', 'Auto-start skipped by managed policy (DO_NOT_TRACK / TRIM_AUTOSTART_DISABLED / autostart.off)'),
  ('CLI_AUTOSTART_MANAGED_OFF_HINT', 'Managed off: unset TRIM_AUTOSTART_DISABLED and remove ~/.config/trim/autostart.off, then trim autostart enable'),
  ('CLI_STOP_ON_DISABLE_OK', 'Requested stop of running local proxy after auto-start disable'),
  ('CLI_DAEMON_RESTART_SEC', '3')
on conflict (code) do nothing;

-- Honest one-click label (control plane copy; browser does not launch Trim).
update public.site_messages
set body = 'Starts local Trim when this IDE opens'
where code = 'PREFERENCES_AUTO_START_LABEL';

update public.site_messages
set body = 'Keeps the local Trim proxy ready while you work (structural Fast path on the proxy - not Deep/ML). Uncheck anytime. The browser cannot start Trim by itself - CLI, daemon, or IDE extension apply this setting on your machine.'
where code = 'PREFERENCES_AUTO_START_HINT';
