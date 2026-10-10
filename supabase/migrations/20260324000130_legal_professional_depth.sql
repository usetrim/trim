-- Professional depth pass for public Privacy/Terms.
-- Aligns with common OSS SaaS legal surfaces (definitions, controller vs
-- customer content, do-not-sell, appeals, retention criteria, license grant,
-- confidentiality, beta, DMCA path, subprocessor notice). Scrubs remaining
-- internal-sounding phrases. Does not invent a legal entity name or street address.

-- Scrub internal-sounding phrases still present in live bodies.
update public.site_legal_sections
set body = replace(
  body,
  'limited device binding signals, IP address, approximate location derived from IP, user agent, and related security telemetry used to protect accounts and free-tier abuse',
  'limited account-security signals (such as IP address, approximate location derived from IP, browser or client type, and session integrity data) used to protect accounts and free-tier abuse'
),
    updated_at = now()
where body like '%device binding signals%';

update public.site_legal_sections
set body = replace(
  body,
  'including CSRF protection where applicable',
  'including session integrity protections where applicable'
),
    updated_at = now()
where body like '%CSRF protection%';

update public.site_legal_sections
set body = replace(body, 'own control plane', 'own Trim deployment'),
    updated_at = now()
where body like '%control plane%';

update public.site_legal_sections
set body = replace(body, 'hosted control plane', 'hosted Service'),
    updated_at = now()
where body like '%control plane%';

-- Tighten categories / retention / subprocessors already present.
update public.site_legal_sections
set body = 'We keep information only as long as needed for the purposes described in this policy, then delete or de-identify it. Typical criteria: account and identity data while the account is active and for a short wind-down after deletion; team membership while you remain on a workspace; usage and metering records for plan enforcement, support, and abuse prevention (commonly up to 24 months unless a longer period is required); billing and tax records for the period required by tax and accounting law (often several years); support correspondence for the life of the request plus a reasonable archive period; and security logs for a limited rolling window. After you delete your account we remove personal account data from active systems, subject to records we must keep for tax, accounting, dispute, or legal retention. Backup copies may take a limited additional time to purge.',
    updated_at = now()
where doc_kind = 'privacy' and heading = 'Retention';

update public.site_legal_sections
set body = 'We use carefully selected service providers to operate the hosted Service. Typical categories for use-trim.com include: cloud hosting and content delivery; managed data storage and authentication infrastructure; payment processing by our Merchant of Record; email delivery for account and support messages; and error or uptime monitoring. Identity providers you choose (such as Google, GitHub, or GitLab) receive authentication data under their own terms when you sign in. Upstream AI model providers you select receive the prompts you send them. We require processors to use personal data only to provide services to us under contract. When we add or replace a core subprocessor that processes personal data for the hosted Service, we will update this policy or provide notice in the product with reasonable advance notice where required, and enterprise customers with a DPA may have additional objection rights under that agreement.',
    updated_at = now()
where doc_kind = 'privacy' and heading = 'Subprocessors and service providers';

update public.site_legal_sections
set body = 'If your organization needs a Data Processing Agreement (DPA) or similar processor terms for GDPR Article 28 style arrangements, contact us using the support email published with this policy. Self-hosted operators who run their own Trim deployment are responsible for their own processor contracts with their users. Until a signed DPA is in place, this Privacy Policy and the Terms describe the hosted Service practices.',
    updated_at = now()
where doc_kind = 'privacy' and heading = 'Enterprise data processing terms';

-- New privacy sections (idempotent by heading).
insert into public.site_legal_sections
  (doc_kind, sort_order, heading, body, contact_lead, contact_trail, uses_support_email, published_at)
