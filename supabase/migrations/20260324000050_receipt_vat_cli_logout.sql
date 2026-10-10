-- Receipt VAT label for first-party PDF (no invent "VAT " English).
-- CLI logout chrome (trim logout clears keychain / credentials file).

insert into public.site_messages (code, body) values
  ('RECEIPT_VAT_ID_PREFIX', 'VAT'),
  ('RECEIPT_PDF_CURRENCY_REQUIRED', 'Receipt currency is missing; cannot render PDF amounts.'),
  ('RECEIPT_PDF_TITLE_REQUIRED', 'Receipt document title is missing; cannot render PDF.'),
  ('CLI_HELP_LOGOUT_SHORT', 'Clear the stored Trim API token from the OS keychain'),
  ('CLI_LOGOUT_SUCCESS', 'Logged out. API token removed from the OS keychain.'),
  ('CLI_LOGOUT_ALREADY', 'Already logged out. No API token was stored.')
on conflict (code) do nothing;
