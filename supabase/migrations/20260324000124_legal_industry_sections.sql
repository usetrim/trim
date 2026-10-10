-- Industry-complete Privacy and Terms sections (Cursor / Vercel style coverage).
-- ASCII punctuation only. No em dashes.

-- Privacy additions
insert into public.site_legal_sections
  (doc_kind, sort_order, heading, body, contact_lead, contact_trail, uses_support_email, published_at)
values
  (
    'privacy', 65, 'What we do not sell',
    'We do not sell your personal information. We do not use prompt contents from the local proxy path for advertising. Optional product telemetry, when enabled, is aggregated and can be disabled.',
    '', '', false, now()
  ),
  (
    'privacy', 85, 'Security measures',
    'We use industry-standard controls for the hosted control plane: TLS in transit, access-controlled databases, keyed API authentication, device binding for keys where required, and operational monitoring. No method of transmission or storage is perfectly secure. Report suspected vulnerabilities through the configured support email.',
    '', '', false, now()
  ),
  (
    'privacy', 95, 'International users',
    'If you access Trim from outside the region where our primary infrastructure runs, your information may be transferred to and processed in that region and in regions operated by our subprocessors. Where required, we rely on contracts and transfer mechanisms those providers offer.',
    '', '', false, now()
  ),
  (
    'privacy', 105, 'Region-specific rights',
    'Depending on where you live, you may have rights under laws such as the GDPR, UK GDPR, or CCPA/CPRA, including access, correction, deletion, portability, and the right to opt out of certain sharing. Use dashboard deletion tools where available, or contact support. We will not discriminate against you for exercising privacy rights.',
    '', '', false, now()
  ),
  (
    'privacy', 115, 'Upstream model providers',
    'After Trim compresses a request locally, the slim prompt is sent to the upstream model provider you configure (for example OpenAI-compatible endpoints). Those providers process the forwarded content under their own terms and privacy policies. Trim does not control how upstream providers retain model inputs.',
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

-- Terms additions
insert into public.site_legal_sections
  (doc_kind, sort_order, heading, body, contact_lead, contact_trail, uses_support_email, published_at)
values
  (
    'terms', 45, 'Teams and seats',
    'Team plans allocate seats according to the catalog. Owners and admins are responsible for invites, role changes, and seat counts. Invited members must sign in with the invited email using an allowed provider. Misuse of seats or sharing credentials may lead to suspension.',
    '', '', false, now()
  ),
  (
    'terms', 55, 'API keys and CLI access',
    'API keys authenticate CLI login and gateway calls. Keys may be device-bound. You must keep secrets confidential. We may revoke keys that leak, violate binding policy, or present fraud signals. Minimum CLI version requirements may block outdated clients for security reasons.',
    '', '', false, now()
  ),
  (
    'terms', 65, 'Open source and self-hosting',
    'Self-hosted deployments of Trim software are governed by the applicable repository licenses in addition to any commercial terms for hosted use-trim.com. Operators who run their own control plane are responsible for their users, compliance, and support.',
    '', '', false, now()
  ),
  (
    'terms', 75, 'Third-party services',
    'Trim integrates with identity providers, Paddle for billing, hosting and CDN vendors, and upstream LLM providers you choose. Your use of those services is also subject to their terms. We are not responsible for outages or policy changes of third parties outside our control.',
    '', '', false, now()
  ),
  (
    'terms', 85, 'Suspension and termination',
    'We may suspend or terminate access for violations of these terms, unpaid invoices, fraud, or risk to the platform. You may stop using the service and delete your account where the product provides that control. Provisions that by nature should survive (including liability limits, indemnity, and governing law) survive termination.',
    '', '', false, now()
  ),
  (
    'terms', 95, 'Export controls and sanctions',
    'You may not use Trim if you are prohibited under applicable export control or sanctions laws. You represent that you are not located in a comprehensively sanctioned jurisdiction and are not a denied party.',
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

update public.site_legal_sections
set published_at = coalesce(published_at, now())
where doc_kind in ('privacy', 'terms') and published_at is null;

update public.site_messages
set body = '27 March 2026', updated_at = now()
where code = 'LEGAL_UPDATED_DATE';
