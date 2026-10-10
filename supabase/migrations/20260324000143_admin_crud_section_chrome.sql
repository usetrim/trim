-- Operator CRUD section chrome (create / edit / danger) for clear form flows.
insert into public.site_messages (code, body) values
  ('ADMIN_SECTION_CREATE', 'Create'),
  ('ADMIN_SECTION_EDIT', 'Edit'),
  ('ADMIN_SECTION_DANGER', 'Danger zone'),
  ('ADMIN_PLAN_EDIT', 'Edit plan'),
  ('ADMIN_ENTERPRISE_EDIT', 'Edit inquiry')
on conflict (code) do nothing;
