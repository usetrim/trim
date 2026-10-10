-- Align receipt chrome with Paddle tax-invoice patterns (Trim seller letterhead).
insert into public.site_messages (code, body) values
  ('RECEIPT_STATUS_COMPLETED', 'PAID'),
  ('RECEIPT_MERCHANT_VIA', 'via Paddle.com'),
  ('RECEIPT_HEADER_META_SEP', ' - '),
  ('RECEIPT_PERIOD_JOIN_FMT', '%s - %s'),
  ('RECEIPT_LABEL_TRANSACTION_ID', 'Transaction'),
  ('RECEIPT_VAT_ID_PREFIX', 'VAT Number')
on conflict (code) do update set body = excluded.body;
