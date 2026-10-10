-- Scrub leftover LEGAL_* site_messages (seeded once, never overwritten by SeedAndRefresh).
-- These are not the public Privacy/Terms bodies (those live in site_legal_sections),
-- but MessageForCode can still surface LEGAL_SUPPORT_EMAIL_MISSING and related chrome.

update public.site_messages set body = 'Support email is not configured for this deployment.', updated_at = now()
where code = 'LEGAL_SUPPORT_EMAIL_MISSING';

update public.site_messages set body = 'This Privacy Policy describes how Trim collects, uses, and shares information when you use the hosted Service at use-trim.com.', updated_at = now()
where code = 'LEGAL_PRIVACY_INTRO';

update public.site_messages set body = 'Account and authentication', updated_at = now()
where code = 'LEGAL_PRIVACY_SEC_AUTH_HEADING';
update public.site_messages set body = 'Sign-in uses identity providers enabled for the Service. We receive account identifiers the provider shares with us, such as account id, email, display name, and avatar. We do not offer password or phone sign-up.', updated_at = now()
where code = 'LEGAL_PRIVACY_SEC_AUTH_BODY';

update public.site_messages set body = 'Usage, billing, and quotas', updated_at = now()
where code = 'LEGAL_PRIVACY_SEC_USAGE_HEADING';
update public.site_messages set body = 'We store metered usage, subscription status, receipts, and credit balances needed to operate quotas and billing. Our payment partner processes payments as Merchant of Record and may issue tax invoices under its terms. Clearing local client files does not reset hosted quotas.', updated_at = now()
where code = 'LEGAL_PRIVACY_SEC_USAGE_BODY';

update public.site_messages set body = 'Security and abuse prevention', updated_at = now()
where code = 'LEGAL_PRIVACY_SEC_DEVICE_HEADING';
update public.site_messages set body = 'To protect accounts and free-tier abuse, we may store limited device binding signals, IP address, and related security telemetry. These signals are used for security enforcement, not for advertising.', updated_at = now()
where code = 'LEGAL_PRIVACY_SEC_DEVICE_BODY';

update public.site_messages set body = 'Local processing', updated_at = now()
where code = 'LEGAL_PRIVACY_SEC_LOCAL_HEADING';
update public.site_messages set body = 'The Trim client can process prompts and project context on your device before requests are sent to the AI model provider you configure. Optional product analytics, when enabled, can be turned off in the product.', updated_at = now()
where code = 'LEGAL_PRIVACY_SEC_LOCAL_BODY';

update public.site_messages set body = 'Your rights', updated_at = now()
where code = 'LEGAL_PRIVACY_SEC_RIGHTS_HEADING';
update public.site_messages set body = 'Depending on where you live, you may request access, correction, deletion, or export of personal account data. The dashboard may provide self-serve tools. Billing records required by law may be retained as needed.', updated_at = now()
where code = 'LEGAL_PRIVACY_SEC_RIGHTS_BODY';

update public.site_messages set body = 'These Terms of Service govern access to and use of Trim''s hosted websites, dashboard, API, CLI authentication, and related services at use-trim.com. Open-source components remain subject to their repository licenses.', updated_at = now()
where code = 'LEGAL_TERMS_INTRO';

update public.site_messages set body = 'You must sign in with an allowed identity provider. You are responsible for activity under your account. We may suspend accounts that violate these Terms or present risk to the Service or other users.', updated_at = now()
where code = 'LEGAL_TERMS_SEC_ACCOUNTS_BODY';

update public.site_messages set body = 'Plan prices and allowances are shown in the product at purchase time. Payments are handled by our payment partner as Merchant of Record. Subscriptions renew until canceled. Exhausted quotas may block paid features until you upgrade or purchase additional capacity.', updated_at = now()
where code = 'LEGAL_TERMS_SEC_BILLING_BODY';

update public.site_messages set body = 'You may not misuse the Service, bypass metering or authentication, attack the Service without authorization, distribute malware, infringe others'' rights, or resell access except as expressly allowed by your plan. We may require clients to meet minimum version requirements for security reasons.', updated_at = now()
where code = 'LEGAL_TERMS_SEC_USE_BODY';

