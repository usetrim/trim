-- Post-Paddle overlay checkout success page + dashboard toast chrome.
-- Paddle.settings.successUrl lands on /billing/success; dashboard shows toast.

insert into public.site_messages (code, body) values
  ('CHECKOUT_SUCCESS_TITLE', 'Payment successful'),
  ('CHECKOUT_SUCCESS_BODY', 'Thanks for your purchase. We are activating your plan and emailing your order details.'),
  ('CHECKOUT_SUCCESS_DASHBOARD_LABEL', 'Go to dashboard'),
  ('CHECKOUT_SUCCESS_RECEIPTS_LABEL', 'View receipts'),
  ('CHECKOUT_SUCCESS_TOAST', 'Payment received. Your plan is updating.')
on conflict (code) do update set body = excluded.body;
