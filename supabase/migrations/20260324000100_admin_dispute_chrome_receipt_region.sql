-- Dispute watching status, chrome message column labels, receipt bill-to region.

insert into public.site_messages (code, body) values
  ('ADMIN_DISPUTE_STATUS_WATCHING', 'Watching'),
  ('ADMIN_CHROME_COL_CODE', 'Code'),
  ('ADMIN_CHROME_COL_BODY', 'Body'),
  ('ADMIN_RECEIPT_BILL_TO_REGION', 'Bill-to region')
on conflict (code) do update set body = excluded.body;
