-- Dashboard usage stacked chart + LOC contribution heatmap chrome (web + admin).

insert into public.site_messages (code, body) values
  ('DASHBOARD_USAGE_TITLE', 'Your Usage'),
  ('DASHBOARD_USAGE_SUBTITLE', 'Your usage per day across this billing period'),
  ('DASHBOARD_USAGE_GROUP_BY_PREFIX', 'Group By:'),
  ('DASHBOARD_USAGE_GROUP_MODEL', 'Model'),
  ('DASHBOARD_USAGE_GROUP_MODE', 'Mode'),
  ('DASHBOARD_USAGE_Y_AXIS', 'Cumulative Tokens'),
  ('DASHBOARD_USAGE_TODAY', 'Today'),
  ('DASHBOARD_HEATMAP_TITLE', 'AI Line Edits'),
  ('DASHBOARD_HEATMAP_SCOPE_ALL', 'All'),
  ('DASHBOARD_HEATMAP_SCOPE_TAB', 'Tab'),
  ('DASHBOARD_HEATMAP_EMPTY_FMT', '{date} - No lines edited'),
  ('DASHBOARD_HEATMAP_VALUE_FMT', '{date} - {count} lines edited'),
  ('ADMIN_USAGE_TITLE', 'Platform Usage'),
  ('ADMIN_USAGE_SUBTITLE', 'Platform usage per day across this window'),
  ('ADMIN_USAGE_GROUP_BY_PREFIX', 'Group By:'),
  ('ADMIN_USAGE_GROUP_MODEL', 'Model'),
  ('ADMIN_USAGE_GROUP_MODE', 'Mode'),
  ('ADMIN_USAGE_Y_AXIS', 'Cumulative Tokens'),
  ('ADMIN_USAGE_TODAY', 'Today'),
  ('ADMIN_HEATMAP_TITLE', 'AI Line Edits'),
  ('ADMIN_HEATMAP_SCOPE_ALL', 'All'),
  ('ADMIN_HEATMAP_SCOPE_TAB', 'Tab'),
  ('ADMIN_HEATMAP_EMPTY_FMT', '{date} - No lines edited'),
  ('ADMIN_HEATMAP_VALUE_FMT', '{date} - {count} lines edited')
on conflict (code) do update set body = excluded.body;
