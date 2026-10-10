-- Invent pass: login default next, API key ellipsis chrome, preview/TUI trunc suffixes.
-- Column is `body` (public.site_messages).

insert into public.site_messages (code, body, updated_at)
values
  ('LOGIN_DEFAULT_NEXT', '/dashboard', now()),
  ('API_KEY_PREFIX_ELLIPSIS', '...', now()),
  ('LOCAL_PREVIEW_TRUNC_SUFFIX', E'\n… (truncated)', now()),
  ('LOCAL_TUI_TRUNC_SUFFIX', '…', now())
on conflict (code) do update
set body = excluded.body, updated_at = now();

comment on table public.site_messages is
  'Backend-owned UI chrome. LOGIN_DEFAULT_NEXT / API_KEY_PREFIX_ELLIPSIS / LOCAL_*_TRUNC_SUFFIX added in migration 35.';
