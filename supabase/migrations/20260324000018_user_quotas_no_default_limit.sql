-- Free credits must come from plan_catalog / signup triggers only.
-- Drop the historical column default so a bare INSERT cannot invent 50 credits.
alter table public.user_quotas
  alter column monthly_credit_limit drop default;
