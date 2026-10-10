-- Admin TOTP step-up, receipt PDF reissue chrome, deep-error / queue alerts.
create table if not exists public.platform_admin_totp (
  user_id uuid primary key references public.profiles(id) on delete cascade,
  secret_ciphertext bytea not null,
  secret_nonce bytea not null,
  pending_ciphertext bytea,
  pending_nonce bytea,
  enrolled_at timestamptz,
  updated_at timestamptz not null default now()
);

comment on table public.platform_admin_totp is
  'AES-GCM encrypted TOTP secrets for platform admin step-up. Null enrolled_at means pending only.';

insert into public.site_messages (code, body) values
  ('ADMIN_TOTP_REQUIRED', 'Enroll TOTP before performing step-up protected actions.'),
  ('ADMIN_TOTP_CODE_INVALID', 'TOTP code is invalid or expired.'),
  ('ADMIN_TOTP_CODE_REQUIRED', 'Enter your authenticator code to verify step-up.'),
  ('ADMIN_TOTP_KEY_MISSING', 'Set TRIM_ADMIN_TOTP_KEY (64 hex chars) before TOTP enroll.'),
  ('ADMIN_TOTP_NOT_ENROLLED', 'TOTP is not enrolled for this admin.'),
  ('ADMIN_TOTP_ALREADY_ENROLLED', 'TOTP is already enrolled. Disable it before re-enrolling.'),
  ('ADMIN_TOTP_ENROLL_TITLE', 'Enroll authenticator (TOTP)'),
  ('ADMIN_TOTP_VERIFY_TITLE', 'Step-up with authenticator'),
  ('ADMIN_TOTP_CODE_LABEL', 'Authenticator code'),
  ('ADMIN_TOTP_SECRET_LABEL', 'Secret (add to authenticator app)'),
  ('ADMIN_TOTP_BEGIN', 'Start TOTP enroll'),
  ('ADMIN_TOTP_CONFIRM', 'Confirm enroll'),
  ('ADMIN_TOTP_DISABLE', 'Disable TOTP'),
  ('ADMIN_TOTP_STATUS_ON', 'TOTP enrolled'),
  ('ADMIN_TOTP_STATUS_OFF', 'TOTP not enrolled'),
  ('ADMIN_ACTION_PDF_REISSUE', 'Reissue invoice PDF'),
  ('ADMIN_PENDING_PDF_REISSUE', 'Refreshing PDF...'),
  ('ADMIN_RECEIPT_PDF_DONE', 'Invoice PDF URL refreshed from Paddle.'),
  ('ADMIN_RECEIPT_PDF_FAILED', 'Could not refresh invoice PDF from Paddle.'),
  ('ADMIN_ALERT_DEEP_ERRORS_24H', 'Deep attach errors (24h)'),
  ('ADMIN_HEALTH_QUEUE_LAG', 'Paddle webhook queue depth'),
  ('ADMIN_HEALTH_GEOLITE', 'GeoLite MMDB paths'),
  ('ADMIN_CHECKLIST_IDP', 'OAuth providers configured'),
  ('ADMIN_CHECKLIST_MIGRATIONS', 'Admin TOTP table present'),
  ('ADMIN_CHECKLIST_TOTP_KEY', 'TRIM_ADMIN_TOTP_KEY configured')
on conflict (code) do nothing;
