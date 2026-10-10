-- Segment filter chrome, admin IP allowlist chrome, pricing unbound gate,
-- Fast mild min lines, legal publish, Paddle refund link, receipt PDF link label.
alter table public.admin_product_settings
  add column if not exists fast_mild_min_lines int;

comment on column public.admin_product_settings.fast_mild_min_lines is
  'Optional override of mild Fast small-file min lines. Null = use mode built-in (15).';

alter table public.site_legal_sections
  add column if not exists updated_at timestamptz not null default now(),
  add column if not exists published_at timestamptz;

comment on column public.site_legal_sections.published_at is
  'When set, section is considered published for public surfaces. Null = draft (fail closed hide).';

-- Existing seeded legal copy is live today; mark published so public pages do not go empty.
update public.site_legal_sections
set published_at = coalesce(published_at, now())
where published_at is null;

alter table public.trim_events
  add column if not exists error_code text;

comment on column public.trim_events.error_code is
  'Optional machine code for failed events (e.g. oom, upstream). Empty on success.';

create index if not exists idx_trim_events_error_code_created
  on public.trim_events (error_code, created_at desc)
  where error_code is not null and error_code <> '';

insert into public.site_messages (code, body) values
  ('ADMIN_FILTER_INTERVAL', 'Billing interval'),
  ('ADMIN_FILTER_CREATED_FROM', 'Created from (YYYY-MM-DD)'),
  ('ADMIN_FILTER_CREATED_TO', 'Created to (YYYY-MM-DD)'),
  ('ADMIN_FILTER_CREDITS_LEFT_MAX', 'Credits left at most'),
  ('ADMIN_SEGMENT_COL_INTERVAL', 'Interval'),
  ('ADMIN_SEGMENT_COL_CHURN', 'Churn risk'),
  ('ADMIN_CHURN_HIGH', 'High'),
  ('ADMIN_CHURN_MEDIUM', 'Medium'),
  ('ADMIN_CHURN_LOW', 'Low'),
  ('ADMIN_CHURN_NONE', 'None'),
  ('ADMIN_PRICING_UNBOUND', 'Pricing is not bound. Self-serve sell paths stay fail-closed until pricing_bound is true.'),
  ('ADMIN_RECEIPT_PDF_OPEN', 'Open PDF'),
  ('ADMIN_PADDLE_REFUND_LINK', 'Open in Paddle'),
  ('ADMIN_PADDLE_TX_URL_FMT', 'https://vendors.paddle.com/transactions-v2/{transaction_id}'),
  ('ADMIN_PRODUCT_FAST_MILD_MIN', 'Fast mild min lines'),
  ('ADMIN_LEGAL_PUBLISH', 'Publish'),
  ('ADMIN_LEGAL_UNPUBLISH', 'Unpublish'),
  ('ADMIN_LEGAL_PUBLISHED', 'Published'),
  ('ADMIN_LEGAL_DRAFT', 'Draft'),
  ('ADMIN_ALERT_OOM_24H', 'Deep OOM / memory errors (24h)'),
  ('ADMIN_ORIGIN_FORBIDDEN', 'Admin origin is not allowed for this request.'),
  ('ADMIN_IP_FORBIDDEN', 'Admin client IP is not on the allow-list.'),
  ('ADMIN_CHECKLIST_ADMIN_IP', 'ADMIN_ALLOWED_CIDRS configured')
on conflict (code) do nothing;
