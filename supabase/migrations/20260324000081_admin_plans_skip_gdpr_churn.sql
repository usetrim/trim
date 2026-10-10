-- Plan catalog skip pagination support (FE uses default_page_size).
-- GDPR export filename from site_messages (no client invent).
-- Churn risk thresholds from admin_product_settings (no invent 0.9/14/30/9999).

alter table public.admin_product_settings
  add column if not exists churn_high_usage_ratio numeric,
  add column if not exists churn_medium_usage_ratio numeric,
  add column if not exists churn_high_idle_days int,
  add column if not exists churn_low_idle_days int;

comment on column public.admin_product_settings.churn_high_usage_ratio is
  'Usage ratio (used/limit) for high churn when also idle. Fail closed when null.';
comment on column public.admin_product_settings.churn_medium_usage_ratio is
  'Usage ratio for medium churn. Fail closed when null.';
comment on column public.admin_product_settings.churn_high_idle_days is
  'Idle days for high churn with high usage. Fail closed when null or <= 0.';
comment on column public.admin_product_settings.churn_low_idle_days is
  'Idle days for low churn alone. Fail closed when null or <= 0.';

update public.admin_product_settings
set
  churn_high_usage_ratio = coalesce(churn_high_usage_ratio, 0.9),
  churn_medium_usage_ratio = coalesce(churn_medium_usage_ratio, 0.7),
  churn_high_idle_days = coalesce(churn_high_idle_days, 14),
  churn_low_idle_days = coalesce(churn_low_idle_days, 30)
where id = 'default';

insert into public.site_messages (code, body) values
  ('ADMIN_GDPR_EXPORT_FILENAME_FMT', 'gdpr-export-{user_id}-{generated_at}.json'),
  ('ADMIN_CHURN_THRESHOLDS_MISSING', 'Set churn_* columns on admin_product_settings before segment churn risk.'),
  ('ADMIN_PRODUCT_CHURN_HIGH_USAGE', 'Churn high usage ratio'),
  ('ADMIN_PRODUCT_CHURN_MED_USAGE', 'Churn medium usage ratio'),
  ('ADMIN_PRODUCT_CHURN_HIGH_IDLE', 'Churn high idle days'),
  ('ADMIN_PRODUCT_CHURN_LOW_IDLE', 'Churn low idle days'),
  ('ADMIN_CHECKLIST_MIGRATIONS_CHURN', 'Churn thresholds / GDPR filename migration applied')
on conflict (code) do update set body = excluded.body;
