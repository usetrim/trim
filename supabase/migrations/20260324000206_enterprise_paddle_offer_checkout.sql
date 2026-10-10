-- Enterprise sales: Paddle-only fulfillment.
-- Admin offers seats (status=offered) → customer pays via Paddle checkout → webhook activates.
-- Admin "Activate" no longer grants entitlements without payment.

alter table public.enterprise_inquiries
  drop constraint if exists enterprise_inquiries_status_check;

alter table public.enterprise_inquiries
  add constraint enterprise_inquiries_status_check
  check (status in ('new', 'contacted', 'offered', 'closed', 'activated'));

alter table public.enterprise_inquiries
  add column if not exists paddle_subscription_id text,
  add column if not exists paddle_transaction_id text;

comment on column public.enterprise_inquiries.paddle_subscription_id is
  'Paddle subscription id written when webhook fulfills an offered inquiry.';
comment on column public.enterprise_inquiries.paddle_transaction_id is
  'Paddle transaction id written when webhook fulfills an offered inquiry.';

insert into public.notification_kinds (
  code, audience, title_message_code, body_message_code, href_ref_kind, href_ref
) values
  (
    'user.enterprise_offer_ready',
    'user',
    'NOTIF_USER_ENTERPRISE_OFFER_TITLE',
    'NOTIF_USER_ENTERPRISE_OFFER_BODY',
    'app_path',
    'APP_PATH_DASHBOARD'
  )
on conflict (code) do update set
  audience = excluded.audience,
  title_message_code = excluded.title_message_code,
  body_message_code = excluded.body_message_code,
  href_ref_kind = excluded.href_ref_kind,
  href_ref = excluded.href_ref;

insert into public.site_messages (code, body) values
  ('ADMIN_ENTERPRISE_STATUS_OFFERED', 'Offered'),
  ('ADMIN_ENTERPRISE_STATUS_DESC', 'Inquiry workflow status (new, contacted, offered, closed, or activated).'),
  ('ADMIN_ENTERPRISE_OFFER', 'Send Paddle offer'),
  ('ADMIN_ENTERPRISE_OFFER_PENDING', 'Sending offer…'),
  ('ADMIN_ENTERPRISE_OFFER_DESC', 'Marks the inquiry as offered and notifies the customer to complete Paddle checkout for the offered seat quantity. Entitlements apply only after payment succeeds.'),
  ('ADMIN_ENTERPRISE_OFFERED', 'Paddle offer sent. Customer must complete checkout before Enterprise entitlements apply.'),
  ('ADMIN_ENTERPRISE_OFFER_FAILED', 'Could not send the Paddle offer for this inquiry.'),
  ('ADMIN_ENTERPRISE_PRICE_MISSING', 'Enterprise plan has no active Paddle price. Sync Plans to Paddle before sending an offer.'),
  ('ADMIN_ENTERPRISE_ALREADY_ACTIVATED', 'This inquiry is already activated after payment.'),
  ('ADMIN_ENTERPRISE_ACTIVATE', 'Send Paddle offer'),
  ('ADMIN_ENTERPRISE_ACTIVATE_PENDING', 'Sending offer…'),
  ('ADMIN_ENTERPRISE_ACTIVATE_DESC', 'Marks the inquiry as offered and notifies the customer to pay via Paddle. Does not grant Enterprise access until checkout completes.'),
  ('ADMIN_ENTERPRISE_ACTIVATED', 'Paddle offer sent. Customer must complete checkout before Enterprise entitlements apply.'),
  ('NOTIF_USER_ENTERPRISE_OFFER_TITLE', 'Enterprise offer ready'),
  ('NOTIF_USER_ENTERPRISE_OFFER_BODY', 'Your Enterprise offer for %s seats is ready. Open the dashboard and complete Paddle checkout to activate.'),
  ('NOTIF_USER_ENTERPRISE_ACTIVE_BODY', 'Your Enterprise plan is active (%s seats). Payment completed via Paddle.'),
  ('ENTERPRISE_INQUIRY_CHECKOUT', 'Complete Enterprise checkout'),
  ('ENTERPRISE_INQUIRY_CHECKOUT_PENDING', 'Opening checkout…'),
  ('ENTERPRISE_OFFER_CHECKOUT', 'ENTERPRISE_OFFER_CHECKOUT'),
  ('ACTION:ENTERPRISE_OFFER_CHECKOUT', 'Pay with Paddle'),
  ('PENDING:ENTERPRISE_OFFER_CHECKOUT', 'Opening checkout…'),
  ('DECIDE_ENTERPRISE_OFFER_CHECKOUT', 'Complete Paddle checkout for your offered Enterprise seats.'),
  ('ENTERPRISE_INQUIRY_REQUIRED', 'A sales offer is required before Enterprise checkout.'),
  ('ENTERPRISE_INQUIRY_NOT_OFFERED', 'This inquiry is not ready for checkout. Wait for a sales offer.'),
  ('ENTERPRISE_INQUIRY_SEATS_MISSING', 'Offered seat quantity is missing on this inquiry.'),
  ('ENTERPRISE_ME_TITLE', 'Enterprise inquiries'),
  ('ENTERPRISE_ME_EMPTY', 'No enterprise inquiries yet. Contact sales from the Enterprise plan card.'),
  ('ENTERPRISE_ME_COL_COMPANY', 'Company'),
  ('ENTERPRISE_ME_COL_STATUS', 'Status'),
  ('ENTERPRISE_ME_COL_REQUESTED', 'Requested seats'),
  ('ENTERPRISE_ME_COL_OFFERED', 'Offered seats'),
  ('ENTERPRISE_ME_COL_CREATED', 'Created'),
  ('ENTERPRISE_ME_COL_UPDATED', 'Updated'),
  ('ENTERPRISE_ME_COL_MESSAGE', 'Message'),
  ('ENTERPRISE_ME_PAY', 'Pay with Paddle'),
  ('ENTERPRISE_ME_PAY_PENDING', 'Opening checkout…'),
  ('ENTERPRISE_ME_VIEW', 'View'),
  ('ENTERPRISE_ME_DETAILS', 'Inquiry details'),
  ('ENTERPRISE_ME_FILTER_STATUS', 'Status'),
  ('ENTERPRISE_ME_FILTER_STATUS_DESC', 'Filter inquiries by workflow status.'),
  ('ENTERPRISE_ME_STATUS_NEW', 'New'),
  ('ENTERPRISE_ME_STATUS_CONTACTED', 'Contacted'),
  ('ENTERPRISE_ME_STATUS_OFFERED', 'Offered - pay to activate'),
  ('ENTERPRISE_ME_STATUS_CLOSED', 'Closed'),
  ('ENTERPRISE_ME_STATUS_ACTIVATED', 'Activated'),
  ('ENTERPRISE_CHECKOUT_INTERVAL_REQUIRED', 'Billing interval is not configured for Enterprise checkout.')
on conflict (code) do update set body = excluded.body, updated_at = now();
