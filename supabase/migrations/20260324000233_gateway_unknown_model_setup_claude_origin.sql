-- Gateway honesty: unknown chat model when adapters are synced (no silent cross-host swap).
-- Fix Claude Code setup copy: ANTHROPIC_BASE_URL must be Trim origin only (no /v1).
-- count_tokens remains Anthropic-native passthrough on /v1/messages/count_tokens.

insert into public.site_messages (code, body, updated_at)
values
  (
    'CLI_PROXY_ERR_UNKNOWN_MODEL_FMT',
    'No provider adapter matched model %q. Pick an id from GET /v1/models, use a DB alias (trim-claude-*, trim-gemini-*, …), or add a match_model_prefixes row in provider_adapters. Trim does not invent a host for unknown models.',
    now()
  ),
  (
    'CLI_SETUP_SHELL_ANTHROPIC_FMT',
    '  export ANTHROPIC_BASE_URL=%q   # Claude Code: Trim listen ORIGIN only (no /v1; Claude Code appends /v1/messages)',
    now()
  ),
  (
    'CLI_SETUP_CLAUDE_CODE_BLOCK_FMT',
    'Claude Code (Door A) - copy/paste:
  export ANTHROPIC_BASE_URL=%q
  export ANTHROPIC_API_KEY=sk-ant-…   # prefer workspace-scoped key
  # optional discovery:
  export CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY=1
  # org/multi-workspace keys only:
  # export TRIM_ANTHROPIC_WORKSPACE_ID=wrkspc_…   # then restart trim start
  # or send header anthropic-workspace-id on each request',
    now()
  ),
  (
    'CLI_PROXY_ADAPTER_ENABLED_FMT',
    'OpenAI-compat provider adapters loaded (%d enabled). Model prefixes route via DB (Claude Messages and same-shape hosts with upstream_base_url). GET /v1/models lists DB provider_discoverable_models. Unmatched chat models return a clear Trim error (no silent host invent).',
    now()
  ),
  (
    'PREFERENCES_CLI_HELP_8',
    'Claude Code: export ANTHROPIC_BASE_URL to the Trim listen origin only (e.g. http://127.0.0.1:8888) with no /v1 suffix - Claude Code appends /v1/messages. Optional: CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY=1 to load models from Trim GET /v1/models. Prefer a workspace-scoped Anthropic key; org keys need TRIM_ANTHROPIC_WORKSPACE_ID or anthropic-workspace-id. Run trim setup to print the correct Claude Code env block.',
    now()
  )
on conflict (code) do update set
  body = excluded.body,
  updated_at = now();
