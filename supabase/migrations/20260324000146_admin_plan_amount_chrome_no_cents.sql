-- Present plan amounts as major currency units in chrome (API remains cents).
update public.site_messages
set body = 'Monthly amount', updated_at = now()
where code = 'ADMIN_PLAN_AMOUNT_MONTHLY';

update public.site_messages
set body = 'Yearly amount', updated_at = now()
where code = 'ADMIN_PLAN_AMOUNT_YEARLY';
