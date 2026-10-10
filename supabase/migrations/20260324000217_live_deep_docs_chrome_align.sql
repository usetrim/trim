-- Align remaining chrome with live Deep Mode (Fast then Deep when preferences say deep).
-- Historical migrations keep old wording; live rows must match product behavior.

update public.site_messages
set body = 'Starts the local proxy for IDE traffic. Fast Mode always runs. When account preferences set Deep Mode on, each request also runs Deep (LLMLingua) after Fast before forwarding upstream.',
    updated_at = now()
where code = 'CLI_HELP_START_LONG';

update public.site_messages
set body = 'Compress a file (Fast or Deep). Live IDE proxy also runs Deep when Preferences Deep is on.',
    updated_at = now()
where code = 'CLI_HELP_COMPRESS_SHORT';

update public.site_messages
set body = 'Fast Mode is low-latency local compression (live proxy and files). Deep Mode is stronger on-machine LLMLingua: live proxy when Preferences Deep is on, and trim compress --deep for file/batch. Engines are not loaded in Trim cloud. Licensed use still requires Trim login and network.',
    updated_at = now()
where code = 'CLI_HELP_COMPRESS_LONG';

update public.site_messages
set body = 'Start with Fast Mode on the live proxy. Turn Deep Mode on in Preferences (then trim config sync) so trim start also runs Deep after Fast. Or use trim compress --deep for file/batch jobs.',
    updated_at = now()
where code = 'DOCS_FAST_VS_DEEP_SUMMARY';

-- Prefer exact codes that exist; upsert summary if used:
insert into public.site_messages (code, body, updated_at)
values (
  'DOCS_FAST_VS_DEEP_SUMMARY',
  'Start with Fast Mode on the live proxy. Turn Deep Mode on in Preferences (then trim config sync) so trim start also runs Deep after Fast. Or use trim compress --deep for file/batch jobs.',
  now()
)
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
