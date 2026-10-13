-- Clarify Cursor Agent vs on-device Base URLs after Trim Cursor Link.
-- Idempotent upsert. No schema drops.

insert into public.site_messages (code, body, updated_at)
values
  (
    'CLI_START_BASE_URL_LABEL',
    'Local Base URL',
    now()
  ),
  (
    'CLI_START_POINT_IDE_FMT',
    E'On-device clients: Continue apiBase / VS Code Chat Custom Endpoint / Claude Code origin use %s://127.0.0.1:%s (Continue and most SDKs need …/v1; VS Code Chat needs …/v1/chat/completions; Claude Code needs origin only, no /v1).',
    now()
  ),
  (
    'CLI_START_CURSOR_AGENT_LABEL',
    'Cursor Agent Base URL',
    now()
  ),
  (
    'CLI_START_CURSOR_AGENT_HINT_OFF',
    'Cursor Agent: do not paste 127.0.0.1 (private networks forbidden). Run: trim cursor-link enable → trim cursor-link url → paste HTTPS …/v1 into Cursor Override OpenAI Base URL. Docs: https://use-trim.com/docs/cli/cursor-link',
    now()
  ),
  (
    'CLI_START_CURSOR_AGENT_TIP_ON',
    'Cursor Agent: paste Cursor Agent Base URL into Override OpenAI Base URL (HTTPS …/v1, not 127.0.0.1).',
    now()
  ),
  (
    'CLI_START_CURSOR_AGENT_LOCAL_TIP',
    'Local tools / Claude Code: keep using http://127.0.0.1:%s as documented.',
    now()
  ),
  (
    'CLI_START_CURSOR_AGENT_CLOUDFLARED_TIP',
    'Install cloudflared or set TRIM_CLOUDFLARED, then: trim cursor-link status',
    now()
  ),
  (
    'CLI_HELP_START_LONG',
    E'Starts the local proxy for IDE traffic, then returns your shell (proxy keeps running in the background). Stop with: trim stop. Use --foreground to keep this terminal attached (Ctrl+C stops).\n\nFast Mode always runs. When account preferences set Deep Mode on, each request also runs Deep (LLMLingua) after Fast before forwarding upstream.\n\nLocal Base URL (127.0.0.1) is for on-device clients (Continue, VS Code Chat, Claude Code). Cursor Agent needs Trim Cursor Link: trim cursor-link enable → trim cursor-link url (HTTPS …/v1; not 127.0.0.1).\n\nAfter traffic flows, check savings with:\n  trim stats\n  http://127.0.0.1:<port>/dashboard → Show savings detail',
    now()
  ),
  (
    'CLI_SETUP_CURSOR_AGENT_HEADER',
    'Cursor Agent (cloud) - use Trim Cursor Link, not localhost:',
    now()
  ),
  (
    'CLI_SETUP_CURSOR_AGENT_BLOCK',
    E'  trim cursor-link enable\n  trim start\n  trim cursor-link url\n  # paste HTTPS …/cl/<token>/v1 into Cursor → Settings → Models → Override OpenAI Base URL\n  # do not paste http://127.0.0.1:…/v1 for Cursor Agent\n  Docs: https://use-trim.com/docs/cli/cursor-link',
    now()
  ),
  (
    'CLI_SETUP_STEP_CURSOR_LINK',
    'Cursor Agent: trim cursor-link enable && trim start && trim cursor-link url (paste HTTPS …/v1; not 127.0.0.1)',
    now()
  ),
  (
    'CLI_SETUP_CURSOR_SKIPPED',
    'Cursor settings.json: skipped writing localhost Base URL (Cursor Agent needs Trim Cursor Link). Continue / VS Code / Windsurf / Zed still use local …/v1 when present.',
    now()
  ),
  (
    'CLI_SETUP_SHELL_OPENAI_FMT',
    '  export OPENAI_BASE_URL=%q   # Continue / OpenAI-compat on-device clients (Gemini, GPT, Claude-via-adapter). Cursor Agent: use trim cursor-link url instead.',
    now()
  ),
  (
    'CLI_SETUP_TLS_POINT_IDE',
    '  On-device: Continue / VS Code Chat at https://localhost:<port>/v1 (or …/v1/chat/completions). Cursor Agent: trim cursor-link url (HTTPS public …/v1).',
    now()
  ),
  (
    'LANDING_USE_2_BODY',
    'Cursor Agent: Override OpenAI Base URL = trim cursor-link url (HTTPS …/v1, not 127.0.0.1). Continue / VS Code: local URL from trim start (example http://127.0.0.1:8888/v1).',
    now()
  ),
  (
    'CLI_HELP_SETUP_LONG',
    E'Writes OpenAI-compatible Base URL overrides for on-device IDEs when present (VS Code, Windsurf, Zed, Continue ~/.continue/config.yaml). Does not write localhost into Cursor (Cursor Agent needs Trim Cursor Link).\n\nPrints copy/paste blocks for:\n  - Cursor Agent (trim cursor-link enable → trim cursor-link url)\n  - Claude Code (ANTHROPIC_BASE_URL = Trim origin, no /v1)\n  - Continue (config.yaml apiBase = Trim …/v1)\n  - VS Code Chat Custom Endpoint (full …/v1/chat/completions + wizard secret)\nUse --tls for local self-signed certs. Docs: /docs/cli/cursor-link /docs/ide/connect /docs/ide/continue /docs/ide/vscode',
    now()
  )
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
