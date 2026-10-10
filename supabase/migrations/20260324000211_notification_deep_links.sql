-- Notification deep links: optional entity id + prefix/query href kinds.
-- UI already follows item.href when present; this wires exact receipt/inquiry targets.

alter table public.notification_kinds
  drop constraint if exists notification_kinds_href_ref_kind_check;

alter table public.notification_kinds
  add constraint notification_kinds_href_ref_kind_check
    check (href_ref_kind in ('none', 'app_path', 'app_path_prefix', 'admin_nav', 'admin_nav_query'));

alter table public.app_notifications
  add column if not exists href_entity_id text not null default '';

comment on column public.app_notifications.href_entity_id is
  'Optional entity id joined into href (receipt uuid, inquiry uuid, break-glass uuid). Empty = static kind href only.';

insert into public.site_messages (code, body) values
  ('APP_PATH_DASHBOARD_ENTERPRISE_INQUIRY_PREFIX', '/dashboard?enterprise_inquiry='),
  ('NOTIF_HREF_QUERY_ID', 'id')
on conflict (code) do update set body = excluded.body, updated_at = now();

update public.notification_kinds set
  href_ref_kind = 'app_path_prefix',
  href_ref = 'APP_PATH_RECEIPTS_PREFIX'
where code = 'user.receipt_ready';

update public.notification_kinds set
  href_ref_kind = 'app_path_prefix',
  href_ref = 'APP_PATH_DASHBOARD_ENTERPRISE_INQUIRY_PREFIX'
where code in ('user.enterprise_offer_ready', 'user.enterprise_activated');

update public.notification_kinds set
  href_ref_kind = 'admin_nav_query',
  href_ref = 'enterprise'
where code = 'admin.enterprise_inquiry';

update public.notification_kinds set
  href_ref_kind = 'admin_nav_query',
  href_ref = 'break_glass'
where code = 'admin.break_glass_request';

-- Backfill entity ids from known dedupe_key shapes (best-effort; new inserts write href_entity_id directly).
update public.app_notifications
set href_entity_id = split_part(dedupe_key, ':', 2)
where coalesce(href_entity_id, '') = ''
  and kind_code = 'user.enterprise_offer_ready'
  and dedupe_key like 'enterprise_offer:%';

update public.app_notifications
set href_entity_id = split_part(dedupe_key, ':', 2)
where coalesce(href_entity_id, '') = ''
  and kind_code = 'user.enterprise_activated'
  and dedupe_key like 'enterprise_activated:%';

update public.app_notifications
set href_entity_id = split_part(dedupe_key, ':', 2)
where coalesce(href_entity_id, '') = ''
  and kind_code = 'admin.enterprise_inquiry'
  and dedupe_key like 'enterprise:%';

update public.app_notifications
set href_entity_id = split_part(dedupe_key, ':', 2)
where coalesce(href_entity_id, '') = ''
  and kind_code = 'admin.break_glass_request'
  and dedupe_key like 'break_glass:%';

update public.app_notifications n
set href_entity_id = r.id::text
from public.billing_receipts r
where coalesce(n.href_entity_id, '') = ''
  and n.kind_code = 'user.receipt_ready'
  and n.dedupe_key = 'receipt:' || r.paddle_transaction_id;
