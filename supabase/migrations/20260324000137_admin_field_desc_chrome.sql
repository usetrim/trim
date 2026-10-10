-- Admin field descriptions / hints and dialog close chrome.
insert into public.site_messages (code, body) values
  (
    'ADMIN_USERS_SEARCH_DESC',
    'Match email, user id, or auth provider.'
  ),
  (
    'ADMIN_FILTER_SEARCH_DESC',
    'Narrow the list by a free-text search.'
  ),
  (
    'ADMIN_FILTER_ACTION_DESC',
    'Filter audit rows by action name.'
  ),
  (
    'ADMIN_FILTER_ACTOR_DESC',
    'Filter by the actor user id.'
  ),
  (
    'ADMIN_FILTER_RESOURCE_DESC',
    'Filter by resource type.'
  ),
  (
    'ADMIN_FILTER_STATUS_DESC',
    'Filter by account or record status.'
  ),
  (
    'ADMIN_FILTER_PLAN_DESC',
    'Filter by plan id or display name.'
  ),
  (
    'ADMIN_FILTER_COUNTRY_DESC',
    'Filter by ISO country code.'
  ),
  (
    'ADMIN_FILTER_PROVIDER_DESC',
    'Filter by auth provider id.'
  ),
  (
    'ADMIN_FILTER_INTERVAL_DESC',
    'Filter by billing interval.'
  ),
  (
    'ADMIN_FILTER_CREATED_FROM_DESC',
    'Accounts created on or after this date.'
  ),
  (
    'ADMIN_FILTER_CREATED_TO_DESC',
    'Accounts created on or before this date.'
  ),
  (
    'ADMIN_FILTER_CREDITS_LEFT_DESC',
    'Accounts with at most this many credits left.'
  ),
  (
    'ADMIN_CREDITS_FILTER_USER_DESC',
    'Show rows for one user id.'
  ),
  (
    'ADMIN_SETTINGS_FIELD_DESC',
    'Changes save after you confirm with step-up.'
  ),
  (
    'ADMIN_NAV_MENU',
    'Menu'
  ),
  (
    'ADMIN_REASON_FIELD_DESC',
    'Required for audited admin actions.'
  ),
  (
    'ADMIN_DIALOG_CLOSE',
    'Close'
  ),
  (
    'ADMIN_USER_NOTES',
    'Admin notes'
  ),
  (
    'ADMIN_USER_NOTES_DESC',
    'Internal notes visible only to platform admins.'
  ),
  (
    'ADMIN_USER_QUOTA_LIMIT',
    'Monthly credit limit'
  ),
  (
    'ADMIN_USER_QUOTA_TOPUP',
    'Purchased top-up credits'
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
