-- Keep usage chart chrome visible with an empty state when there is no series data.

insert into public.site_messages (code, body) values
  (
    'DASHBOARD_USAGE_EMPTY',
    'No usage recorded in this period yet. Charts fill in after Trim records proxy events from the CLI or cloud gateway.'
  ),
  (
    'ADMIN_USAGE_EMPTY',
    'No platform usage recorded in this window yet. Charts fill in after Trim records proxy events.'
  )
on conflict (code) do update set body = excluded.body;
