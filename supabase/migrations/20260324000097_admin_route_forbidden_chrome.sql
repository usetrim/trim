-- Route-level forbidden chrome for admins deep-linking without nav permission.

insert into public.site_messages (code, body) values
  ('ADMIN_ROUTE_FORBIDDEN', 'You do not have permission to open this admin page.')
on conflict (code) do update set body = excluded.body;
