-- Segmented Allowlist Deep: CLI start help DB-wins (detach + Cursor Link + Who this is for),
-- --foreground flag chrome, landing flow/why copy. Idempotent. No em dashes.

update public.site_messages
set body = $body$Starts the local proxy for IDE traffic, then returns your shell (proxy keeps running in the background). Stop with: trim stop. Use --foreground to keep this terminal attached (Ctrl+C stops).

Fast Mode always runs. When account preferences set Deep Mode on, each request also runs Segmented Allowlist Deep after Fast: freeze tools/system/paths/code/JSON, compress each last-user prose island in place (order preserved beside paths), validate, else fail-closed Fast.

Local Base URL (127.0.0.1) is for on-device clients (Continue, VS Code Chat, Claude Code). Cursor Agent needs Trim Cursor Link: trim cursor-link enable → trim cursor-link url (HTTPS …/v1; not 127.0.0.1).

After traffic flows, check savings with:
  trim stats
  http://127.0.0.1:<port>/dashboard → Show savings detail

Who this is for: Continue, VS Code Chat, Claude Code (local Base URL), and Cursor Agent via Trim Cursor Link (trim cursor-link url). Whole-body Saved % can look low on agent chrome; Deep stage (prose) and Deep status explain why.$body$,
    updated_at = now()
where code = 'CLI_HELP_START_LONG';

insert into public.site_messages (code, body, updated_at)
values (
  'CLI_HELP_START_FLAG_FOREGROUND',
  'Keep the proxy in this terminal (Ctrl+C to stop); default detaches and returns the shell',
  now()
)
on conflict (code) do update
set body = excluded.body,
    updated_at = now();

update public.site_messages
set body = 'Chat and agents POST through the local proxy. Fast Mode always runs. When Preferences Deep is on, Segmented Allowlist Deep also compresses each last-user prose island in place (order preserved beside paths); tools and paths stay frozen. Fast Mode drops vendor dumps, lockfiles, and log spam while keeping the files and stacks you are editing.',
    updated_at = now()
where code = 'LANDING_FLOW_3_BODY';

update public.site_messages
set body = 'Fast Mode compresses on your machine before anything is forwarded. When Deep Mode is on, Segmented Allowlist Deep runs after Fast on eligible last-user prose (order preserved beside paths). Trim cloud meters the account; it does not need the raw pack.',
    updated_at = now()
where code = 'LANDING_WHY_1_BODY';
