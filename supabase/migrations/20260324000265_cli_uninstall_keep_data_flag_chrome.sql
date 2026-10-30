-- Ensure keep-data flag chrome is present (CLI also embeds the same default offline).
-- Idempotent upsert.

insert into public.site_messages (code, body, updated_at)
values
  (
    'CLI_HELP_UNINSTALL_FLAG_KEEP_DATA',
    'Keep ~/.trim, ~/.config/trim, Deep Mode caches, the CLI binary/package, and Windows LocalAppData Trim (only stop / unregister Always-on / logout / setup revert)',
    now()
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
