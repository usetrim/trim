-- Keep-data wording + ensure scheduled/process chrome codes stay accurate.
-- Idempotent upsert (safe to re-apply).

insert into public.site_messages (code, body, updated_at)
values
  (
    'CLI_HELP_UNINSTALL_FLAG_KEEP_DATA',
    'Keep ~/.trim, ~/.config/trim, Deep Mode caches, the CLI binary/package, and Windows LocalAppData Trim (only stop / unregister Always-on / logout / setup revert)',
    now()
  ),
  (
    'CLI_HELP_UNINSTALL_LONG',
    E'Fully removes Trim from this machine:\n  - stops the local proxy, Cursor Link tunnel, and other Trim processes\n  - removes the OS login daemon\n  - clears CLI credentials\n  - reverts IDE Base URL overrides from trim setup\n  - deletes ~/.trim and ~/.config/trim, Deep caches, sidecars, and the CLI binary/PATH\n  - when installed via npm / Scoop / Homebrew / winget, schedules that package uninstall after exit (so Windows can release the running binary)\n\nUse --keep-data to skip deleting local data directories, Deep caches, the CLI binary/package, and Windows LocalAppData Trim.\n\nIDE extension is separate: Trim: Clear API Key, then uninstall Trim IDE from the Extensions view.\nFallback if a package manager was not auto-detected:\n  brew uninstall trim\n  scoop uninstall trim\n  winget uninstall Trim.CLI\n  npm uninstall -g @usetrim/trim',
    now()
  ),
  (
    'CLI_UNINSTALL_PROCESSES_STOPPED',
    'Stopped other Trim processes on this machine.',
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
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
