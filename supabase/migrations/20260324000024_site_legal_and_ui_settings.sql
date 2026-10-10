-- Legal policy/terms body lives in Postgres (editable); public auth-providers reads DB only.
-- Also: default list page size and max team seat qty (no client invent).

create table if not exists public.site_legal_sections (
  id uuid primary key default gen_random_uuid(),
  doc_kind text not null check (doc_kind in ('privacy', 'terms')),
  sort_order int not null,
  heading text not null default '',
  body text not null default '',
  contact_lead text not null default '',
  contact_trail text not null default '',
  uses_support_email boolean not null default false,
  unique (doc_kind, sort_order)
);

comment on table public.site_legal_sections is
  'Privacy and Terms section bodies. Operators update rows; API never invents legal copy.';

alter table public.billing_settings
  add column if not exists default_page_size int not null default 20
    check (default_page_size >= 1 and default_page_size <= 200);

alter table public.billing_settings
  add column if not exists max_seat_quantity int not null default 500
    check (max_seat_quantity >= 1 and max_seat_quantity <= 10000);

comment on column public.billing_settings.default_page_size is
  'Recommended skip/limit page size for dashboard tables (events, receipts, keys, team).';

comment on column public.billing_settings.max_seat_quantity is
  'Upper bound for team seat quantity input in plan checkout.';

insert into public.site_legal_sections
  (doc_kind, sort_order, heading, body, contact_lead, contact_trail, uses_support_email)
values
  (
    'privacy', 10, '',
    'Trim (use-trim.com) is an open-core context optimization product. This policy describes what we collect when you use the hosted dashboard, API, and authenticated CLI.',
    '', '', false
  ),
  (
    'privacy', 20, 'Account and authentication',
    'Sign-in is social only, using providers enabled by the operator (for example the configured OAuth allow-list). We store your account id, email, display name, avatar URL, and authenticated provider name from the identity provider. We do not offer email/password or phone sign-up.',
    '', '', false
  ),
  (
    'privacy', 30, 'Usage, billing, and quotas',
    'Metered CLI and API usage, subscription status, receipts, and credit balances are stored server-side in PostgreSQL and Redis. Clearing local CLI files or reinstalling the binary does not reset quotas. Paddle (Merchant of Record) processes payments and may provide tax invoices under their terms.',
    '', '', false
  ),
  (
    'privacy', 40, 'Device and fraud signals',
    'To limit free-tier multi-account abuse, the authenticated CLI may send a hardware-derived identifier (hashed machine id). We may also associate TLS/JA4 fingerprints and IP reputation signals (including Cloudflare edge headers when present) with accounts for fraud prevention. These signals are used for security enforcement, not for advertising.',
    '', '', false
  ),
  (
    'privacy', 50, 'Local proxy and telemetry',
    'The local Trim proxy processes prompts on your machine. Optional anonymous product telemetry respects DO_NOT_TRACK=1 and trim telemetry disable. Self-hosted deployments can disable cloud anti-fraud and billing checks via deployment mode.',
    '', '', false
  ),
  (
    'privacy', 60, 'Your rights',
    'You may request export or deletion of personal account data from the dashboard (GDPR-style account deletion). Billing records required for tax law may be retained in anonymized form.',
    'Contact', 'for privacy requests.', true
  ),
  (
    'terms', 10, '',
    'These terms govern use of the Trim hosted service (use-trim.com), the authenticated CLI when connected to our API, and related dashboards. The open-source CLI client and backend source are also subject to their repository licenses (MIT-oriented CLI packaging intent; AGPL-style protection for the hosted backend).',
    '', '', false
  ),
  (
    'terms', 20, 'Accounts',
    'You must sign in with an allowed social provider configured by the Trim operator. One natural person should not create multiple free accounts to bypass quotas. We may suspend accounts that abuse hardware, network, or identity signals described in the Privacy Policy.',
    '', '', false
  ),
  (
    'terms', 30, 'Plans and billing',
    'Plan prices, annual discounts, credits, and Paddle price identifiers are defined in our database catalog and billed through Paddle as Merchant of Record. Self-serve downgrades may be disabled while a paid period remains unexpired; upgrades may use Paddle proration. Exhausted quotas return payment-required errors until you upgrade or purchase top-ups.',
    '', '', false
  ),
  (
    'terms', 40, 'Acceptable use',
    'Do not reverse-engineer canary keys, attack the API, resell free-tier access, or use Trim to violate third-party LLM provider terms. We may enforce a minimum CLI version kill-switch for security patches.',
    '', '', false
  ),
  (
    'terms', 50, 'Disclaimer',
    'Context trimming changes payloads before they reach upstream models. Active-file protection and heuristics reduce risk, but output quality can still vary. The service is provided as available without warranties beyond those required by law.',
    '', '', false
  ),
  (
    'terms', 60, 'Contact',
    '',
    'Questions:', '.', true
  )
on conflict (doc_kind, sort_order) do nothing;

alter table public.site_legal_sections enable row level security;

create policy "site_legal_sections_public_read" on public.site_legal_sections
  for select using (true);
