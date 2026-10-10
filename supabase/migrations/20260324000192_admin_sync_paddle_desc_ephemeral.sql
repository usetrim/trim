-- Clarify Sync to Paddle is a per-save flag (not stored on plan_catalog).
-- See apps/admin/README.md “Admin CRUD UX” and root README Paddle §3.

update public.site_messages
set body = 'When enabled for this save, push product and price changes to Paddle. Not stored on the plan.'
where code = 'ADMIN_PLAN_SYNC_PADDLE_DESC';
