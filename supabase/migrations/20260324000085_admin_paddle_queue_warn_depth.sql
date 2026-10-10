-- Fail-closed paddle webhook queue health threshold (no invent depth=100).

alter table public.admin_product_settings
  add column if not exists paddle_webhook_queue_warn_depth integer;

comment on column public.admin_product_settings.paddle_webhook_queue_warn_depth is
  'Dashboard health marks paddle queue ok when depth is below this. Null or 0 fails closed (not ok).';

insert into public.site_messages (code, body) values
  ('ADMIN_PRODUCT_QUEUE_WARN_DEPTH', 'Paddle webhook queue warn depth'),
  ('ADMIN_CHECKLIST_MIGRATIONS_QUEUE_WARN', 'Paddle queue warn depth migration applied')
on conflict (code) do update set body = excluded.body;
