-- Align remaining help chrome with Deep keep-rate + rates CLI subcommand.

update public.site_messages
set body = 'Local Fast/Deep preferences (get | set | sync | rates)',
    updated_at = now()
where code = 'CLI_HELP_CONFIG_SHORT';

update public.site_messages
set body = 'Set a preference (example: trim config set compression-rate 0.7)',
    updated_at = now()
where code = 'CLI_HELP_CONFIG_SET_SHORT';

insert into public.site_messages (code, body, updated_at) values
  ('CLI_HELP_COMPRESS_FLAG_RATE', 'Deep keep-rate after/before (Microsoft LLMLingua); omit to use synced preferences', now())
on conflict (code) do update set body = excluded.body, updated_at = now();

update public.site_messages
set body = 'Deep keep-rate after/before (Microsoft LLMLingua); omit to use synced preferences',
    updated_at = now()
where code = 'CLI_HELP_COMPRESS_FLAG_TARGET';
