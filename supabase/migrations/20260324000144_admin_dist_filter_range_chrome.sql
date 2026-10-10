-- Distribution range filter field chrome (distinct from column Day label).
insert into public.site_messages (code, body) values
  ('ADMIN_DIST_FILTER_RANGE', 'Range'),
  ('ADMIN_DIST_FILTER_RANGE_DESC', 'Aggregate window for distribution metrics.')
on conflict (code) do nothing;
