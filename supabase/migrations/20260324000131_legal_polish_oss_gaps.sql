-- Professional polish: scrub remaining ops-sounding phrases, add OSS-world
-- gaps (definitions in Terms, contributions, GPC, breach notice), merge
-- overlapping children sections. No entity name/address invented.

-- Soften product/ops jargon still present in public legal bodies.
update public.site_legal_sections
set body = replace(body, 'free-tier abuse', 'unauthorized or abusive use'),
    updated_at = now()
where body like '%free-tier abuse%';

update public.site_legal_sections
set body = replace(body, 'metered usage related to the Service, plan and subscription status, credit balances, receipts, team membership, roles, seat counts, and preference settings needed to operate quotas and features', 'usage measurements related to the Service, plan and subscription status, credit balances, receipts, team membership, roles, seat counts, and preference settings needed to operate plan limits and features'),
    updated_at = now()
where body like '%metered usage related to the Service%';

update public.site_legal_sections
set body = replace(body, 'hosting, database, identity, email delivery, content delivery, observability, and payment processing', 'hosting, data storage, identity, email delivery, content delivery, reliability monitoring, and payment processing'),
    updated_at = now()
where body like '%observability%';

update public.site_legal_sections
set body = replace(body, 'usage and metering records for plan enforcement, support, and abuse prevention', 'usage records for plan enforcement, support, and abuse prevention'),
    updated_at = now()
where body like '%metering records%';

update public.site_legal_sections
set body = replace(body, 'plans, metering, and support', 'plans, usage, and support'),
    updated_at = now()
where body like '%plans, metering, and support%';

update public.site_legal_sections
set body = replace(body, 'help reduce AI coding context size, meter usage, manage plans, and administer teams', 'help reduce AI coding context size, track plan usage, manage plans, and administer teams'),
    updated_at = now()
where body like '%meter usage%';

update public.site_legal_sections
set body = replace(body, 'bypass metering or authentication', 'bypass usage limits or authentication'),
    updated_at = now()
where body like '%bypass metering%';

update public.site_legal_sections
set body = replace(body, 'including Base URL and upstream credentials', 'including client connection settings and upstream credentials'),
    updated_at = now()
where body like '%Base URL%';

update public.site_legal_sections
set body = replace(body, 'typically account id, email address, display name, and avatar', 'typically an account identifier, email address, display name, and profile image'),
    updated_at = now()
where body like '%account id%';

update public.site_legal_sections
set body = replace(body, 'enforce plans and quotas', 'enforce plans and usage limits'),
    updated_at = now()
where body like '%enforce plans and quotas%';

update public.site_legal_sections
set body = replace(body, 'enforce quotas, detect abuse', 'enforce usage limits, detect abuse'),
    updated_at = now()
where body like '%enforce quotas%';

update public.site_legal_sections
set body = replace(body, 'usage events for quotas and security', 'usage events for plan limits and security'),
    updated_at = now()
where body like '%usage events for quotas%';

update public.site_legal_sections
set body = replace(body, 'evade bans, quotas, or payment', 'evade bans, plan limits, or payment'),
    updated_at = now()
where body like '%evade bans, quotas%';

update public.site_legal_sections
set body = replace(body, 'Clearing local client files does not reset hosted quotas or billing records', 'Clearing local client files does not reset hosted plan limits or billing records'),
    updated_at = now()
where body like '%hosted quotas%';

update public.site_legal_sections
set body = replace(body, 'disable telemetry in the product', 'disable optional product analytics in the product'),
    updated_at = now()
where body like '%disable telemetry%';

update public.site_legal_sections
set body = replace(body, 'local Fast Mode compression', 'local compression features'),
    updated_at = now()
where body like '%Fast Mode compression%';

update public.site_legal_sections
set body = replace(body, 'non-payment, fraud, legal risk', 'non-payment, abuse, legal risk'),
    updated_at = now()
where body like '%non-payment, fraud%';

-- Merge overlapping children sections into one clear notice.
update public.site_legal_sections
set body = 'The Service is intended for professionals and organizations. It is not directed to children under 13, and we do not knowingly collect personal information from children under 13. Where a higher age of digital consent applies in your region (for example 16 under GDPR in some countries), that higher age controls. If you believe we have collected information from a child, contact us and we will delete it.',
    updated_at = now()
