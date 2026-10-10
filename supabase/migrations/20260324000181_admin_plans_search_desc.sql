-- Plans list search description (not the users email/id/name copy).
insert into public.site_messages (code, body) values
  ('ADMIN_PLANS_SEARCH_DESC', 'Search by plan id or display name shown in the list.')
on conflict (code) do update
  set body = excluded.body
  where public.site_messages.body is distinct from excluded.body;
