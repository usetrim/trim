-- Usage tooltip + heatmap scope-specific copy + activity stats (web + admin).
-- Fail-closed: clients render only when these codes are present (no invent).

insert into public.site_messages (code, body) values
  ('DASHBOARD_USAGE_TOOLTIP_BREAKDOWN', 'Daily breakdown'),
  ('DASHBOARD_USAGE_TOOLTIP_DAILY_TOTAL', 'Daily total'),
  ('DASHBOARD_USAGE_TOOLTIP_CUMULATIVE_TOTAL', 'Cumulative total'),
  ('DASHBOARD_USAGE_TOOLTIP_SHARE_FMT', '{pct}%'),
  ('ADMIN_USAGE_TOOLTIP_BREAKDOWN', 'Daily breakdown'),
  ('ADMIN_USAGE_TOOLTIP_DAILY_TOTAL', 'Daily total'),
  ('ADMIN_USAGE_TOOLTIP_CUMULATIVE_TOTAL', 'Cumulative total'),
  ('ADMIN_USAGE_TOOLTIP_SHARE_FMT', '{pct}%'),

  ('DASHBOARD_HEATMAP_VALUE_FMT_ALL', '{date}' || chr(10) || '{count} lines edited'),
  ('DASHBOARD_HEATMAP_EMPTY_FMT_ALL', '{date}' || chr(10) || 'No lines edited'),
  ('DASHBOARD_HEATMAP_VALUE_FMT_TAB', '{date}' || chr(10) || '{count} tab accepts'),
  ('DASHBOARD_HEATMAP_EMPTY_FMT_TAB', '{date}' || chr(10) || 'No tab accepts'),
  ('ADMIN_HEATMAP_VALUE_FMT_ALL', '{date}' || chr(10) || '{count} lines edited'),
  ('ADMIN_HEATMAP_EMPTY_FMT_ALL', '{date}' || chr(10) || 'No lines edited'),
  ('ADMIN_HEATMAP_VALUE_FMT_TAB', '{date}' || chr(10) || '{count} tab accepts'),
  ('ADMIN_HEATMAP_EMPTY_FMT_TAB', '{date}' || chr(10) || 'No tab accepts'),

  ('DASHBOARD_HEATMAP_STAT_MOST_ACTIVE_MONTH', 'Most Active Month'),
  ('DASHBOARD_HEATMAP_STAT_MOST_ACTIVE_DAY', 'Most Active Day'),
  ('DASHBOARD_HEATMAP_STAT_LONGEST_STREAK', 'Longest Streak'),
  ('DASHBOARD_HEATMAP_STAT_CURRENT_STREAK', 'Current Streak'),
  ('DASHBOARD_HEATMAP_STREAK_FMT', '{count}d'),
  ('ADMIN_HEATMAP_STAT_MOST_ACTIVE_MONTH', 'Most Active Month'),
  ('ADMIN_HEATMAP_STAT_MOST_ACTIVE_DAY', 'Most Active Day'),
  ('ADMIN_HEATMAP_STAT_LONGEST_STREAK', 'Longest Streak'),
  ('ADMIN_HEATMAP_STAT_CURRENT_STREAK', 'Current Streak'),
  ('ADMIN_HEATMAP_STREAK_FMT', '{count}d')
on conflict (code) do update set body = excluded.body;

-- Keep legacy codes aligned with All-scope copy for any leftover readers.
update public.site_messages
set body = (select body from public.site_messages where code = 'DASHBOARD_HEATMAP_VALUE_FMT_ALL')
where code = 'DASHBOARD_HEATMAP_VALUE_FMT';

update public.site_messages
set body = (select body from public.site_messages where code = 'DASHBOARD_HEATMAP_EMPTY_FMT_ALL')
where code = 'DASHBOARD_HEATMAP_EMPTY_FMT';

update public.site_messages
set body = (select body from public.site_messages where code = 'ADMIN_HEATMAP_VALUE_FMT_ALL')
where code = 'ADMIN_HEATMAP_VALUE_FMT';

update public.site_messages
set body = (select body from public.site_messages where code = 'ADMIN_HEATMAP_EMPTY_FMT_ALL')
where code = 'ADMIN_HEATMAP_EMPTY_FMT';
