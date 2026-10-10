-- Autostart enforcement polish: quit policy, fail-soft status, unset chrome, shutdown URL.

insert into public.site_messages (code, body) values
  ('IDE_AUTOSTART_STOP_ON_QUIT', 'false'),
  ('IDE_PROXY_SHUTDOWN_URL', 'http://127.0.0.1:8000/v1/control/shutdown'),
  ('IDE_AUTOSTART_STATUS_OK', 'Trim · proxy ready'),
  ('IDE_AUTOSTART_STATUS_FAILED', 'Trim · proxy unavailable'),
  ('IDE_AUTOSTART_WARN_FAILED', 'Trim could not start the local proxy. Check Output → Trim. The IDE was not blocked.'),
  ('IDE_AUTOSTART_SKIPPED_LOCAL_OFF', 'Auto-start overridden off in Trim extension settings'),
  ('IDE_AUTOSTART_SKIPPED_MANAGED', 'Auto-start skipped by managed policy (DO_NOT_TRACK / TRIM_AUTOSTART_DISABLED / autostart.off)'),
  ('IDE_AUTOSTART_SHUTDOWN_TIMEOUT_MS', '5000'),
  ('CLI_CONFIG_GET_AUTOSTART_UNSET', 'unset'),
  ('CLI_AUTOSTART_SYNC_DAEMON_OK', 'Local daemon install/uninstall aligned with synced auto-start preference.'),
  ('CLI_HELP_STOP_SHORT', 'Stop the local Trim proxy (localhost control endpoint)'),
  ('CLI_STOP_OK', 'Local Trim proxy stop requested'),
  ('CLI_STOP_FAILED_FMT', 'Could not stop local Trim proxy (%s)'),
  ('CLI_STOP_URL_MISSING', 'Proxy shutdown URL is not configured (site_messages IDE_PROXY_SHUTDOWN_URL / CLI chrome)')
on conflict (code) do nothing;
