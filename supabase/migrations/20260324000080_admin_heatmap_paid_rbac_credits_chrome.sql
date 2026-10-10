-- Paid-by-country heatmap chrome + role delete / credit grant labels.

insert into public.site_messages (code, body) values
  ('ADMIN_OBS_HEATMAP_COL_PAID', 'Paid seats'),
  ('ADMIN_OBS_HEATMAP_PAID_TITLE', 'Logins and paid by country'),
  ('ADMIN_RBAC_DELETE_ROLE', 'Delete role'),
  ('ADMIN_ROLE_IN_USE', 'Reassign or remove admins on this role before deleting it.'),
  ('ADMIN_CREDITS_GRANT_TITLE', 'Manual credit grant'),
  ('ADMIN_CREDITS_GRANT_USER', 'User id'),
  ('ADMIN_CREDITS_GRANT_AMOUNT', 'Credits'),
  ('ADMIN_CREDITS_GRANT_ACTION', 'Grant credits')
on conflict (code) do update set body = excluded.body;
