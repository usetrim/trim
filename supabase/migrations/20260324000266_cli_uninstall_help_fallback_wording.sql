-- Align uninstall long-help fallback wording with CLI next-steps (detected channels).
-- Idempotent upsert.

insert into public.site_messages (code, body, updated_at)
values
  (
    'CLI_HELP_UNINSTALL_LONG',
    E'Fully removes Trim from this machine:\n  - stops the local proxy, Cursor Link tunnel, and other Trim processes\n  - removes the OS login daemon\n  - clears CLI credentials\n  - reverts IDE Base URL overrides from trim setup\n  - deletes ~/.trim and ~/.config/trim, Deep caches, sidecars, and the CLI binary (plus Windows user PATH / Apps entry)\n  - when installed via npm / Scoop / Homebrew / winget, schedules that package uninstall after exit (so Windows can release the running binary)\n\nUse --keep-data to skip deleting local data directories, Deep caches, the CLI binary/package, and Windows LocalAppData Trim.\n\nIDE extension is separate: Trim: Clear API Key, then uninstall Trim IDE from the Extensions view.\nIf a detected package install was not auto-removed after exit:\n  brew uninstall trim\n  scoop uninstall trim\n  winget uninstall Trim.CLI\n  npm uninstall -g @usetrim/trim',
    now()
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
