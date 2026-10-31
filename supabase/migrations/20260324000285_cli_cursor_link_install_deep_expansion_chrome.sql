-- Cursor Link install/tunnel + Deep expansion / start worker chrome (DB-driven). Idempotent. No em dashes.

insert into public.site_messages (code, body, updated_at) values
  ('CLI_PROXY_DEEP_EXPANSION_FAIL_CLOSED_FMT', 'expansion fail-closed (%d->%d stage tokens; kept Fast)', now()),
  ('CLI_PROXY_WORKER_EXITED_EARLY_FMT', 'proxy worker exited early: %v', now()),
  ('CLI_PROXY_NOT_HEALTHY', 'proxy did not become healthy in time', now()),
  ('CLI_CURSOR_LINK_DOWNLOADING_FMT', 'Cursor Link: downloading cloudflared to %s', now()),
  ('CLI_CURSOR_LINK_DOWNLOAD_ERR_FMT', 'download cloudflared: %v' || E'\n' || 'Install manually: %s' || E'\n' || 'Or set TRIM_CLOUDFLARED to a cloudflared binary', now()),
  ('CLI_CURSOR_LINK_DOWNLOAD_HTTP_FMT', 'download cloudflared: HTTP %s' || E'\n' || 'Install manually: %s' || E'\n' || 'Or set TRIM_CLOUDFLARED', now()),
  ('CLI_CURSOR_LINK_EXTRACT_ERR_FMT', 'extract cloudflared: %v' || E'\n' || 'Run trim cursor-link enable && trim start again, or brew install cloudflared if Homebrew exists, or set TRIM_CLOUDFLARED', now()),
  ('CLI_CURSOR_LINK_ARCHIVE_BIN_MISSING', 'cloudflared binary not found inside archive', now()),
  ('CLI_CURSOR_LINK_UNSUPPORTED_PLATFORM_FMT', 'unsupported platform %s/%s for auto-download; %s; or set TRIM_CLOUDFLARED', now()),
  ('CLI_CURSOR_LINK_HINT_WINDOWS', 'Windows: run trim cursor-link enable && trim start (auto-download), or download cloudflared-windows-amd64.exe into %USERPROFILE%\.trim\bin, or winget install --id Cloudflare.cloudflared if winget exists', now()),
  ('CLI_CURSOR_LINK_HINT_DARWIN', 'macOS: run trim cursor-link enable && trim start (auto-download), or brew install cloudflared if Homebrew exists, or set TRIM_CLOUDFLARED', now()),
  ('CLI_CURSOR_LINK_HINT_LINUX', 'Linux: run trim cursor-link enable && trim start (auto-download), or install from https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/, or set TRIM_CLOUDFLARED', now()),
  ('CLI_CURSOR_LINK_HINT_OTHER', 'install cloudflared from https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/ or set TRIM_CLOUDFLARED', now()),
  ('CLI_CURSOR_LINK_NOT_FOUND_FMT', 'cloudflared not found (Trim can auto-download on Windows/macOS/Linux, or install from %s, or set TRIM_CLOUDFLARED)', now()),
  ('CLI_CURSOR_LINK_PORT_REQUIRED', 'local Trim port is required for Cursor Link', now()),
  ('CLI_CURSOR_LINK_START_CLOUDFLARED_FMT', 'start cloudflared: %v', now()),
  ('CLI_CURSOR_LINK_TUNNEL_TIMEOUT', 'cloudflared did not publish a trycloudflare.com URL in time', now()),
  ('CLI_CURSOR_LINK_COMBINE_ERR_HINT_FMT', '%v' || E'\n' || '%s', now()),
  ('CLI_CURSOR_LINK_ENV_MISSING_FMT', 'TRIM_CLOUDFLARED points to a missing binary: %s', now())
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
