-- Historical seed narrative (ops overwrite required).
-- Early migrations (e.g. 03) may have left annual_discount_percent=20 and
-- default_currency='USD' on billing_settings. Migration 28 dropped invent
-- column defaults; 34/48 neutralize unbound catalog cents and require
-- pricing_bound before checkout. This migration does not invent new prices.
-- It only documents and asserts: unbound rows must not pretend to be live.

COMMENT ON COLUMN public.billing_settings.annual_discount_percent IS
  'Operator-owned annual savings percent for UI math. Historical seeds may show 20 until overwritten. Checkout still requires pricing_bound + live pri_*.';

COMMENT ON COLUMN public.billing_settings.default_currency IS
  'Operator-owned ISO currency. Historical seeds may show USD until overwritten. Fail closed when empty or invent defaults removed (migration 28+).';

COMMENT ON COLUMN public.billing_settings.pricing_bound IS
  'False until scripts/bind-paddle-prices.sql (or equivalent) binds real Paddle pri_* and valid annual_discount_percent + default_currency. Never invent.';

-- Keep any still-unbound install fail-closed (idempotent).
UPDATE public.billing_settings
SET pricing_bound = false
WHERE pricing_bound IS DISTINCT FROM true
   AND (
     annual_discount_percent IS NULL
     OR annual_discount_percent <= 0
     OR COALESCE(NULLIF(TRIM(default_currency), ''), '') = ''
   );
