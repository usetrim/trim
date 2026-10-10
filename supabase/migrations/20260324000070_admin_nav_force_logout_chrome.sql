-- DB-driven admin nav catalog (href + permission + chrome code). Labels stay in site_messages.
create table if not exists public.admin_nav_items (
  id text primary key,
  href text not null,
  chrome_code text not null,
  permission_code text not null,
  sort_order int not null default 0,
  is_active boolean not null default true
);

comment on table public.admin_nav_items is
  'Platform admin sidebar entries. Labels come from site_messages via chrome_code; empty label hides the link.';

insert into public.admin_nav_items (id, href, chrome_code, permission_code, sort_order) values
  ('dashboard', '/', 'ADMIN_NAV_DASHBOARD', 'dashboard.read', 10),
  ('users', '/users', 'ADMIN_NAV_USERS', 'users.read', 20),
  ('segments', '/segments', 'ADMIN_NAV_SEGMENTS', 'segments.read', 30),
  ('rbac', '/rbac', 'ADMIN_NAV_RBAC', 'rbac.read', 40),
  ('plans', '/billing/plans', 'ADMIN_NAV_PLANS', 'billing.read', 50),
  ('billing_settings', '/billing/settings', 'ADMIN_NAV_SETTINGS_BILLING', 'billing.settings', 60),
  ('subscriptions', '/billing/subscriptions', 'ADMIN_NAV_SUBSCRIPTIONS', 'billing.read', 70),
  ('receipts', '/billing/receipts', 'ADMIN_NAV_RECEIPTS', 'billing.receipts', 80),
  ('enterprise', '/enterprise', 'ADMIN_NAV_ENTERPRISE', 'enterprise.read', 90),
  ('denylist', '/denylist', 'ADMIN_NAV_DENYLIST', 'denylist.read', 100),
  ('product', '/product', 'ADMIN_NAV_PRODUCT', 'product.read', 110),
  ('auth', '/auth', 'ADMIN_NAV_AUTH', 'auth.read', 120),
  ('chrome', '/chrome', 'ADMIN_NAV_CHROME', 'chrome.read', 130),
  ('email', '/email', 'ADMIN_NAV_EMAIL', 'chrome.read', 140),
  ('observability', '/observability', 'ADMIN_NAV_OBSERVABILITY', 'observability.read', 150),
  ('distribution', '/distribution', 'ADMIN_NAV_DISTRIBUTION', 'distribution.read', 160),
  ('audit', '/audit', 'ADMIN_NAV_AUDIT', 'audit.read', 170),
  ('compliance', '/compliance', 'ADMIN_NAV_COMPLIANCE', 'compliance.read', 180),
  ('break_glass', '/break-glass', 'ADMIN_NAV_BREAK_GLASS', 'admins.break_glass', 190)
on conflict (id) do nothing;

insert into public.site_messages (code, body) values
  ('ADMIN_SUB_LAST_WEBHOOK', 'Last webhook'),
  ('ADMIN_OBS_CLI_VERSION', 'Minimum CLI version'),
  ('ADMIN_OBS_CLI_FORCE_UPGRADE', 'CLI force-upgrade notice'),
  ('AUTH_FORCE_LOGOUT', 'Your sessions were ended by an operator. Sign in again.')
on conflict (code) do nothing;
