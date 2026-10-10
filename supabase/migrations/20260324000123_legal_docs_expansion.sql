-- Expand Privacy Policy and Terms of Service for public site_legal_sections.
-- No invented operator copy at runtime: rows are published and fail-closed when draft.
-- Copy uses ASCII punctuation only (no em dashes).

-- Privacy: refresh core sections
update public.site_legal_sections
set
  body = 'Trim (use-trim.com) is an open-core context optimization product. This Privacy Policy explains what we collect when you use the hosted dashboard, public API, authenticated CLI, and related websites. It also explains how the local Trim proxy handles prompts on your machine. By using Trim you agree to this policy.',
  updated_at = now(),
  published_at = coalesce(published_at, now())
where doc_kind = 'privacy' and sort_order = 10;

update public.site_legal_sections
set
  heading = 'Who we are',
  body = 'The hosted service is operated under the company legal name configured for the deployment (COMPANY_LEGAL_NAME). Support contact is the configured support email. Self-hosted operators who run their own Trim control plane are independent controllers for the data on their stack.',
  updated_at = now(),
  published_at = coalesce(published_at, now())
where doc_kind = 'privacy' and sort_order = 20;

update public.site_legal_sections
set
  heading = 'Account and authentication',
  body = 'Sign-in is social only, using providers enabled by the operator (for example Google, GitHub, and GitLab via the configured OAuth allow-list). We store your account id, email, display name, avatar URL, and authenticated provider identity from the identity provider. We do not offer email/password or phone sign-up. When Automatic Account Linking is enabled at the identity provider, verified emails across providers may join one Trim account.',
  updated_at = now(),
  published_at = coalesce(published_at, now())
where doc_kind = 'privacy' and sort_order = 30;

update public.site_legal_sections
set
  heading = 'Usage, billing, and quotas',
  body = 'Metered CLI and API usage, subscription status, receipts, credit balances, and plan entitlements are stored server-side in PostgreSQL and Redis. Clearing local CLI files or reinstalling the binary does not reset quotas. Paddle acts as Merchant of Record for payments and may provide tax invoices under Paddle terms. Plan prices and Paddle price identifiers live in our plan catalog, not in client code.',
  updated_at = now(),
  published_at = coalesce(published_at, now())
where doc_kind = 'privacy' and sort_order = 40;

update public.site_legal_sections
set
  heading = 'Device and fraud signals',
  body = 'To limit free-tier multi-account abuse, the authenticated CLI may send a hardware-derived identifier (hashed machine id). API keys can be device-bound. We may also associate TLS/JA4 fingerprints, IP addresses, ASN reputation, and Cloudflare edge headers (when present) with accounts for fraud prevention and security enforcement. These signals are not used for advertising.',
  updated_at = now(),
  published_at = coalesce(published_at, now())
where doc_kind = 'privacy' and sort_order = 50;

update public.site_legal_sections
set
  heading = 'Local proxy, prompts, and telemetry',
  body = 'The local Trim proxy (trim start) processes prompts on your machine before they are forwarded to your chosen upstream model provider. Trim does not require uploading full prompt bodies to Trim cloud for Fast Mode compression. Optional anonymous product telemetry respects DO_NOT_TRACK=1 and trim telemetry disable. Self-hosted deployments can disable cloud anti-fraud and billing checks via deployment mode. Deep Mode (LLMLingua family) runs only on trim compress on the developer machine; the hosted cloud path does not load those models.',
  updated_at = now(),
  published_at = coalesce(published_at, now())
where doc_kind = 'privacy' and sort_order = 60;

-- Additional privacy sections
insert into public.site_legal_sections
  (doc_kind, sort_order, heading, body, contact_lead, contact_trail, uses_support_email, published_at)
values
  (
    'privacy', 70, 'Cookies and similar technologies',
    'The dashboard and marketing site may use essential cookies or local storage for authentication sessions, CSRF protection, theme preference, and soft-navigation cache behavior. We do not run third-party advertising cookies on use-trim.com.',
    '', '', false, now()
  ),
  (
    'privacy', 80, 'Data retention',
    'Account profile data is retained while your account is active. Usage events and receipts are retained for metering, support, and legal record-keeping. After account deletion we remove personal account data from active systems, subject to billing and tax retention requirements which may keep anonymized or legally required records.',
    '', '', false, now()
  ),
  (
    'privacy', 90, 'Subprocessors and transfers',
    'We use infrastructure and payment processors such as our database host, Redis provider, identity providers you choose at sign-in, Paddle for billing, and CDN or hosting vendors for the website and API. Data may be processed in the regions those providers operate. Where required, we rely on appropriate transfer mechanisms offered by those providers.',
    '', '', false, now()
  ),
  (
    'privacy', 100, 'Children',
    'Trim is built for professional software developers and organizations. It is not directed to children under 16, and we do not knowingly collect personal data from children.',
    '', '', false, now()
  ),
  (
    'privacy', 110, 'Your rights',
    'Depending on your location you may have rights to access, correct, export, or delete personal account data. The dashboard supports GDPR-style account deletion flows where enabled. You may also disconnect linked social identities, except you cannot remove the last remaining sign-in method on an account.',
    'Contact', 'for privacy requests.', true, now()
  ),
  (
    'privacy', 120, 'Changes to this policy',
    'We may update this Privacy Policy as the product evolves. Material changes will be reflected by updating the "Last updated" date on this page. Continued use of the service after an update constitutes acceptance of the revised policy where permitted by law.',
    '', '', false, now()
  )
on conflict (doc_kind, sort_order) do update set
  heading = excluded.heading,
  body = excluded.body,
  contact_lead = excluded.contact_lead,
  contact_trail = excluded.contact_trail,
  uses_support_email = excluded.uses_support_email,
  updated_at = now(),
  published_at = coalesce(public.site_legal_sections.published_at, excluded.published_at);

