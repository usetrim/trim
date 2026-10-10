-- Continue + VS Code Chat setup chrome: exact files, wizard secrets, config.yaml.
-- Aligns trim setup / trim help setup with proven IDE paths.

INSERT INTO public.site_messages (code, body, updated_at)
VALUES
  (
    'CLI_SETUP_CONTINUE_BLOCK_FMT',
    E'Continue (OpenAI door) - edit ~/.continue/config.yaml (not config.json on modern Continue):\n  provider: openai\n  model: trim-claude-sonnet   # or id from GET /v1/models\n  apiBase: %q\n  apiKey: <Anthropic key for Claude / OpenAI key for GPT>\n  capabilities: [tool_use]\n  useResponsesApi: false\n  Then: Continue sidebar → Reload config → pick Claude via Trim. Docs: /docs/ide/continue',
    now()
  ),
  (
    'CLI_SETUP_VSCODE_CHAT_BLOCK_FMT',
    E'VS Code Chat Custom Endpoint (Ask) - Chat: Manage Language Models → Add Models → Custom Endpoint:\n  url: %q   # must be full …/v1/chat/completions (not only …/v1)\n  id: trim-claude-sonnet\n  apiKey: wizard ${input:chat.lm.secret…} only (never paste raw sk- into the JSON)\n  File: Code/User/chatLanguageModels.json\n  Prefer Continue for Agent file edits. Docs: /docs/ide/vscode',
    now()
  ),
  (
    'CLI_SETUP_CONTINUE_NO_MODEL',
    'Continue config has no openai-compatible model with apiBase. Edit ~/.continue/config.yaml (see trim setup Continue block / docs/ide/continue), or set TRIM_SETUP_DEFAULT_MODEL and TRIM_SETUP_DEFAULT_MODEL_TITLE then re-run trim setup.',
    now()
  ),
  (
    'CLI_START_POINT_IDE_FMT',
    'Point IDE Base URL at %s://localhost:%s/v1 (Cursor Override / Continue apiBase). VS Code Chat Custom Endpoint needs …/v1/chat/completions - see docs.',
    now()
  ),
  (
    'CLI_HELP_SETUP_SHORT',
    'Point Cursor, Continue, VS Code Chat, Claude Code, and shell env at Trim',
    now()
  ),
  (
    'CLI_HELP_SETUP_LONG',
    E'Writes OpenAI-compatible Base URL overrides for installed IDEs (Cursor, VS Code settings, Windsurf, Zed, Continue ~/.continue/config.yaml when present) and prints copy/paste blocks for:\n  - Claude Code (ANTHROPIC_BASE_URL = Trim origin, no /v1)\n  - Continue (config.yaml apiBase = Trim …/v1)\n  - VS Code Chat Custom Endpoint (full …/v1/chat/completions + wizard secret)\nUse --tls for local self-signed certs. Docs: /docs/ide/connect /docs/ide/continue /docs/ide/vscode',
    now()
  ),
  (
    'CLI_SETUP_SHELL_HEADER',
    'Shell / Claude Code / Continue / Aider (same Trim listen host; different paths and env):',
    now()
  )
ON CONFLICT (code) DO UPDATE
SET body = EXCLUDED.body,
    updated_at = EXCLUDED.updated_at;
