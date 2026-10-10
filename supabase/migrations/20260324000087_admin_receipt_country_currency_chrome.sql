-- Receipts admin: bill-to country column chrome (API already returns bill_to_country).

insert into public.site_messages (code, body) values
  ('ADMIN_RECEIPT_BILL_TO_COUNTRY', 'Bill-to country'),
  ('ADMIN_RECEIPT_CURRENCY', 'Currency')
on conflict (code) do update set body = excluded.body;
