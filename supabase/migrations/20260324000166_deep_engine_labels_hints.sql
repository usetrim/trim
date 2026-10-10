-- Clear Deep engine labels + per-engine hints (chrome-first).
-- v2 (LLMLingua-2) stays DEFAULT_DEEP_ENGINE and is labeled recommended.
-- Existing rows are updated; new hint codes are inserted.

insert into public.site_messages (code, body) values
  ('ENGINE_V2_HINT', 'Usually faster and lighter. Best default for most Deep compress jobs on your machine.'),
  ('ENGINE_LONG_HINT', 'Question-aware ranking for long documents. Requires a question (CLI: --question).'),
  ('ENGINE_V1_HINT', 'Classic LLMLingua. Strong general compression when you want the original engine.')
on conflict (code) do update set body = excluded.body, updated_at = now();

update public.site_messages
set body = 'LLMLingua-2 (v2) · recommended',
    updated_at = now()
where code = 'ENGINE_V2_LABEL';

update public.site_messages
set body = 'LongLLMLingua (long)',
    updated_at = now()
where code = 'ENGINE_LONG_LABEL';

update public.site_messages
set body = 'LLMLingua (v1)',
    updated_at = now()
where code = 'ENGINE_V1_LABEL';

update public.site_messages
set body = 'Pick one Deep engine for trim compress on your machine. LLMLingua-2 (v2) is recommended for most jobs. Only applies when Deep Mode is on.',
    updated_at = now()
where code = 'PREFERENCES_ENGINE_HINT';

update public.site_messages
set body = 'Deep engine: v2 (recommended, usually faster) | long (needs --question) | v1 (classic). Else prefs deep-engine.',
    updated_at = now()
where code = 'CLI_HELP_COMPRESS_FLAG_ENGINE';

update public.site_messages
set body = $long$Fast Mode is low-latency local compression for the live proxy or files. Deep Mode is stronger on-machine file/batch compression; engines are not loaded in Trim cloud.

Deep engines (pick one): v2 LLMLingua-2 (recommended, usually faster), long LongLLMLingua (question-aware; requires --question), v1 LLMLingua (classic). Licensed use still requires Trim login and network.$long$,
    updated_at = now()
where code = 'CLI_HELP_COMPRESS_LONG';

update public.site_messages
set body = 'Default Deep engine for new accounts (v1, long, or v2). LLMLingua-2 (v2) is recommended.',
    updated_at = now()
where code = 'ADMIN_PRODUCT_ENGINE_DESC';

-- Keep product default explicit (already v2 in most envs).
update public.site_messages
set body = 'v2',
    updated_at = now()
where code = 'DEFAULT_DEEP_ENGINE'
  and body is distinct from 'v2';