select v.doc_kind, v.sort_order, v.heading, v.body, v.contact_lead, v.contact_trail, v.uses_support_email, now()
from (values
(
  'privacy'::text, 15, 'Definitions',
  'Personal information / personal data means information that identifies or can reasonably be linked to an individual. Processing means any operation on personal information (collection, storage, use, disclosure, deletion, and similar). Controller means the party that decides why and how personal information is processed. Processor means a party that processes personal information on behalf of a controller. Service Data means content and materials you or your organization submit through your tools and workflows (for example prompts, project context, or repository materials you choose to process). Account Data means information about your Trim account, authentication, plans, metering, and support interactions with us.',
  '', '', false
),
(
  'privacy', 35, 'Account Data and Service Data',
  'For Account Data about individuals using the hosted Service, Trim is typically the controller (or equivalent). For Service Data you submit through your tools, your organization is generally the controller of that content, and Trim processes it only as needed to provide the features you use on the hosted Service. Local compression on your device does not require uploading full prompt bodies to Trim solely to run that local path. Content sent to upstream AI providers is governed by those providers'' terms. If your organization needs processor terms for Service Data, request a DPA as described in Enterprise data processing terms.',
  '', '', false
),
(
  'privacy', 125, 'Retention criteria by category',
  'Retention follows purpose and legal need rather than a single global clock: identifiers and profile fields for active accounts; commercial and subscription records for billing and tax; usage events for quotas and security; and communication records for support quality. When you close an account we delete or de-identify Account Data from production systems on a commercially reasonable schedule, except where retention is required or permitted by law (for example disputed charges, security investigations, or statutory recordkeeping).',
  '', '', false
),
(
  'privacy', 172, 'Do not sell or share; advertising',
  'Trim does not sell personal information for money. Trim does not share personal information for cross-context behavioral advertising as those terms are commonly used under CCPA/CPRA. We do not use personal information from the hosted Service to build advertising profiles. If that ever changes, we will update this policy and provide required opt-out mechanisms (including a clear Do Not Sell or Share control where required). You may still contact us to state a preference.',
  'Email', 'to state a do-not-sell or do-not-share preference.', true
),
(
  'privacy', 166, 'Appeals and complaints',
  'If we deny a privacy request in whole or in part, you may ask us to reconsider by replying to our decision with additional information. Where your local law provides an appeal right (for example certain US state privacy laws), we will honor that process and timelines. You may also lodge a complaint with a data protection supervisory authority in your country or state of residence.',
  '', '', false
),
(
  'privacy', 112, 'EU, UK, and other representatives',
  'If we are required to appoint an EU or UK representative or a data protection officer, we will publish those contact details on this page or in a linked notice. Until such an appointment is published, use the support email in this policy for privacy matters. International transfers continue to rely on appropriate safeguards offered by our providers where required.',
  '', '', false
),
(
  'terms', 25, 'License and access',
  'Subject to these Terms and your plan, Trim grants you a limited, non-exclusive, non-transferable, non-sublicensable, revocable right to access and use the hosted Service for your internal purposes during your subscription or free-tier eligibility. You do not receive ownership of the Service, and except for open-source components under their licenses you may not copy, modify, distribute, rent, or create derivative works of proprietary hosted Service software. Access ends when your account is closed or suspended.',
  '', '', false
),
(
  'terms', 88, 'Confidentiality',
  'Each party may receive non-public information from the other that is marked confidential or would reasonably be understood as confidential (Confidential Information). The receiving party will use it only to perform under these Terms and will protect it with reasonable care. Confidential Information does not include information that is public through no fault of the receiver, independently developed, or rightfully received from a third party without duty. Trim may process Service Data and Account Data as described in the Privacy Policy. Compelled disclosures may be made when required by law, with notice to the other party when legally permitted.',
  '', '', false
),
(
  'terms', 87, 'Beta and experimental features',
  'We may offer beta, preview, or experimental features. Those features are provided as-is, may be unstable, may change or end without notice, may be subject to additional terms shown in the product, and are excluded from any availability commitment unless a separate written agreement says otherwise. Feedback you provide about beta features is covered by the feedback license in these Terms.',
  '', '', false
),
(
  'terms', 92, 'Copyright complaints (DMCA)',
  'If you believe material on the hosted Service infringes your copyright, send a notice to the support email published with these Terms that includes: your contact information; a description of the copyrighted work; the location of the allegedly infringing material; a statement that you have a good-faith belief the use is not authorized; a statement under penalty of perjury that the information is accurate and that you are the owner or authorized to act; and your physical or electronic signature. We may remove or disable access to material and, where appropriate, terminate repeat infringers.',
  'Email', 'for copyright notices.', true
),
(
  'terms', 175, 'Notices',
  'We may provide notices under these Terms by email to the address associated with your account, by in-product message, or by posting to the Service. You are responsible for keeping your account email current. Notices to Trim must be sent to the support email published with these Terms unless a separate contract specifies another address.',
  '', '', false
)
) as v(doc_kind, sort_order, heading, body, contact_lead, contact_trail, uses_support_email)
where not exists (
  select 1 from public.site_legal_sections s
  where s.doc_kind = v.doc_kind and s.heading = v.heading
);

-- Soften leftover LEGAL_* chrome snippets if any still sound internal.
update public.site_messages
set body = 'To protect accounts and free-tier abuse, we may store limited account-security signals such as IP address and session integrity data. These signals are used for security enforcement, not for advertising.',
    updated_at = now()
where code = 'LEGAL_PRIVACY_SEC_DEVICE_BODY'
  and body like '%device%';

update public.site_messages
set body = to_char(now() at time zone 'utc', 'YYYY-MM-DD'),
    updated_at = now()
where code = 'LEGAL_UPDATED_DATE';
