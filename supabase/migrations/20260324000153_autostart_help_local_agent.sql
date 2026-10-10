-- Autostart help chrome + daemon run gate + local-agent online window (DB-driven).

insert into public.site_messages (code, body) values
  ('CLI_HELP_AUTOSTART_SHORT', 'Everyday proxy auto-start with IDE (synced preference)'),
  ('CLI_HELP_AUTOSTART_STATUS_SHORT', 'Show whether auto-start with IDE is on'),
  ('CLI_HELP_AUTOSTART_ENABLE_SHORT', 'Enable auto-start and install OS login daemon when supported'),
  ('CLI_HELP_AUTOSTART_DISABLE_SHORT', 'Disable auto-start and uninstall OS login daemon when supported'),
  ('CLI_HELP_DAEMON_RUN_SHORT', 'Start proxy only when auto-start preference is on (used by OS login service)'),
  ('CLI_AUTOSTART_DAEMON_SKIPPED_OFF', 'Auto-start with IDE is off - daemon run exiting without starting proxy'),
  ('CLI_AUTOSTART_DAEMON_SKIPPED_UNSET', 'Auto-start preference unset - fail-closed, daemon run exiting'),
  ('LOCAL_AGENT_ONLINE_WITHIN_SEC', '900'),
  ('LOCAL_AGENT_ONLINE_WITHIN_SEC_MISSING', 'LOCAL_AGENT_ONLINE_WITHIN_SEC is not configured in site_messages'),
  ('LOCAL_AGENT_ONLINE_WITHIN_SEC_INVALID', 'LOCAL_AGENT_ONLINE_WITHIN_SEC must be a positive integer (seconds)'),
  ('PREFERENCES_LOCAL_AGENT_TITLE', 'Local agent'),
  ('PREFERENCES_LOCAL_AGENT_ONLINE', 'Online (device seen recently)'),
  ('PREFERENCES_LOCAL_AGENT_OFFLINE', 'Offline (no recent device activity)'),
  ('PREFERENCES_LOCAL_AGENT_UNKNOWN', 'Unknown (no registered devices yet)'),
  ('PREFERENCES_LOCAL_AGENT_HINT', 'Status uses last cloud heartbeat from a registered CLI or IDE device. The browser cannot probe localhost; open your IDE or run trim start on the machine to refresh.'),
  ('PREFERENCES_LOCAL_AGENT_UNAVAILABLE', 'Local agent status chrome is incomplete in site_messages'),
  ('IDE_AUTOSTART_SETTLE_MS', '1500')
on conflict (code) do nothing;
