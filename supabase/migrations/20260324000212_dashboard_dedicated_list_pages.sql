-- Dedicated customer dashboard routes: Traces, Receipts list, Enterprise inquiries.
-- Update enterprise offer deep-link prefix to the new Enterprise page.

insert into public.site_messages (code, body) values
  ('APP_PATH_TRACES', '/dashboard/traces'),
  ('APP_PATH_RECEIPTS', '/dashboard/receipts'),
  ('APP_PATH_ENTERPRISE', '/dashboard/enterprise'),
  ('DASHBOARD_NAV_TRACES', 'Traces'),
  ('DASHBOARD_NAV_RECEIPTS', 'Receipts'),
  ('DASHBOARD_NAV_ENTERPRISE', 'Enterprise'),
  ('APP_PATH_DASHBOARD_ENTERPRISE_INQUIRY_PREFIX', '/dashboard/enterprise?id=')
on conflict (code) do update set body = excluded.body, updated_at = now();

update public.notification_kinds
set href_ref = 'APP_PATH_DASHBOARD_ENTERPRISE_INQUIRY_PREFIX',
    href_ref_kind = 'app_path_prefix'
where code in ('user.enterprise_offer_ready', 'user.enterprise_activated');
