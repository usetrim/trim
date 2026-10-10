-- Platform operator console (apps/admin): RBAC, audit, account status, OSS distribution.
-- Instance-local only. Never a Trim-Inc backdoor into other people's installs.
-- Workspace owner/admin/member stays separate.

-- ---------------------------------------------------------------------------
-- Account status on profiles (support actions)
-- ---------------------------------------------------------------------------
alter table public.profiles
  add column if not exists account_status text not null default 'active',
  add column if not exists account_status_reason text,
  add column if not exists account_status_changed_at timestamptz,
  add column if not exists account_status_changed_by uuid references public.profiles(id) on delete set null,
  add column if not exists last_login_country text,
  add column if not exists last_login_at timestamptz,
  add column if not exists admin_notes text;

alter table public.profiles
  drop constraint if exists profiles_account_status_check;
alter table public.profiles
  add constraint profiles_account_status_check
  check (account_status in ('active', 'suspended', 'banned', 'pending_delete', 'shadowbanned'));

comment on column public.profiles.account_status is
  'Platform support status; independent of workspace roles and subscription status.';
comment on column public.profiles.last_login_country is
  'Coarse CF/Geo country code only; never street address.';

-- ---------------------------------------------------------------------------
-- Fixed permission catalog (API enforces these codes only; no free-typed invent)
-- ---------------------------------------------------------------------------
create table if not exists public.platform_permission_catalog (
  code text primary key,
  description text not null,
  category text not null,
  step_up_required boolean not null default false,
  sort_order int not null default 100,
  created_at timestamptz not null default now()
);

comment on table public.platform_permission_catalog is
  'Canonical admin capabilities. Roles may only grant codes present here.';

insert into public.platform_permission_catalog (code, description, category, step_up_required, sort_order) values
  ('admin.access', 'Enter the operator console', 'core', false, 1),
  ('dashboard.read', 'View command-center KPIs and health', 'core', false, 10),
  ('users.read', 'List and view user profiles', 'users', false, 20),
  ('users.suspend', 'Suspend or restore accounts', 'users', true, 21),
  ('users.ban', 'Ban or shadowban accounts', 'users', true, 22),
  ('users.notes', 'Edit admin notes on accounts', 'users', false, 23),
  ('users.quota', 'Adjust user quota / credits', 'users', true, 24),
  ('users.keys', 'Revoke API keys cross-user', 'users', true, 25),
  ('users.gdpr', 'Trigger GDPR export or erase', 'users', true, 26),
  ('users.force_logout', 'Invalidate sessions / force logout', 'users', true, 27),
  ('segments.read', 'View individuals / teams / enterprise segments', 'segments', false, 30),
  ('enterprise.read', 'View enterprise inquiries', 'segments', false, 31),
  ('enterprise.write', 'Update enterprise inquiry status and notes', 'segments', false, 32),
  ('billing.read', 'View plans, subscriptions, receipts', 'billing', false, 40),
  ('billing.plans', 'Edit plan_catalog', 'billing', true, 41),
  ('billing.settings', 'Edit billing_settings', 'billing', true, 42),
  ('billing.receipts', 'Resync receipts / PDF ops', 'billing', false, 43),
  ('billing.credits', 'Manual credit grant / top-up ledger', 'billing', true, 44),
  ('product.read', 'View compression and fraud settings', 'product', false, 50),
  ('product.write', 'Edit product / rate-limit / fraud thresholds', 'product', true, 51),
  ('auth.read', 'View auth_settings and OAuth chrome', 'auth', false, 60),
  ('auth.write', 'Toggle allowed auth providers', 'auth', true, 61),
  ('denylist.read', 'View email / IP / ASN denylists', 'fraud', false, 70),
  ('denylist.write', 'Edit denylists', 'fraud', true, 71),
  ('chrome.read', 'View site_messages', 'chrome', false, 80),
  ('chrome.write', 'Edit site_messages and legal chrome', 'chrome', true, 81),
  ('observability.read', 'View global events and webhooks', 'ops', false, 90),
  ('webhooks.replay', 'Replay paddle webhook events', 'ops', true, 91),
  ('distribution.read', 'View OSS clone / download / install stats', 'ops', false, 92),
  ('distribution.sync', 'Sync GitHub / release distribution stats', 'ops', false, 93),
  ('audit.read', 'Read admin audit log', 'security', false, 100),
  ('audit.export', 'Export audit log', 'security', false, 101),
  ('rbac.read', 'View roles, permissions, platform admins', 'security', false, 110),
  ('rbac.write', 'Create/edit roles and permission grants', 'security', true, 111),
  ('admins.invite', 'Invite or remove platform admins', 'security', true, 112),
  ('admins.break_glass', 'Request or approve break-glass elevation', 'security', true, 113),
  ('compliance.read', 'View retention and access-review data', 'compliance', false, 120),
  ('compliance.write', 'Edit retention policy / run access review export', 'compliance', true, 121)
