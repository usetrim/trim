-- CLI help chrome: surface cursor-link, config sync, and package-manager uninstall
-- (including npm -g) in root / command help. Idempotent upsert.

insert into public.site_messages (code, body, updated_at)
values
  (
    'CLI_HELP_ROOT_LONG',
    E'Trim keeps a local Fast Mode proxy on your machine so IDE traffic uses less noisy context.\n\nEveryday: start, stop, status, stats, upgrade, autostart, cursor-link, login, logout, uninstall, setup, version\nAdvanced: tui, compress, daemon, telemetry, config (get | set | sync)\n\nCursor Agent: trim cursor-link enable → trim start → trim cursor-link url (paste HTTPS …/v1; not 127.0.0.1).\n\nSimple savings check after trim start:\n  trim stats\n  open http://127.0.0.1:<port>/dashboard → Show savings detail\n\ntrim status = cloud quota. trim stats = local Deep status / savings meter.\n\nFull remove: trim uninstall (then brew/scoop/winget/npm uninstall if you installed that way).\n\nRun "trim help <command>" for details. Prefer trim start / autostart for daily use; compress and Deep Mode are file/batch tools, not the everyday IDE path.',
    now()
  ),
  (
    'CLI_HELP_UNINSTALL_LONG',
    E'Fully removes Trim from this machine:\n  - stops the local proxy and Cursor Link tunnel\n  - removes the OS login daemon\n  - clears CLI credentials\n  - reverts IDE Base URL overrides from trim setup\n  - deletes ~/.trim and ~/.config/trim, Deep caches, sidecars, and the CLI binary/PATH when possible\n\nUse --keep-data to skip deleting local data directories, Deep caches, and the binary.\n\nIf you installed via a package manager, also uninstall that package after trim uninstall:\n  brew uninstall trim\n  scoop uninstall trim\n  winget uninstall Trim.CLI\n  npm uninstall -g @usetrim/trim',
    now()
  ),
  (
    'CLI_UNINSTALL_NEXT_PKG_NPM',
    'If installed via npm: npm uninstall -g @usetrim/trim',
    now()
  ),
  (
    'CLI_HELP_CONFIG_SHORT',
    'Local Fast/Deep preferences (get | set | sync)',
    now()
  ),
  (
    'CLI_HELP_CONFIG_LONG',
    E'Manage local Fast/Deep preferences under ~/.config/trim/.\n\n  trim config get\n  trim config get default-tier\n  trim config set default-tier deep\n  trim config sync\n\nget - show local preferences\nset - write a local preference key\nsync - pull Fast/Deep preferences and provider adapters from your Trim cloud profile (requires trim login)',
    now()
  ),
  (
    'CLI_HELP_CONFIG_SYNC_SHORT',
    'Pull Fast/Deep preferences and provider adapters from your Trim cloud profile into ~/.config/trim/',
    now()
  ),
  (
    'CLI_HELP_CURSOR_LINK_SHORT',
    'Public HTTPS URL to local Trim for Cursor Agent (managed tunnel)',
    now()
  ),
  (
    'CLI_HELP_CURSOR_LINK_LONG',
    E'Trim Cursor Link exposes your local Trim proxy through an automatic Cloudflare quick tunnel so Cursor Agent (cloud) can reach it without 127.0.0.1.\n\nCompression, meters, and adapters still run on this PC. The public hostname is only a door. Lifecycle matches trim start / trim stop when enabled.\n\nTypical flow:\n  trim cursor-link enable\n  trim start\n  trim cursor-link url\n  # paste …/cl/<token>/v1 into Cursor Override OpenAI Base URL\n  trim stop\n\nSubcommands: enable, disable, status, url, rotate',
    now()
  ),
  (
    'CLI_HELP_CURSOR_LINK_ENABLE_SHORT',
    'Enable Cursor Link (auto tunnel while trim start is running)',
    now()
  ),
  (
    'CLI_HELP_CURSOR_LINK_DISABLE_SHORT',
    'Disable Cursor Link and stop any running tunnel',
    now()
  ),
  (
    'CLI_HELP_CURSOR_LINK_STATUS_SHORT',
    'Show Cursor Link preference and live public URL',
    now()
  ),
  (
    'CLI_HELP_CURSOR_LINK_URL_SHORT',
    'Print the live Cursor Agent Base URL (…/v1) for copy/paste',
    now()
  ),
  (
    'CLI_HELP_CURSOR_LINK_ROTATE_SHORT',
    'Rotate the Cursor Link access token (revokes the previous public URL path)',
    now()
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
