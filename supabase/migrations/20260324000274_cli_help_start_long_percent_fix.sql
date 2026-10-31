-- CLI_HELP_START_LONG is shown raw (cobra Long), not via Printf.
-- Migration 00267 stored %% which users saw as literal %%. Fix to a single %.
-- Idempotent. No em dashes.

update public.site_messages
set body = $body$Starts the local proxy for IDE traffic. Fast Mode always runs. When account preferences set Deep Mode on, each request also runs Segmented Allowlist Deep after Fast: freeze tools/system/paths/code/JSON, compress last-user prose islands only, validate, else fail-closed Fast.

After traffic flows, check savings clearly with:
  trim stats
  http://127.0.0.1:<port>/dashboard → Show savings detail

Whole-body Saved % can look low on agent chrome; Deep stage (prose) and Deep status explain why.$body$,
    updated_at = now()
where code = 'CLI_HELP_START_LONG';
