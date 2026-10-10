-- Privacy clarity for Unlimited metering (plan_catalog.unlimited).
-- Align Payments + Automated processing with Terms (migration 189).

update public.site_legal_sections
set body = 'Paid plans and top-ups are processed by our payment partner acting as Merchant of Record. That partner may collect billing name, address, tax information, and payment method details under its own privacy notice. Trim stores subscription status, entitlements (including whether Unlimited metering is enabled on your plan), credit balances, and receipt metadata needed for your account. Tax invoices may be issued by the payment partner under its terms. Clearing local client files does not reset hosted plan limits, Unlimited flags, or billing records.',
    updated_at = now()
where doc_kind = 'privacy' and heading = 'Payments and Merchant of Record';

update public.site_legal_sections
set body = 'We use automated systems to enforce usage limits, detect abuse, and protect the Service. When Unlimited metering is enabled on your plan by the operator, cloud requests are not debited while that flag remains on; when the flag is off, normal credit metering and payment-required exhaustion rules apply. These systems may affect access to free or paid features. They are not used to produce legal or similarly significant decisions about you solely by automated means without human review where such review is required by law. You may contact us to contest an access restriction tied to your account.',
    updated_at = now()
where doc_kind = 'privacy' and heading = 'Automated processing';
