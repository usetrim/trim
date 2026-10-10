-- Fix migration 203 (used invent column "message"; live schema is site_messages.body).
-- Sales & revenue IA: nav sections, revenue permission/page, range presets, settled-receipt analytics chrome.

-- ---------------------------------------------------------------------------
-- Repair enterprise detail chrome from 203
-- ---------------------------------------------------------------------------
insert into public.site_messages (code, body) values
  ('ADMIN_ENTERPRISE_COL_SEATS', 'Requested seats'),
  ('ADMIN_ENTERPRISE_COL_CREATED', 'Created'),
  ('ADMIN_ENTERPRISE_COL_MESSAGE', 'Message'),
  ('ADMIN_ENTERPRISE_DETAILS', 'Inquiry details'),
  ('ADMIN_ENTERPRISE_INQUIRY_MESSAGE', 'Customer message'),
  ('ADMIN_ENTERPRISE_INQUIRY_MESSAGE_DESC', 'Message the customer submitted with this enterprise inquiry.'),
  ('ADMIN_ENTERPRISE_ESTIMATED_SEATS', 'Requested seats'),
  ('ADMIN_ENTERPRISE_ESTIMATED_SEATS_DESC', 'Seat estimate the customer entered when submitting the inquiry (estimated_seats).'),
  ('ADMIN_ENTERPRISE_USER', 'User'),
  ('ADMIN_ENTERPRISE_USER_DESC', 'Signed-in account that submitted the inquiry.'),
  ('ADMIN_ENTERPRISE_CREATED', 'Created'),
  ('ADMIN_ENTERPRISE_UPDATED', 'Updated'),
  ('ADMIN_ENTERPRISE_CONTRACT_NOTES_DESC', 'Internal contract or sales notes stored on the enterprise inquiry (contract_notes).'),
  ('ADMIN_ENTERPRISE_OFFERED_SEATS_DESC', 'Seat quantity offered in the proposal (offered_seat_quantity).'),
  ('ADMIN_ENTERPRISE_STATUS_DESC', 'Inquiry workflow status (new, contacted, closed, or activated).')
on conflict (code) do update set body = excluded.body, updated_at = now();

-- ---------------------------------------------------------------------------
-- Nav sections (docs-style accordion groups)
-- ---------------------------------------------------------------------------
create table if not exists public.admin_nav_sections (
  id text primary key,
  chrome_code text not null,
  sort_order int not null default 0,
  is_active boolean not null default true
);

comment on table public.admin_nav_sections is
  'Platform admin sidebar accordion sections. Labels from site_messages via chrome_code; empty label hides the section.';

alter table public.admin_nav_items
  add column if not exists section_id text references public.admin_nav_sections(id) on delete set null;

create index if not exists idx_admin_nav_items_section
  on public.admin_nav_items (section_id, sort_order, id);

insert into public.admin_nav_sections (id, chrome_code, sort_order) values
  ('overview', 'ADMIN_NAV_SECTION_OVERVIEW', 10),
  ('customers', 'ADMIN_NAV_SECTION_CUSTOMERS', 20),
  ('sales', 'ADMIN_NAV_SECTION_SALES', 30),
  ('catalog', 'ADMIN_NAV_SECTION_CATALOG', 40),
  ('access', 'ADMIN_NAV_SECTION_ACCESS', 50),
  ('product', 'ADMIN_NAV_SECTION_PRODUCT', 60),
  ('platform', 'ADMIN_NAV_SECTION_PLATFORM', 70)
on conflict (id) do update set
  chrome_code = excluded.chrome_code,
  sort_order = excluded.sort_order,
  is_active = true;

-- ---------------------------------------------------------------------------
-- Permission + role grants for revenue analytics
-- ---------------------------------------------------------------------------
insert into public.platform_permission_catalog (code, description, category, step_up_required, sort_order) values
  ('billing.revenue', 'View settled revenue analytics (receipts ledger)', 'billing', false, 42)
