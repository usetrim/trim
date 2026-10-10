-- Contribution-calendar weekday labels + popover fmt without em dash (web + admin).

insert into public.site_messages (code, body) values
  ('DASHBOARD_HEATMAP_WD_MON', 'M'),
  ('DASHBOARD_HEATMAP_WD_WED', 'W'),
  ('DASHBOARD_HEATMAP_WD_FRI', 'F'),
  ('DASHBOARD_HEATMAP_EMPTY_FMT', E'{date}\nNo lines edited'),
  ('DASHBOARD_HEATMAP_VALUE_FMT', E'{date}\n{count} lines edited'),
  ('ADMIN_HEATMAP_WD_MON', 'M'),
  ('ADMIN_HEATMAP_WD_WED', 'W'),
  ('ADMIN_HEATMAP_WD_FRI', 'F'),
  ('ADMIN_HEATMAP_EMPTY_FMT', E'{date}\nNo lines edited'),
  ('ADMIN_HEATMAP_VALUE_FMT', E'{date}\n{count} lines edited')
on conflict (code) do update set body = excluded.body;
