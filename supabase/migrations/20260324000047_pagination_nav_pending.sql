-- Migration 47: First / Prev / Next / Last pagination button pending labels.
-- Matches SkipPagination isLoading + pendingLabel (same pattern as skip-to Go).

insert into public.site_messages (code, body) values
  ('PENDING:PAGINATION_FIRST', 'Loading first...'),
  ('PENDING:PAGINATION_PREV', 'Loading previous...'),
  ('PENDING:PAGINATION_NEXT', 'Loading next...'),
  ('PENDING:PAGINATION_LAST', 'Loading last...')
on conflict (code) do nothing;

comment on column public.site_messages.body is
  'Operator-owned chrome. PENDING:PAGINATION_* drive nav button loading copy.';
