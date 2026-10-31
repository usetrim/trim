-- Landing install / get-started chrome: bootstrap before start (match Quickstart).
-- Idempotent. No em dashes.

insert into public.site_messages (code, body, updated_at) values
  (
    'LANDING_INSTALL_SNIPPET',
    $body$curl -fsSL https://use-trim.com/install.sh | sh
trim compress --bootstrap
trim login
trim start
# Cursor Agent: trim cursor-link url → Override Base URL (HTTPS …/v1)
# On-device: http://127.0.0.1:8888/v1$body$,
    now()
  ),
  (
    'LANDING_FLOW_1_TITLE',
    'Install, bootstrap Deep, then start Trim',
    now()
  ),
  (
    'LANDING_FLOW_1_BODY',
    'Run the one-line installer, then trim compress --bootstrap once (green success banner; local llmlingua, not Trim cloud), trim login, and trim start. Trim listens on TRIM_PORT (example http://127.0.0.1:8888/v1) as an OpenAI-compatible endpoint. Your code stays on this machine.',
    now()
  ),
  (
    'LANDING_USE_1_TITLE',
    'Install, bootstrap, then trim start',
    now()
  ),
  (
    'LANDING_USE_1_BODY',
    'curl the install script (or use your package manager), run trim compress --bootstrap once, then start the local proxy.',
    now()
  ),
  (
    'LANDING_HOW_1_TITLE',
    'Install, bootstrap Deep, start the local proxy',
    now()
  ),
  (
    'LANDING_HOW_1_BODY',
    'After install, run trim compress --bootstrap once (package managers do not pip-install llmlingua). Then trim start binds an OpenAI-compatible proxy on localhost. Compression runs locally, then the slim prompt goes to your model provider.',
    now()
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
