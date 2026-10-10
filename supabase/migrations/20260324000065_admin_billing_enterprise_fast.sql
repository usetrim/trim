-- Billing chrome labels, ARR/free-to-paid KPIs, enterprise contract ops, credit ledger, Fast min-line caps.
alter table public.enterprise_inquiries
  add column if not exists contract_notes text,
  add column if not exists offered_seat_quantity int;

comment on column public.enterprise_inquiries.contract_notes is
  'Operator notes for custom enterprise contracts. Empty is allowed until filled.';
comment on column public.enterprise_inquiries.offered_seat_quantity is
  'Seat quantity offered in custom enterprise deal. Null until operator sets it.';

alter table public.admin_product_settings
  add column if not exists fast_balanced_min_lines int,
  add column if not exists fast_aggressive_min_lines int;

comment on column public.admin_product_settings.fast_balanced_min_lines is
  'Optional override of balanced Fast small-file min lines. Null = use mode built-in (15).';
comment on column public.admin_product_settings.fast_aggressive_min_lines is
  'Optional override of aggressive Fast small-file min lines. Null = use mode built-in (5).';

insert into public.site_messages (code, body) values
  ('ADMIN_KPI_ARR_CENTS', 'ARR proxy (cents)'),
  ('ADMIN_KPI_FREE_TO_PAID_30D', 'Free to paid (30d)'),
  ('ADMIN_BILLING_ANNUAL_DISCOUNT', 'Annual discount %'),
  ('ADMIN_BILLING_CURRENCY', 'Default currency'),
  ('ADMIN_BILLING_DEFAULT_PAGE_SIZE', 'Default page size'),
  ('ADMIN_BILLING_MAX_PAGE_SIZE', 'Max page size'),
  ('ADMIN_BILLING_DEFAULT_SEATS', 'Default seat quantity'),
  ('ADMIN_BILLING_MIN_SEATS', 'Minimum seat quantity'),
  ('ADMIN_BILLING_DEEP_TARGET_DEFAULT', 'Default deep target tokens'),
  ('ADMIN_BILLING_DEEP_TARGET_MIN', 'Deep target token min'),
  ('ADMIN_BILLING_DEEP_TARGET_MAX', 'Deep target token max'),
  ('ADMIN_BILLING_ALLOW_DOWNGRADES', 'Allow plan downgrades'),
  ('ADMIN_BILLING_PRORATION_MODE', 'Upgrade proration mode'),
  ('ADMIN_BILLING_MONTHLY_TO_ANNUAL', 'Treat monthly to annual as upgrade'),
  ('ADMIN_BILLING_SKIP_TO_MAX', 'Pagination skip-to max pages'),
  ('ADMIN_BILLING_CHART_TOP_N', 'Chart top N'),
  ('ADMIN_BILLING_CHART_DAYS', 'Chart series days'),
  ('ADMIN_BILLING_DATE_RANGE_MONTHS', 'Date range months'),
  ('ADMIN_BILLING_PLAN_INTERVAL', 'Default plan interval'),
  ('ADMIN_ENTERPRISE_CONTRACT_NOTES', 'Contract notes'),
  ('ADMIN_ENTERPRISE_OFFERED_SEATS', 'Offered seats'),
  ('ADMIN_CREDIT_LEDGER_TITLE', 'Credit grant ledger'),
  ('ADMIN_PRODUCT_FAST_BALANCED_MIN', 'Fast balanced min lines'),
  ('ADMIN_PRODUCT_FAST_AGGRESSIVE_MIN', 'Fast aggressive min lines'),
  ('ADMIN_PRORATION_PRORATED_IMMEDIATELY', 'Prorated immediately'),
  ('ADMIN_PRORATION_FULL_IMMEDIATELY', 'Full immediately'),
  ('ADMIN_PRORATION_PRORATED_NEXT', 'Prorated next period'),
  ('ADMIN_PRORATION_FULL_NEXT', 'Full next period'),
  ('ADMIN_PRORATION_DO_NOT_BILL', 'Do not bill')
on conflict (code) do nothing;
