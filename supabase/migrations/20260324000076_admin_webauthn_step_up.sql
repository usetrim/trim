-- WebAuthn (passkey) credentials for platform admin step-up.
-- TOTP remains supported; either factor satisfies step-up enrollment.

create table if not exists public.platform_admin_webauthn_credentials (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references public.profiles(id) on delete cascade,
  credential_id bytea not null,
  public_key bytea not null,
  attestation_type text not null default '',
  transport text[] not null default '{}',
  sign_count bigint not null default 0,
  clone_warning boolean not null default false,
  aaguid bytea,
  friendly_name text not null default '',
  created_at timestamptz not null default now(),
  last_used_at timestamptz,
  unique (credential_id)
);

create index if not exists idx_platform_admin_webauthn_user
  on public.platform_admin_webauthn_credentials (user_id);

comment on table public.platform_admin_webauthn_credentials is
  'Passkey / WebAuthn credentials for platform admin step-up. Instance-local; no invent defaults.';

insert into public.site_messages (code, body) values
  ('ADMIN_WEBAUTHN_STATUS_ON', 'Passkey enrolled'),
  ('ADMIN_WEBAUTHN_STATUS_OFF', 'Passkey not enrolled'),
  ('ADMIN_WEBAUTHN_ENROLL_TITLE', 'Enroll passkey (WebAuthn)'),
  ('ADMIN_WEBAUTHN_VERIFY_TITLE', 'Step-up with passkey'),
  ('ADMIN_WEBAUTHN_BEGIN_REGISTER', 'Register passkey'),
  ('ADMIN_WEBAUTHN_BEGIN_ASSERT', 'Verify with passkey'),
  ('ADMIN_WEBAUTHN_REMOVE', 'Remove passkey'),
  ('ADMIN_WEBAUTHN_NAME_LABEL', 'Passkey label'),
  ('ADMIN_WEBAUTHN_COL_NAME', 'Label'),
  ('ADMIN_WEBAUTHN_COL_CREATED', 'Created'),
  ('ADMIN_WEBAUTHN_COL_LAST_USED', 'Last used'),
  ('ADMIN_WEBAUTHN_RP_MISSING', 'Set ADMIN_WEBAUTHN_RP_ID or ADMIN_ALLOWED_ORIGINS before passkey enroll.'),
  ('ADMIN_WEBAUTHN_UNAVAILABLE', 'Passkey step-up is unavailable.'),
  ('ADMIN_WEBAUTHN_REGISTER_FAILED', 'Passkey registration failed.'),
  ('ADMIN_WEBAUTHN_ASSERT_FAILED', 'Passkey verification failed.'),
  ('ADMIN_WEBAUTHN_NOT_ENROLLED', 'No passkey enrolled for this admin.'),
  ('ADMIN_WEBAUTHN_ALREADY', 'A passkey with this credential already exists.'),
  ('ADMIN_WEBAUTHN_BROWSER_UNSUPPORTED', 'This browser does not support WebAuthn.'),
  ('ADMIN_STEP_UP_FACTOR_REQUIRED', 'Enroll TOTP or a passkey before step-up protected actions.'),
  ('ADMIN_CHECKLIST_WEBAUTHN_RP', 'WebAuthn RP id configured')
on conflict (code) do update set body = excluded.body;
