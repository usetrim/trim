-- Lock offered Enterprise inquiries: no silent seat edits / re-offers.
-- Sales must move Offered → Contacted to revise, or → Closed to withdraw.

insert into public.site_messages (code, body) values
  (
    'ADMIN_ENTERPRISE_ALREADY_OFFERED',
    'This inquiry already has an active Paddle offer. Move status to Contacted to revise seats, or Closed to withdraw.'
  ),
  (
    'ADMIN_ENTERPRISE_INQUIRY_CLOSED',
    'Closed inquiries cannot receive a Paddle offer. Set status to Contacted first.'
  ),
  (
    'ADMIN_ENTERPRISE_OFFER_LOCKED_DESC',
    'Offer terms are locked while status is Offered. Move to Contacted to revise seats and re-send, or Closed to withdraw. Entitlements still apply only after the customer pays.'
  )
on conflict (code) do update set body = excluded.body, updated_at = now();
