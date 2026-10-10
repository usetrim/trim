-- Soften internal-sounding phrases with clear public SaaS wording.
update public.site_legal_sections
set body = replace(body, 'the Trim CLI when connected to our hosted control plane', 'the Trim CLI when signed in to the hosted Service'),
    updated_at = now()
where body like '%hosted control plane%';

update public.site_legal_sections
set body = replace(body, 'prevent fraud and abuse', 'prevent abuse and protect accounts'),
    updated_at = now()
where body like '%prevent fraud and abuse%';

update public.site_legal_sections
set body = 'These Terms of Service (''Terms'') govern your access to and use of Trim''s hosted websites, dashboard, API, CLI authentication when signed in to the hosted Service, and related services at use-trim.com (the ''Service''). By creating an account or using the Service you agree to these Terms and our Privacy Policy. If you use the Service on behalf of an organization, you represent that you have authority to bind that organization, and ''you'' includes that organization. If you do not agree, do not use the Service.',
    updated_at = now()
where doc_kind = 'terms' and heading = 'Agreement';

update public.site_legal_sections
set body = 'Trim provides developer tools and hosted account features that help reduce AI coding context size, meter usage, manage plans, and administer teams. Features, interfaces, and integrations may change as we improve the product. We do not guarantee that any particular model provider, IDE, or third-party integration will remain available or unchanged. Beta or experimental features may be offered without the same availability or support expectations as generally available features and may be modified or withdrawn at any time.',
    updated_at = now()
where doc_kind = 'terms' and heading = 'The Service';

update public.site_legal_sections
set body = 'Self-hosted deployments of Trim software are governed by the applicable repository licenses and any commercial agreement you enter for hosted use-trim.com. If you operate your own Trim deployment, you are responsible for your users, compliance, security, support, and any legal notices you publish. Hosted Service terms apply to use-trim.com accounts even if you also run open-source components locally.',
    updated_at = now()
where doc_kind = 'terms' and heading = 'Open source and self-hosting';

update public.site_legal_sections
set body = replace(body, 'Self-hosted operators who run their own control plane', 'Self-hosted operators who run their own Trim deployment'),
    updated_at = now()
where body like '%own control plane%';

update public.site_messages
set body = to_char(now() at time zone 'utc', 'YYYY-MM-DD'),
    updated_at = now()
where code = 'LEGAL_UPDATED_DATE';
