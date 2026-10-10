-- Operator chrome UI map (no client invent limit:500).
-- Audit export filename + row cap from DB (no hardcoded limit 5000 / filename).
-- Access-review attestations list cap from DB (no hardcoded limit 20).

alter table public.admin_retention_settings
  add column if not exists audit_export_max_rows int,
  add column if not exists access_review_attestations_limit int;

comment on column public.admin_retention_settings.audit_export_max_rows is
  'Max rows returned by GET /api/v1/admin/audit/export. Fail closed when null or <= 0.';
comment on column public.admin_retention_settings.access_review_attestations_limit is
  'Max attestation rows on GET /api/v1/admin/compliance/access-review. Fail closed when null or <= 0.';

update public.admin_retention_settings
set
  audit_export_max_rows = coalesce(audit_export_max_rows, 5000),
  access_review_attestations_limit = coalesce(access_review_attestations_limit, 20)
where id = 'default';

insert into public.site_messages (code, body) values
  ('ADMIN_AUDIT_EXPORT_FILENAME_FMT', 'admin-audit-{generated_at}.json'),
  ('ADMIN_AUDIT_EXPORT_LIMIT_MISSING', 'Set admin_retention_settings.audit_export_max_rows before exporting audit logs.'),
  ('ADMIN_ACCESS_REVIEW_ATTEST_LIMIT_MISSING', 'Set admin_retention_settings.access_review_attestations_limit before loading access review.'),
  ('ADMIN_CHECKLIST_MIGRATIONS_AUDIT_EXPORT', 'Audit export / chrome UI map migration applied'),
  ('ADMIN_COMPLIANCE_AUDIT_EXPORT_MAX', 'Audit export max rows'),
  ('ADMIN_COMPLIANCE_ATTEST_LIMIT', 'Access review attestations limit')
on conflict (code) do update set body = excluded.body;
