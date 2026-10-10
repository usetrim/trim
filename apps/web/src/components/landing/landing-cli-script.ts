/**
 * Shared live-CLI narrative for all landing terminal surfaces.
 * Keep product story identical: install → proxy → trim request → status meter.
 */

export type CliTone = "dim" | "ok" | "warn" | "saved" | "label";

export type CliScriptLine =
  | { kind: "in"; text: string; delay?: number }
  | { kind: "out"; text: string; tone?: CliTone }
  | { kind: "blank" };

export const CLI_SCRIPT_UNIX: CliScriptLine[] = [
  { kind: "in", text: "curl -fsSL https://use-trim.com/install.sh | sh" },
  { kind: "out", text: "→ fetching trim-cli 0.9.2 (darwin-arm64)", tone: "dim" },
  { kind: "out", text: "→ verifying sha256… ok", tone: "dim" },
  { kind: "out", text: "→ installing to /usr/local/bin/trim", tone: "dim" },
  { kind: "out", text: "✓ trim 0.9.2 ready", tone: "ok" },
  { kind: "blank" },
  { kind: "in", text: "trim login", delay: 380 },
  { kind: "out", text: "→ open browser · device code TRM-7K2Q", tone: "dim" },
  { kind: "out", text: "✓ signed in as you@team.dev", tone: "ok" },
  { kind: "blank" },
  { kind: "in", text: "trim start", delay: 420 },
  { kind: "out", text: "starting local OpenAI-compatible proxy", tone: "label" },
  { kind: "out", text: "  listen     http://127.0.0.1:8888/v1", tone: "ok" },
  { kind: "out", text: "  mode       Fast Mode (skeleton + drop noise)", tone: "dim" },
  { kind: "out", text: "  deep       off (toggle: trim mode deep)", tone: "dim" },
  {
    kind: "out",
    text: "  privacy    Trim cloud does not receive prompts for compression",
    tone: "dim",
  },
  { kind: "out", text: "  upstream   provider after compress", tone: "dim" },
  {
    kind: "out",
    text: "✓ proxy live - set Cursor or VS Code Base URL to the listen URL",
    tone: "ok",
  },
  { kind: "blank" },
  { kind: "out", text: "← POST /v1/chat/completions   model=gpt-4.1   Composer", tone: "label" },
  { kind: "out", text: "  capture   noisy workspace context", tone: "dim" },
  { kind: "out", text: "  tokens in 12,840", tone: "dim" },
  { kind: "out", text: "  ── Fast Mode pass ─────────────────", tone: "label" },
  { kind: "out", text: "  drop  vendor/react/cjs/*           −3,120 tok", tone: "dim" },
  { kind: "out", text: "  drop  node_modules/lodash/*        −4,080 tok", tone: "dim" },
  { kind: "out", text: "  drop  .next/cache + webpack noise  −1,400 tok", tone: "dim" },
  { kind: "out", text: "  drop  DEBUG/INFO log spam          −1,100 tok", tone: "dim" },
  { kind: "out", text: "  keep  src/auth/session.ts", tone: "ok" },
  { kind: "out", text: "  keep  stack frames + task files", tone: "ok" },
  { kind: "out", text: "  skeleton  heavy imports → stubs", tone: "ok" },
  { kind: "out", text: "→ forward  2,140 tokens to model", tone: "saved" },
  { kind: "out", text: "← reply    streamed (unchanged quality)", tone: "ok" },
  { kind: "blank" },
  { kind: "in", text: "trim status", delay: 380 },
  { kind: "out", text: "trim status · last request + today", tone: "label" },
  { kind: "out", text: "  tokens    12,840 → 2,140            83% smaller", tone: "saved" },
  { kind: "out", text: "  usd       $0.0384 → $0.0064         $0.032 saved", tone: "saved" },
  { kind: "out", text: "  latency   +12ms trim overhead", tone: "dim" },
  { kind: "out", text: "  mode      Fast Mode · local", tone: "dim" },
  { kind: "out", text: "  today     1.24M tok trimmed · ≈ $4.80", tone: "ok" },
  { kind: "out", text: "  dashboard https://use-trim.com/dashboard", tone: "dim" },
];

