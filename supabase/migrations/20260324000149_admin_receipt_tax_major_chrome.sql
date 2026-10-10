-- Receipt tax column label: major-unit display in admin (API remains cents).
update public.site_messages
set body = 'Tax', updated_at = now()
where code = 'ADMIN_RECEIPT_TAX_CENTS';