-- Re-point old "Your rights" row if it still exists at 60 with old heading after updates above.
-- Terms: refresh core sections
update public.site_legal_sections
set
  body = 'These Terms of Service govern access to and use of the Trim hosted service (use-trim.com), the authenticated CLI when connected to our API, the dashboard, and related websites. The open-source CLI client and backend source remain subject to their repository licenses. By creating an account or using the hosted service you agree to these terms.',
  updated_at = now(),
  published_at = coalesce(published_at, now())
where doc_kind = 'terms' and sort_order = 10;

update public.site_legal_sections
set
  heading = 'Accounts and eligibility',
  body = 'You must sign in with an allowed social provider configured by the Trim operator. You are responsible for activity under your account. One natural person should not create multiple free accounts to bypass quotas. We may suspend or terminate accounts that abuse hardware, network, identity, or payment signals described in the Privacy Policy, or that violate these terms.',
  updated_at = now(),
  published_at = coalesce(published_at, now())
where doc_kind = 'terms' and sort_order = 20;

update public.site_legal_sections
set
  heading = 'Plans, billing, and credits',
  body = 'Plan prices, annual display discounts, credits, seat allocations, and Paddle price identifiers are defined in our database catalog and billed through Paddle as Merchant of Record. Self-serve downgrades may be disabled while a paid period remains unexpired. Upgrades may use Paddle proration according to billing settings. Exhausted quotas return payment-required errors until you upgrade or purchase top-ups. Top-up packs do not change your plan tier.',
  updated_at = now(),
  published_at = coalesce(published_at, now())
where doc_kind = 'terms' and sort_order = 30;

update public.site_legal_sections
set
  heading = 'Acceptable use',
  body = 'You may not reverse-engineer canary keys, attack or overload the API, resell free-tier access, bypass device binding, or use Trim to violate third-party LLM provider terms or applicable law. We may enforce a minimum CLI version kill-switch for security patches. Automated scraping of the dashboard or docs in a way that degrades service is prohibited.',
  updated_at = now(),
  published_at = coalesce(published_at, now())
where doc_kind = 'terms' and sort_order = 40;

update public.site_legal_sections
set
  heading = 'Service description and changes',
  body = 'Trim provides context optimization middleware: a local OpenAI-compatible proxy, optional cloud metering, and dashboard controls. Features may change as we improve Fast Mode heuristics, billing, and admin tooling. We may modify or discontinue features with reasonable notice when practical. Beta or preview features are provided as available.',
  updated_at = now(),
  published_at = coalesce(published_at, now())
where doc_kind = 'terms' and sort_order = 50;

update public.site_legal_sections
set
  heading = 'Intellectual property',
  body = 'Trim branding, hosted software, and documentation are owned by the operator or its licensors. You retain ownership of your code and prompts. Feedback you submit may be used to improve the product without obligation to you.',
  contact_lead = '',
  contact_trail = '',
  uses_support_email = false,
  updated_at = now(),
  published_at = coalesce(published_at, now())
where doc_kind = 'terms' and sort_order = 60;

insert into public.site_legal_sections
  (doc_kind, sort_order, heading, body, contact_lead, contact_trail, uses_support_email, published_at)
values
  (
    'terms', 70, 'Disclaimer of warranties',
    'Context trimming changes payloads before they reach upstream models. Active-file protection and heuristics reduce risk, but output quality can still vary. Except where prohibited by law, the service is provided "as available" without warranties of merchantability, fitness for a particular purpose, or non-infringement.',
    '', '', false, now()
  ),
  (
    'terms', 80, 'Limitation of liability',
    'To the maximum extent permitted by law, Trim and its operators are not liable for indirect, incidental, special, consequential, or punitive damages, or for lost profits, lost data, or model output quality issues arising from use of the service. Aggregate liability for claims relating to the hosted service is limited to the amounts you paid to Trim for the service in the three months before the claim.',
    '', '', false, now()
  ),
  (
    'terms', 90, 'Indemnity',
    'You agree to defend and indemnify Trim against claims arising from your misuse of the service, your violation of these terms, or your violation of third-party rights or provider terms in connection with prompts you send through Trim.',
    '', '', false, now()
  ),
  (
    'terms', 100, 'Governing law',
    'Unless a mandatory local law requires otherwise, these terms are governed by the laws applicable to the operator''s principal place of business, excluding conflict-of-law rules. Courts in that venue have exclusive jurisdiction, subject to consumer protections that cannot be waived.',
    '', '', false, now()
  ),
  (
    'terms', 110, 'Changes and contact',
    'We may update these Terms of Service by posting a revised version with an updated "Last updated" date. Material changes may also be communicated in product or by email when appropriate.',
    'Questions:', '.', true, now()
  )
on conflict (doc_kind, sort_order) do update set
  heading = excluded.heading,
  body = excluded.body,
  contact_lead = excluded.contact_lead,
  contact_trail = excluded.contact_trail,
  uses_support_email = excluded.uses_support_email,
  updated_at = now(),
  published_at = coalesce(public.site_legal_sections.published_at, excluded.published_at);

-- Ensure every legal row used publicly is published
update public.site_legal_sections
set published_at = coalesce(published_at, now())
where doc_kind in ('privacy', 'terms') and published_at is null;

update public.site_messages
set body = '27 March 2026'
where code = 'LEGAL_UPDATED_DATE';

insert into public.site_messages (code, body) values
  ('LEGAL_UPDATED_DATE', '27 March 2026')
on conflict (code) do update set body = excluded.body;
