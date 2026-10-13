-- Clarify cloudflared tip: Trim auto-downloads on all OS; winget/brew optional.
-- Idempotent upsert. No schema drops.

insert into public.site_messages (code, body, updated_at)
values
  (
    'CLI_START_CURSOR_AGENT_CLOUDFLARED_TIP',
    'Trim auto-downloads cloudflared to ~/.trim/bin when missing (all OS). Optional: brew/winget only if installed. Or set TRIM_CLOUDFLARED, then: trim cursor-link status',
    now()
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
