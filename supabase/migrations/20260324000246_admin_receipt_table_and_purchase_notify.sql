-- Admin receipts list: lean columns + view detail. Admin purchase notifications → receipts?id=.

insert into public.site_messages (code, body) values
  ('ADMIN_RECEIPT_COL_WHEN', 'Date'),
  ('ADMIN_RECEIPT_COL_INVOICE', 'Invoice'),
  ('ADMIN_RECEIPT_COL_CUSTOMER', 'Customer'),
  ('ADMIN_RECEIPT_COL_STATUS', 'Status'),
  ('ADMIN_RECEIPT_COL_TOTAL', 'Total'),
  ('ADMIN_RECEIPT_VIEW_DETAIL', 'View detail'),
  ('ADMIN_RECEIPT_PDF_OPEN', 'View detail'),
  ('NOTIF_ADMIN_RECEIPT_TITLE', 'Payment received'),
  ('NOTIF_ADMIN_RECEIPT_BODY', 'A completed receipt is ready to review.')
on conflict (code) do update set body = excluded.body;

insert into public.notification_kinds (
  code, audience, title_message_code, body_message_code, href_ref_kind, href_ref
) values (
  'admin.receipt_ready',
  'admin',
  'NOTIF_ADMIN_RECEIPT_TITLE',
  'NOTIF_ADMIN_RECEIPT_BODY',
  'admin_nav_query',
  'receipts'
)
on conflict (code) do update set
  audience = excluded.audience,
  title_message_code = excluded.title_message_code,
  body_message_code = excluded.body_message_code,
  href_ref_kind = excluded.href_ref_kind,
  href_ref = excluded.href_ref;
