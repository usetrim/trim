-- Product settings sub-tabs and admin chrome cleanup.
insert into public.site_messages (code, body) values
  ('ADMIN_PRODUCT_TAB_DEFAULTS', 'Defaults'),
  ('ADMIN_PRODUCT_TAB_LIMITS', 'Limits'),
  ('ADMIN_PRODUCT_TAB_CLI', 'CLI'),
  ('ADMIN_PRODUCT_TAB_CHURN', 'Churn'),
  ('ADMIN_PRODUCT_TAB_FAST', 'Fast mode')
on conflict (code) do nothing;

update public.site_messages
set body = 'Recurring monthly price in major currency units (for example 9.99).', updated_at = now()
where code = 'ADMIN_PLAN_PRICE_MONTHLY_DESC';

update public.site_messages
set body = 'Recurring yearly price in major currency units (for example 99.99).', updated_at = now()
where code = 'ADMIN_PLAN_PRICE_YEARLY_DESC';

update public.site_messages
set body = 'MRR proxy', updated_at = now()
where code = 'ADMIN_KPI_MRR_CENTS';

update public.site_messages
set body = 'Search by email, id, or name shown in the list.', updated_at = now()
where code = 'ADMIN_FILTER_SEARCH_DESC';
