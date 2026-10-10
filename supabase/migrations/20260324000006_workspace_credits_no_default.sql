-- Remove hardcoded default on workspace pool credits.
-- Credits must come from plan_catalog at insert / Paddle sync time.

alter table public.workspace_quotas
  alter column monthly_shared_credits drop default;

comment on column public.workspace_quotas.monthly_shared_credits is
  'Shared pool size. Set from plan_catalog.credits_monthly on workspace create and Paddle team sync. No silent default.';
