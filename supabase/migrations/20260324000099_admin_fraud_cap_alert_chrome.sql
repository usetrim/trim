-- Fraud-cap breach alert chrome (threshold from admin_product_settings).

insert into public.site_messages (code, body) values
  ('ADMIN_ALERT_JA4_CAP_BREACH', 'JA4 fingerprints over account cap'),
  ('ADMIN_ALERT_HARDWARE_CAP_BREACH', 'Hardware fingerprints over account cap')
on conflict (code) do update set body = excluded.body;
