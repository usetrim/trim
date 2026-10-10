-- Clearer Sync to Paddle operator copy for open-source Admin Plans modal.
-- Per-save flag (not plan_catalog); modal close + checkbox reset after save is expected.
-- See apps/admin/README.md “Admin CRUD UX”.

update public.site_messages
set body = 'Per-save action only - not stored on the plan. Check when pushing amounts/currency to Paddle; leave off for catalog-only edits such as Unlimited or Popular. After a successful save the modal closes and this checkbox resets; that is expected.'
where code = 'ADMIN_PLAN_SYNC_PADDLE_DESC';
