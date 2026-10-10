-- Compliance break-glass TTL label, audit column chrome, dispute list chrome, retention missing.

insert into public.site_messages (code, body) values
  ('ADMIN_COMPLIANCE_BREAK_GLASS_TTL', 'Break-glass TTL (minutes)'),
  ('ADMIN_RETENTION_SETTINGS_MISSING', 'Admin retention settings row is missing.'),
  ('ADMIN_AUDIT_COL_ACTOR', 'Actor'),
  ('ADMIN_AUDIT_COL_REASON', 'Reason'),
  ('ADMIN_AUDIT_COL_STEP_UP', 'Step-up'),
  ('ADMIN_AUDIT_STEP_UP_YES', 'Yes'),
  ('ADMIN_AUDIT_STEP_UP_NO', 'No'),
  ('ADMIN_DISPUTE_LIST_TITLE', 'Dispute notes'),
  ('ADMIN_DISPUTE_COL_WHEN', 'When'),
  ('ADMIN_DISPUTE_COL_CREATED_BY', 'Created by')
on conflict (code) do nothing;
