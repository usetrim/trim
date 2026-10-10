-- While IDE stays open: re-check auto_start_with_ide and stop/start proxy (DB-driven poll ms).
-- 0 = poll disabled (activate-only). Positive = interval in milliseconds.

insert into public.site_messages (code, body) values
  ('IDE_AUTOSTART_PREF_POLL_MS', '60000'),
  ('IDE_AUTOSTART_PREF_POLL_MS_INVALID', 'IDE_AUTOSTART_PREF_POLL_MS must be 0 or 15000-600000')
on conflict (code) do nothing;

-- Honest control-plane note: local enforcer re-checks while IDE is open.
update public.site_messages
set body = 'This preference syncs to your device via trim config sync. Local enforcers: trim autostart, trim daemon, and the Trim IDE extension (re-checks while the IDE stays open). The browser cannot start or stop Trim by itself.'
where code = 'PREFERENCES_AUTO_START_NOTE';