on conflict (code) do update set
  description = excluded.description,
  category = excluded.category,
  sort_order = excluded.sort_order;

insert into public.platform_role_permissions (role_id, permission_code)
select 'a0000000-0000-4000-8000-000000000001', 'billing.revenue'
on conflict do nothing;

insert into public.platform_role_permissions (role_id, permission_code)
select 'a0000000-0000-4000-8000-000000000002', 'billing.revenue'
on conflict do nothing;

insert into public.platform_role_permissions (role_id, permission_code)
select 'a0000000-0000-4000-8000-000000000004', 'billing.revenue'
on conflict do nothing;

-- ---------------------------------------------------------------------------
-- Revenue range presets (API-driven; no invent option lists in UI)
-- ---------------------------------------------------------------------------
create table if not exists public.admin_revenue_range_presets (
  id text primary key,
  chrome_code text not null,
  sort_order int not null default 0,
  is_active boolean not null default true
);

comment on table public.admin_revenue_range_presets is
  'Sales revenue dashboard range selectors. Labels from site_messages; empty label hides the preset.';

insert into public.admin_revenue_range_presets (id, chrome_code, sort_order) values
  ('7d', 'ADMIN_REVENUE_RANGE_7D', 10),
  ('30d', 'ADMIN_REVENUE_RANGE_30D', 20),
  ('mtd', 'ADMIN_REVENUE_RANGE_MTD', 30),
  ('ytd', 'ADMIN_REVENUE_RANGE_YTD', 40),
  ('custom', 'ADMIN_REVENUE_RANGE_CUSTOM', 50)
on conflict (id) do update set
  chrome_code = excluded.chrome_code,
  sort_order = excluded.sort_order,
  is_active = true;

-- ---------------------------------------------------------------------------
-- Chrome labels
-- ---------------------------------------------------------------------------
insert into public.site_messages (code, body) values
  ('ADMIN_NAV_SECTION_OVERVIEW', 'Overview'),
  ('ADMIN_NAV_SECTION_CUSTOMERS', 'Customers'),
  ('ADMIN_NAV_SECTION_SALES', 'Sales and revenue'),
  ('ADMIN_NAV_SECTION_CATALOG', 'Catalog and billing config'),
  ('ADMIN_NAV_SECTION_ACCESS', 'Access and security'),
  ('ADMIN_NAV_SECTION_PRODUCT', 'Product and growth'),
  ('ADMIN_NAV_SECTION_PLATFORM', 'Platform'),
  ('ADMIN_NAV_REVENUE', 'Revenue'),
  ('ADMIN_REVENUE_TITLE', 'Revenue'),
  ('ADMIN_REVENUE_INTRO', 'Settled revenue from completed receipts only. Catalog price times seats is not used here.'),
  ('ADMIN_REVENUE_KPI_TOTAL', 'Settled revenue'),
  ('ADMIN_REVENUE_KPI_RECEIPTS', 'Completed receipts'),
  ('ADMIN_REVENUE_KPI_REFUNDED', 'Refunded receipts'),
  ('ADMIN_REVENUE_KPI_FREE_ACCOUNTS', 'Free accounts'),
  ('ADMIN_REVENUE_KPI_PAID_ACCOUNTS', 'Paid accounts'),
  ('ADMIN_REVENUE_CHART_TITLE', 'Settled revenue by plan'),
  ('ADMIN_REVENUE_BY_PLAN_TITLE', 'By plan'),
  ('ADMIN_REVENUE_COL_PLAN', 'Plan'),
  ('ADMIN_REVENUE_COL_KIND', 'Kind'),
  ('ADMIN_REVENUE_COL_REVENUE', 'Revenue'),
  ('ADMIN_REVENUE_COL_RECEIPTS', 'Receipts'),
  ('ADMIN_REVENUE_DRILL_TITLE', 'Receipts in range'),
  ('ADMIN_REVENUE_DRILL_LINK', 'Open receipts'),
  ('ADMIN_REVENUE_EMPTY', 'No completed receipts in this range.'),
  ('ADMIN_REVENUE_RANGE_LABEL', 'Range'),
  ('ADMIN_REVENUE_RANGE_DESC', 'Filter settled receipt revenue by time range.'),
  ('ADMIN_REVENUE_RANGE_7D', 'Last 7 days'),
  ('ADMIN_REVENUE_RANGE_30D', 'Last 30 days'),
  ('ADMIN_REVENUE_RANGE_MTD', 'Month to date'),
  ('ADMIN_REVENUE_RANGE_YTD', 'Year to date'),
  ('ADMIN_REVENUE_RANGE_CUSTOM', 'Custom calendar'),
  ('ADMIN_REVENUE_UNKNOWN_PLAN', 'Unmapped'),
  ('ADMIN_DASHBOARD_REVENUE_HINT', 'Deep settled revenue by plan lives under Sales and revenue.'),
  ('ADMIN_DASHBOARD_REVENUE_LINK', 'Open revenue')
