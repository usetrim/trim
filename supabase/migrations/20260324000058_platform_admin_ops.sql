-- Admin fail-closed chrome + break-glass TTL (no invent hour).
alter table public.admin_retention_settings
  add column if not exists break_glass_ttl_minutes int;

comment on column public.admin_retention_settings.break_glass_ttl_minutes is
  'Approved break-glass elevation length in minutes. Required before approve; no invent default.';

insert into public.site_messages (code, body) values
  ('AUTH_ACCOUNT_DISABLED', 'This account is disabled by the platform operator.'),
  ('ADMIN_STEP_UP_TTL_MISSING', 'Set TRIM_ADMIN_STEP_UP_TTL_SEC (60-3600) before step-up can run.'),
  ('ADMIN_INSTALL_PATH_REQUIRED', 'Query path is required for install-hit (no invent default).'),
  ('ADMIN_INSTALL_PATH_INVALID', 'Install-hit path is invalid.'),
  ('ADMIN_BREAK_GLASS_TTL_MISSING', 'Set admin_retention_settings.break_glass_ttl_minutes before approving break-glass.'),
  ('ADMIN_GDPR_EXPORT_READY', 'GDPR export packaged.'),
  ('ADMIN_GDPR_ERASE_DONE', 'GDPR erase completed.'),
  ('ADMIN_FORCE_LOGOUT_DONE', 'API keys revoked and force-logout marker set.'),
  ('ADMIN_WEBHOOK_REPLAY_DONE', 'Webhook marked for reprocess.'),
  ('ADMIN_WEBHOOK_NOT_FOUND', 'Webhook event not found.'),
  ('ADMIN_PRODUCT_SETTINGS_MISSING', 'Product operator settings row missing.'),
  ('ADMIN_CHECKLIST_POSTGRES', 'PostgreSQL reachable'),
  ('ADMIN_CHECKLIST_REDIS', 'Redis reachable'),
  ('ADMIN_CHECKLIST_OWNERS', 'Active platform owner assigned'),
  ('ADMIN_CHECKLIST_GITHUB', 'GitHub distribution sync configured'),
  ('ADMIN_CHECKLIST_PRICING', 'Billing pricing bound'),
  ('ADMIN_CHECKLIST_COMPANY', 'COMPANY_LEGAL_NAME set'),
  ('ADMIN_CHECKLIST_GEOLITE', 'GeoLite path configured'),
  ('ADMIN_SEGMENT_INDIVIDUALS', 'Individuals'),
  ('ADMIN_SEGMENT_TEAMS', 'Teams'),
  ('ADMIN_SEGMENT_ENTERPRISE', 'Enterprise'),
  ('ADMIN_NAV_DASHBOARD', 'Command center'),
  ('ADMIN_NAV_USERS', 'Users'),
  ('ADMIN_NAV_SEGMENTS', 'Segments'),
  ('ADMIN_NAV_RBAC', 'Roles and admins'),
  ('ADMIN_NAV_BILLING', 'Billing'),
  ('ADMIN_NAV_PLANS', 'Plans'),
  ('ADMIN_NAV_SETTINGS_BILLING', 'Billing settings'),
  ('ADMIN_NAV_SUBSCRIPTIONS', 'Subscriptions'),
  ('ADMIN_NAV_RECEIPTS', 'Receipts'),
  ('ADMIN_NAV_ENTERPRISE', 'Enterprise'),
  ('ADMIN_NAV_DENYLIST', 'Denylist'),
  ('ADMIN_NAV_PRODUCT', 'Product'),
  ('ADMIN_NAV_AUTH', 'Auth'),
  ('ADMIN_NAV_CHROME', 'Chrome'),
  ('ADMIN_NAV_OBSERVABILITY', 'Observability'),
  ('ADMIN_NAV_DISTRIBUTION', 'Distribution'),
  ('ADMIN_NAV_AUDIT', 'Audit'),
  ('ADMIN_NAV_COMPLIANCE', 'Compliance'),
  ('ADMIN_NAV_BREAK_GLASS', 'Break-glass'),
  ('ADMIN_THEME_SYSTEM', 'System'),
  ('ADMIN_THEME_LIGHT', 'Light'),
  ('ADMIN_THEME_DARK', 'Dark'),
  ('ADMIN_LOGIN_REQUIRED', 'Sign in with an allowed provider to open the operator console.'),
  ('ADMIN_FORBIDDEN_PAGE', 'You are signed in but not a platform admin on this instance.')
on conflict (code) do nothing;

-- Operator-tunable product knobs (env remains source for process limits; DB mirrors for admin UI).
create table if not exists public.admin_product_settings (
  id text primary key default 'default',
  default_compression_mode text,
  default_deep_engine text,
  treesitter_required boolean,
  deep_attach_default boolean,
  model_routing_enabled boolean,
  updated_at timestamptz not null default now(),
  updated_by uuid references public.profiles(id) on delete set null
);

insert into public.admin_product_settings (id)
values ('default')
on conflict (id) do nothing;

create table if not exists public.admin_dispute_notes (
  id uuid primary key default gen_random_uuid(),
  paddle_transaction_id text,
  user_id uuid references public.profiles(id) on delete set null,
  note text not null,
  status text not null default 'open',
  created_by uuid references public.profiles(id) on delete set null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint admin_dispute_notes_status_check check (status in ('open', 'watching', 'closed'))
);

create table if not exists public.admin_credit_grants (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references public.profiles(id) on delete cascade,
  credits int not null check (credits <> 0),
  reason text not null,
  created_by uuid references public.profiles(id) on delete set null,
  created_at timestamptz not null default now()
);
