-- Export downloads as Excel (.xlsx), not JSON.
update public.site_messages
set body = replace(body, '.json', '.xlsx'),
    updated_at = now()
where code in (
  'ADMIN_AUDIT_EXPORT_FILENAME_FMT',
  'ADMIN_ACCESS_REVIEW_FILENAME_FMT',
  'ADMIN_GDPR_EXPORT_FILENAME_FMT'
)
  and body like '%.json';

insert into public.site_messages (code, body)
values
  ('ADMIN_AUDIT_EXPORT_FILENAME_FMT', 'admin-audit-{generated_at}.xlsx'),
  ('ADMIN_ACCESS_REVIEW_FILENAME_FMT', 'access-review-{generated_at}.xlsx'),
  ('ADMIN_GDPR_EXPORT_FILENAME_FMT', 'gdpr-export-{user_id}-{generated_at}.xlsx')
on conflict (code) do update
set body = excluded.body,
    updated_at = now()
where public.site_messages.body like '%.json'
   or public.site_messages.body is distinct from excluded.body;
