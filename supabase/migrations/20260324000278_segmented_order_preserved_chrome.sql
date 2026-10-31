-- Segmented Allowlist Deep: prose islands compressed in place (order preserved beside paths).
-- Align savings-detail / help chrome with per-island reassembly. Idempotent. No em dashes.

update public.site_messages
set body = 'Whole-request Saved % can look low when IDE agent chrome is frozen. Deep compresses each last-user prose island in place (order preserved beside paths); paths, tools, and schemas stay verbatim. Deep stage shows savings on that prose.',
    updated_at = now()
where code = 'LOCAL_SAVINGS_DETAIL_LEAD';

update public.site_messages
set body = $body$Starts the local proxy for IDE traffic. Fast Mode always runs. When account preferences set Deep Mode on, each request also runs Segmented Allowlist Deep after Fast: freeze tools/system/paths/code/JSON, compress each last-user prose island in place (order preserved beside paths), validate, else fail-closed Fast.

After traffic flows, check savings clearly with:
  trim stats
  http://127.0.0.1:<port>/dashboard → Show savings detail

Who this is for: Continue, VS Code Chat, Claude Code (local Base URL), and Cursor Agent via Trim Cursor Link (trim cursor-link url). Whole-body Saved % can look low on agent chrome; Deep stage (prose) and Deep status explain why.$body$,
    updated_at = now()
where code = 'CLI_HELP_START_LONG';