on conflict (code) do nothing;

-- ---------------------------------------------------------------------------
-- Roles (system seeds + owner-created custom roles)
-- ---------------------------------------------------------------------------
create table if not exists public.platform_roles (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  slug text not null unique,
  description text not null default '',
  is_system boolean not null default false,
  is_owner bool not null default false,
  created_by uuid references public.profiles(id) on delete set null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists public.platform_role_permissions (
  role_id uuid not null references public.platform_roles(id) on delete cascade,
  permission_code text not null references public.platform_permission_catalog(code) on delete cascade,
  primary key (role_id, permission_code)
);

create table if not exists public.platform_admins (
  user_id uuid primary key references public.profiles(id) on delete cascade,
  role_id uuid not null references public.platform_roles(id) on delete restrict,
  status text not null default 'active',
  invited_by uuid references public.profiles(id) on delete set null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint platform_admins_status_check check (status in ('active', 'disabled'))
);

create table if not exists public.platform_admin_invites (
  id uuid primary key default gen_random_uuid(),
  email text not null,
  role_id uuid not null references public.platform_roles(id) on delete cascade,
  token_hash text not null unique,
  invited_by uuid references public.profiles(id) on delete set null,
  expires_at timestamptz not null,
  accepted_at timestamptz,
  created_at timestamptz not null default now()
);

create index if not exists idx_platform_admin_invites_email
  on public.platform_admin_invites (lower(email));

-- ---------------------------------------------------------------------------
-- Immutable audit log
-- ---------------------------------------------------------------------------
create table if not exists public.admin_audit_log (
  id bigserial primary key,
  actor_user_id uuid references public.profiles(id) on delete set null,
  action text not null,
  resource_type text not null,
  resource_id text,
  before_json jsonb,
  after_json jsonb,
  reason text,
  ip text,
  ja4 text,
  country text,
  step_up_used boolean not null default false,
  created_at timestamptz not null default now()
);

create index if not exists idx_admin_audit_created on public.admin_audit_log (created_at desc);
create index if not exists idx_admin_audit_actor on public.admin_audit_log (actor_user_id, created_at desc);

comment on table public.admin_audit_log is
  'Append-only operator audit trail. Do not update or delete rows from app code.';

-- ---------------------------------------------------------------------------
-- Break-glass (time-boxed dual control)
-- ---------------------------------------------------------------------------
create table if not exists public.admin_break_glass (
  id uuid primary key default gen_random_uuid(),
  requester_id uuid not null references public.profiles(id) on delete cascade,
  approver_id uuid references public.profiles(id) on delete set null,
  reason text not null,
  status text not null default 'pending',
  elevates_permission text,
  starts_at timestamptz,
  ends_at timestamptz,
  created_at timestamptz not null default now(),
  constraint admin_break_glass_status_check
    check (status in ('pending', 'approved', 'denied', 'expired', 'revoked'))
);

-- ---------------------------------------------------------------------------
-- Compliance / retention
-- ---------------------------------------------------------------------------
create table if not exists public.admin_retention_settings (
  id text primary key default 'default',
  trim_events_ttl_days int,
  audit_log_ttl_days int,
  updated_at timestamptz not null default now(),
  updated_by uuid references public.profiles(id) on delete set null
);

insert into public.admin_retention_settings (id, trim_events_ttl_days, audit_log_ttl_days)
values ('default', null, null)
on conflict (id) do nothing;

-- ---------------------------------------------------------------------------
-- OSS distribution (aggregate only; country coarse; no street address)
-- ---------------------------------------------------------------------------
create table if not exists public.distribution_daily_stats (
  day date not null,
  source text not null,
  metric text not null,
  country text not null default '',
  value bigint not null default 0,
  primary key (day, source, metric, country)
);

comment on table public.distribution_daily_stats is
  'Aggregated GitHub traffic, release downloads, install.sh hits. Country is ISO code or empty.';

create table if not exists public.distribution_sync_state (
  source text primary key,
  last_synced_at timestamptz,
  last_error text,
  meta jsonb
);

-- Optional install beacon from use-trim.com/install.sh (country from CF-IPCountry only)
create table if not exists public.install_hits (
  id bigserial primary key,
  hit_at timestamptz not null default now(),
  country text not null default '',
  path text not null default '/install.sh',
  user_agent_hash text
);

-- Index hit_at directly: (timestamptz::date) is not IMMUTABLE (session TimeZone),
-- so it cannot be used in an expression index. Day buckets use range predicates on hit_at.
create index if not exists idx_install_hits_hit_at on public.install_hits (hit_at);

-- ---------------------------------------------------------------------------
-- IP / ASN denylist (email denylist already exists)
-- ---------------------------------------------------------------------------
create table if not exists public.ip_denylist (
  cidr cidr primary key,
  reason text not null default '',
  created_by uuid references public.profiles(id) on delete set null,
  created_at timestamptz not null default now()
);

create table if not exists public.asn_denylist (
  asn bigint primary key,
  reason text not null default '',
  created_by uuid references public.profiles(id) on delete set null,
  created_at timestamptz not null default now()
);

-- ---------------------------------------------------------------------------
-- Seed system roles
-- ---------------------------------------------------------------------------
insert into public.platform_roles (id, name, slug, description, is_system, is_owner)
values
  ('a0000000-0000-4000-8000-000000000001', 'Owner', 'owner', 'Full platform control including RBAC', true, true),
  ('a0000000-0000-4000-8000-000000000002', 'Viewer', 'viewer', 'Read-only command center and directories', true, false),
  ('a0000000-0000-4000-8000-000000000003', 'Support', 'support', 'User support actions', true, false),
  ('a0000000-0000-4000-8000-000000000004', 'Billing', 'billing', 'Plans, receipts, credits', true, false),
  ('a0000000-0000-4000-8000-000000000005', 'Ops', 'ops', 'Product settings, webhooks, distribution', true, false),
  ('a0000000-0000-4000-8000-000000000006', 'Security', 'security', 'Fraud, denylist, audit, break-glass', true, false)
on conflict (slug) do nothing;

-- Owner: all permissions
insert into public.platform_role_permissions (role_id, permission_code)
select 'a0000000-0000-4000-8000-000000000001', code from public.platform_permission_catalog
on conflict do nothing;

-- Viewer
insert into public.platform_role_permissions (role_id, permission_code)
select 'a0000000-0000-4000-8000-000000000002', code from public.platform_permission_catalog
where code in (
  'admin.access', 'dashboard.read', 'users.read', 'segments.read', 'enterprise.read',
  'billing.read', 'product.read', 'auth.read', 'denylist.read', 'chrome.read',
  'observability.read', 'distribution.read', 'audit.read', 'rbac.read', 'compliance.read'
)
on conflict do nothing;

-- Support
insert into public.platform_role_permissions (role_id, permission_code)
select 'a0000000-0000-4000-8000-000000000003', code from public.platform_permission_catalog
where code in (
  'admin.access', 'dashboard.read', 'users.read', 'users.suspend', 'users.notes',
  'users.quota', 'users.keys', 'users.force_logout', 'segments.read', 'billing.read',
  'observability.read', 'audit.read'
)
on conflict do nothing;

-- Billing
insert into public.platform_role_permissions (role_id, permission_code)
select 'a0000000-0000-4000-8000-000000000004', code from public.platform_permission_catalog
where code in (
  'admin.access', 'dashboard.read', 'users.read', 'segments.read', 'enterprise.read',
  'enterprise.write', 'billing.read', 'billing.plans', 'billing.settings',
  'billing.receipts', 'billing.credits', 'audit.read'
)
on conflict do nothing;

-- Ops
insert into public.platform_role_permissions (role_id, permission_code)
select 'a0000000-0000-4000-8000-000000000005', code from public.platform_permission_catalog
where code in (
  'admin.access', 'dashboard.read', 'product.read', 'product.write', 'auth.read',
  'auth.write', 'chrome.read', 'chrome.write', 'observability.read', 'webhooks.replay',
  'distribution.read', 'distribution.sync', 'audit.read'
)
on conflict do nothing;

-- Security
insert into public.platform_role_permissions (role_id, permission_code)
select 'a0000000-0000-4000-8000-000000000006', code from public.platform_permission_catalog
where code in (
  'admin.access', 'dashboard.read', 'users.read', 'users.ban', 'users.gdpr',
  'denylist.read', 'denylist.write', 'audit.read', 'audit.export', 'rbac.read',
  'admins.break_glass', 'compliance.read', 'compliance.write', 'observability.read'
)
on conflict do nothing;

-- ---------------------------------------------------------------------------
-- site_messages chrome for admin API (fail-closed; operators may edit)
-- ---------------------------------------------------------------------------
insert into public.site_messages (code, body) values
  ('ADMIN_FORBIDDEN', 'Platform admin access required.'),
  ('ADMIN_PERMISSION_DENIED', 'Missing permission for this admin action.'),
  ('ADMIN_STEP_UP_REQUIRED', 'Step-up verification required for this action. Re-authenticate and retry.'),
  ('ADMIN_STEP_UP_INVALID', 'Step-up token invalid or expired.'),
  ('ADMIN_ROLE_NOT_FOUND', 'Platform role not found.'),
  ('ADMIN_ROLE_SYSTEM_LOCKED', 'System roles cannot be deleted.'),
  ('ADMIN_ROLE_OWNER_LOCKED', 'Owner role permissions cannot be reduced.'),
  ('ADMIN_PERMISSION_UNKNOWN', 'Unknown permission code.'),
  ('ADMIN_ADMIN_NOT_FOUND', 'Platform admin not found.'),
  ('ADMIN_CANNOT_REMOVE_LAST_OWNER', 'Cannot remove or demote the last active owner.'),
  ('ADMIN_USER_NOT_FOUND', 'User not found.'),
  ('ADMIN_STATUS_INVALID', 'Invalid account status.'),
  ('ADMIN_REASON_REQUIRED', 'A reason is required for this action.'),
  ('ADMIN_BOOTSTRAP_REQUIRED', 'No platform owner configured. Set TRIM_PLATFORM_OWNER_EMAILS and restart API.'),
  ('ADMIN_INVITE_INVALID', 'Admin invite invalid or expired.'),
  ('ADMIN_BREAK_GLASS_NOT_FOUND', 'Break-glass request not found.'),
  ('ADMIN_BREAK_GLASS_SELF_APPROVE', 'Requester cannot approve their own break-glass request.'),
  ('ADMIN_GITHUB_NOT_CONFIGURED', 'GitHub distribution sync is not configured (TRIM_GITHUB_TOKEN / TRIM_GITHUB_REPO).'),
  ('ADMIN_DISTRIBUTION_SYNC_FAILED', 'Distribution sync failed.'),
  ('ADMIN_PENDING_SAVING', 'Saving...'),
  ('ADMIN_PENDING_SUSPENDING', 'Suspending...'),
  ('ADMIN_PENDING_RESTORING', 'Restoring...'),
  ('ADMIN_PENDING_INVITING', 'Inviting...'),
  ('ADMIN_PENDING_SYNCING', 'Syncing...'),
  ('ADMIN_PENDING_DELETING', 'Deleting...'),
  ('ADMIN_BRAND', 'Trim Operator'),
  ('ADMIN_TAGLINE', 'Instance operator console')
on conflict (code) do nothing;
