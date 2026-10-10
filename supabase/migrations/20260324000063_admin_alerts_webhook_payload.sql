-- Webhook inspector payload + admin alerts / segment filter / user section chrome.
alter table public.paddle_webhook_events
  add column if not exists payload jsonb,
  add column if not exists process_status text,
  add column if not exists last_error text;

comment on column public.paddle_webhook_events.payload is
  'Raw Paddle webhook JSON body for operator inspector. Stored only when event is claimed.';
comment on column public.paddle_webhook_events.process_status is
  'ok | failed. Empty on legacy rows. Failed rows may be cleared for replay.';
comment on column public.paddle_webhook_events.last_error is
  'Last processing error message when process_status = failed.';

insert into public.site_messages (code, body) values
  ('ADMIN_ALERTS_TITLE', 'Alerts'),
  ('ADMIN_ALERT_EVENT_ERRORS_24H', 'Event errors (24h)'),
  ('ADMIN_ALERT_QUOTA_EXHAUSTED', 'Quota exhausted accounts'),
  ('ADMIN_ALERT_JA4_LINKED', 'JA4 linked account clusters'),
  ('ADMIN_ALERT_SUSPENDED_7D', 'Suspended / banned (7d)'),
  ('ADMIN_ALERT_WEBHOOK_FAILED', 'Failed webhook events'),
  ('ADMIN_KPI_MRR_CENTS', 'MRR proxy (cents)'),
  ('ADMIN_KPI_CHURN_30D', 'Canceled subs (30d)'),
  ('ADMIN_KPI_NEW_PAID_30D', 'New paid subs (30d)'),
  ('ADMIN_HEALTH_PADDLE_WEBHOOKS', 'Paddle webhooks (last event age)'),
  ('ADMIN_HEALTH_ERROR_RATE', 'Event error rate (24h %)'),
  ('ADMIN_FILTER_PLAN', 'Plan'),
  ('ADMIN_FILTER_STATUS', 'Status'),
  ('ADMIN_FILTER_COUNTRY', 'Country'),
  ('ADMIN_FILTER_SEARCH', 'Search'),
  ('ADMIN_USER_SECTION_IDENTITIES', 'Identities'),
  ('ADMIN_USER_SECTION_WORKSPACES', 'Workspaces'),
  ('ADMIN_USER_SECTION_DEVICES', 'Devices'),
  ('ADMIN_USER_SECTION_JA4', 'JA4 fingerprints'),
  ('ADMIN_USER_SECTION_LINKED', 'Linked by JA4'),
  ('ADMIN_OBS_COL_STATUS', 'Status'),
  ('ADMIN_OBS_PAYLOAD_TITLE', 'Webhook payload'),
  ('ADMIN_RECEIPTS_SEARCH', 'Search receipts')
on conflict (code) do nothing;
