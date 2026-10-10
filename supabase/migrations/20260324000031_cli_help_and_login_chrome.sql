-- Locale + plan-not-found chrome. Full CLI_HELP_* and login OAuth chrome rows are
-- upserted by SeedAndRefreshSiteMessages on API boot (server/internal/subscriptions).

INSERT INTO public.site_messages (code, body) VALUES
  ('SITE_HTML_LANG', 'en'),
  ('PLAN_NOT_FOUND', 'plan not found'),
  ('LOCAL_HTML_LANG', 'en')
ON CONFLICT (code) DO UPDATE
SET body = EXCLUDED.body,
    updated_at = now();
