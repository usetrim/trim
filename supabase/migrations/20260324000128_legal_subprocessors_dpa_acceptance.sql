-- Worldwide SaaS/OSS checklist gaps: named subprocessors (public vendors, not
-- internal ops), DPA availability, clearer acceptance, COPPA age floor.
-- Does not expose fraud fingerprints, env vars, table names, or admin tooling.

insert into public.site_legal_sections
  (doc_kind, sort_order, heading, body, contact_lead, contact_trail, uses_support_email, published_at)
values
(
  'privacy', 105, 'Subprocessors and service providers',
  'We use carefully selected service providers to operate the hosted Service. Typical categories and examples for use-trim.com include: cloud hosting and content delivery; managed database and authentication infrastructure; payment processing by our Merchant of Record; email delivery for account and support messages; and error or uptime monitoring. Identity providers you choose (such as Google, GitHub, or GitLab) receive authentication data under their own terms when you sign in. Upstream AI model providers you select receive the prompts you send them. We require processors to use personal data only to provide services to us under contract. A current summary of categories is maintained in this policy; material changes to core subprocessors may be reflected here or announced in the product when required.',
  '', '', false, now()
),
(
  'privacy', 168, 'Enterprise data processing terms',
  'If your organization needs a Data Processing Agreement (DPA) or similar processor terms for GDPR Article 28 style arrangements, contact us using the support email published with this policy. Self-hosted operators who run their own control plane are responsible for their own processor contracts with their users.',
  'Email', 'to request enterprise privacy or DPA paperwork.', true, now()
),
(
  'privacy', 152, 'Age and children (COPPA)',
  'The Service is not directed to children under 13, and we do not knowingly collect personal information from children under 13. Where a higher age of digital consent applies (for example 16 in some regions), that higher age controls as stated in our Children section. If you believe we have collected information from a child, contact us so we can delete it.',
  '', '', false, now()
),
(
  'terms', 12, 'Acceptance',
  'By creating an account, completing checkout, clicking an accept control, or otherwise using the Service, you agree to these Terms and acknowledge the Privacy Policy. If you do not agree, do not create an account or use the Service. Continued use after we post material updates constitutes acceptance of the revised Terms to the extent permitted by law.',
  '', '', false, now()
),
(
  'terms', 105, 'Customer content ownership',
  'As between you and Trim, you retain ownership of your prompts, project materials, and other content you submit through your tools. Trim does not claim ownership of your repositories. Open-source components remain under their licenses. Hosted Service software and branding remain Trim''s or its licensors'' property.',
  '', '', false, now()
);

update public.site_messages
set body = to_char(now() at time zone 'utc', 'YYYY-MM-DD'),
    updated_at = now()
where code = 'LEGAL_UPDATED_DATE';