export const CLI_SCRIPT_POWERSHELL: CliScriptLine[] = [
  { kind: "in", text: "irm https://use-trim.com/install.ps1 | iex" },
  { kind: "out", text: "→ fetching trim-cli 0.9.2 (win-x64)", tone: "dim" },
  { kind: "out", text: "→ verifying sha256… ok", tone: "dim" },
  { kind: "out", text: "→ installing to %LOCALAPPDATA%\\trim", tone: "dim" },
  { kind: "out", text: "✓ trim 0.9.2 ready", tone: "ok" },
  { kind: "blank" },
  { kind: "in", text: "trim login", delay: 380 },
  { kind: "out", text: "→ open browser · device code TRM-7K2Q", tone: "dim" },
  { kind: "out", text: "✓ signed in as you@team.dev", tone: "ok" },
  { kind: "blank" },
  { kind: "in", text: "trim start", delay: 420 },
  { kind: "out", text: "starting local OpenAI-compatible proxy", tone: "label" },
  { kind: "out", text: "  listen     http://127.0.0.1:8888/v1", tone: "ok" },
  { kind: "out", text: "  mode       Fast Mode (skeleton + drop noise)", tone: "dim" },
  { kind: "out", text: "  deep       off (toggle: trim mode deep)", tone: "dim" },
  {
    kind: "out",
    text: "  privacy    Trim cloud does not receive prompts for compression",
    tone: "dim",
  },
  { kind: "out", text: "  upstream   provider after compress", tone: "dim" },
  {
    kind: "out",
    text: "✓ proxy live - set Cursor or VS Code Base URL to the listen URL",
    tone: "ok",
  },
  { kind: "blank" },
  { kind: "out", text: "← POST /v1/chat/completions   model=gpt-4.1   Composer", tone: "label" },
  { kind: "out", text: "  capture   noisy workspace context", tone: "dim" },
  { kind: "out", text: "  tokens in 12,840", tone: "dim" },
  { kind: "out", text: "  -- Fast Mode pass -----------------", tone: "label" },
  { kind: "out", text: "  drop  vendor/react/cjs/*           -3,120 tok", tone: "dim" },
  { kind: "out", text: "  drop  node_modules/lodash/*        -4,080 tok", tone: "dim" },
  { kind: "out", text: "  drop  .next/cache + webpack noise  -1,400 tok", tone: "dim" },
  { kind: "out", text: "  drop  DEBUG/INFO log spam          -1,100 tok", tone: "dim" },
  { kind: "out", text: "  keep  src/auth/session.ts", tone: "ok" },
  { kind: "out", text: "  keep  stack frames + task files", tone: "ok" },
  { kind: "out", text: "  skeleton  heavy imports → stubs", tone: "ok" },
  { kind: "out", text: "→ forward  2,140 tokens to model", tone: "saved" },
  { kind: "out", text: "← reply    streamed (unchanged quality)", tone: "ok" },
  { kind: "blank" },
  { kind: "in", text: "trim status", delay: 380 },
  { kind: "out", text: "trim status · last request + today", tone: "label" },
  { kind: "out", text: "  tokens    12,840 → 2,140            83% smaller", tone: "saved" },
  { kind: "out", text: "  usd       $0.0384 → $0.0064         $0.032 saved", tone: "saved" },
  { kind: "out", text: "  latency   +12ms trim overhead", tone: "dim" },
  { kind: "out", text: "  mode      Fast Mode · local", tone: "dim" },
  { kind: "out", text: "  today     1.24M tok trimmed · ≈ $4.80", tone: "ok" },
  { kind: "out", text: "  dashboard https://use-trim.com/dashboard", tone: "dim" },
];

/** @deprecated use CLI_SCRIPT_UNIX - kept as alias for hero */
export const CLI_SCRIPT_HERO = CLI_SCRIPT_UNIX.map((line) => {
  if (line.kind === "in") return { k: "in" as const, t: line.text };
  if (line.kind === "blank") return { k: "out" as const, t: "", tone: "dim" as const };
  return { k: "out" as const, t: line.text, tone: line.tone };
}).filter((l) => !(l.k === "out" && l.t === ""));
