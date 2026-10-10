-- Homepage interactive demo + how-it-works + pricing section chrome (fail-closed; no client invent).
insert into public.site_messages (code, body) values
  ('LANDING_NAV_PRICING', 'Pricing'),
  ('LANDING_DEMO_TITLE', 'See what Trim cuts before the model sees it'),
  ('LANDING_DEMO_SUBTITLE', 'A live walkthrough of noisy agent context becoming a tight, billable prompt.'),
  ('LANDING_DEMO_BEFORE_LABEL', 'Before Trim'),
  ('LANDING_DEMO_AFTER_LABEL', 'After Trim'),
  ('LANDING_DEMO_RUN_LABEL', 'Run Trim'),
  ('LANDING_DEMO_REPLAY_LABEL', 'Replay'),
  ('LANDING_DEMO_TOKENS_IN_FMT', '%s tokens in'),
  ('LANDING_DEMO_TOKENS_OUT_FMT', '%s tokens out'),
  ('LANDING_DEMO_SAVED_FMT', '%s%% less context'),
  ('LANDING_DEMO_BEFORE_BODY', E'// vendor/react/index.js (pulled into context)\nexport function createElement() { /* 400 lines */ }\n\n// build/logs/ci-run-88421.txt\n[INFO] compiling...\n[WARN] deprecated dep\n[DEBUG] cache miss x 220\n\n// node_modules/.cache/noise.ts\nconst _unused = Array(500).fill(0);\n\n// src/auth/session.ts (signal)\nexport async function requireUser(req) {\n  const session = await getSession(req);\n  if (!session) throw new AuthError("unauthorized");\n  return session.user;\n}'),
  ('LANDING_DEMO_AFTER_BODY', E'// src/auth/session.ts (kept)\nexport async function requireUser(req) {\n  const session = await getSession(req);\n  if (!session) throw new AuthError("unauthorized");\n  return session.user;\n}\n\n// vendor/react: skeletonized\n// createElement() { /* omitted */ }\n\n// build logs: pruned\n// [noise lines removed]'),
  ('LANDING_HOW_TITLE', 'How Trim works'),
  ('LANDING_HOW_1_TITLE', 'Install the local proxy'),
  ('LANDING_HOW_1_BODY', 'Point Cursor or your IDE at Trim on localhost. Requests stay on your machine.'),
  ('LANDING_HOW_2_TITLE', 'AST-aware compression'),
  ('LANDING_HOW_2_BODY', 'Fast Mode skeletonizes background code and drops log noise before tokens leave.'),
  ('LANDING_HOW_3_TITLE', 'Cloud meters usage only'),
  ('LANDING_HOW_3_BODY', 'Auth, quotas, billing, and receipts live in the cloud. Your source stays local.'),
  ('LANDING_PRICING_TITLE', 'Pricing'),
  ('LANDING_PRICING_SUBTITLE', 'Start free. Upgrade when your team needs more credits.'),
  ('LANDING_PRICING_CTA', 'Choose plan'),
  ('LANDING_PRICING_ANNUAL', 'Annual'),
  ('LANDING_PRICING_MONTHLY', 'Monthly'),
  ('LANDING_INSTALL_TITLE', 'Install in one minute'),
  ('LANDING_CTA_DEMO', 'See live demo'),
  ('LANDING_HERO_NOTE', 'Local proxy. Cloud control plane. Your code never uploads for trimming.')
on conflict (code) do update set body = excluded.body;

update public.site_messages set body = 'Open-core context optimization'
  where code = 'LANDING_EYEBROW';
update public.site_messages set body = 'Trim the tokens. Keep the signal.'
  where code = 'LANDING_HEADLINE';
update public.site_messages set body = 'Local AST heuristics plus optional Deep Mode on your machine. Cloud meters usage and billing only.'
  where code = 'LANDING_TAGLINE';
