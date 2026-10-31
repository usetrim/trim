-- Correct CLI_PROXY_DEEP_COMPACT_STUB: gate chrome only (not history). History uses CLI_PROXY_HISTORY_COMPACTED_STUB on Fast. Idempotent. No em dashes.

insert into public.site_messages (code, body, updated_at) values
  ('CLI_PROXY_DEEP_COMPACT_STUB', '[trim] Segmented Allowlist Deep chrome ready', now())
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
