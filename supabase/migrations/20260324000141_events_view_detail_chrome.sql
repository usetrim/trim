-- Trace (events) detail view chrome for dashboard table + preview dialog.
insert into public.site_messages (code, body) values
  (
    'EVENTS_COL_STATUS',
    'Status'
  ),
  (
    'EVENTS_COL_REQUEST_ID',
    'Request id'
  ),
  (
    'EVENTS_COL_VIEW',
    'View'
  ),
  (
    'EVENTS_OPEN_LABEL',
    'View'
  ),
  (
    'EVENTS_PREVIEW_FIELD_DESC',
    'Trace field from this run. Use search to find other traces.'
  )
on conflict (code) do update set body = excluded.body;
