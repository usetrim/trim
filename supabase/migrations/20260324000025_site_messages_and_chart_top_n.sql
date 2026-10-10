-- Operator-editable public site chrome (landing + legal titles/meta).
-- Bodies for privacy/terms already live in site_legal_sections.
-- Chart top-N for dashboard breakdowns (no invent limit 20 in SQL).

create table if not exists public.site_messages (
  code text primary key,
  body text not null,
  updated_at timestamptz not null default now()
);

comment on table public.site_messages is
  'Public marketing/legal chrome strings. API reads these; operators edit rows. No invent fallback when empty.';

alter table public.billing_settings
  add column if not exists chart_top_n int not null default 20
    check (chart_top_n >= 1 and chart_top_n <= 100);

alter table public.billing_settings
  add column if not exists chart_series_days int not null default 30
    check (chart_series_days >= 1 and chart_series_days <= 366);

comment on column public.billing_settings.chart_top_n is
  'Max rows for dashboard model/mode breakdown charts (GROUP BY ... LIMIT).';

comment on column public.billing_settings.chart_series_days is
  'Lookback days for dashboard token series (created_at >= now() - interval).';

insert into public.site_messages (code, body) values
  ('LANDING_BRAND', 'Trim'),
  ('LANDING_NAV_DASHBOARD', 'Dashboard'),
  ('LANDING_NAV_SIGN_IN', 'Sign in'),
  ('LANDING_EYEBROW', 'Open-core context optimization'),
  ('LANDING_HEADLINE', 'Trim the tokens. Keep the signal.'),
  ('LANDING_TAGLINE', 'Local AST heuristics plus optional Deep Mode on your machine. Cloud meters usage and billing only.'),
  ('LANDING_CTA_START', 'Get started'),
  ('LANDING_CTA_SOURCE', 'View source'),
  ('LANDING_FOOTER_PRIVACY', 'Privacy'),
  ('LANDING_FOOTER_TERMS', 'Terms'),
  ('LANDING_FOOTER_GITHUB', 'GitHub'),
  ('LANDING_COPYRIGHT_FMT', E'\u00a9 %d Trim'),
  ('LANDING_SOURCE_URL', 'https://github.com/usetrim/trim'),
  ('LANDING_INSTALL_SNIPPET', E'curl -fsSL https://use-trim.com/install.sh | sh\ntrim start\n# Cursor → Override Base URL → http://localhost:8000/v1'),
  ('LEGAL_PRIVACY_TITLE', 'Privacy Policy'),
  ('LEGAL_TERMS_TITLE', 'Terms of Service'),
  ('LEGAL_UPDATED_PREFIX', 'Last updated:'),
  ('LEGAL_UPDATED_DATE', '24 March 2026'),
  ('LEGAL_LINK_PRIVACY', 'Privacy Policy'),
  ('LEGAL_LINK_TERMS', 'Terms of Service'),
  ('LEGAL_LINK_HOME', 'Home'),
  ('LOGIN_TITLE', 'Sign in to Trim'),
  ('AUTH_PROVIDERS_EMPTY', 'No sign-in providers are enabled. Set ALLOWED_AUTH_PROVIDERS on the API.'),
  ('LOGIN_PROVIDER_DISABLED', 'That sign-in provider is not enabled for this deployment.'),
  ('LOGIN_FAILED', 'Sign in failed.'),
  ('AUTH_NOT_SIGNED_IN', 'Not signed in'),
  ('DASHBOARD_META_SEP', ' · '),
  ('CLI_LOGIN_PHRASE_OR', ' or '),
  ('CLI_LOGIN_PHRASE_COMMA', ', or '),
  ('DIALOG_CLOSE', 'Close')
on conflict (code) do nothing;

alter table public.site_messages enable row level security;

create policy "site_messages_public_read" on public.site_messages
  for select using (true);
