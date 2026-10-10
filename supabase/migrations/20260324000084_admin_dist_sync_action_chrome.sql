-- Distribution sync idle action label (never reuse ADMIN_PENDING_SYNCING as idle copy).

insert into public.site_messages (code, body) values
  ('ADMIN_DIST_SYNC', 'Sync distribution stats')
on conflict (code) do update set body = excluded.body;
