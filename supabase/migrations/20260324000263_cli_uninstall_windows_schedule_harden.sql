-- Reinforce uninstall chrome after Windows post-exit helper harden
-- (temp .ps1 + MoveFileEx reboot fallback). Idempotent upsert.

insert into public.site_messages (code, body, updated_at)
values
  (
    'CLI_UNINSTALL_BINARY_MANUAL_FMT',
    'Could not schedule binary removal. Delete manually after closing all Trim windows (or reboot if Windows deferred the delete): %s',
    now()
  ),
  (
    'CLI_UNINSTALL_BINARY_SCHEDULED_FMT',
    'Scheduled CLI binary removal after exit: %s',
    now()
  ),
  (
    'CLI_UNINSTALL_PKG_SCHEDULED_FMT',
    'Scheduled package removal after exit: %s',
    now()
  ),
  (
    'CLI_HELP_UNINSTALL_SHORT',
    'Fully remove Trim from this machine (proxy, daemon, data, Deep caches, binary, package install)',
    now()
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
