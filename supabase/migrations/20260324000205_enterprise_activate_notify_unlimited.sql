-- Enterprise activation completeness: fail-closed credits, customer notify, contract status chrome.
-- Sales-led enterprise with credits_monthly=0 must be unlimited (otherwise activate yields 0/0 exhausted).

-- Catalog repair: metered enterprise with 0 monthly credits is unusable after sales activate.
update public.plan_catalog
set unlimited = true,
    updated_at = now()
where is_active = true
  and plan_kind = 'enterprise'
  and credits_monthly = 0
  and coalesce(unlimited, false) = false;

insert into public.notification_kinds (
  code, audience, title_message_code, body_message_code, href_ref_kind, href_ref
) values
  (
    'user.enterprise_activated',
    'user',
    'NOTIF_USER_ENTERPRISE_ACTIVE_TITLE',
    'NOTIF_USER_ENTERPRISE_ACTIVE_BODY',
    'app_path',
    'APP_PATH_DASHBOARD'
  )
on conflict (code) do update set
  audience = excluded.audience,
  title_message_code = excluded.title_message_code,
  body_message_code = excluded.body_message_code,
  href_ref_kind = excluded.href_ref_kind,
  href_ref = excluded.href_ref;

insert into public.site_messages (code, body) values
  ('STATUS_ENTERPRISE_CONTRACT', 'Enterprise contract'),
  ('ADMIN_ENTERPRISE_CREDITS_REQUIRED', 'Enterprise plan has no monthly credits and is not Unlimited. Set credits_monthly or turn Unlimited on in Plans before activating.'),
  ('ADMIN_ENTERPRISE_STATUS_DESC', 'Inquiry workflow status (new, contacted, closed, or activated).'),
  ('NOTIF_USER_ENTERPRISE_ACTIVE_TITLE', 'Enterprise plan activated'),
  ('NOTIF_USER_ENTERPRISE_ACTIVE_BODY', 'Your enterprise contract is active (%s seats). Cloud access follows your Enterprise plan entitlements.')
on conflict (code) do update set body = excluded.body, updated_at = now();
