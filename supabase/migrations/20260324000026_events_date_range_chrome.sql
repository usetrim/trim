-- Events date-range filter chrome (dashboard Calendar / DateRangePicker).
insert into public.site_messages (code, body) values
  ('EVENTS_DATE_RANGE_PLACEHOLDER', 'Filter by date'),
  ('EVENTS_DATE_RANGE_CLEAR', 'Clear dates'),
  ('EVENTS_DATE_RANGE_APPLY', 'Done'),
  ('EVENTS_DATE_FROM_INVALID', 'from must be YYYY-MM-DD'),
  ('EVENTS_DATE_TO_INVALID', 'to must be YYYY-MM-DD'),
  ('EVENTS_DATE_RANGE_ORDER', 'from must be on or before to')
on conflict (code) do nothing;
