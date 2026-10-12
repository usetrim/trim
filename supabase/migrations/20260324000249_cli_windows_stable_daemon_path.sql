-- Windows OS logon uses HKCU Startup Run → %LOCALAPPDATA%\Trim\bin\trim.exe
-- (stable copy outside npm/Scoop so package uninstall never EPERM-locks vendor\trim.exe).
-- Idempotent upsert: some OS-hook codes may exist only as MessageForCode fallbacks.

insert into public.site_messages (code, body) values
  (
    'CLI_DAEMON_WINDOWS_OK',
    'Windows logon autostart registered (Startup Run → %LOCALAPPDATA%\Trim\bin - no admin)'
  ),
  (
    'CLI_AUTOSTART_OS_HOOK_ACCESS_DENIED_BODY',
    E'Windows blocked the logon registration (Access denied). Trim uses a per-user Startup entry pointing at %LOCALAPPDATA%\\Trim\\bin\\trim.exe (no Administrator on a normal PC). Retry: trim autostart enable. If it still fails, your organization may block Startup programs - use trim start or the Trim IDE extension.'
  ),
  (
    'CLI_HELP_AUTOSTART_LONG',
    E'Synced Always-on preference (Start Trim with your IDE).\n\n  trim autostart status\n  trim autostart enable\n  trim autostart disable\n\nenable turns the preference on and best-effort registers OS logon:\n  macOS launchd · Linux systemd --user · Windows Startup Run → %LOCALAPPDATA%\\Trim\\bin (no Administrator on a normal PC)\n\nIf OS registration fails, your preference stays on. The CLI prints a yellow warning, muted guidance, and next steps (trim start / IDE). Use -v for raw OS detail.\n\nPrefer this over trim daemon for everyday use.\nDocs: https://use-trim.com/docs/cli/autostart'
  ),
  (
    'CLI_HELP_AUTOSTART_ENABLE_SHORT',
    'Enable auto-start and best-effort OS logon (Windows Startup Run → LocalAppData, no admin)'
  ),
  (
    'CLI_HELP_DAEMON_LONG',
    E'Advanced OS logon entrypoint (same hook as trim autostart enable).\n\n  trim daemon status\n  trim daemon install\n  trim daemon uninstall\n  trim daemon run\n\nPlatforms:\n  macOS: launchd LaunchAgent (per-user)\n  Linux: systemd --user (per-user)\n  Windows: HKCU Startup Run → %LOCALAPPDATA%\\Trim\\bin\\trim.exe (stable copy outside npm/Scoop; no Administrator on a normal PC)\n\ndaemon run only starts when auto-start is on, and attaches if a healthy proxy is already listening.\nPrefer trim autostart enable for everyday Always-on.\nDocs: https://use-trim.com/docs/cli/autostart'
  )
on conflict (code) do update set body = excluded.body, updated_at = now();
