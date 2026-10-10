-- DataTable selection / row-menu / bulk action chrome (admin + shared web codes).

insert into public.site_messages (code, body) values
  ('ADMIN_TABLE_SELECT_ALL', 'Select all rows on this page'),
  ('ADMIN_TABLE_SELECT_ROW', 'Select row'),
  ('ADMIN_TABLE_SELECTED_FMT', '{count} selected'),
  ('ADMIN_TABLE_ROW_ACTIONS', 'Row actions'),
  ('ADMIN_TABLE_BULK_REMOVE', 'Remove selected'),
  ('ADMIN_TABLE_CLEAR_SELECTION', 'Clear selection'),
  ('ADMIN_ACTION_VIEW', 'View'),
  ('ADMIN_FILTER_ACTION', 'Action'),
  ('ADMIN_FILTER_ACTOR', 'Actor user id'),
  ('ADMIN_FILTER_RESOURCE', 'Resource type'),
  ('TABLE_SELECT_ALL', 'Select all rows on this page'),
  ('TABLE_SELECT_ROW', 'Select row'),
  ('TABLE_SELECTED_FMT', '{count} selected'),
  ('TABLE_ROW_ACTIONS', 'Row actions'),
  ('TABLE_BULK_REVOKE', 'Revoke selected'),
  ('TABLE_BULK_REMOVE', 'Remove selected'),
  ('TABLE_CLEAR_SELECTION', 'Clear selection')
on conflict (code) do update set body = excluded.body;
