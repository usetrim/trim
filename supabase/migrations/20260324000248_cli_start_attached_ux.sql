-- Clear UX when trim start attaches to an already-running local proxy.
-- Coloring matches status/autostart: Success ● + Table rows + Muted tip.

insert into public.site_messages (code, body) values
  (
    'CLI_START_ATTACHED_OK_FMT',
    'Trim is already running on %s://127.0.0.1:%s'
  ),
  (
    'CLI_START_ATTACHED_HINT',
    'Attached to the existing local proxy - no second process started.'
  ),
  (
    'CLI_START_DASHBOARD_LABEL',
    'Dashboard'
  )
on conflict (code) do update set body = excluded.body, updated_at = now();

update public.site_messages
set body = 'Trim proxy running on %s://127.0.0.1:%s', updated_at = now()
where code = 'CLI_START_RUNNING_FMT';

update public.site_messages
set body = 'Local dashboard: %s://127.0.0.1:%s/dashboard', updated_at = now()
where code = 'CLI_START_DASHBOARD_FMT';

update public.site_messages
set body = 'Point your IDE Base URL at %s://127.0.0.1:%s/v1 (Cursor Override / Continue apiBase). VS Code Chat Custom Endpoint needs …/v1/chat/completions - see docs.',
    updated_at = now()
where code = 'CLI_START_POINT_IDE_FMT';
