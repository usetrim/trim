-- User profile country chrome (login + bill-to from receipts).

insert into public.site_messages (code, body) values
  ('ADMIN_USER_LOGIN_COUNTRY', 'Last login country'),
  ('ADMIN_USER_BILL_TO_COUNTRY', 'Bill-to country')
on conflict (code) do update set body = excluded.body;
