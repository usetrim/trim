-- Status upgrade action, config sync action labels, daemon platform chrome. Idempotent. No em dashes.

insert into public.site_messages (code, body, updated_at) values
  ('CLI_STATUS_UPGRADE_CMD', 'trim upgrade', now()),
  ('CLI_CONFIG_SYNC_ACTION_AUTO_START', 'patch auto_start_with_ide', now()),
  ('CLI_CONFIG_SYNC_ACTION_COMPRESSION', 'patch compression preferences', now()),
  ('CLI_CHROME_TIMEOUT_INVALID_FMT', '%s missing or invalid', now()),
  ('CLI_DAEMON_WINDOWS_ONLY', 'windows autostart is only available on Windows', now())
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
