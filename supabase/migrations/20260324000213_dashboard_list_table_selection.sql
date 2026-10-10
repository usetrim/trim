-- DataTable selection chrome + bulk-delete confirms for dedicated Traces / Enterprise pages.
-- Receipts remain an immutable ledger (search / view / sync only - no customer delete).

insert into public.site_messages (code, body) values
  ('EVENTS_DELETE', 'Delete'),
  ('EVENTS_DELETE_PENDING', 'Deleting…'),
  ('EVENTS_DELETE_CONFIRM', 'Delete this trace permanently? This cannot be undone.'),
  ('EVENTS_BULK_DELETE_CONFIRM', 'Delete the selected traces permanently? This cannot be undone.'),
  ('EVENTS_DELETED', 'Trace deleted.'),
  ('EVENTS_BULK_DELETED', 'Selected traces deleted.'),
  ('EVENTS_DELETE_FAILED', 'Could not delete traces.'),
  ('EVENTS_IDS_REQUIRED', 'Select at least one trace.'),
  ('EVENTS_IDS_INVALID', 'One or more trace ids are invalid.'),
  ('ENTERPRISE_ME_SEARCH', 'Search inquiries'),
  ('ENTERPRISE_ME_SEARCH_DESC', 'Filters by company or message as you type.'),
  ('ENTERPRISE_ME_DELETE', 'Delete'),
  ('ENTERPRISE_ME_DELETE_PENDING', 'Deleting…'),
  ('ENTERPRISE_ME_DELETE_CONFIRM', 'Delete this inquiry permanently? Activated inquiries cannot be deleted.'),
  ('ENTERPRISE_ME_BULK_DELETE_CONFIRM', 'Delete the selected inquiries permanently? Activated inquiries are skipped. This cannot be undone.'),
  ('ENTERPRISE_ME_DELETED', 'Inquiry deleted.'),
  ('ENTERPRISE_ME_BULK_DELETED', 'Selected inquiries deleted.'),
  ('ENTERPRISE_ME_DELETE_FAILED', 'Could not delete inquiries.'),
  ('ENTERPRISE_ME_DELETE_ACTIVATED', 'Activated inquiries cannot be deleted.'),
  ('ENTERPRISE_ME_IDS_REQUIRED', 'Select at least one inquiry.'),
  ('ENTERPRISE_ME_IDS_INVALID', 'One or more inquiry ids are invalid.')
on conflict (code) do update set body = excluded.body, updated_at = now();
