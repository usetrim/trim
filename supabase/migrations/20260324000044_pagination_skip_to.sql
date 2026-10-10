-- Migration 44: skip-to-page chrome for database skip/limit pagination.
-- First / last / jump labels live in site_messages only.

insert into public.site_messages (code, body) values
  ('PAGINATION_FIRST', 'First'),
  ('PAGINATION_LAST', 'Last'),
  ('PAGINATION_SKIP_TO_LABEL', 'Skip to page'),
  ('PAGINATION_SKIP_TO_INVALID', 'That page is outside the available range.'),
  ('ACTION:PAGINATION_SKIP_TO', 'Go'),
  ('PENDING:PAGINATION_SKIP_TO', 'Skipping...')
on conflict (code) do nothing;

comment on column public.site_messages.body is
  'Operator-owned chrome. PAGINATION_FIRST / LAST / SKIP_TO_* drive skip pagination.';