where doc_kind = 'privacy' and heading = 'Children';

delete from public.site_legal_sections
where doc_kind = 'privacy' and heading = 'Age and children (COPPA)';

-- Cookies: add GPC / opt-out signal language (best practice).
update public.site_legal_sections
set body = 'We use essential cookies or local storage for authentication, security (including session integrity protections where applicable), theme preference, and basic product function. We do not run third-party advertising cookies on use-trim.com. Your browser may offer Do Not Track or Global Privacy Control (GPC) signals; we treat optional product analytics as off when you disable optional product analytics in the product or through supported client settings, and we honor GPC as an opt-out of sale or sharing where those laws apply and we would otherwise engage in such activity. If we add optional analytics cookies in the future, we will update this policy and provide choices where required by law.',
    updated_at = now()
where doc_kind = 'privacy' and heading = 'Cookies and similar technologies';

-- Security: clearer breach notice without internals.
update public.site_legal_sections
set body = 'We use administrative, technical, and organizational safeguards appropriate to a hosted developer service, including encryption in transit, access-controlled systems, and authenticated API access. No method of transmission or storage is completely secure. Please use strong account practices with your identity provider and protect API keys. Report suspected vulnerabilities through our support contact. If we become aware of a personal data breach affecting you, we will notify you and regulators when required by law, without undue delay and within applicable legal timelines.',
    updated_at = now()
where doc_kind = 'privacy' and heading = 'Security';

-- New sections (idempotent by heading).
insert into public.site_legal_sections
  (doc_kind, sort_order, heading, body, contact_lead, contact_trail, uses_support_email, published_at)
select v.doc_kind, v.sort_order, v.heading, v.body, v.contact_lead, v.contact_trail, v.uses_support_email, now()
from (values
(
  'privacy'::text, 132, 'Security incidents',
  'We maintain processes to detect, investigate, and respond to security incidents that may involve personal information. Where a breach notification law applies, we will provide the information required by that law, which typically includes the nature of the incident, likely consequences, and measures taken or proposed. You can report suspected security issues using the support email published with this policy.',
  'Email', 'to report a security concern.', true
),
(
  'terms', 14, 'Definitions',
  'Service means the hosted websites, dashboard, API, authenticated CLI access, and related services at use-trim.com. Account means your registered hosted Service account. Organization means a company or other entity on whose behalf you use the Service. Content means materials you submit through your tools and workflows. Open-source components means Trim software distributed under public repository licenses. Payment partner means our Merchant of Record for paid checkout. Privacy Policy means the privacy notice published for the hosted Service.',
  '', '', false
),
(
  'terms', 108, 'Open-source contributions',
  'Contributions you submit to public Trim repositories are governed by each repository''s license and contribution guidelines. Unless a repository states otherwise or a separate contributor agreement applies, you license your contributions under the same terms as that repository (the common inbound-equals-outbound practice). Hosted Service accounts and paid features remain governed by these Terms even if you also contribute to open-source repositories.',
  '', '', false
),
(
  'terms', 145, 'Mutual responsibility for AI use',
  'You are responsible for how your organization uses model outputs, for configuring which upstream providers receive Content, and for complying with those providers'' terms. Trim is responsible for operating the hosted Service accounts, billing integration through our payment partner, and the features we publish for authenticated users. Neither party is responsible for the independent policies of third-party model providers you choose.',
  '', '', false
)
) as v(doc_kind, sort_order, heading, body, contact_lead, contact_trail, uses_support_email)
where not exists (
  select 1 from public.site_legal_sections s
  where s.doc_kind = v.doc_kind and s.heading = v.heading
);

update public.site_messages
set body = 'To protect accounts and prevent unauthorized use, we may store limited account-security signals such as IP address and session integrity data. These signals are used for security enforcement, not for advertising.',
    updated_at = now()
where code = 'LEGAL_PRIVACY_SEC_DEVICE_BODY';

update public.site_messages
set body = to_char(now() at time zone 'utc', 'YYYY-MM-DD'),
    updated_at = now()
where code = 'LEGAL_UPDATED_DATE';
