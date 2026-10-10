-- Curated CLI help UX: everyday vs advanced groups + clearer root long help.
-- Copy is site_messages-only (CLI reads via auth-providers chrome; no invent in binary).

insert into public.site_messages (code, body) values
  ('CLI_HELP_GROUP_EVERYDAY', 'Everyday Commands:'),
  ('CLI_HELP_GROUP_ADVANCED', 'Advanced Commands:')
on conflict (code) do update set body = excluded.body, updated_at = now();

update public.site_messages
set body = 'Local Fast Mode proxy for AI coding tools',
    updated_at = now()
where code = 'CLI_HELP_ROOT_SHORT';

update public.site_messages
set body = $long$Trim keeps a local Fast Mode proxy on your machine so IDE traffic uses less noisy context.

Everyday: start, stop, status, autostart, login, setup
Advanced: stats, tui, compress, daemon, telemetry, config

Run "trim help <command>" for details. Prefer trim start / autostart for daily use; compress and Deep Mode are file/batch tools, not the everyday IDE path.$long$,
    updated_at = now()
where code = 'CLI_HELP_ROOT_LONG';

update public.site_messages
set body = 'Start the local Trim Fast Mode proxy',
    updated_at = now()
where code = 'CLI_HELP_START_SHORT';

update public.site_messages
set body = 'Starts the local Fast Mode proxy for IDE traffic. Deep Mode is trim compress (file/batch), not every proxied request.',
    updated_at = now()
where code = 'CLI_HELP_START_LONG';

update public.site_messages
set body = 'Stop the local Trim proxy',
    updated_at = now()
where code = 'CLI_HELP_STOP_SHORT';

update public.site_messages
set body = 'Show auth and quota status',
    updated_at = now()
where code = 'CLI_HELP_STATUS_SHORT';

update public.site_messages
set body = 'Auto-start proxy with your IDE (synced preference)',
    updated_at = now()
where code = 'CLI_HELP_AUTOSTART_SHORT';

update public.site_messages
set body = 'Sign in via browser',
    updated_at = now()
where code = 'CLI_HELP_LOGIN_SHORT';

update public.site_messages
set body = 'Sign out and clear the stored API token',
    updated_at = now()
where code = 'CLI_HELP_LOGOUT_SHORT';

update public.site_messages
set body = 'Point IDEs and shell env at the local Trim proxy',
    updated_at = now()
where code = 'CLI_HELP_SETUP_SHORT';

update public.site_messages
set body = 'Print CLI version',
    updated_at = now()
where code = 'CLI_HELP_VERSION_SHORT';

update public.site_messages
set body = 'Show local proxy stats',
    updated_at = now()
where code = 'CLI_HELP_STATS_SHORT';

update public.site_messages
set body = 'Open the terminal stats dashboard',
    updated_at = now()
where code = 'CLI_HELP_TUI_SHORT';

update public.site_messages
set body = 'Compress a file (Fast or Deep Mode; not the live IDE proxy)',
    updated_at = now()
where code = 'CLI_HELP_COMPRESS_SHORT';

update public.site_messages
set body = 'Background OS service for auto-start',
    updated_at = now()
where code = 'CLI_HELP_DAEMON_SHORT';

update public.site_messages
set body = 'Anonymous usage telemetry opt-in/out',
    updated_at = now()
where code = 'CLI_HELP_TELEMETRY_SHORT';

update public.site_messages
set body = 'Local Fast/Deep preferences',
    updated_at = now()
where code = 'CLI_HELP_CONFIG_SHORT';
