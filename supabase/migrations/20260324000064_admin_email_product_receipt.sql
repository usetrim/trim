-- Product rate/fraud overrides, email template catalog, CLI force-upgrade chrome, receipt resync chrome.
alter table public.admin_product_settings
  add column if not exists rate_limit_ip_per_min int,
  add column if not exists rate_limit_user_per_min int,
  add column if not exists pow_difficulty int,
  add column if not exists max_accounts_per_hardware int,
  add column if not exists max_accounts_per_ja4 int,
  add column if not exists cf_threat_score_min int;

comment on column public.admin_product_settings.rate_limit_ip_per_min is
  'Optional override of TRIM_RATE_LIMIT_IP_PER_MIN. Null = use env Config only.';
comment on column public.admin_product_settings.rate_limit_user_per_min is
  'Optional override of TRIM_RATE_LIMIT_USER_PER_MIN. Null = use env Config only.';
comment on column public.admin_product_settings.pow_difficulty is
  'Optional override of TRIM_POW_DIFFICULTY. Null = use env Config only.';
comment on column public.admin_product_settings.max_accounts_per_hardware is
  'Optional override of TRIM_MAX_ACCOUNTS_PER_HARDWARE. Null = use env.';
comment on column public.admin_product_settings.max_accounts_per_ja4 is
  'Optional override of TRIM_MAX_ACCOUNTS_PER_JA4. Null = use env.';
comment on column public.admin_product_settings.cf_threat_score_min is
  'Optional override of TRIM_CF_THREAT_SCORE_MIN. Null = use env.';

create table if not exists public.admin_email_template_catalog (
  code text primary key,
  kind text not null check (kind in ('invite', 'suspend', 'receipt')),
  sort_order int not null default 0
);

comment on table public.admin_email_template_catalog is
  'Operator email template codes that must exist in site_messages. Empty body fails closed at send.';

insert into public.site_messages (code, body) values
  ('EMAIL_SUSPEND_SUBJECT', 'Your Trim account was suspended'),
  ('EMAIL_SUSPEND_BODY_FMT', 'Your Trim account ({email}) was suspended.\n\nReason: {reason}\n\nContact support if you believe this is an error.'),
  ('EMAIL_RECEIPT_SUBJECT_FMT', 'Your Trim receipt {invoice}'),
  ('EMAIL_RECEIPT_BODY_FMT', 'Thanks for your payment.\n\nInvoice: {invoice}\nTotal: {total}\n\nView receipts in your dashboard.'),
  ('CLI_FORCE_UPGRADE_NOTICE', ''),
  ('ADMIN_NAV_EMAIL', 'Email templates'),
  ('ADMIN_EMAIL_KIND_INVITE', 'Invite'),
  ('ADMIN_EMAIL_KIND_SUSPEND', 'Suspend'),
  ('ADMIN_EMAIL_KIND_RECEIPT', 'Receipt'),
  ('ADMIN_ACTION_RESYNC', 'Resync from Paddle'),
  ('ADMIN_PENDING_RESYNCING', 'Resyncing...'),
  ('ADMIN_RECEIPT_VAT_LABEL', 'Tax / VAT id'),
  ('ADMIN_RECEIPT_TAX_CENTS', 'Tax (cents)'),
  ('ADMIN_RECEIPT_PDF_URL', 'Invoice PDF URL'),
  ('ADMIN_PRODUCT_RATE_IP', 'Rate limit per IP / min'),
  ('ADMIN_PRODUCT_RATE_USER', 'Rate limit per user / min'),
  ('ADMIN_PRODUCT_POW', 'PoW difficulty'),
  ('ADMIN_PRODUCT_MAX_HW', 'Max accounts per hardware'),
  ('ADMIN_PRODUCT_MAX_JA4', 'Max accounts per JA4'),
  ('ADMIN_PRODUCT_CF_THREAT', 'CF threat score min'),
  ('ADMIN_PRODUCT_CLI_NOTICE', 'CLI force-upgrade notice'),
  ('ADMIN_SMTP_STATUS_ON', 'SMTP configured'),
  ('ADMIN_SMTP_STATUS_OFF', 'SMTP not configured'),
  ('ADMIN_RECEIPT_RESYNC_DONE', 'Receipt refreshed from Paddle.'),
  ('ADMIN_RECEIPT_RESYNC_FAILED', 'Could not resync receipt from Paddle.'),
  ('ADMIN_PADDLE_UNAVAILABLE', 'Paddle API is not configured for this deployment.'),
  ('ADMIN_EMAIL_TEMPLATE_NOT_FOUND', 'Email template code is not in the catalog.')
on conflict (code) do nothing;

insert into public.admin_email_template_catalog (code, kind, sort_order) values
  ('WORKSPACE_INVITE_EMAIL_SUBJECT_FALLBACK', 'invite', 10),
  ('WORKSPACE_INVITE_EMAIL_SUBJECT_NAMED_FMT', 'invite', 20),
  ('WORKSPACE_INVITE_EMAIL_BODY_FMT', 'invite', 30),
  ('EMAIL_SUSPEND_SUBJECT', 'suspend', 10),
  ('EMAIL_SUSPEND_BODY_FMT', 'suspend', 20),
  ('EMAIL_RECEIPT_SUBJECT_FMT', 'receipt', 10),
  ('EMAIL_RECEIPT_BODY_FMT', 'receipt', 20)
on conflict (code) do nothing;
