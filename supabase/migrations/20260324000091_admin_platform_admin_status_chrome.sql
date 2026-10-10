-- Platform admin roster + break-glass status labels (no client invent of raw codes).

insert into public.site_messages (code, body) values
  ('ADMIN_PLATFORM_ADMIN_STATUS_ACTIVE', 'Active'),
  ('ADMIN_PLATFORM_ADMIN_STATUS_DISABLED', 'Disabled'),
  ('ADMIN_BREAK_GLASS_STATUS_PENDING', 'Pending'),
  ('ADMIN_BREAK_GLASS_STATUS_APPROVED', 'Approved'),
  ('ADMIN_BREAK_GLASS_STATUS_DENIED', 'Denied'),
  ('ADMIN_BREAK_GLASS_STATUS_REVOKED', 'Revoked')
on conflict (code) do update set body = excluded.body;
