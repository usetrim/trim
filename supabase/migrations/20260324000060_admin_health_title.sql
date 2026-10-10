-- Health section title for admin command center (fail-closed if empty).
insert into public.site_messages (code, body) values
  ('ADMIN_HEALTH_TITLE', 'Health')
on conflict (code) do nothing;
