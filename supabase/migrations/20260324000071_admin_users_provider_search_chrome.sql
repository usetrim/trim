-- User directory provider search chrome.
insert into public.site_messages (code, body) values
  ('ADMIN_FILTER_PROVIDER', 'Provider'),
  ('ADMIN_USERS_COL_PROVIDER', 'Provider'),
  ('ADMIN_USERS_SEARCH', 'Search email, id, or provider')
on conflict (code) do nothing;
