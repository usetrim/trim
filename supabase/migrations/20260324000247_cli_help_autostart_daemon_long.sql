-- Long help for trim autostart / trim daemon (Windows Startup Run + soft-fail UX).

insert into public.site_messages (code, body) values
  (
    'CLI_HELP_AUTOSTART_LONG',
    E'Synced Always-on preference (Start Trim with your IDE).\n\n  trim autostart status\n  trim autostart enable\n  trim autostart disable\n\nenable turns the preference on and best-effort registers OS logon:\n  macOS launchd · Linux systemd --user · Windows per-user Startup Run (no Administrator on a normal PC)\n\nIf OS registration fails, your preference stays on. The CLI prints a yellow warning, muted guidance, and next steps (trim start / IDE). Use -v for raw OS detail.\n\nPrefer this over trim daemon for everyday use.\nDocs: https://use-trim.com/docs/cli/autostart'
  ),
  (
    'CLI_HELP_DAEMON_LONG',
    E'Advanced OS logon entrypoint (same hook as trim autostart enable).\n\n  trim daemon status\n  trim daemon install\n  trim daemon uninstall\n  trim daemon run\n\nPlatforms:\n  macOS: launchd LaunchAgent (per-user)\n  Linux: systemd --user (per-user)\n  Windows: HKCU Startup Run key - no Administrator on a normal PC (not an admin Scheduled Task)\n\ndaemon run only starts when auto-start is on, and attaches if a healthy proxy is already listening.\nPrefer trim autostart enable for everyday Always-on.\nDocs: https://use-trim.com/docs/cli/autostart'
  )
on conflict (code) do update set body = excluded.body, updated_at = now();

update public.site_messages set body = 'OS logon autostart (launchd / systemd user / Windows Startup Run)', updated_at = now()
where code = 'CLI_HELP_DAEMON_SHORT';

update public.site_messages set body = 'Install Trim proxy to start on login (launchd / systemd --user / Windows Startup Run)', updated_at = now()
where code = 'CLI_HELP_DAEMON_INSTALL_SHORT';

update public.site_messages set body = 'Remove OS logon autostart (Startup Run / launchd / systemd user)', updated_at = now()
where code = 'CLI_HELP_DAEMON_UNINSTALL_SHORT';

update public.site_messages set body = 'Show whether OS logon autostart is installed', updated_at = now()
where code = 'CLI_HELP_DAEMON_STATUS_SHORT';

update public.site_messages set body = 'Enable auto-start and best-effort OS logon (Windows Startup Run, no admin)', updated_at = now()
where code = 'CLI_HELP_AUTOSTART_ENABLE_SHORT';

update public.site_messages set body = 'Disable auto-start and uninstall OS logon when supported', updated_at = now()
where code = 'CLI_HELP_AUTOSTART_DISABLE_SHORT';
