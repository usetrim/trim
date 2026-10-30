-- Full uninstall: stop other Trim processes, schedule running-binary + package-manager
-- removal after exit (Windows cannot delete a mapped exe). Idempotent upsert.

insert into public.site_messages (code, body, updated_at)
values
  (
    'CLI_HELP_UNINSTALL_SHORT',
    'Fully remove Trim from this machine (proxy, daemon, data, Deep caches, binary, package install)',
    now()
  ),
  (
    'CLI_HELP_UNINSTALL_LONG',
    E'Fully removes Trim from this machine:\n  - stops the local proxy, Cursor Link tunnel, and other Trim processes\n  - removes the OS login daemon\n  - clears CLI credentials\n  - reverts IDE Base URL overrides from trim setup\n  - deletes ~/.trim and ~/.config/trim, Deep caches, sidecars, and the CLI binary/PATH\n  - when installed via npm / Scoop / Homebrew / winget, schedules that package uninstall after exit (so Windows can release the running binary)\n\nUse --keep-data to skip deleting local data directories, Deep caches, and the CLI binary/package.\n\nIDE extension is separate: Trim: Clear API Key, then uninstall Trim IDE from the Extensions view.\nFallback if a package manager was not auto-detected:\n  brew uninstall trim\n  scoop uninstall trim\n  winget uninstall Trim.CLI\n  npm uninstall -g @usetrim/trim',
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
  ),
  (
    'CLI_UNINSTALL_BINARY_MANUAL_FMT',
    'Could not schedule binary removal. Delete manually after closing all Trim windows: %s',
    now()
  ),
  (
    'CLI_UNINSTALL_NEXT_PKG_BREW',
    'Fallback if Homebrew install was not auto-removed: brew uninstall trim',
    now()
  ),
  (
    'CLI_UNINSTALL_NEXT_PKG_SCOOP',
    'Fallback if Scoop install was not auto-removed: scoop uninstall trim',
    now()
  ),
  (
    'CLI_UNINSTALL_NEXT_PKG_WINGET',
    'Fallback if winget install was not auto-removed: winget uninstall Trim.CLI',
    now()
  ),
  (
    'CLI_UNINSTALL_NEXT_PKG_NPM',
    'Fallback if npm install was not auto-removed: npm uninstall -g @usetrim/trim',
    now()
  ),
  (
    'CLI_HELP_ROOT_LONG',
    E'Trim keeps a local Fast Mode proxy on your machine so IDE traffic uses less noisy context.\n\nEveryday: start, stop, status, stats, upgrade, autostart, cursor-link, login, logout, uninstall, setup, version\nAdvanced: tui, compress, daemon, telemetry, config (get | set | sync | rates | models)\n\nFast vs Deep (new accounts default to Deep = Fast then Deep):\n  trim config get default-tier\n  trim config set default-tier deep\n  trim config set default-tier fast\n  trim config rates\n  trim config set compression-rate 0.7\n  trim config models\n  trim config set deep-v2-backbone mbert-base\n  trim config sync\n\nDeep keep-rate (after/before, Microsoft LLMLingua): Dashboard slider or trim config rates / set compression-rate (billing min/max/step).\nDeep Base Model (Microsoft LLMLingua-2): Dashboard picker or trim config models / set deep-v2-backbone (billing catalog; default mbert-base, optional xlm-roberta-large).\n\nCursor Agent: trim cursor-link enable → trim start → trim cursor-link url (paste HTTPS …/v1; not 127.0.0.1).\n\nSimple savings check after trim start:\n  trim stats\n  open http://127.0.0.1:<port>/dashboard → Show savings detail\n\ntrim status = cloud quota. trim stats = local Deep status / savings meter.\n\nFull remove: trim uninstall (stops Trim processes, deletes local data/binary, and auto-removes npm/Scoop/Homebrew/winget installs when detected).\n\nRun "trim help <command>" for details. Prefer trim start / autostart for daily use; compress and Deep Mode are file/batch tools, not the everyday IDE path.',
    now()
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
