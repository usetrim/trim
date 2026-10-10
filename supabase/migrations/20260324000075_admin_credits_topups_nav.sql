-- Global credits / top-ups ledger nav + column chrome (DB-driven; fail-closed if empty).

insert into public.site_messages (code, body) values
  ('ADMIN_NAV_CREDITS', 'Credits and top-ups'),
  ('ADMIN_CREDITS_TAB_GRANTS', 'Manual grants'),
  ('ADMIN_CREDITS_TAB_TOPUPS', 'Top-up ledger'),
  ('ADMIN_CREDITS_COL_USER', 'User id'),
  ('ADMIN_CREDITS_COL_CREDITS', 'Credits'),
  ('ADMIN_CREDITS_COL_REASON', 'Reason'),
  ('ADMIN_CREDITS_COL_BY', 'Granted by'),
  ('ADMIN_CREDITS_COL_WHEN', 'When'),
  ('ADMIN_TOPUP_COL_USER', 'User id'),
  ('ADMIN_TOPUP_COL_TX', 'Paddle transaction'),
  ('ADMIN_TOPUP_COL_PRICE', 'Price id'),
  ('ADMIN_TOPUP_COL_PLAN', 'Plan hint'),
  ('ADMIN_TOPUP_COL_CREDITS', 'Credits'),
  ('ADMIN_TOPUP_COL_QTY', 'Qty'),
  ('ADMIN_TOPUP_COL_WHEN', 'When'),
  ('ADMIN_CREDITS_FILTER_USER', 'Filter by user id')
on conflict (code) do update set body = excluded.body;

insert into public.admin_nav_items (id, href, chrome_code, permission_code, sort_order, is_active)
values
  ('credits', '/billing/credits', 'ADMIN_NAV_CREDITS', 'billing.credits', 85, true)
on conflict (id) do update set
  href = excluded.href,
  chrome_code = excluded.chrome_code,
  permission_code = excluded.permission_code,
  sort_order = excluded.sort_order,
  is_active = excluded.is_active;
