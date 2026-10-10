-- Distinct clear/"all" label for optional enum filter Selects (not the field title).
insert into public.site_messages (code, body) values
  ('ADMIN_FILTER_ALL', 'All')
on conflict (code) do nothing;
