-- Creating / Updating pending chrome (Deleting already seeded). Date-range picker labels for admin filters.
insert into public.site_messages (code, body) values
  ('ADMIN_PENDING_CREATING', 'Creating...'),
  ('ADMIN_PENDING_UPDATING', 'Updating...'),
  ('ADMIN_DATE_RANGE_PLACEHOLDER', 'Filter by date'),
  ('ADMIN_DATE_RANGE_CLEAR', 'Clear dates'),
  ('ADMIN_DATE_RANGE_APPLY', 'Done')
on conflict (code) do nothing;
