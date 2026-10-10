-- Ensure DOCS_FAST_VS_DEEP_LIVE exists (used by product copy / builtins alignment).

insert into public.site_messages (code, body, updated_at)
values (
  'DOCS_FAST_VS_DEEP_LIVE',
  'Live proxy runs Fast Mode on every turn. When account preferences set compression_tier=deep, the local proxy also runs Deep Mode (LLMLingua) after Fast before forwarding upstream.',
  now()
)
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
