-- Clearer Refunds, Billing FAQ, and Support for Terms (MoR / Paddle-aligned).
-- Does not invent a legal entity street address or a seller fee percentage
-- (none is configured in Trim; customers pay catalog prices shown at checkout).

-- Strengthen billing overview with product-true upgrade/cancel facts.
update public.site_legal_sections
set body = 'Paid features require an active plan. Prices, included allowances, and plan descriptions are shown in the product at purchase time and may change prospectively for future periods. Payments, tax collection, and buyer checkout are handled by our payment partner acting as Merchant of Record (currently Paddle for hosted use-trim.com checkout). Subscriptions renew for the interval you select until you cancel in the product (for example Manage billing / cancel at period end) or in the payment partner customer portal. Upgrades on an active plan may credit unused time using the payment partner''s proration rules as previewed in the product before you confirm. Downgrades may be restricted while a paid period remains active. Taxes may be added at checkout by the payment partner where required. Failure to pay may result in suspension of paid features. Top-ups and credits, if offered, follow the rules shown at purchase and do not necessarily change your plan tier. The price you see and approve at checkout is the customer price for that purchase; any fees Trim pays the payment partner as seller are not a separate customer line item unless shown at checkout (for example tax).',
    updated_at = now()
where doc_kind = 'terms' and heading = 'Plans, billing, upgrades, and cancellations';

update public.site_legal_sections
set body = 'If a free trial or promotional period is offered, it ends when the stated period ends unless you cancel or convert as described in the product. Paid subscriptions renew automatically for the selected interval until canceled. Prices exclude taxes unless stated otherwise; our payment partner may collect applicable taxes and issue tax invoices under its Merchant of Record terms. Keep a valid payment method on file for renewals. Chargebacks or payment disputes filed in bad faith may result in suspension pending review.',
    updated_at = now()
where doc_kind = 'terms' and heading = 'Trials, renewals, and taxes';

-- New dedicated sections (idempotent by heading).
insert into public.site_legal_sections
  (doc_kind, sort_order, heading, body, contact_lead, contact_trail, uses_support_email, published_at)
select v.doc_kind, v.sort_order, v.heading, v.body, v.contact_lead, v.contact_trail, v.uses_support_email, now()
from (
  values
  (
    'terms'::text, 52, 'Refunds and payment disputes',
    'Unless required by applicable law or expressly stated otherwise at checkout, subscription fees and one-time purchases are non-refundable once the applicable billing period or purchase has started. That means canceling stops future renewals; it does not automatically refund time already paid for the current period. If you believe you were charged in error, contact us using the support email published with these Terms and include your account email, approximate charge date, and receipt or transaction reference from the product or payment partner. We will review in good faith with our payment partner. Refunds, if issued, are processed by the Merchant of Record to the original payment method under its timelines and buyer terms. Statutory consumer rights (including any mandatory cooling-off or withdrawal rights in your region) are not limited by this section where they cannot be waived. Do not open a chargeback before contacting support except where your bank or card network requires it; unresolved good-faith disputes may still be escalated through the payment partner.',
    '', '', false
  ),
  (
    'terms'::text, 56, 'Billing FAQ',
    'Where are prices? In the product pricing and checkout screens at purchase time. Who charges my card? Our payment partner as Merchant of Record (Paddle for hosted use-trim.com), not a separate Trim card processor. How do I cancel? Use Manage billing / cancel at period end in the product when available, or the payment partner customer portal; access typically continues until the end of the paid period. What happens on upgrade? The product may show a proration preview before you confirm; unused time on the current plan can be credited under the payment partner''s proration mode. Can I downgrade anytime? Self-serve downgrades may be blocked while a paid period is active; see the product message for your plan. What about annual vs monthly? Switching monthly to annual on the same plan is treated as an upgrade with proration where the product allows it. What about top-ups? They add capacity as described at purchase and usually do not change your subscription tier. Where are receipts? In the dashboard receipts area; tax invoices may also come from the payment partner. Questions? Use the Support section and support email on this page.',
    '', '', false
  ),
  (
    'terms'::text, 198, 'Support',
    'For account access, billing questions, refund or charge inquiries, security reports, and general hosted Service help, contact us using the support email published with these Terms. Include your account email and enough detail to reproduce the issue. We aim to respond within a commercially reasonable time for a hosted developer service; we do not publish a guaranteed response SLA for free or self-serve plans unless a separate enterprise agreement says otherwise. Payment-method updates, tax invoice copies, and some refunds may also be handled directly in the payment partner customer portal under Merchant of Record buyer terms. Privacy requests are described in the Privacy Policy.',
    'Support', '', true
  ),
  (
    'privacy'::text, 22, 'Support and how to reach us',
    'For privacy requests, security reports, and account questions related to the hosted Service, use the support email published with this policy. We do not publish a guaranteed support SLA for self-serve accounts. If we appoint a postal address, data protection officer, or EU/UK representative, we will add those details here or in a linked notice. Until then, email is the primary contact channel.',
    'Support', '', true
  )
) as v(doc_kind, sort_order, heading, body, contact_lead, contact_trail, uses_support_email)
where not exists (
  select 1 from public.site_legal_sections s
  where s.doc_kind = v.doc_kind and s.heading = v.heading
);

-- Point Terms Contact at Support section for clarity.
update public.site_legal_sections
set body = 'Questions about these Terms, billing, refunds, or the hosted Service can be sent to our support email. See also the Support and Refunds and payment disputes sections above.',
    updated_at = now()
where doc_kind = 'terms' and heading = 'Contact';
