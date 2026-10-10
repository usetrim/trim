-- Fail-closed receipt money locale chrome (SITE_HTML_LANG empty must not invent Intl locale).
-- site_messages.body is the column (see migration 25); never use invent "message".

INSERT INTO public.site_messages (code, body) VALUES
  ('RECEIPT_MONEY_LOCALE_REQUIRED', 'Receipt money locale is not configured (SITE_HTML_LANG).'),
  ('RECEIPT_PDF_MONEY_LOCALE_REQUIRED', 'Receipt PDF money locale is missing; cannot format amounts.')
ON CONFLICT (code) DO UPDATE
SET body = EXCLUDED.body,
    updated_at = now();
