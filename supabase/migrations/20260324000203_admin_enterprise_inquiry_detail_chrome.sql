-- Enterprise inquiry list/detail chrome: surface customer-submitted fields.
-- NOTE: site_messages column is body (not message).

insert into public.site_messages (code, body) values
  ('ADMIN_ENTERPRISE_COL_SEATS', 'Requested seats'),
  ('ADMIN_ENTERPRISE_COL_CREATED', 'Created'),
  ('ADMIN_ENTERPRISE_COL_MESSAGE', 'Message'),
  ('ADMIN_ENTERPRISE_DETAILS', 'Inquiry details'),
  ('ADMIN_ENTERPRISE_INQUIRY_MESSAGE', 'Customer message'),
  ('ADMIN_ENTERPRISE_INQUIRY_MESSAGE_DESC', 'Message the customer submitted with this enterprise inquiry.'),
  ('ADMIN_ENTERPRISE_ESTIMATED_SEATS', 'Requested seats'),
  ('ADMIN_ENTERPRISE_ESTIMATED_SEATS_DESC', 'Seat estimate the customer entered when submitting the inquiry (estimated_seats).'),
  ('ADMIN_ENTERPRISE_USER', 'User'),
  ('ADMIN_ENTERPRISE_USER_DESC', 'Signed-in account that submitted the inquiry.'),
  ('ADMIN_ENTERPRISE_CREATED', 'Created'),
  ('ADMIN_ENTERPRISE_UPDATED', 'Updated'),
  ('ADMIN_ENTERPRISE_CONTRACT_NOTES_DESC', 'Internal contract or sales notes stored on the enterprise inquiry (contract_notes).'),
  ('ADMIN_ENTERPRISE_OFFERED_SEATS_DESC', 'Seat quantity offered in the proposal (offered_seat_quantity).')
on conflict (code) do update set body = excluded.body, updated_at = now();

update public.site_messages
set body = 'Inquiry workflow status (new, contacted, closed, or activated).',
    updated_at = now()
where code = 'ADMIN_ENTERPRISE_STATUS_DESC';
