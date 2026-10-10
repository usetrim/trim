-- Public Privacy Policy and Terms of Service rewrite for use-trim.com.
-- Professional open-source SaaS coverage aligned with common public policies
-- (identity, categories, lawful bases, rights, MoR billing, OSS/self-host,
-- warranties, liability, export). No env var names, table names, internal
-- ops jargon, fraud fingerprint detail, or admin tooling.

delete from public.site_legal_sections
where doc_kind in ('privacy', 'terms');

insert into public.site_legal_sections
  (doc_kind, sort_order, heading, body, contact_lead, contact_trail, uses_support_email, published_at)
values
-- Privacy
(
  'privacy', 10, 'Introduction',
  'This Privacy Policy describes how Trim ("Trim", "we", "us", or "our") collects, uses, discloses, and otherwise processes personal information when you use use-trim.com, the Trim dashboard, the public API, the Trim CLI when connected to our hosted control plane, and related websites and communications (together, the "Service"). Trim helps developers reduce AI coding context size. By using the Service you acknowledge this policy. If you do not agree, do not use the Service. This policy applies to the hosted Service operated by Trim. Self-hosted operators who run their own deployment are independent controllers for data on their infrastructure unless they republish this policy for their users.',
  '', '', false, now()
),
(
  'privacy', 20, 'Who we are and how to contact us',
  'Trim operates the hosted Service at use-trim.com. For privacy questions, data subject requests, security reports, or complaints, contact us using the support email published with this policy. If we appoint a data protection officer or EU/UK representative, we will publish those details on this page or in product notices.',
  'Email', 'for privacy requests.', true, now()
),
(
  'privacy', 30, 'Scope and roles',
  'For personal information we collect about individual users of the hosted Service, Trim acts as a controller (or equivalent under applicable law). When you use Trim on behalf of an organization, that organization may also be a controller of account and workspace data it manages. Our payment partner acts as an independent Merchant of Record for paid checkout. Upstream AI model providers you select are independent controllers or processors of content you send them under their own terms.',
  '', '', false, now()
),
(
  'privacy', 40, 'Categories of information we collect',
  'Account and identity data: identifiers your identity provider shares with us when you sign in (typically account id, email address, display name, and avatar), and provider name. Usage and product data: metered usage related to the Service, plan and subscription status, credit balances, receipts, team membership, roles, seat counts, and preference settings needed to operate quotas and features. Device and security data: limited device binding signals, IP address, approximate location derived from IP, user agent, and related security telemetry used to protect accounts and free-tier abuse. Support and communications: contents of emails or tickets you send us, and service notices we send you. Website and session data: standard server logs and essential cookies or local storage needed for sessions, security, and preferences. We do not require password or phone sign-up on the hosted product path.',
  '', '', false, now()
),
(
  'privacy', 50, 'Information we do not need for local compression',
  'The Trim client can process prompts and project context on your device before requests are sent to the AI model provider you configure. Hosted Trim does not need full prompt bodies uploaded to Trim cloud solely to perform local Fast Mode compression. Optional product analytics, when enabled, are designed to be limited and can be turned off in the product. Content you send to upstream model providers is governed by those providers'' policies, not this policy alone.',
  '', '', false, now()
),
(
  'privacy', 60, 'Sources of information',
  'We collect information directly from you, from your identity provider when you authenticate, from your devices and clients when you use the Service, from our payment partner about subscription status and receipt metadata, from team admins who invite you, and from security and infrastructure providers that help us operate and protect the Service.',
  '', '', false, now()
),
(
  'privacy', 70, 'How we use information and lawful bases',
  'We use information to provide, maintain, and improve the Service; authenticate users; enforce plans and quotas; process payments; prevent fraud and abuse; provide support; communicate service-related notices; comply with law; and protect rights and safety. Where GDPR or UK GDPR applies, we rely on one or more lawful bases: performance of a contract (providing the Service you request), legitimate interests (securing the Service, improving reliability, preventing abuse, in ways that do not override your rights), consent where we ask for it, and legal obligation where we must retain or disclose records. We do not sell your personal information. We do not use your prompt contents from the local compression path for advertising.',
  '', '', false, now()
),
(
  'privacy', 80, 'Payments and Merchant of Record',
  'Paid plans and top-ups are processed by our payment partner acting as Merchant of Record. That partner may collect billing name, address, tax information, and payment method details under its own privacy notice. Trim stores subscription status, entitlements, and receipt metadata needed for your account. Tax invoices may be issued by the payment partner under its terms. Clearing local client files does not reset hosted quotas or billing records.',
  '', '', false, now()
),
(
  'privacy', 90, 'Cookies and similar technologies',
  'We use essential cookies or local storage for authentication, security (including CSRF protection where applicable), theme preference, and basic product function. We do not run third-party advertising cookies on use-trim.com. Your browser may offer Do Not Track controls; we treat optional product analytics as off when you disable telemetry in the product or through supported client settings. If we add optional analytics cookies in the future, we will update this policy and provide choices where required by law.',
  '', '', false, now()
),
(
  'privacy', 100, 'When we share information',
  'We share information with service providers that help us run the Service (for example hosting, database, identity, email delivery, content delivery, observability, and payment processing), with upstream AI providers you choose when you send them requests, with team admins when you join a workspace, with professional advisors under confidentiality, and when required by law, legal process, or to protect rights, safety, and the integrity of the Service. We require processors to use personal data only to provide services to us under contract. We may share aggregated or de-identified information that does not reasonably identify you.',
  '', '', false, now()
),
(
  'privacy', 110, 'International transfers',
  'We may process information in the United States and other countries where we or our providers operate. Where required, we use appropriate transfer safeguards offered by those providers (such as standard contractual clauses or equivalent mechanisms). By using the Service you understand that your information may be transferred to a country with different data-protection rules than your own.',
  '', '', false, now()
),
(
  'privacy', 120, 'Retention',
  'We keep account data while your account remains active and for a reasonable period afterward as needed to close the account, resolve disputes, and meet legal requirements. Usage and billing records are retained as needed for metering, support, audits, tax, and legal obligations. After you delete your account we remove personal account data from active systems, subject to records we must keep for tax, accounting, dispute, or legal retention requirements. Backup systems may take a limited additional time to purge.',
  '', '', false, now()
),
(
  'privacy', 130, 'Security',
  'We use administrative, technical, and organizational safeguards appropriate to a hosted developer service, including encryption in transit, access-controlled systems, and authenticated API access. No method of transmission or storage is completely secure. Please use strong account practices with your identity provider and protect API keys. Report suspected vulnerabilities through our support contact. We will notify affected users and regulators of personal data breaches when required by law.',
  '', '', false, now()
),
(
  'privacy', 140, 'Automated processing',
  'We use automated systems to enforce quotas, detect abuse, and protect the Service. These systems may affect access to free or paid features. They are not used to produce legal or similarly significant decisions about you solely by automated means without human review where such review is required by law. You may contact us to contest an access restriction tied to your account.',
  '', '', false, now()
),
(
  'privacy', 150, 'Children',
  'The Service is intended for professionals and organizations. It is not directed to children under 16 (or the higher age required in your jurisdiction), and we do not knowingly collect personal information from children. If you believe a child has provided us personal information, contact us and we will take appropriate steps to delete it.',
  '', '', false, now()
),
(
  'privacy', 160, 'Your privacy rights',
  'Depending on where you live, you may have rights to access, correct, delete, or export personal data, to object to or restrict certain processing, to withdraw consent where processing is consent-based, and to opt out of certain sharing under laws such as the GDPR, UK GDPR, or CCPA/CPRA and similar US state laws. The dashboard may provide self-serve deletion and identity disconnect tools. You may also contact us to exercise rights. We will not discriminate against you for exercising privacy rights. If we decline a request, we will explain why where required. You may have the right to lodge a complaint with a supervisory authority.',
  'Contact', 'to exercise privacy rights.', true, now()
),
(
  'privacy', 170, 'California and similar US state notices',
  'If you are a resident of California or another US state with consumer privacy laws, you may have rights to know, delete, correct, and opt out of sale or sharing of personal information, and to limit use of sensitive personal information where applicable. Trim does not sell personal information as that term is commonly defined for advertising. We may use limited data for security and service operation. Categories collected generally include identifiers, commercial information related to subscriptions, internet or electronic activity related to Service use, and inferences used only for security and product operation. To submit a request, use the dashboard tools or email support. We may need to verify your identity before fulfilling a request. Authorized agents may submit requests where the law allows, subject to verification.',
  '', '', false, now()
),
(
  'privacy', 180, 'Upstream AI providers',
  'After Trim prepares a request on your machine or through your configured path, content may be sent to the model provider you select. Those providers process content under their own terms and privacy policies. Trim does not control how upstream providers retain or use model inputs once received. Choose providers and retention settings that match your organization''s requirements.',
  '', '', false, now()
),
(
  'privacy', 190, 'Changes to this policy',
  'We may update this Privacy Policy from time to time. We will post the updated version on this page and revise the "Last updated" date. Material changes may also be communicated through the Service or email when appropriate. Continued use after the effective date means you accept the updated policy to the extent permitted by law.',
  '', '', false, now()
),
(
  'privacy', 200, 'Contact',
  'For privacy questions or requests related to the hosted Service, contact us at the support email below. Please include enough detail for us to verify and respond to your request.',
  'Support', '', true, now()
),
-- Terms
(
  'terms', 10, 'Agreement',
  'These Terms of Service ("Terms") govern your access to and use of Trim''s hosted websites, dashboard, API, CLI authentication when connected to our control plane, and related services at use-trim.com (the "Service"). By creating an account or using the Service you agree to these Terms and our Privacy Policy. If you use the Service on behalf of an organization, you represent that you have authority to bind that organization, and "you" includes that organization. If you do not agree, do not use the Service.',
  '', '', false, now()
),
(
  'terms', 20, 'The Service',
  'Trim provides developer tools and a control plane that help reduce AI coding context size, meter usage, manage plans, and administer teams. Features, interfaces, and integrations may change as we improve the product. We do not guarantee that any particular model provider, IDE, or third-party integration will remain available or unchanged. Beta or experimental features may be offered without the same availability or support expectations as generally available features and may be modified or withdrawn at any time.',
  '', '', false, now()
),
(
  'terms', 30, 'Eligibility and accounts',
  'You must be able to form a binding contract and must sign in with an allowed identity provider. You are responsible for activity under your account and for keeping access to your identity provider secure. You must provide accurate information and must not impersonate others or create accounts to evade bans, quotas, or payment. We may refuse, suspend, or terminate accounts that violate these Terms or present risk to the Service or other users. Organizations are responsible for their members'' compliance.',
  '', '', false, now()
),
(
  'terms', 40, 'Acceptable use',
  'You may not misuse the Service. Prohibited conduct includes attempting to bypass metering or authentication; attacking, probing, or overloading the Service without authorization; distributing malware; infringing others'' rights; using the Service for unlawful content or activity; scraping in a way that harms the Service; reverse engineering proprietary hosted components except as allowed by law; or reselling access except as expressly allowed by your plan. You must also comply with third-party LLM provider terms applicable to content you send upstream. We may investigate and take action including suspension or reporting to authorities when appropriate. Clients may need to meet minimum version requirements for security reasons.',
  '', '', false, now()
),
(
  'terms', 50, 'Plans, billing, upgrades, and cancellations',
  'Paid features require an active plan. Prices, included allowances, and plan descriptions are shown in the product at purchase time and may change prospectively. Payments are handled by our payment partner as Merchant of Record. Subscriptions renew according to the interval you select until canceled through the product or payment partner portal. Upgrades may be prorated according to the rules shown in the product. Downgrades may be restricted while a paid period remains active. Unless required by law or stated otherwise at checkout, fees are non-refundable once a billing period has started. Taxes may apply. Failure to pay may result in suspension of paid features. Top-ups and credits, if offered, follow the rules shown at purchase and do not necessarily change your plan tier.',
  '', '', false, now()
),
(
  'terms', 60, 'Teams and seats',
  'Team plans allocate seats as described in the product. Workspace owners and admins are responsible for invitations, roles, and seat counts. Invited members must use the invited email with an allowed sign-in method. Sharing credentials or seating in bad faith may result in suspension. You are responsible for content and activity in workspaces you control.',
  '', '', false, now()
),
(
  'terms', 70, 'API keys and client access',
  'API keys and CLI login credentials authenticate access to the Service. You must keep secrets confidential and revoke keys you believe are compromised. We may revoke keys that are leaked, abused, or used in ways that violate these Terms or threaten platform security. You are responsible for configuring clients safely, including Base URL and upstream credentials.',
  '', '', false, now()
),
(
  'terms', 80, 'Your content and feedback',
  'You retain rights to content you submit through your tools and workflows. You grant Trim a limited worldwide license to host, process, transmit, and display that content solely as needed to operate and secure the Service as you direct. You represent that you have the rights needed to submit that content. If you send feedback or suggestions, you grant Trim a royalty-free, perpetual, irrevocable right to use them without obligation to you.',
  '', '', false, now()
),
(
  'terms', 90, 'Intellectual property',
  'The Service, branding, documentation, and Trim-owned software are protected by intellectual property laws. These Terms do not transfer ownership of Trim IP to you. Open-source components are provided under their respective licenses in our public repositories. You may not copy, modify, or distribute proprietary hosted Service components except as allowed by law or a separate license. If you believe content on the Service infringes your copyright, contact us with a notice that includes your contact information, a description of the work, the location of the material, and a statement made under penalty of perjury that you are authorized to act.',
  '', '', false, now()
),
(
  'terms', 100, 'Open source and self-hosting',
  'Self-hosted deployments of Trim software are governed by the applicable repository licenses and any commercial agreement you enter for hosted use-trim.com. If you operate your own control plane, you are responsible for your users, compliance, security, support, and any legal notices you publish. Hosted Service terms apply to use-trim.com accounts even if you also run open-source components locally.',
  '', '', false, now()
),
(
  'terms', 110, 'Third-party services',
  'The Service integrates with identity providers, payment processors, hosting providers, and AI model providers you choose. Your use of those services is also subject to their terms and policies. Trim is not responsible for outages, data practices, billing mistakes of the Merchant of Record outside our reasonable control, or policy changes of third parties.',
  '', '', false, now()
),
(
  'terms', 120, 'Disclaimer of warranties',
  'THE SERVICE IS PROVIDED "AS IS" AND "AS AVAILABLE." TO THE MAXIMUM EXTENT PERMITTED BY LAW, TRIM DISCLAIMS ALL WARRANTIES, EXPRESS OR IMPLIED, INCLUDING MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, TITLE, AND NON-INFRINGEMENT. We do not warrant that the Service will be uninterrupted, error-free, secure, or free of harmful components, or that compression will achieve any particular savings or model quality outcome. No advice or information obtained from Trim creates any warranty not expressly stated in these Terms.',
  '', '', false, now()
),
(
  'terms', 130, 'Limitation of liability',
  'TO THE MAXIMUM EXTENT PERMITTED BY LAW, TRIM AND ITS SUPPLIERS WILL NOT BE LIABLE FOR INDIRECT, INCIDENTAL, SPECIAL, CONSEQUENTIAL, EXEMPLARY, OR PUNITIVE DAMAGES, OR ANY LOSS OF PROFITS, REVENUE, DATA, GOODWILL, OR BUSINESS INTERRUPTION, ARISING FROM YOUR USE OF THE SERVICE, WHETHER BASED IN CONTRACT, TORT, OR OTHERWISE, EVEN IF ADVISED OF THE POSSIBILITY. TRIM''S TOTAL LIABILITY FOR CLAIMS RELATING TO THE SERVICE WILL NOT EXCEED THE AMOUNTS YOU PAID TO TRIM FOR THE SERVICE IN THE THREE MONTHS BEFORE THE CLAIM AROSE (OR, IF GREATER AND REQUIRED BY LAW, THE MINIMUM LIABILITY THAT CANNOT BE LIMITED). Some jurisdictions do not allow certain limitations; in those cases our liability is limited to the fullest extent permitted.',
  '', '', false, now()
),
(
  'terms', 140, 'Indemnity',
  'You will defend and indemnify Trim and its affiliates, officers, and employees against claims, damages, losses, and expenses (including reasonable attorneys'' fees) arising from your misuse of the Service, your content, your violation of these Terms, or your violation of law or third-party rights.',
  '', '', false, now()
),
(
  'terms', 150, 'Suspension and termination',
  'You may stop using the Service at any time and may delete your account where the product provides that control. We may suspend or terminate access for violations of these Terms, non-payment, fraud, legal risk, or risk to the platform or other users. Provisions that by their nature should survive (including IP, disclaimers, liability limits, indemnity, and governing law) survive termination. Upon termination, your right to use the hosted Service ends; open-source licenses continue according to their terms.',
  '', '', false, now()
),
(
  'terms', 160, 'Export controls and sanctions',
  'You may not use the Service if you are prohibited under applicable export control or sanctions laws. You represent that you are not located in a comprehensively sanctioned jurisdiction and are not a denied or restricted party, and that you will not use the Service to violate export or sanctions rules.',
  '', '', false, now()
),
(
  'terms', 170, 'Governing law and disputes',
  'These Terms are governed by the laws applicable to the operator of use-trim.com, without regard to conflict-of-law rules, except where mandatory consumer protections of your country apply. Courts in that jurisdiction will have exclusive venue for disputes that are not resolved amicably, except that either party may seek injunctive relief in any competent court for IP or unauthorized access claims. Before filing a claim, you agree to try to resolve the dispute informally by contacting support.',
  '', '', false, now()
),
(
  'terms', 180, 'General',
  'If any provision of these Terms is unenforceable, the remaining provisions remain in effect. These Terms are the entire agreement between you and Trim regarding the Service and supersede prior agreements on that subject. Our failure to enforce a provision is not a waiver. You may not assign these Terms without our consent; we may assign them in connection with a merger, acquisition, or sale of assets. Notices may be provided by email, in-product message, or posting to the Service. Headings are for convenience only. Force majeure events beyond reasonable control excuse delayed performance for the duration of the event.',
  '', '', false, now()
),
(
  'terms', 190, 'Changes',
  'We may update these Terms by posting a revised version on this page and updating the "Last updated" date. Material changes may also be announced in the product or by email. Continued use after the effective date constitutes acceptance of the revised Terms to the extent permitted by law. If you do not agree, stop using the Service and delete your account.',
  '', '', false, now()
),
(
  'terms', 200, 'Contact',
  'Questions about these Terms for the hosted Service can be sent to our support email.',
  'Support', '', true, now()
);

update public.site_messages
set body = to_char(now() at time zone 'utc', 'YYYY-MM-DD'),
    updated_at = now()
where code = 'LEGAL_UPDATED_DATE';
