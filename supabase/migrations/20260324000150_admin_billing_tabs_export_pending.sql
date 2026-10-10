-- Audit export pending label and billing settings tab chrome.
insert into public.site_messages (code, body) values
  ('ADMIN_PENDING_EXPORTING', 'Exporting...'),
  ('ADMIN_BILLING_TAB_PRICING', 'Pricing'),
  ('ADMIN_BILLING_TAB_DEEP', 'Deep targets'),
  ('ADMIN_BILLING_TAB_LISTS', 'Lists and charts'),
  ('ADMIN_BILLING_TAB_POLICY', 'Policy')
on conflict (code) do nothing;
