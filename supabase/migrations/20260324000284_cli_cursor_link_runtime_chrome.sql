-- Cursor Link runtime UX chrome (DB-driven; no invent English in CLI). Idempotent. No em dashes.

insert into public.site_messages (code, body, updated_at) values
  ('CLI_CURSOR_LINK_ENABLED_OK', 'Cursor Link enabled', now()),
  ('CLI_CURSOR_LINK_DISABLED_OK', 'Cursor Link disabled (tunnel stopped if it was running)', now()),
  ('CLI_CURSOR_LINK_ROTATED_OK', 'Cursor Link token rotated', now()),
  ('CLI_CURSOR_LINK_ENABLE_NEXT', 'Next: trim start  →  then trim cursor-link url  →  paste into Cursor Override OpenAI Base URL (…/v1)', now()),
  ('CLI_CURSOR_LINK_DISABLED_MUTED', 'Cursor Agent will not get a public Trim URL until you enable again. Local 127.0.0.1 still works for on-device clients.', now()),
  ('CLI_CURSOR_LINK_STATUS_PASTE', 'Paste Public Base URL into Cursor → Settings → Models → Override OpenAI Base URL', now()),
  ('CLI_CURSOR_LINK_STATUS_PRIVACY', 'Privacy: URL is public; Fast/Deep compression stays on this PC', now()),
  ('CLI_CURSOR_LINK_STATUS_ENABLE_HINT', 'Enable with: trim cursor-link enable', now()),
  ('CLI_CURSOR_LINK_ROTATE_MUTED', 'Old public path is revoked immediately (Trim reads the new token live). Update Cursor Override OpenAI Base URL.', now()),
  ('CLI_CURSOR_LINK_DISABLED_ERR', 'Cursor Link is disabled - run: trim cursor-link enable', now()),
  ('CLI_CURSOR_LINK_DISABLED_START_ERR', 'Cursor Link is disabled - run: trim cursor-link enable && trim start', now()),
  ('CLI_CURSOR_LINK_NO_URL_FMT', 'no public URL yet (%v) - is Trim running? trim start', now()),
  ('CLI_CURSOR_LINK_LABEL_STATE', 'Cursor Link', now()),
  ('CLI_CURSOR_LINK_STATE_ON', 'on', now()),
  ('CLI_CURSOR_LINK_STATE_OFF', 'off', now()),
  ('CLI_CURSOR_LINK_LABEL_PUBLIC_URL', 'Public Base URL', now()),
  ('CLI_CURSOR_LINK_PUBLIC_URL_PENDING', '(not up - run trim start)', now()),
  ('CLI_CURSOR_LINK_LABEL_LOCAL_PORT', 'Local port', now()),
  ('CLI_CURSOR_LINK_LABEL_TUNNEL_PID', 'Tunnel PID', now()),
  ('CLI_CURSOR_LINK_LABEL_NEW_URL', 'New Cursor Agent Base URL', now()),
  ('CLI_CURSOR_LINK_WARN_FMT', 'Cursor Link: %v', now()),
  ('CLI_CURSOR_LINK_TUNNEL_NOT_READY_FMT', 'Trim is up but tunnel not ready yet: %v', now()),
  ('CLI_CURSOR_LINK_TUNNEL_RETRY', 'Run trim start again (Trim auto-downloads cloudflared when needed), or: trim cursor-link status', now())
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
