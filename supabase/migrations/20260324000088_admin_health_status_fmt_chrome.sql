-- Health status_label formats (no invent bare numbers).

insert into public.site_messages (code, body) values
  ('ADMIN_HEALTH_ERROR_RATE_FMT', '{pct}%'),
  ('ADMIN_HEALTH_QUEUE_DEPTH_FMT', '{depth}')
on conflict (code) do update set body = excluded.body;
