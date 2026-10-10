-- Webhook process status labels (no client invent of ok/failed/pending).

insert into public.site_messages (code, body) values
  ('ADMIN_WEBHOOK_STATUS_OK', 'OK'),
  ('ADMIN_WEBHOOK_STATUS_FAILED', 'Failed'),
  ('ADMIN_WEBHOOK_STATUS_PENDING', 'Pending')
on conflict (code) do update set body = excluded.body;
