-- Success toast bodies for admin + web CRUD mutations. Handlers return these as
-- `message` so clients never invent success copy (fail-closed to empty).

insert into public.site_messages (code, body) values
  -- admin denylist
  ('ADMIN_DENYLIST_EMAIL_ADDED', 'Email domain added to denylist.'),
  ('ADMIN_DENYLIST_EMAIL_REMOVED', 'Email domain removed from denylist.'),
  ('ADMIN_DENYLIST_IP_ADDED', 'IP range added to denylist.'),
  ('ADMIN_DENYLIST_IP_REMOVED', 'IP range removed from denylist.'),
  ('ADMIN_DENYLIST_ASN_ADDED', 'ASN added to denylist.'),
  ('ADMIN_DENYLIST_ASN_REMOVED', 'ASN removed from denylist.'),
  -- admin settings saves
  ('ADMIN_CHROME_MESSAGE_SAVED', 'Site message saved.'),
  ('ADMIN_LEGAL_SECTION_SAVED', 'Legal section saved.'),
  ('ADMIN_EMAIL_TEMPLATE_SAVED', 'Email template saved.'),
  ('ADMIN_PRODUCT_SETTINGS_SAVED', 'Product settings saved.'),
  ('ADMIN_AUTH_SETTINGS_SAVED', 'Auth settings saved.'),
  ('ADMIN_BILLING_SETTINGS_SAVED', 'Billing settings saved.'),
  ('ADMIN_COMPLIANCE_RETENTION_SAVED', 'Retention settings saved.'),
  -- admin plans / credits / disputes
  ('ADMIN_PLAN_CREATED', 'Plan created.'),
  ('ADMIN_PLAN_SAVED', 'Plan saved.'),
  ('ADMIN_PADDLE_CATALOG_SYNCED', 'Paddle catalog synced.'),
  ('ADMIN_CREDIT_GRANTED', 'Credits granted.'),
  ('ADMIN_DISPUTE_NOTE_SAVED', 'Dispute note saved.'),
  -- admin users
  ('ADMIN_USER_STATUS_UPDATED', 'Account status updated.'),
  ('ADMIN_USER_NOTES_SAVED', 'Admin notes saved.'),
  ('ADMIN_USER_QUOTA_UPDATED', 'User quota updated.'),
  ('ADMIN_USER_KEYS_REVOKED', 'API keys revoked.'),
  -- admin enterprise / break-glass / rbac
  ('ADMIN_ENTERPRISE_INQUIRY_UPDATED', 'Enterprise inquiry updated.'),
  ('ADMIN_BREAK_GLASS_REQUESTED', 'Break-glass request submitted.'),
  ('ADMIN_BREAK_GLASS_REVOKED', 'Break-glass access revoked.'),
  ('ADMIN_BREAK_GLASS_RESOLVED', 'Break-glass request resolved.'),
  ('ADMIN_ROLE_CREATED', 'Role created.'),
  ('ADMIN_ROLE_SAVED', 'Role updated.'),
  ('ADMIN_ROLE_DELETED', 'Role deleted.'),
  ('ADMIN_ADMIN_INVITED', 'Admin invited.'),
  ('ADMIN_ADMIN_REMOVED', 'Admin removed.'),
  -- admin distribution / exports
  ('ADMIN_DISTRIBUTION_SYNCED', 'Distribution metrics synced.'),
  ('ADMIN_AUDIT_EXPORT_READY', 'Audit export ready.'),
  ('ADMIN_ACCESS_REVIEW_EXPORT_READY', 'Access review export ready.'),
  -- web account / keys
  ('API_KEY_CREATED', 'API key created.'),
  ('API_KEY_REVOKED', 'API key revoked.'),
  ('API_KEY_DEVICE_REGISTERED', 'Device registered.'),
  ('API_KEY_DEVICE_REMOVED', 'Device removed.'),
  -- web workspaces
  ('WORKSPACE_CREATED', 'Workspace created.'),
  ('WORKSPACE_INVITE_REVOKED', 'Invite revoked.'),
  ('WORKSPACE_MEMBER_REMOVED', 'Member removed.'),
  ('WORKSPACE_INVITE_ACCEPTED', 'Invite accepted.')
on conflict (code) do update set body = excluded.body, updated_at = now();