on conflict (code) do update set body = excluded.body, updated_at = now();

-- ---------------------------------------------------------------------------
-- Assign nav items to sections + sales route hrefs + revenue item
-- ---------------------------------------------------------------------------
update public.admin_nav_items set section_id = 'overview', sort_order = 10 where id = 'dashboard';

update public.admin_nav_items set section_id = 'customers', sort_order = 10 where id = 'users';
update public.admin_nav_items set section_id = 'customers', sort_order = 20 where id = 'segments';

update public.admin_nav_items
set section_id = 'sales', href = '/sales/receipts', sort_order = 20
where id = 'receipts';
update public.admin_nav_items
set section_id = 'sales', href = '/sales/subscriptions', sort_order = 30
where id = 'subscriptions';
update public.admin_nav_items
set section_id = 'sales', href = '/sales/credits', sort_order = 40
where id = 'credits';
update public.admin_nav_items
set section_id = 'sales', href = '/sales/enterprise', sort_order = 50
where id = 'enterprise';

insert into public.admin_nav_items (id, href, chrome_code, permission_code, sort_order, section_id, is_active) values
  ('revenue', '/sales/revenue', 'ADMIN_NAV_REVENUE', 'billing.revenue', 10, 'sales', true)
on conflict (id) do update set
  href = excluded.href,
  chrome_code = excluded.chrome_code,
  permission_code = excluded.permission_code,
  sort_order = excluded.sort_order,
  section_id = excluded.section_id,
  is_active = true;

update public.admin_nav_items set section_id = 'catalog', sort_order = 10 where id = 'plans';
update public.admin_nav_items set section_id = 'catalog', sort_order = 20 where id = 'billing_settings';

update public.admin_nav_items set section_id = 'access', sort_order = 10 where id = 'rbac';
update public.admin_nav_items set section_id = 'access', sort_order = 20 where id = 'auth';
update public.admin_nav_items set section_id = 'access', sort_order = 30 where id = 'denylist';
update public.admin_nav_items set section_id = 'access', sort_order = 40 where id = 'break_glass';

update public.admin_nav_items set section_id = 'product', sort_order = 10 where id = 'product';
update public.admin_nav_items set section_id = 'product', sort_order = 20 where id = 'distribution';
update public.admin_nav_items set section_id = 'product', sort_order = 30 where id = 'email';

update public.admin_nav_items set section_id = 'platform', sort_order = 10 where id = 'chrome';
update public.admin_nav_items set section_id = 'platform', sort_order = 20 where id = 'observability';
update public.admin_nav_items set section_id = 'platform', sort_order = 30 where id = 'audit';
update public.admin_nav_items set section_id = 'platform', sort_order = 40 where id = 'compliance';
