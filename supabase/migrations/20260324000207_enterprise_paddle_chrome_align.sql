-- Align leftover free-activate chrome with Paddle-only Enterprise.
-- STATUS_ENTERPRISE_CONTRACT is unused after paid-subscription status path;
-- keep body honest if any older client still reads the code.

insert into public.site_messages (code, body) values
  ('STATUS_ENTERPRISE_CONTRACT', 'Enterprise'),
  ('ADMIN_ENTERPRISE_STATUS_DESC', 'Inquiry workflow status (new, contacted, offered, closed, or activated).'),
  ('NOTIF_USER_ENTERPRISE_ACTIVE_BODY', 'Your Enterprise plan is active (%s seats). Payment completed via Paddle.')
on conflict (code) do update set body = excluded.body, updated_at = now();
