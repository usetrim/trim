-- In-app notifications (web users + platform admins). Chrome and kind copy live in site_messages.
-- No invent: empty chrome codes fail closed in API/UI.

create table if not exists public.notification_kinds (
  code text primary key,
  audience text not null check (audience in ('user', 'admin')),
  title_message_code text not null,
  body_message_code text not null,
  -- app_path: site_messages APP_PATH_* code; admin_nav: admin_nav_items.id; none: no href
  href_ref_kind text not null default 'none'
    check (href_ref_kind in ('none', 'app_path', 'admin_nav')),
  href_ref text not null default '',
  created_at timestamptz not null default now()
);

comment on table public.notification_kinds is
  'Catalog of in-app notification kinds. Titles/bodies/paths resolve from site_messages or admin_nav_items.';

create table if not exists public.app_notifications (
  id uuid primary key default gen_random_uuid(),
  recipient_id uuid not null references public.profiles(id) on delete cascade,
  audience text not null check (audience in ('user', 'admin')),
  kind_code text not null references public.notification_kinds(code),
  -- Ordered string args for fmt.Sprintf against body_message_code (empty = body as-is)
  body_args jsonb not null default '[]'::jsonb,
  read_at timestamptz,
  created_at timestamptz not null default now(),
  -- Optional idempotency key scoped to recipient
  dedupe_key text
);

comment on table public.app_notifications is
  'Per-recipient in-app notifications for web (audience=user) and admin console (audience=admin).';

create unique index if not exists app_notifications_recipient_dedupe_uidx
  on public.app_notifications (recipient_id, dedupe_key)
  where dedupe_key is not null and dedupe_key <> '';

create index if not exists app_notifications_recipient_created_idx
  on public.app_notifications (recipient_id, created_at desc);

create index if not exists app_notifications_recipient_unread_idx
  on public.app_notifications (recipient_id)
  where read_at is null;

alter table public.app_notifications enable row level security;

-- API uses service role / pool; no direct anon policies.

insert into public.notification_kinds (
  code, audience, title_message_code, body_message_code, href_ref_kind, href_ref
) values
  ('user.workspace_invite', 'user', 'NOTIF_USER_WS_INVITE_TITLE', 'NOTIF_USER_WS_INVITE_BODY', 'app_path', 'APP_PATH_TEAM'),
  ('user.workspace_joined', 'user', 'NOTIF_USER_WS_JOINED_TITLE', 'NOTIF_USER_WS_JOINED_BODY', 'app_path', 'APP_PATH_TEAM'),
  ('user.subscription_active', 'user', 'NOTIF_USER_SUB_ACTIVE_TITLE', 'NOTIF_USER_SUB_ACTIVE_BODY', 'app_path', 'APP_PATH_DASHBOARD'),
  ('user.subscription_canceled', 'user', 'NOTIF_USER_SUB_CANCELED_TITLE', 'NOTIF_USER_SUB_CANCELED_BODY', 'app_path', 'APP_PATH_DASHBOARD'),
  ('user.receipt_ready', 'user', 'NOTIF_USER_RECEIPT_TITLE', 'NOTIF_USER_RECEIPT_BODY', 'app_path', 'APP_PATH_DASHBOARD'),
  ('user.account_status', 'user', 'NOTIF_USER_ACCOUNT_STATUS_TITLE', 'NOTIF_USER_ACCOUNT_STATUS_BODY', 'app_path', 'APP_PATH_DASHBOARD'),
  ('admin.enterprise_inquiry', 'admin', 'NOTIF_ADMIN_ENTERPRISE_TITLE', 'NOTIF_ADMIN_ENTERPRISE_BODY', 'admin_nav', 'enterprise'),
  ('admin.break_glass_request', 'admin', 'NOTIF_ADMIN_BREAK_GLASS_TITLE', 'NOTIF_ADMIN_BREAK_GLASS_BODY', 'admin_nav', 'break_glass')
on conflict (code) do nothing;

insert into public.site_messages (code, body) values
  ('NOTIF_BELL_ARIA', 'Notifications'),
  ('NOTIF_PANEL_TITLE', 'Notifications'),
  ('NOTIF_EMPTY', 'No notifications yet.'),
  ('NOTIF_MARK_ALL_READ', 'Mark all read'),
  ('NOTIF_MARK_ALL_PENDING', 'Marking…'),
  ('NOTIF_MARK_READ', 'Mark read'),
  ('NOTIF_UNREAD_LABEL', 'Unread'),
  ('NOTIF_POLL_INTERVAL_MS', '60000'),
  ('NOTIF_USER_WS_INVITE_TITLE', 'Workspace invite'),
  ('NOTIF_USER_WS_INVITE_BODY', 'You were invited to join %s as %s.'),
  ('NOTIF_USER_WS_JOINED_TITLE', 'New team member'),
  ('NOTIF_USER_WS_JOINED_BODY', '%s joined %s as %s.'),
  ('NOTIF_USER_SUB_ACTIVE_TITLE', 'Subscription updated'),
  ('NOTIF_USER_SUB_ACTIVE_BODY', 'Your %s plan is now %s.'),
  ('NOTIF_USER_SUB_CANCELED_TITLE', 'Subscription canceled'),
  ('NOTIF_USER_SUB_CANCELED_BODY', 'Your subscription was canceled.'),
  ('NOTIF_USER_RECEIPT_TITLE', 'Receipt ready'),
  ('NOTIF_USER_RECEIPT_BODY', 'A new receipt is available on your dashboard.'),
  ('NOTIF_USER_ACCOUNT_STATUS_TITLE', 'Account status'),
  ('NOTIF_USER_ACCOUNT_STATUS_BODY', 'Your account status is now %s.'),
  ('NOTIF_ADMIN_ENTERPRISE_TITLE', 'Enterprise inquiry'),
  ('NOTIF_ADMIN_ENTERPRISE_BODY', 'New inquiry from %s (%s).'),
  ('NOTIF_ADMIN_BREAK_GLASS_TITLE', 'Break-glass request'),
  ('NOTIF_ADMIN_BREAK_GLASS_BODY', '%s requested elevation: %s.'),
  ('NOTIF_LIST_FAILED', 'Could not load notifications.'),
  ('NOTIF_MARK_FAILED', 'Could not update notification.'),
  ('NOTIF_COUNT_FAILED', 'Could not load unread count.'),
  ('NOTIF_LIST_INVALID', 'Invalid notification request.'),
  ('NOTIF_INSERT_INVALID', 'Invalid notification payload.'),
  ('NOTIF_INSERT_FAILED', 'Could not create notification.'),
  ('NOTIF_ADMINS_LOAD_FAILED', 'Could not load admin recipients.')
on conflict (code) do nothing;
