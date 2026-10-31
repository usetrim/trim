-- Segmented Allowlist Deep: remaining chrome aligned to per-island reassembly
-- (order preserved beside paths). Idempotent. No em dashes.

update public.site_messages
set body = 'Live proxy runs Fast Mode on every turn. When account preferences set compression_tier=deep, the local proxy also runs Segmented Allowlist Deep after Fast: freeze tools/system/paths/code/JSON, compress each last-user prose island in place (order preserved beside paths) at your compression rate, validate, else fail-closed Fast. Models and OOM policy come from billing settings via trim config sync.',
    updated_at = now()
where code in ('PREFERENCES_PAGE_DESCRIPTION', 'DOCS_FAST_VS_DEEP_LIVE');

update public.site_messages
set body = 'When Deep Mode is on, the live IDE proxy (trim start) runs Fast then Segmented Allowlist Deep (each last-user prose island in place, order preserved beside paths; tools/paths frozen) using your engine and compression rate from this page. Off = Fast only. Trim cloud never loads Deep engines; auth, quotas, and billing still use Trim cloud.',
    updated_at = now()
where code = 'PREFERENCES_DEEP_HINT';

update public.site_messages
set body = 'Start with Fast Mode on the live proxy. Turn Deep Mode on in Preferences (then trim config sync) so trim start runs Segmented Allowlist Deep after Fast (each last-user prose island in place, order preserved beside paths). Or use trim compress --deep for file/batch jobs.',
    updated_at = now()
where code = 'DOCS_FAST_VS_DEEP_SUMMARY';

update public.site_messages
set body = $body$Fast Mode is low-latency local compression (live proxy and files). Deep Mode is stronger on-machine LLMLingua: live proxy runs Segmented Allowlist Deep when Preferences Deep is on (each last-user prose island in place, order preserved beside paths; tools/paths/schemas frozen), and trim compress --deep for file/batch. Engines are not loaded in Trim cloud.

Deep engines (pick one): v2 LLMLingua-2 (recommended; uses synced Base Model from trim config models), long LongLLMLingua (question-aware; requires --question), v1 LLMLingua (classic).

Keep-rate: --rate or synced compression-rate (Dashboard / trim config rates).
Base Model (v2 only): Dashboard picker or trim config models / set deep-v2-backbone (billing catalog; default mbert-base, optional xlm-roberta-large). No free-text Hugging Face box.

Licensed use still requires Trim login and network.$body$,
    updated_at = now()
where code = 'CLI_HELP_COMPRESS_LONG';

update public.site_messages
set body = $body$Reads the live local proxy (/v1/stats) and prints clear Deep status + Deep stage savings, then the raw JSON.

Useful when Claude Code / Cursor shows 0% whole-body Saved: check Deep status and Deep stage (Segmented Allowlist Deep = each prose island in place, order preserved beside paths; tools/paths/schemas stay frozen).

Everyday:
  trim start
  trim stats
  open http://127.0.0.1:<TRIM_PORT>/dashboard → Show savings detail

Use --tui for the Bubble Tea live dashboard (same as trim tui).$body$,
    updated_at = now()
where code = 'CLI_HELP_STATS_LONG';
