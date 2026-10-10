-- Dispute writes require step-up; align catalog with dangerous billing ops.

update public.platform_permission_catalog
set step_up_required = true
where code = 'billing.receipts'
  and step_up_required = false;
