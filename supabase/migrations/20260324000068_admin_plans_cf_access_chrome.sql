-- Plan editor chrome, credit packs, segment savings, CF Access checklist, retention purge.
insert into public.site_messages (code, body) values
  ('ADMIN_PLAN_DESCRIPTION', 'Description'),
  ('ADMIN_PLAN_RANK', 'Plan rank'),
  ('ADMIN_PLAN_SORT_ORDER', 'Sort order'),
  ('ADMIN_PLAN_KIND', 'Plan kind'),
  ('ADMIN_PLAN_KIND_SUBSCRIPTION', 'Subscription'),
  ('ADMIN_PLAN_KIND_TOPUP', 'Credit pack'),
  ('ADMIN_PLAN_KIND_ENTERPRISE', 'Enterprise'),
  ('ADMIN_SEGMENT_COL_SAVINGS', 'Tokens saved'),
  ('ADMIN_TOPUP_LEDGER_TITLE', 'Top-up ledger'),
  ('ADMIN_CHECKLIST_CF_ACCESS', 'Cloudflare Access JWT configured'),
  ('ADMIN_CF_ACCESS_REQUIRED', 'Cloudflare Access assertion is required.'),
  ('ADMIN_CF_ACCESS_INVALID', 'Cloudflare Access assertion is invalid.'),
  ('ADMIN_RETENTION_PURGE', 'Purge expired data'),
  ('ADMIN_RETENTION_PURGE_DONE', 'Retention purge completed.'),
  ('ADMIN_RETENTION_TTL_REQUIRED', 'Set positive retention TTL days before purge.')
on conflict (code) do nothing;