update public.site_messages set body = 'Context optimization changes payloads before they reach upstream models. The Service is provided as available without warranties beyond those required by law. We do not guarantee particular savings or model quality outcomes.', updated_at = now()
where code = 'LEGAL_TERMS_SEC_DISCLAIMER_BODY';

-- Gap fill: professional SaaS / OSS checklist items still thin in public legal sections.
insert into public.site_legal_sections
  (doc_kind, sort_order, heading, body, contact_lead, contact_trail, uses_support_email, published_at)
values
(
  'privacy', 45, 'Sensitive personal information',
  'We do not require sensitive personal information (as defined under CPRA and similar laws) to use the core Service. Please do not submit government ID numbers, precise geolocation, health data, or similar sensitive data in support tickets unless we expressly ask for it to verify an account request. If we process sensitive personal information only as needed to provide a requested service or as permitted by law, we will limit use and disclosure accordingly.',
  '', '', false, now()
),
(
  'privacy', 165, 'How to submit a privacy request',
  'You can use in-product account controls where available, or email the support address published with this policy. We may need to verify your identity (for example by confirming control of the account email) before fulfilling a request. We aim to respond within the time required by applicable law (commonly 30 to 45 days, with permitted extensions when the request is complex). Authorized agents may submit requests where the law allows, subject to verification. If we cannot fulfill a request, we will explain why where required.',
  'Email', 'to submit a privacy request.', true, now()
),
(
  'privacy', 175, 'Business transfers',
  'If Trim is involved in a merger, acquisition, financing, reorganization, bankruptcy, or sale of assets, personal information may be transferred as part of that transaction. We will require the recipient to honor privacy commitments consistent with this policy or provide notice and choices where required by law.',
  '', '', false, now()
),
(
  'privacy', 185, 'Links to other websites',
  'The Service may link to third-party sites or services (including identity providers, payment portals, and AI model providers). Their privacy practices are governed by their own policies. We are not responsible for those third-party practices.',
  '', '', false, now()
),
(
  'privacy', 95, 'Marketing communications',
  'We may send service-related messages (security, billing, product notices) that are not marketing. If we send optional product updates or marketing email, you can unsubscribe using the link in those messages or by contacting support. Essential service messages may continue while your account remains active.',
  '', '', false, now()
),
(
  'terms', 15, 'Privacy Policy',
  'Our Privacy Policy explains how we collect and process personal information for the hosted Service. By using the Service you also acknowledge that policy. If there is a conflict about personal data practices, the Privacy Policy controls for privacy disclosures; these Terms control for contractual rights and obligations.',
  '', '', false, now()
),
(
  'terms', 55, 'Trials, renewals, and taxes',
  'If a free trial or promotional period is offered, it ends when the stated period ends unless you cancel or convert as described in the product. Paid subscriptions renew automatically for the selected interval until canceled. Prices exclude taxes unless stated otherwise; our payment partner may collect applicable taxes. Chargebacks or payment disputes filed in bad faith may result in suspension.',
  '', '', false, now()
),
(
  'terms', 85, 'AI outputs and professional advice',
  'Trim optimizes context sent to AI systems you choose. Model outputs can be wrong, incomplete, or unsafe to rely on without review. The Service does not provide legal, medical, financial, or other professional advice. You are responsible for reviewing outputs and for compliance with your organization''s policies before using them in production systems.',
  '', '', false, now()
),
(
  'terms', 115, 'Availability and no SLA',
  'Unless you have a separate written enterprise agreement that expressly includes a service level commitment, the hosted Service is provided without a formal uptime SLA. We may perform maintenance, roll out changes, or experience outages. Credits or remedies for downtime apply only if a separate agreement says so.',
  '', '', false, now()
);

update public.site_messages
set body = to_char(now() at time zone 'utc', 'YYYY-MM-DD'),
    updated_at = now()
where code = 'LEGAL_UPDATED_DATE';
