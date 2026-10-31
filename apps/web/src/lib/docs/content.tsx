import type { ReactNode } from "react";

export type DocBlock =
  | { type: "p"; text: string }
  | { type: "h2"; id: string; text: string }
  | { type: "h3"; id: string; text: string }
  | { type: "ul"; items: string[] }
  | { type: "ol"; items: string[] }
  | { type: "code"; code: string; language?: string }
  | { type: "table"; headers: string[]; rows: string[][] }
  | { type: "callout"; title?: string; text: string }
  | { type: "note"; text: string };

export type DocPage = {
  title: string;
  description: string;
  headings: { id: string; title: string }[];
  blocks: DocBlock[];
};

function page(title: string, description: string, blocks: DocBlock[]): DocPage {
  const headings = blocks
    .filter((b): b is Extract<DocBlock, { type: "h2" }> => b.type === "h2")
    .map((b) => ({ id: b.id, title: b.text }));
  return { title, description, headings, blocks };
}

export const DOC_PAGES: Record<string, DocPage> = {
  "": page(
    "Introduction",
    "Plain map of Trim: what it is, where settings live, and which guide to open for your tool.",
    [
      {
        type: "p",
        text: "Trim runs a small local helper on your computer. Your coding tool (Cursor, Claude Code, VS Code, …) sends chat through Trim first. Trim removes noisy context on your machine, then sends a smaller prompt to your AI provider. Sign-in, plans, and usage charts live on use-trim.com.",
      },
      {
        type: "callout",
        title: "Read this first ",
        text: "You will use four places: (1) Terminal - install and run Trim. (2) Your coding tool - tell it Trim’s address. (3) Local meter - http://127.0.0.1:8888/dashboard to see savings on this PC. (4) Cloud dashboard - https://use-trim.com/dashboard for login, Preferences, Billing, and API keys. Every docs code block has a Copy button.",
      },
      {
        type: "h2",
        id: "choose-tool",
        text: "Choose your tool (pick one path)",
      },
      {
        type: "table",
        headers: ["I use…", "Open this guide next", "Where you change settings"],
        rows: [
          [
            "Claude Code only (no Cursor / no VS Code)",
            "Claude Code (CLI only)",
            "Terminal env vars (ANTHROPIC_BASE_URL) - not IDE Models",
          ],
          [
            "Cursor (Agent)",
            "Cursor setup + Trim Cursor Link",
            "Override Base URL = trim cursor-link url (https://…/v1), not 127.0.0.1",
          ],
          [
            "Continue (VS Code / Cursor extension)",
            "Quickstart → Continue setup",
            "~/.continue/config.yaml",
          ],
          ["VS Code built-in Chat", "Quickstart → VS Code setup", "Chat: Manage Language Models"],
          ["Not sure yet", "Quickstart", "Shared install + login first, then pick a path"],
        ],
      },
      {
        type: "h2",
        id: "how-it-works",
        text: "How it works end to end (simple)",
      },
      {
        type: "ol",
        items: [
          "Install the Trim CLI on your PC (puts the trim command on PATH)",
          "Sign in once: trim login (browser opens; cloud account for quotas/billing)",
          "Start Trim on this PC: trim start (or trim autostart enable so it comes back after reboot)",
          "Point your coding tool at Trim’s address (different tools use a slightly different address - see the table above)",
          "Chat as usual → check local meter → optional cloud Usage / Billing",
        ],
      },
      {
        type: "h2",
        id: "where-settings",
        text: "Where settings live (map)",
      },
      {
        type: "table",
        headers: ["What", "Where"],
        rows: [
          [
            "Install / start / stop Trim",
            "Terminal: trim version, trim start, trim stop, trim status",
          ],
          [
            "Always-on preference",
            "https://use-trim.com/dashboard → Preferences → Start Trim with your IDE",
          ],
          ["Plans / prices / quotas", "https://use-trim.com/dashboard → Billing"],
          [
            "Trim IDE extension API key (optional)",
            "Dashboard → Settings → API keys (not your Anthropic/OpenAI key)",
          ],
          [
            "Provider AI keys (Anthropic / OpenAI / Google)",
            "Inside your coding tool (or shell env for Claude Code)",
          ],
          ["Local savings meter", "http://127.0.0.1:8888/dashboard"],
        ],
      },
      {
        type: "callout",
        title: "Start here",
        text: "Go to Quickstart now. If you only use Claude Code, jump straight to Claude Code (CLI only) after install + login.",
      },
      { type: "h2", id: "open-core", text: "Open core" },
      {
        type: "p",
        text: "The CLI and much of the backend are open for inspection and self-hosting. Hosted use-trim.com adds managed auth, quotas, fraud controls, and billing.",
      },
    ],
  ),

  quickstart: page(
    "Quickstart",
    "Shared install → pick your tool path → verify. Clear where / what / how for non-technical and technical users.",
    [
      {
        type: "p",
        text: "Do steps 1-3 on every computer. Then open only the Step 4 path that matches your tool. Do not mix Cursor settings with Claude Code settings. Every code block has a Copy button.",
      },
      {
        type: "table",
        headers: ["Step", "What you do", "Where you do it", "What you should see"],
        rows: [
          ["1", "Install Trim", "Terminal", "trim version prints a version number"],
          ["2", "Sign in", "Terminal → browser", "Browser login completes; terminal confirms"],
          [
            "3",
            "Start Trim on this PC",
            "Terminal (+ optional Dashboard Preferences)",
            "trim start ready UI / local dashboard loads",
          ],
          [
            "4",
            "Connect your coding tool",
            "Only your tool’s settings (pick one path below)",
            "Chats go through Trim",
          ],
          ["5", "Verify", "Browser + terminal", "Local Requests increase; optional cloud Usage"],
        ],
      },
      {
        type: "h2",
        id: "before",
        text: "Before you start",
      },
      {
        type: "ul",
        items: [
          "You need a terminal (macOS Terminal, Windows PowerShell, or Linux shell)",
          "You need your AI provider key later (Anthropic for Claude Code / Claude models; OpenAI for GPT; Google for Gemini)",
          "You do not paste Trim HMAC secrets or internal API URLs for normal use",
        ],
      },
      { type: "h2", id: "install", text: "1. Install the CLI (Terminal)" },
      {
        type: "p",
        text: "Where: Terminal. What: put the trim command on your PATH. How: copy one installer, run it, then confirm.",
      },
      {
        type: "code",
        language: "bash",
        code: `# macOS / Linux
curl -fsSL https://use-trim.com/install.sh | sh
trim version

# Windows (PowerShell)
irm https://use-trim.com/install.ps1 | iex
trim version

# npm (any OS with Node)
npm install -g @usetrim/trim
trim version

# Homebrew (macOS)
brew install usetrim/tap/trim
trim version`,
      },
      {
        type: "p",
        text: "After this: if trim version fails, close and reopen the terminal (PATH refresh), then retry. Platform detail: Installation.",
      },
      {
        type: "h2",
        id: "login",
        text: "2. Sign in (Terminal → browser)",
      },
      {
        type: "p",
        text: "Where: Terminal, then the browser window that opens. What: link this PC to your use-trim.com account (quotas, Billing, Usage). How:",
      },
      {
        type: "code",
        language: "bash",
        code: "trim login",
      },
      {
        type: "p",
        text: "After this: keep the provider AI key for Step 4 (Trim login is not your Anthropic/OpenAI/Google key).",
      },
      {
        type: "h2",
        id: "start-trim",
        text: "3. Start Trim on this PC (Terminal)",
      },
      {
        type: "p",
        text: "Where: Terminal. Optional cloud toggle: https://use-trim.com/dashboard → Preferences → Start Trim with your IDE (leave checked). The website cannot start Trim by itself - the CLI does. How:",
      },
      {
        type: "code",
        language: "bash",
        code: `# One-shot: start Trim now (prints ready UI, returns your shell; proxy stays up)
trim start

# Recommended: also survive reboot / logon (no admin on a normal Windows PC)
trim autostart enable
trim autostart status`,
      },
      {
        type: "p",
        text: "After this: open http://127.0.0.1:8888/dashboard in a browser. If it loads, Trim is listening on this PC. Claude Code users: you still need Step 4B (env vars). IDE users: Step 4A/4C/4D.",
      },
      {
        type: "callout",
        title: "Already running?",
        text: "If Trim is already healthy, trim start confirms and exits 0 - it does not start a second copy. Prefer one Trim proxy per machine.",
      },
      {
        type: "h2",
        id: "connect-tool",
        text: "4. Connect your coding tool (pick ONE path)",
      },
      {
        type: "p",
        text: "Do not configure every tool. Open only the page for the tool you actually use.",
      },
      {
        type: "table",
        headers: ["Path", "Who", "Where you set it", "Full guide"],
        rows: [
          [
            "4A Cursor",
            "Cursor Agent / Composer (cloud)",
            "trim cursor-link enable → trim start → paste trim cursor-link url (https://…/v1) into Override OpenAI Base URL - not 127.0.0.1",
            "Cursor setup",
          ],
          [
            "4B Claude Code only",
            "claude CLI / Claude Code (no IDE)",
            "Terminal env ANTHROPIC_BASE_URL = http://127.0.0.1:8888 (no /v1) + ANTHROPIC_API_KEY",
            "Claude Code (CLI only)",
          ],
          [
            "4C Continue",
            "Continue sidebar",
            "~/.continue/config.yaml apiBase + model",
            "Continue setup",
          ],
          [
            "4D VS Code Chat",
            "Built-in VS Code Chat",
            "Chat: Manage Language Models → Custom Endpoint → …/v1/chat/completions",
            "VS Code setup",
          ],
        ],
      },
      {
        type: "h3",
        id: "copy-addresses",
        text: "Copy: Trim addresses (use the one for your path)",
      },
      {
        type: "code",
        language: "text",
        code: `Path 4A - Cursor Agent (REQUIRED) - run: trim cursor-link url
https://YOUR_SUBDOMAIN.trycloudflare.com/cl/YOUR_TOKEN/v1

Path 4C / most on-device OpenAI IDE settings
http://127.0.0.1:8888/v1

Path 4D - VS Code Chat Custom Endpoint only
http://127.0.0.1:8888/v1/chat/completions

Path 4B - Claude Code only (origin, NO /v1)
http://127.0.0.1:8888`,
      },
      {
        type: "callout",
        title: "Claude Code users (important)",
        text: "If you only use Claude Code, skip Cursor / VS Code model menus entirely. Open Claude Code (CLI only) now and finish Step 4B there: Windows / macOS / Linux copy blocks, permanent shell profile, verify, everyday loop, and troubleshooting. Every block has a Copy button.",
      },
      {
        type: "callout",
        title: "IDE custom model names",
        text: "For Cursor / Continue / VS Code you can add friendly ids like trim-claude-sonnet. That is an IDE Models setting - not used by Claude Code’s Anthropic door. Guide: Custom models.",
      },
      { type: "h2", id: "verify", text: "5. Verify (after Step 4)" },
      {
        type: "p",
        text: "Where: Terminal + browser. What: prove traffic hits Trim. How:",
      },
      {
        type: "ol",
        items: [
          "Send one short message in your tool (“Reply with exactly: ok”)",
          "Open http://127.0.0.1:8888/dashboard - Requests should increase",
          "Optional: trim status and trim stats in Terminal",
          "Optional cloud: https://use-trim.com/dashboard → Usage / Traces",
          "Plans / prices: https://use-trim.com/dashboard → Billing",
        ],
      },
      {
        type: "code",
        language: "bash",
        code: `trim status
trim stats`,
      },
      {
        type: "h2",
        id: "after",
        text: "After setup (everyday)",
      },
      {
        type: "ul",
        items: [
          "IDE users: open the editor; with Always-on, Trim stays ready (see Always-on with IDE)",
          "Claude Code users: keep ANTHROPIC_BASE_URL set in that terminal session (or shell profile), run trim start if the local dashboard is down, then run claude as usual",
          "If savings disappear after an editor update: re-check Step 4 address",
          "To leave a machine clean: Uninstall",
        ],
      },
      {
        type: "note",
        text: "Next detail pages: Installation, Claude Code (CLI only), Cursor setup, Continue setup, VS Code setup, Custom models.",
      },
    ],
  ),

  installation: page(
    "Installation",
    "Install Trim on macOS, Linux, or Windows, then finish Always-on + IDE model setup with copyable steps.",
    [
      {
        type: "p",
        text: "Prefer the one-line installers from use-trim.com. They fetch the matching release for your OS and architecture and place trim on your PATH. The same installers work worldwide; they pick the correct OS/CPU asset automatically. Every code block below has a Copy button.",
      },
      {
        type: "table",
        headers: ["Path", "Who it is for", "Command"],
        rows: [
          ["curl installer", "macOS / Linux", "curl -fsSL https://use-trim.com/install.sh | sh"],
          ["PowerShell installer", "Windows", "irm https://use-trim.com/install.ps1 | iex"],
          ["npm", "Any OS with Node", "npm install -g @usetrim/trim"],
          ["Homebrew", "macOS", "brew install usetrim/tap/trim"],
        ],
      },
      { type: "h2", id: "unix", text: "macOS and Linux" },
      {
        type: "code",
        language: "bash",
        code: `curl -fsSL https://use-trim.com/install.sh | sh
trim version`,
      },
      { type: "h2", id: "windows", text: "Windows" },
      {
        type: "code",
        language: "powershell",
        code: `irm https://use-trim.com/install.ps1 | iex
trim version`,
      },
      {
        type: "p",
        text: "On Windows, the installer also registers Trim under Settings → Apps when cloud chrome is reachable, so you can remove Trim later from Add or remove programs (same full purge as trim uninstall).",
      },
      { type: "h2", id: "packages", text: "Package managers" },
      {
        type: "code",
        language: "bash",
        code: `# npm (downloads the official Release binary + SHA-256 verify)
npm install -g @usetrim/trim

# Homebrew
brew install usetrim/tap/trim

# Optional later: Scoop / winget
scoop bucket add trim https://github.com/usetrim/scoop-trim
scoop install trim
winget install Trim.CLI`,
      },
      {
        type: "p",
        text: "npm (@usetrim/trim) and Homebrew (usetrim/tap/trim) are live. curl/irm installers remain supported. Scoop/winget are optional. Maintainer ops: packaging/README.md.",
      },
      { type: "h2", id: "after-install", text: "After install: Always-on (required)" },
      {
        type: "p",
        text: "Official installs need zero environment variables for Trim Cloud. Do these in order. Exact dashboard place for the preference: https://use-trim.com/dashboard → Preferences → Start Trim with your IDE (leave checked).",
      },
      {
        type: "ol",
        items: [
          "Confirm install: trim version",
          "Sign in: trim login",
          "Enable Always-on: trim autostart enable",
          "Confirm: trim autostart status",
          "Optional smoke: trim start (ready UI, then your shell returns; proxy stays listening)",
          "Sync adapters + model aliases: trim config sync",
        ],
      },
      {
        type: "code",
        language: "bash",
        code: `trim version
trim login
trim autostart enable
trim autostart status
trim config sync

# Optional advanced OS logon check
trim daemon status

# One-shot manual session (any time)
trim start`,
      },
      {
        type: "callout",
        title: "Windows: no Administrator",
        text: "trim autostart enable registers a per-user Startup Run key (HKCU) pointing at a stable copy under %LOCALAPPDATA%\\Trim\\bin\\trim.exe - the same industry pattern as browsers and editors (never locks npm’s vendor\\trim.exe). No admin Scheduled Task. If OS registration is blocked by policy, your preference stays on - use trim start or the Trim IDE extension. See trim autostart.",
      },
      {
        type: "code",
        language: "powershell",
        code: `# Confirm Windows Startup Run → LocalAppData after enable (optional)
Get-ItemProperty HKCU:\\Software\\Microsoft\\Windows\\CurrentVersion\\Run |
  Select-Object TrimProxy
Test-Path \"$env:LOCALAPPDATA\\Trim\\bin\\trim.exe\"`,
      },
      {
        type: "h2",
        id: "ide-and-models",
        text: "After install: connect your tool (pick ONE)",
      },
      {
        type: "p",
        text: "Installing the CLI alone does not compress chat. Next: trim login → trim start → connect exactly one tool. Open Quickstart Step 4, or jump straight to your guide below.",
      },
      {
        type: "table",
        headers: ["I use…", "Where I set Trim’s address", "Full guide"],
        rows: [
          [
            "Claude Code only (no IDE)",
            "Terminal env ANTHROPIC_BASE_URL = http://127.0.0.1:8888 (no /v1)",
            "Claude Code (CLI only)",
          ],
          [
            "Cursor Agent",
            "Override OpenAI Base URL = trim cursor-link url (https://…/v1), not 127.0.0.1",
            "Cursor setup",
          ],
          ["Continue", "~/.continue/config.yaml → apiBase", "Continue setup"],
          ["VS Code Chat", "Custom Endpoint url = …/v1/chat/completions", "VS Code setup"],
          ["Friendly IDE model names", "Same IDE model settings", "Custom models"],
        ],
      },
      {
        type: "code",
        language: "text",
        code: `Cursor Agent - run: trim cursor-link url
https://YOUR_SUBDOMAIN.trycloudflare.com/cl/YOUR_TOKEN/v1

Continue / most on-device OpenAI IDE settings
http://127.0.0.1:8888/v1

VS Code Chat Custom Endpoint only
http://127.0.0.1:8888/v1/chat/completions

Claude Code only (origin - NO /v1)
http://127.0.0.1:8888`,
      },
      {
        type: "p",
        text: "After connecting: open http://127.0.0.1:8888/dashboard and send one short chat. Plans / prices: https://use-trim.com/dashboard → Billing. Trim IDE extension keys (optional): Dashboard → Settings → API keys. Provider keys stay in your tool (or Claude Code shell env).",
      },
      { type: "h2", id: "from-source", text: "From source (Go)" },
      {
        type: "code",
        language: "bash",
        code: `git clone https://github.com/usetrim/trim.git
cd trim/cli
go install ./cmd/trim
# or: go run ./cmd/trim start`,
      },
      {
        type: "p",
        text: "Remote go install …@vX.Y.Z is not supported while cli/go.mod uses a local replace for the server module. Clone the repo (above) or use Release / npm / Homebrew installers.",
      },
      {
        type: "p",
        text: "Building from source is useful when contributing. For day-to-day use, install the release binary.",
      },
      { type: "h2", id: "remove", text: "Remove Trim" },
      {
        type: "p",
        text: "To fully remove Trim, local data, and Deep Mode caches from a machine, see Uninstall. Prefer trim uninstall (or Windows Settings → Apps); it also schedules npm / Scoop / Homebrew / winget package removal when that install channel is detected.",
      },
      { type: "h2", id: "updates", text: "Keeping Trim updated" },
      {
        type: "p",
        text: "Hosted Trim may require a recent CLI for security and compatibility. If the API rejects an old binary, re-run the installer to upgrade.",
      },
    ],
  ),

  uninstall: page(
    "Uninstall",
    "Fully remove Trim from a machine worldwide: one command (or Windows Apps & features) clears proxy, other Trim processes, daemon, credentials, setup overrides, local data, Deep Mode caches, the CLI binary, and detected npm / Scoop / Homebrew / winget installs.",
    [
      {
        type: "p",
        text: "Trim is designed for a professional one-shot leave: stop using Trim, or wipe the machine clean. Runtime labels and Deep Mode purge lists come from Trim cloud site_messages (fail-closed). Full purge is the default.",
      },
      {
        type: "callout",
        title: "Recommended",
        text: "On any OS: run trim uninstall. On Windows you can also use Settings → Apps → Trim. Both run the same full purge, including %USERPROFILE%\\.trim (or ~/.trim) and Deep Mode model caches listed in site_messages. On Windows, the running binary is renamed and package removal is scheduled after exit so the file lock cannot leave trim on PATH.",
      },
      { type: "h2", id: "one-shot", text: "One-shot full uninstall" },
      {
        type: "code",
        language: "bash",
        code: "trim uninstall",
      },
      {
        type: "p",
        text: "That single command stops the local proxy and other Trim processes, removes the OS login daemon, clears CLI credentials, reverts IDE Base URL overrides written by trim setup, deletes local config/data directories from site_messages, cleans Deep Mode Hugging Face hub dirs and pip packages from site_messages, removes Deep sidecars next to the binary, deletes or schedules removal of the CLI binary and Windows user PATH entry, removes the Windows Apps & features registration, and when the install path is npm / Scoop / Homebrew / winget schedules that package uninstall after exit.",
      },
      {
        type: "code",
        language: "bash",
        code: `# Keep local data, Deep caches, CLI binary, and package install
# (stop / unregister Always-on / logout / setup revert only)
trim uninstall --keep-data`,
      },
      { type: "h2", id: "helpers", text: "Helper scripts" },
      {
        type: "code",
        language: "bash",
        code: `# macOS / Linux
curl -fsSL https://use-trim.com/uninstall.sh | sh

# Windows (PowerShell)
irm https://use-trim.com/uninstall.ps1 | iex`,
      },
      {
        type: "p",
        text: "Helpers only invoke trim uninstall when the binary is on PATH (fail-closed; no invent path lists in the scripts).",
      },
      { type: "h2", id: "windows", text: "Windows Apps & features" },
      {
        type: "ol",
        items: [
          "Open Settings → Apps → Installed apps (or Add or remove programs)",
          "Find Trim",
          "Choose Uninstall",
        ],
      },
      {
        type: "p",
        text: "Windows runs the same UninstallString as trim uninstall, so user data and Deep Mode weights are swept with the product - not left behind after Remove Programs.",
      },
      { type: "h2", id: "packages", text: "Package managers" },
      {
        type: "code",
        language: "bash",
        code: `trim uninstall
# If a detected channel was not auto-removed after exit:
# brew uninstall trim
# scoop uninstall trim
# winget uninstall Trim.CLI
# npm uninstall -g @usetrim/trim`,
      },
      {
        type: "p",
        text: "trim uninstall detects npm / Scoop / Homebrew / winget from the running binary path and by probing each package manager, then schedules that package removal after exit (required on Windows while the CLI file is locked). The CLI also prints the matching fallback for each detected channel in case the post-exit helper or package manager fails.",
      },
      { type: "h2", id: "extension", text: "IDE extension" },
      {
        type: "ol",
        items: [
          "Command Palette → Trim: Clear API Key (clears Secret Storage, pending counters, and cached IDE chrome)",
          "Command Palette → Extensions → uninstall Trim IDE",
        ],
      },
      {
        type: "note",
        text: "Editors do not allow a third-party CLI to force-uninstall extensions. Clear API Key plus Extensions UI is the supported host path.",
      },
      { type: "h2", id: "cloud", text: "Cloud account" },
      {
        type: "p",
        text: "On the home Dashboard (/dashboard), use Delete account (confirm dialog) to remove hosted account data. It does not delete local CLI files. Run trim uninstall on each machine you used. Account delete is not under Settings.",
      },
      { type: "h2", id: "checklist", text: "What a full uninstall covers" },
      {
        type: "ul",
        items: [
          "Local proxy stop",
          "OS login daemon removal",
          "CLI API credentials",
          "IDE Base URL overrides from trim setup",
          "Local data dirs from site_messages (typically ~/.trim and ~/.config/trim, including metrics)",
          "macOS Trim log files from site_messages when present",
          "Deep Mode Hugging Face hub dirs, pip packages, and sidecars from site_messages",
          "CLI binary, Windows user PATH / Apps entry, %LOCALAPPDATA%\\Trim, and detected npm / Scoop / Homebrew / winget installs (scheduled after exit on Windows)",
        ],
      },
      {
        type: "p",
        text: "After uninstall, only optional leftover items are the IDE extension (remove in the editor) and a cloud account (delete in the dashboard if you want hosted data gone too).",
      },
    ],
  ),

  "concepts/overview": page(
    "How Trim works",
    "End-to-end path from IDE request to upstream model: local compression, then upstream network; Trim cloud for auth, quotas, and billing.",
    [
      {
        type: "p",
        text: "Every request that targets the Trim listen URL hits the local proxy first. Trim runs Fast Mode every turn, and when Preferences Deep is on also runs Segmented Allowlist Deep (freeze tools/paths/schemas; compress each last-user prose island in place, order preserved beside paths; validate or fail-closed Fast), then forwards upstream. One Base URL serves two client dialects: OpenAI /v1/chat/completions and Anthropic /v1/messages. On the OpenAI door, DB-driven provider adapters route by model prefix: anthropic_messages for Claude, openai_compat for GPT/Gemini/DeepSeek/Mistral hosts. No OpenRouter. No daily UPSTREAM_OPENAI_URL flipping.",
      },
      { type: "h2", id: "pipeline", text: "Request pipeline" },
      {
        type: "ol",
        items: [
          "On-device IDE or Claude Code POSTs to the local Trim URL from trim start (example http://127.0.0.1:8888/v1/chat/completions or /v1/messages). Cursor Agent POSTs to trim cursor-link url (HTTPS …/cl/<token>/v1)",
          "Trim captures workspace context and classifies vendor vs product signal",
          "Fast Mode drops vendor dumps, lockfiles, and log spam; keeps active files and stacks",
          "When Preferences Deep is on, Segmented Allowlist Deep runs after Fast: freeze tools/paths/code/JSON; compress each last-user prose island in place (order preserved beside paths); validate or fail-closed Fast. Engine v2 uses LLMLingua-2 with the synced Base Model (catalog id such as mbert-base or xlm-roberta-large from trim config sync)",
          "If the OpenAI-door model matches an enabled provider_adapters prefix, Trim applies that dialect (anthropic_messages translate, or openai_compat host route via upstream_base_url), then calls that upstream",
          "Otherwise, if provider adapters are synced and no prefix matches: clear Trim error (no silent host invent). If adapters are not synced: same-shape forward to UPSTREAM_OPENAI_URL (OpenAI door) or UPSTREAM_ANTHROPIC_URL (Anthropic door)",
          "When logged in, usage is metered against your plan quotas",
        ],
      },
      { type: "h2", id: "what-stays-local", text: "What stays local" },
      {
        type: "p",
        text: "Prompt compression for the live proxy runs on your machine. The slim prompt still goes to your upstream model over the network. Trim cloud stores account, billing, and metering metadata. Deep Mode engines are not loaded in Trim cloud.",
      },
      {
        type: "callout",
        title: "Design goal",
        text: "Same task quality with far fewer input tokens. Dashboard shows tokens in to out, percent cut, and USD direction so the win is measurable.",
      },
      {
        type: "h2",
        id: "always-on-link",
        text: "Always-on with IDE",
      },
      {
        type: "p",
        text: "Everyday product feel is one synced preference (Start Trim with your IDE), enforced locally by the IDE extension and/or CLI daemon. The dashboard owns the switch and local-agent status; it does not spawn Trim in the browser. See Always-on with IDE.",
      },
    ],
  ),

  "concepts/always-on": page(
    "Always-on with IDE",
    "Keep Trim running on this PC when you open your editor - or when you use Claude Code from the terminal.",
    [
      {
        type: "p",
        text: "Always-on means: Trim should already be listening on this computer when you start working. You should not need to remember a new ritual every morning. Live chat always runs Fast; Deep also runs when Preferences Deep is on.",
      },
      {
        type: "table",
        headers: ["I use…", "How Always-on helps", "What you still set"],
        rows: [
          [
            "Cursor / VS Code",
            "Trim IDE extension and/or trim autostart start the proxy when the editor opens",
            "Correct Base URL for your tool + model (Cursor Agent: Cursor Link; others: Quickstart 4A/4C/4D)",
          ],
          [
            "Claude Code only",
            "trim autostart (or trim start) keeps the proxy up - no IDE extension required",
            "Terminal ANTHROPIC_BASE_URL + ANTHROPIC_API_KEY (see Claude Code CLI only)",
          ],
        ],
      },
      { type: "h2", id: "one-preference", text: "One preference (exact place)" },
      {
        type: "p",
        text: "Where: https://use-trim.com/dashboard → Preferences → Start Trim with your IDE. Default is on. Uncheck anytime to stop auto-starting. The website cannot start Trim by itself - your PC’s CLI or Trim IDE extension does.",
      },
      {
        type: "ul",
        items: [
          "Dashboard: checkbox + local agent online/offline",
          "CLI: trim autostart enable|disable|status (works for IDE and Claude Code users)",
          "IDE extension: on editor startup, if preference is on, ensure the proxy is healthy",
        ],
      },
      {
        type: "callout",
        title: "Hard constraint",
        text: "The web dashboard cannot start a process on your laptop by itself. Cloud only stores the boolean and syncs it down. CLI, daemon, or the IDE extension apply it on the machine.",
      },
      { type: "h2", id: "os-logon", text: "OS logon (optional)" },
      {
        type: "p",
        text: "trim autostart enable also best-effort registers a per-user logon hook so Trim can start with your OS session. This is the same industry pattern used by editors and browsers - never a machine-wide admin Scheduled Task on Windows.",
      },
      {
        type: "ul",
        items: [
          "macOS: launchd LaunchAgent",
          "Linux: systemd --user",
          "Windows: HKCU Startup Run → %LOCALAPPDATA%\\Trim\\bin\\trim.exe (no Administrator on a normal PC)",
        ],
      },
      {
        type: "p",
        text: "If logon registration fails, your preference stays on. The CLI shows a yellow warning, muted guidance, and a next-step table (trim start / keep IDE Always-on). Use trim -v only when you need raw OS detail.",
      },
      { type: "h2", id: "daily", text: "Daily flow" },
      {
        type: "ol",
        items: [
          "Install the CLI (and optionally the Trim IDE extension)",
          "Sign in when you need quotas and the dashboard",
          "Point on-device IDEs at the Local Base URL trim start printed (example http://127.0.0.1:8888/v1). Cursor Agent: trim cursor-link url (HTTPS …/v1; not 127.0.0.1)",
          "Leave Start Trim with your IDE checked (or run trim autostart enable)",
          "Open the IDE - Trim should already be trimming (Fast; Deep if Preferences Deep is on)",
          "Optional: trim compress / Deep when you want file or batch compression",
          "Anytime: uncheck the preference - extension and daemon honor it and stop auto-starting",
        ],
      },
      { type: "h2", id: "attach", text: "Two IDEs, one port" },
      {
        type: "p",
        text: "If a healthy Trim proxy already owns the listen port, a second IDE or trim start attaches instead of binding again. That avoids double-bind races when Cursor and VS Code are both open.",
      },
      { type: "h2", id: "opt-out", text: "Opt-out and managed off" },
      {
        type: "ul",
        items: [
          "Dashboard checkbox (one click)",
          "trim autostart disable",
          "Extension setting trim.autoStartWithIde = off",
          "Enterprise-style gates: DO_NOT_TRACK=1, TRIM_AUTOSTART_DISABLED=1, or ~/.config/trim/autostart.off",
        ],
      },
      {
        type: "note",
        text: "Always-on starts the local proxy with your IDE. Deep Mode still only runs when Preferences set Deep on (Fast then Segmented Allowlist Deep per request). After changing compression rate or Base model, run trim config sync on each machine. Everyday loop = open IDE (or trim start).",
      },
    ],
  ),

  "concepts/fast-vs-deep": page(
    "Fast Mode and Deep Mode",
    "Fast Mode always runs on the live proxy. Deep Mode is on-machine LLMLingua (live proxy when prefs say deep, plus trim compress).",
    [
      { type: "h2", id: "fast", text: "Fast Mode" },
      {
        type: "p",
        text: "Fast Mode powers the live proxy on every turn. When Dashboard preferences set Deep Mode on (compression_tier=deep), trim start also runs Segmented Allowlist Deep after Fast: freeze tools, system, paths, code fences, URLs, and dense JSON; compress each last-user prose island in place (order preserved beside paths); validate or fail-closed Fast.",
      },
      {
        type: "ul",
        items: [
          "Drops vendor and node_modules noise",
          "Collapses DEBUG/INFO spam while keeping ERROR stacks",
          "Protects active files and task-relevant paths",
          "Reduces heavy imports when configured",
        ],
      },
      { type: "h2", id: "deep", text: "Deep Mode" },
      {
        type: "p",
        text: "Deep Mode runs stronger compression on your machine (LLMLingua). With Deep Mode on in Preferences, the live IDE proxy runs Segmented Allowlist Deep after Fast: freeze agent control, tools, paths, code fences, URLs, and dense JSON; compress each last-user prose island in place (order preserved beside paths); validate or fail-closed Fast. New accounts default to engine v2 (LLMLingua-2); choose Base model in Settings (see Base Model). trim compress --deep remains available for file/batch jobs. First run may download local dependencies (mbert-base is smaller; xlm-roberta-large is larger). Trim cloud never loads Deep Mode engines.",
      },
      { type: "h2", id: "compression-rate", text: "Compression rate" },
      {
        type: "p",
        text: "Deep uses a keep-rate (tokens after / tokens before), the same knob as Microsoft LLMLingua-2 on Hugging Face. Default is 0.7. Drag the Dashboard slider within billing min/max by billing step (usually 0.3 to 1.0 step 0.1). Lower keeps less context; 1.0 keeps almost everything. Trim floors at billing min (typically 0.3) so a demo-style 0.1 wipe is not offered by default.",
      },
      {
        type: "ol",
        items: [
          "Open Dashboard → Settings",
          "Confirm Deep Mode is on (new accounts start on Deep)",
          "Drag Compression rate left for more savings (for example 0.5) or right toward 1.0 to keep more text; or type the value next to the slider",
          "If engine is v2, confirm Base model (Dashboard or trim config models)",
          "Save",
          "On each machine: trim config sync, then trim start again",
          "Optional CLI check: trim config rates (shows billing default and selected), trim config get compression-rate",
        ],
      },
      {
        type: "code",
        language: "bash",
        code: `trim config rates
trim config set compression-rate 0.7
trim config get compression-rate
trim compress docs.txt --deep --rate 0.7`,
      },
      { type: "h2", id: "base-model", text: "Base Model (LLMLingua-2)" },
      {
        type: "p",
        text: "When Deep engine is v2, Trim uses the Microsoft LLMLingua-2 Base Model catalog from Admin billing (same idea as the Hugging Face Space Base Model radio: mbert-base and xlm-roberta-large). Catalog rows live in billing deep_v2_backbone_options as {id,label,model}; billing default_deep_v2_backbone stores the default catalog id (seeded mbert-base, matching the Space). xlm-roberta-large is larger, uses more RAM, and is slower. Pick in Dashboard → Settings or with trim config models. Live Deep and trim compress --deep --engine v2 use the synced Base Model (resolved Hugging Face id from the catalog), not a free-text model box.",
      },
      {
        type: "ol",
        items: [
          "Open Dashboard → Settings",
          "Confirm Deep Mode is on and engine is LLMLingua-2 (v2)",
          "Choose Base model from the Admin catalog (ids such as mbert-base or xlm-roberta-large when enabled)",
          "Save",
          "On each machine: trim config sync, then trim start again",
          "Optional CLI: trim config models, trim config set deep-v2-backbone <id>, trim config get deep-v2-backbone",
        ],
      },
      {
        type: "code",
        language: "bash",
        code: `trim config models
trim config set deep-v2-backbone mbert-base
trim config set deep-v2-backbone xlm-roberta-large
trim config get deep-v2-backbone
trim compress docs.txt --deep --engine v2 --rate 0.7`,
      },
      { type: "h2", id: "engines", text: "Deep engines (pick one)" },
      {
        type: "ul",
        items: [
          "LLMLingua-2 (v2) · recommended - default selected for new accounts; usually faster and lighter for most Deep jobs; Base Model picker applies here",
          "LongLLMLingua (long) - question-aware for long documents; requires --question",
          "LLMLingua (v1) - classic engine for strong general compression",
        ],
      },
      {
        type: "code",
        language: "bash",
        code: `trim compress path/to/file.go --mode fast
trim compress docs.txt --deep --engine v2 --rate 0.7
trim compress docs.txt --deep --engine long --question "Where is auth handled?" --rate 0.7
trim compress --bootstrap`,
      },
      {
        type: "note",
        text: "New accounts default to Deep Mode (live proxy runs Fast then Segmented Allowlist Deep). Switch to Fast anytime in Dashboard → Settings or with trim config set default-tier fast. Deep engine defaults to LLMLingua-2 (v2) with Base Model mbert-base. Run trim config sync on each machine after saving. List keep-rates with trim config rates; list Base Models with trim config models.",
      },
      { type: "h2", id: "history-window", text: "Optional history window" },
      {
        type: "p",
        text: "For very long chats, optional history_keep_turns can stub older eligible text turns while protecting tools, files, and agent chrome. Default is off. See history_keep_turns.",
      },
      { type: "h2", id: "reading-savings", text: "Reading Saved % on the local meter" },
      {
        type: "p",
        text: "On http://127.0.0.1:8888/dashboard, whole-request Saved % can look low (even 0%) on Claude Code or Cursor agent turns. That is often normal: system reminders, tools, git status, and other agent chrome stay frozen so the loop stays safe. Live Deep uses Segmented Allowlist Deep: compress each last-user prose island in place (order preserved beside paths); paths, code fences, URLs, and dense JSON stay verbatim. Open Show savings detail to see Deep status and Deep stage tokens (prose islands only).",
      },
      {
        type: "ul",
        items: [
          "Deep applied: live Deep rewrote eligible last-user prose at your compression rate (paths/tools/schemas frozen)",
          "Deep kept Fast (expansion fail-closed): Deep would have grown or mangled the payload, so Trim kept Fast",
          "Deep stage saved: savings on prose islands only, separate from whole-body %",
        ],
      },
    ],
  ),

  "concepts/privacy": page(
    "Privacy model",
    "How Trim separates local prompt compression from a thin cloud control plane.",
    [
      {
        type: "p",
        text: "Compression runs on your machine. Trim cloud handles identity, quotas, fraud controls, and billing, not your raw context pack. Upstream model providers still receive the slim prompt you forward.",
      },
      { type: "h2", id: "local", text: "Local processing" },
      {
        type: "ul",
        items: [
          "Fast Mode compresses prompts locally in trim start before upstream forward",
          "Heavy context stays on the developer machine while trimming",
          "Deep Mode artifacts stay on the machine that runs Deep (live proxy or trim compress)",
          "Optional telemetry can be disabled in the product or via supported client settings",
        ],
      },
      { type: "h2", id: "cloud", text: "Cloud data" },
      {
        type: "ul",
        items: [
          "Account profile from social login",
          "Usage events, receipts, credits, plan state",
          "Device-bound key metadata and limited security signals used to protect accounts",
        ],
      },
      { type: "h2", id: "upstream", text: "What still leaves the machine" },
      {
        type: "p",
        text: "After local compression, the smaller prompt goes to the model provider you configured. That is intentional: Trim shrinks noise; it does not replace your LLM.",
      },
      {
        type: "p",
        text: "Read the full Privacy Policy for rights, retention, and how we share information with processors.",
      },
      { type: "h2", id: "wipe", text: "Wiping a machine" },
      {
        type: "p",
        text: "Local data and Deep Mode caches are removed with trim uninstall (or Windows Settings → Apps). Hosted account deletion is separate on the home Dashboard (/dashboard) confirm dialog, not under Settings. See Uninstall.",
      },
    ],
  ),

  "concepts/metering": page(
    "Metering and quotas",
    "How tokens, credits, and plan ranks gate the authenticated CLI and API.",
    [
      {
        type: "p",
        text: "When you are logged in, Trim meters usage against your workspace plan. Exhausted quotas return payment-required style errors until you upgrade or buy top-ups.",
      },
      { type: "h2", id: "ranks", text: "Plan ranks" },
      {
        type: "p",
        text: "Plans are ordered by rank (for example free, pro, team, and enterprise). Self-serve downgrades can be blocked while a paid period is unexpired. Upgrades may prorate through our payment partner.",
      },
      { type: "h2", id: "credits", text: "Cloud credits" },
      {
        type: "p",
        text: "One cloud credit is one metered proxy request against your personal pool or a shared workspace pool when you send X-Workspace-Id. Monthly grants come from plan_catalog.credits_monthly; top-up packs add purchased capacity without changing your tier.",
      },
      { type: "h2", id: "unlimited", text: "Unlimited metering" },
      {
        type: "p",
        text: "Operators can enable Unlimited metering on any plan in the admin catalog (including Free). When your active plan has that flag on, cloud requests are not debited, the dashboard and CLI show the Unlimited label from the Service, and pricing cards show an Unlimited cloud metering badge only for plans where the flag is on. When the flag is off, normal credit metering applies. This is an operator growth dial, not a permanent promise that every plan is free forever.",
      },
      { type: "h2", id: "topups", text: "Top-ups" },
      {
        type: "p",
        text: "Credit packs are always a new checkout and do not change your tier. They extend capacity inside the current plan.",
      },
      {
        type: "note",
        text: "Clearing local CLI state does not reset server-side quotas.",
      },
    ],
  ),

  "cli/overview": page(
    "CLI overview",
    "What the trim command does, and which CLI guide to open next (including Claude Code-only).",
    [
      {
        type: "p",
        text: "The Trim CLI is the program you install on your computer. It starts the local Trim helper, signs you in, and prints the addresses your coding tools need. IDEs and Claude Code both depend on this CLI - they are not separate Trim installs.",
      },
      {
        type: "table",
        headers: ["I want to…", "Open"],
        rows: [
          ["Finish first-time setup", "Quickstart"],
          ["Use Claude Code only (no Cursor / VS Code)", "Claude Code (CLI only)"],
          ["Start / stop the local helper", "trim start"],
          ["Cursor Agent public HTTPS link", "Trim Cursor Link"],
          ["Survive reboot", "trim autostart"],
          [
            "Sync cloud Fast/Deep prefs (rates + v2 Base Models)",
            "trim config sync / rates / models",
          ],
          ["Sign in again", "trim login"],
          ["Fully remove Trim", "Uninstall"],
        ],
      },
      {
        type: "code",
        language: "bash",
        code: `trim help
trim version
trim login
trim start
trim setup
trim autostart enable
trim cursor-link enable
trim cursor-link url
trim status
trim stats
trim config sync
trim stop
trim uninstall`,
      },
      {
        type: "callout",
        title: "trim setup (helpful for Claude Code)",
        text: "trim setup prints copy/paste blocks for Claude Code (ANTHROPIC_BASE_URL origin, no /v1), Continue, and VS Code Chat. It can also write common IDE settings files when those products are installed. Full Claude Code walkthrough: Claude Code (CLI only).",
      },
      { type: "h2", id: "roles", text: "What the CLI does" },
      {
        type: "ul",
        items: [
          "Installs and runs the local Trim listen address on this PC",
          "Signs in to use-trim.com for quotas and Billing",
          "Optional OS logon via trim autostart",
          "Fast Mode (and Deep when Preferences Deep is on) on your machine",
          "trim setup copy/paste for Claude Code and IDEs",
          "Full remove via trim uninstall",
        ],
      },
      { type: "h2", id: "typical-day", text: "Typical day" },
      {
        type: "ol",
        items: [
          "If local dashboard is down: trim start (or rely on trim autostart)",
          "IDE path: open Cursor / VS Code / Continue with Base URL already set",
          "Claude Code path: same terminal has ANTHROPIC_BASE_URL + ANTHROPIC_API_KEY, then run claude",
          "Check http://127.0.0.1:8888/dashboard or trim stats after big tasks",
          "Plans / prices: https://use-trim.com/dashboard → Billing",
        ],
      },
      {
        type: "callout",
        title: "Hosted auth vs on-machine compression",
        text: "Compression runs on your machine; chat still needs an upstream model key. Sign in for Trim cloud auth, quotas, usage sync, and paid entitlements.",
      },
    ],
  ),

  "cli/claude-code": page(
    "Claude Code (CLI only)",
    "World-wide Claude Code / claude CLI path (Windows, macOS, Linux). No Cursor. No VS Code. Where / what / how / why / what happens / what next - every block has a Copy button.",
    [
      {
        type: "p",
        text: "This guide is only for people who use Anthropic’s Claude Code (the claude command in a terminal). You never open Cursor Settings → Models. You never open VS Code Chat. You never paste a …/v1 Base URL into an IDE. Full path: install Claude Code → get Anthropic API key → install Trim → trim login → trim start → set ANTHROPIC_BASE_URL + ANTHROPIC_API_KEY (shell and/or ~/.claude/settings.json) → run claude → verify on the local meter. Every code block below has a Copy icon - use it, then paste into your terminal.",
      },
      {
        type: "callout",
        title: "Who should leave this page",
        text: "If you use Cursor Composer, Continue, or VS Code Chat, leave now and open Cursor setup, Continue setup, or VS Code setup. Those tools use Trim’s OpenAI door (…/v1 or …/v1/chat/completions). Claude Code uses Trim’s Anthropic door (listen origin only, no /v1).",
      },
      {
        type: "h2",
        id: "what-happens",
        text: "What happens when it works (plain English)",
      },
      {
        type: "ol",
        items: [
          "You type in Claude Code (claude) as usual",
          "Claude Code reads ANTHROPIC_BASE_URL and sends the request to Trim on this PC (http://127.0.0.1:8888) instead of straight to api.anthropic.com",
          "Claude Code itself appends /v1/messages - so the real URL becomes http://127.0.0.1:8888/v1/messages",
          "Trim Fast Mode (and Deep when Preferences Deep is on) shrinks noisy context on your machine",
          "Trim forwards the smaller request to Anthropic using your ANTHROPIC_API_KEY",
          "The reply comes back through Trim → Claude Code → your terminal",
          "The local meter at http://127.0.0.1:8888/dashboard counts the request and shows savings",
        ],
      },
      {
        type: "code",
        language: "text",
        code: `You (terminal: claude)
        │
        │  ANTHROPIC_BASE_URL = http://127.0.0.1:8888   ← origin only, NO /v1
        │  ANTHROPIC_API_KEY  = sk-ant-…
        ▼
Trim on this PC  (http://127.0.0.1:8888)
  • receives POST /v1/messages   (Claude Code appended /v1/messages)
  • Fast Mode (+ optional Segmented Allowlist Deep: each prose island in place, order preserved) on your machine
  • tools/paths/schemas stay frozen; check Deep stage on the local meter
  • local meter: http://127.0.0.1:8888/dashboard
        │
        ▼
Anthropic API  (api.anthropic.com)
        │
        ▼
Reply back → Trim → Claude Code → your terminal`,
      },
      {
        type: "h2",
        id: "why-no-v1",
        text: "Why the address has NO /v1 (critical)",
      },
      {
        type: "p",
        text: "Where this matters: the value of ANTHROPIC_BASE_URL only. What Claude Code does: it always appends /v1/messages itself. Why Trim’s value must be the origin: if you set http://127.0.0.1:8888/v1, the real path becomes /v1/v1/messages and Claude Code often shows a confusing “model may not exist” error. Correct value (copy this exactly):",
      },
      {
        type: "code",
        language: "text",
        code: "http://127.0.0.1:8888",
      },
      {
        type: "table",
        headers: ["Value you set", "What Claude Code actually calls", "Result"],
        rows: [
          [
            "http://127.0.0.1:8888  (correct)",
            "http://127.0.0.1:8888/v1/messages",
            "Works through Trim",
          ],
          [
            "http://127.0.0.1:8888/v1  (wrong)",
            "http://127.0.0.1:8888/v1/v1/messages",
            "Broken / “model may not exist”",
          ],
          [
            "https://api.anthropic.com  (default, bypasses Trim)",
            "https://api.anthropic.com/v1/messages",
            "Works, but no Trim savings",
          ],
        ],
      },
      {
        type: "h2",
        id: "map",
        text: "Where everything lives (map)",
      },
      {
        type: "table",
        headers: ["What", "Where you do it", "Example / value", "Why"],
        rows: [
          [
            "Claude Code app",
            "Terminal: Anthropic installer (Step 0)",
            "claude --version",
            "You need the claude command first",
          ],
          [
            "Anthropic API key",
            "Browser: console.anthropic.com (Step 0b)",
            "sk-ant-…",
            "Your provider secret (not Trim login)",
          ],
          [
            "Install Trim",
            "Terminal (PowerShell / Terminal.app / Linux / WSL)",
            "curl … | sh  or  irm … | iex",
            "Puts the trim command on PATH",
          ],
          [
            "Trim Cloud account",
            "Terminal: trim login → browser",
            "use-trim.com sign-in",
            "Quotas, billing, Preferences",
          ],
          [
            "Trim running on this PC",
            "Terminal: trim start / trim autostart",
            "http://127.0.0.1:8888/dashboard",
            "Local proxy must be listening",
          ],
          [
            "Claude Code → Trim address",
            "Shell env and/or ~/.claude/settings.json env",
            "http://127.0.0.1:8888 (NO /v1)",
            "Intercepts Claude Code traffic",
          ],
          [
            "Your Anthropic API key (runtime)",
            "Shell env and/or settings.json env",
            "sk-ant-…",
            "Trim forwards to Anthropic with your key",
          ],
          [
            "Optional model list helper",
            "CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY=1",
            "1",
            "Lets Claude Code discover models via Trim",
          ],
          [
            "Copy/paste helper",
            "Terminal: trim setup",
            "Prints Claude Code block for current port",
            "Avoid typing the URL by hand",
          ],
          [
            "Plans / prices",
            "Browser: use-trim.com/dashboard → Billing",
            "Your plan",
            "Cloud billing, not local",
          ],
          [
            "Local savings meter",
            "Browser: http://127.0.0.1:8888/dashboard",
            "Requests / Saved",
            "Proof traffic hit this PC’s Trim",
          ],
          [
            "Deep Mode toggle (optional)",
            "Browser: use-trim.com/dashboard → Preferences",
            "Deep on/off; v2 Base model (mbert-base default)",
            "Extra compression when enabled; change Base model under Settings when engine is v2",
          ],
        ],
      },
      {
        type: "h2",
        id: "before-you-start",
        text: "Before you start (checklist)",
      },
      {
        type: "ul",
        items: [
          "Terminal: Windows → use PowerShell (not old cmd.exe). macOS → Terminal / iTerm / Warp. Linux → bash or zsh. WSL on Windows → use the Linux copy blocks, not PowerShell",
          "You will install Claude Code (claude) in Step 0 if you do not have it yet",
          "You will create an Anthropic API key (sk-ant-…) in Step 0b - prefer a workspace-scoped key in the Anthropic Console",
          "You are on this page because you do not need Cursor or VS Code for this Claude Code-only path",
          "Every code block on this page has a Copy icon in the top-right of the block",
        ],
      },
      {
        type: "h2",
        id: "when",
        text: "When to do each step",
      },
      {
        type: "table",
        headers: ["When", "Step", "Where"],
        rows: [
          [
            "First time on this PC",
            "0 → 0b → 1 → 2 → 3 → 4 (or 4b/4c) → 5 → Verify",
            "Terminal + browser",
          ],
          [
            "Every day after setup",
            "Everyday loop (trim start if meter down → claude)",
            "Terminal",
          ],
          [
            "After reboot",
            "If autostart enabled, Trim may already be up; else trim start. Env must still be set (4b/4c)",
            "Terminal + local dashboard",
          ],
          [
            "New terminal window",
            "If you only did session exports (Step 4), paste Step 4 again - or use permanent 4b/4c",
            "That new terminal",
          ],
          [
            "Port is not 8888",
            "Use the origin printed by trim start / trim setup (still NO /v1)",
            "Terminal",
          ],
        ],
      },
      {
        type: "h2",
        id: "steps",
        text: "Setup steps (do in order)",
      },
      {
        type: "p",
        text: "Do Step 0 → 0b → 1 → 2 → 3 → 4 → 5 on every computer the first time. Use only the copy block for your OS. Do not mix Windows PowerShell syntax into bash, or bash export into PowerShell. WSL users: follow Linux blocks.",
      },
      {
        type: "h3",
        id: "step0",
        text: "Step 0 - Install Claude Code (Anthropic)",
      },
      {
        type: "p",
        text: "Where: Terminal on this PC. What: install Anthropic’s claude command (separate from Trim). How: Copy the block for your OS → paste → Enter. What happens: the Claude Code installer puts claude on your PATH. What next: claude --version (or claude -v) should print a version. Why first: Trim only intercepts Claude Code after claude exists. Official Anthropic installers (native recommended; npm is deprecated by Anthropic).",
      },
      {
        type: "code",
        language: "powershell",
        code: `# Windows PowerShell - Claude Code (Anthropic) - Copy this block
irm https://claude.ai/install.ps1 | iex
claude --version

# Alternative (WinGet):
# winget install Anthropic.ClaudeCode`,
      },
      {
        type: "code",
        language: "bash",
        code: `# macOS - Claude Code (Anthropic) - Copy this block
curl -fsSL https://claude.ai/install.sh | bash
claude --version

# Alternative (Homebrew cask):
# brew install --cask claude-code`,
      },
      {
        type: "code",
        language: "bash",
        code: `# Linux / WSL - Claude Code (Anthropic) - Copy this block
curl -fsSL https://claude.ai/install.sh | bash
claude --version`,
      },
      {
        type: "h3",
        id: "step0b",
        text: "Step 0b - Get your Anthropic API key",
      },
      {
        type: "p",
        text: "Where: browser → https://console.anthropic.com → Settings → API keys. What: create a key (prefer inside a workspace = workspace-scoped). How: copy the sk-ant-… secret once and keep it ready for Step 4. Why: Trim forwards to Anthropic with YOUR key - Trim Cloud login (Step 2) is a different account. When: do this once; rotate the key if it leaks.",
      },
      {
        type: "code",
        language: "text",
        code: "https://console.anthropic.com",
      },
      {
        type: "h3",
        id: "step1",
        text: "Step 1 - Install Trim",
      },
      {
        type: "p",
        text: "Where: Terminal on this PC. What: install the trim command onto your PATH. How: Copy the block for your OS → paste → Enter. What happens: the installer downloads Trim and registers the trim command. Why: Claude Code talks to Trim’s local helper - without the trim CLI you cannot start or meter that helper. What next: trim version must print a version number. If the command is not found, close the terminal, open a new one, and try again (PATH refresh).",
      },
      {
        type: "code",
        language: "powershell",
        code: `# Windows (PowerShell) - Copy this block
irm https://use-trim.com/install.ps1 | iex
trim version`,
      },
      {
        type: "code",
        language: "bash",
        code: `# macOS (Terminal / iTerm / Warp) - Copy this block
curl -fsSL https://use-trim.com/install.sh | sh
trim version`,
      },
      {
        type: "code",
        language: "bash",
        code: `# Linux / WSL (bash / zsh) - Copy this block
curl -fsSL https://use-trim.com/install.sh | sh
trim version`,
      },
      {
        type: "h3",
        id: "step2",
        text: "Step 2 - Sign in to Trim Cloud",
      },
      {
        type: "p",
        text: "Where: same Terminal. What: link this PC to your Trim account. How: Copy → paste → Enter. What happens: a browser window opens for sign-in; when you finish, the terminal confirms. Why: cloud quotas, Billing, and Preferences. What next: leave the Anthropic key for Step 4 - trim login is NOT your Anthropic key.",
      },
      {
        type: "code",
        language: "bash",
        code: `# Windows / macOS / Linux / WSL - Copy this
trim login`,
      },
      {
        type: "h3",
        id: "step3",
        text: "Step 3 - Start Trim on this PC",
      },
      {
        type: "p",
        text: "Where: same Terminal. What: start the local Trim helper so it listens on this PC (default port 8888). How: Copy → paste → Enter. What happens: Trim starts in the background (or shows a ready UI); the local meter becomes available. Why: Claude Code can only reach Trim if something is listening on 127.0.0.1:8888. What next: open the local dashboard URL below in a browser - if it loads, Step 3 succeeded. Optional: trim autostart enable so Trim comes back after reboot. Non-default port: if trim start prints a different port, use that host:port as the origin in Step 4 (still NO /v1) - easiest: run trim setup and copy its Claude Code block.",
      },
      {
        type: "code",
        language: "bash",
        code: `# Windows / macOS / Linux / WSL - Copy this
trim start
trim autostart enable`,
      },
      {
        type: "code",
        language: "text",
        code: "http://127.0.0.1:8888/dashboard",
      },
      {
        type: "p",
        text: "Optional helper - Where: Terminal. What: print ready-made Claude Code copy/paste lines for the port Trim is actually using (best when the port is not 8888). How:",
      },
      {
        type: "code",
        language: "bash",
        code: `# Windows / macOS / Linux / WSL - Copy this
trim setup`,
      },
      {
        type: "h3",
        id: "step4",
        text: "Step 4 - Point Claude Code at Trim (intercept)",
      },
      {
        type: "p",
        text: "Where: the same terminal window where you will run claude (session), or permanent options in Step 4b (shell/User env) / Step 4c (Claude Code settings.json - recommended for background agents). What: set ANTHROPIC_BASE_URL to Trim’s origin and ANTHROPIC_API_KEY to your Anthropic key. How: Copy only your OS block → paste → replace sk-ant-YOUR_KEY with your real key → Enter. What happens: from this moment, any claude process started in that terminal talks to Trim first. Why: without these two variables, Claude Code goes straight to Anthropic and Trim never sees the traffic.",
      },
      {
        type: "callout",
        title: "Replace the placeholder",
        text: "Change sk-ant-YOUR_KEY to your real Anthropic API key before you run the block. Prefer creating the key inside a workspace in the Anthropic Console (workspace-scoped keys need no extra Trim workspace setting).",
      },
      {
        type: "code",
        language: "powershell",
        code: `# Windows PowerShell - current session only - Copy this block
$env:ANTHROPIC_BASE_URL = "http://127.0.0.1:8888"
$env:ANTHROPIC_API_KEY = "sk-ant-YOUR_KEY"
$env:CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY = "1"

# Confirm (should print http://127.0.0.1:8888 with NO /v1)
echo $env:ANTHROPIC_BASE_URL`,
      },
      {
        type: "code",
        language: "bash",
        code: `# macOS (zsh / bash) - current session only - Copy this block
export ANTHROPIC_BASE_URL="http://127.0.0.1:8888"
export ANTHROPIC_API_KEY="sk-ant-YOUR_KEY"
export CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY=1

# Confirm (should print http://127.0.0.1:8888 with NO /v1)
echo "$ANTHROPIC_BASE_URL"`,
      },
      {
        type: "code",
        language: "bash",
        code: `# Linux (bash / zsh) - current session only - Copy this block
export ANTHROPIC_BASE_URL="http://127.0.0.1:8888"
export ANTHROPIC_API_KEY="sk-ant-YOUR_KEY"
export CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY=1

# Confirm (should print http://127.0.0.1:8888 with NO /v1)
echo "$ANTHROPIC_BASE_URL"`,
      },
      {
        type: "p",
        text: "Org / multi-workspace Anthropic keys only (most users can skip). Where: User environment (once) or shell profile. What: tell Trim which Anthropic workspace id to send. How: set TRIM_ANTHROPIC_WORKSPACE_ID, then open a new terminal and run trim start again.",
      },
      {
        type: "code",
        language: "powershell",
        code: `# Windows - org / multi-workspace keys ONLY (once). Then close/reopen PowerShell + trim start
[System.Environment]::SetEnvironmentVariable("TRIM_ANTHROPIC_WORKSPACE_ID","wrkspc_YOUR_ID","User")`,
      },
      {
        type: "code",
        language: "bash",
        code: `# macOS (zsh) / Linux zsh - org / multi-workspace keys ONLY (once). Then new terminal + trim start
echo 'export TRIM_ANTHROPIC_WORKSPACE_ID="wrkspc_YOUR_ID"' >> ~/.zshrc
source ~/.zshrc`,
      },
      {
        type: "code",
        language: "bash",
        code: `# Linux bash - org / multi-workspace keys ONLY (once). Then new terminal + trim start
echo 'export TRIM_ANTHROPIC_WORKSPACE_ID="wrkspc_YOUR_ID"' >> ~/.bashrc
source ~/.bashrc`,
      },
      {
        type: "h3",
        id: "step4-permanent",
        text: "Step 4b - Make the intercept permanent in your shell (optional)",
      },
      {
        type: "p",
        text: "Where: your shell startup file (macOS/Linux) or Windows User environment variables. What: so every new terminal keeps ANTHROPIC_BASE_URL pointed at Trim. Why: session-only exports disappear when you close the window. How: Copy your OS block. Security note: storing ANTHROPIC_API_KEY in a startup file is convenient but less safe than a secret manager - many teams set the Base URL permanently and paste the key per session. Prefer Step 4c if you also use Claude Code background agents. macOS note: default shell is zsh (~/.zshrc). If you use bash on macOS, append to ~/.bash_profile or ~/.bashrc instead.",
      },
      {
        type: "code",
        language: "powershell",
        code: `# Windows - permanent User env (survives new PowerShell windows)
# Copy → run → fully close PowerShell → open a new window
[System.Environment]::SetEnvironmentVariable("ANTHROPIC_BASE_URL","http://127.0.0.1:8888","User")
[System.Environment]::SetEnvironmentVariable("CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY","1","User")
# Optional (stores the key for your Windows user):
# [System.Environment]::SetEnvironmentVariable("ANTHROPIC_API_KEY","sk-ant-YOUR_KEY","User")`,
      },
      {
        type: "code",
        language: "bash",
        code: `# macOS (zsh default) - permanent - Copy this block, then open a new Terminal tab
echo 'export ANTHROPIC_BASE_URL="http://127.0.0.1:8888"' >> ~/.zshrc
echo 'export CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY=1' >> ~/.zshrc
# Optional (stores the key in your profile - understand the risk first):
# echo 'export ANTHROPIC_API_KEY="sk-ant-YOUR_KEY"' >> ~/.zshrc
source ~/.zshrc
echo "$ANTHROPIC_BASE_URL"`,
      },
      {
        type: "code",
        language: "bash",
        code: `# macOS (bash only) - permanent - Copy this if your macOS shell is bash
echo 'export ANTHROPIC_BASE_URL="http://127.0.0.1:8888"' >> ~/.bash_profile
echo 'export CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY=1' >> ~/.bash_profile
source ~/.bash_profile
echo "$ANTHROPIC_BASE_URL"`,
      },
      {
        type: "code",
        language: "bash",
        code: `# Linux (bash) - permanent - Copy this block, then open a new terminal
echo 'export ANTHROPIC_BASE_URL="http://127.0.0.1:8888"' >> ~/.bashrc
echo 'export CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY=1' >> ~/.bashrc
# Optional (stores the key in your profile - understand the risk first):
# echo 'export ANTHROPIC_API_KEY="sk-ant-YOUR_KEY"' >> ~/.bashrc
source ~/.bashrc
echo "$ANTHROPIC_BASE_URL"`,
      },
      {
        type: "code",
        language: "bash",
        code: `# Linux (zsh) / WSL zsh - permanent - Copy this block if your login shell is zsh
echo 'export ANTHROPIC_BASE_URL="http://127.0.0.1:8888"' >> ~/.zshrc
echo 'export CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY=1' >> ~/.zshrc
source ~/.zshrc
echo "$ANTHROPIC_BASE_URL"`,
      },
      {
        type: "h3",
        id: "step4c-settings",
        text: "Step 4c - Make the intercept permanent in Claude Code settings (recommended)",
      },
      {
        type: "p",
        text: "Where: Claude Code user settings file on this PC. What: put ANTHROPIC_BASE_URL (and optionally the key) in the env block so every Claude Code session - including background agents - uses Trim. Why: Anthropic’s gateway docs prefer settings-file env over shell-only exports when you want it everywhere; if both are set, the settings-file value wins. How: create or edit the file below, paste the JSON (replace the key), save, then start a new claude session. Confirm inside Claude Code with /status (Base URL should show Trim’s origin).",
      },
      {
        type: "table",
        headers: ["OS", "File to edit"],
        rows: [
          ["macOS / Linux / WSL", "~/.claude/settings.json"],
          ["Windows", "%USERPROFILE%\\.claude\\settings.json"],
        ],
      },
      {
        type: "p",
        text: "Where to create/open the file (Copy your OS block first if the folder does not exist yet):",
      },
      {
        type: "code",
        language: "powershell",
        code: `# Windows PowerShell - create folder + open settings.json in Notepad - Copy this
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\\.claude" | Out-Null
if (-not (Test-Path "$env:USERPROFILE\\.claude\\settings.json")) {
  Set-Content -Path "$env:USERPROFILE\\.claude\\settings.json" -Value "{}"
}
notepad "$env:USERPROFILE\\.claude\\settings.json"`,
      },
      {
        type: "code",
        language: "bash",
        code: `# macOS - create folder + open settings.json - Copy this
mkdir -p ~/.claude
test -f ~/.claude/settings.json || echo '{}' > ~/.claude/settings.json
open -e ~/.claude/settings.json`,
      },
      {
        type: "code",
        language: "bash",
        code: `# Linux / WSL - create folder + open settings.json - Copy this
mkdir -p ~/.claude
test -f ~/.claude/settings.json || echo '{}' > ~/.claude/settings.json
# Use your editor (examples):
nano ~/.claude/settings.json
# code ~/.claude/settings.json
# vim ~/.claude/settings.json`,
      },
      {
        type: "p",
        text: "Paste this JSON into the file (replace sk-ant-YOUR_KEY), save, then start a new claude session:",
      },
      {
        type: "code",
        language: "json",
        code: `{
  "env": {
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:8888",
    "ANTHROPIC_API_KEY": "sk-ant-YOUR_KEY",
    "CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY": "1"
  }
}`,
      },
      {
        type: "callout",
        title: "Security for settings.json",
        text: "This file can store your Anthropic key on disk for convenience. If your org forbids that, keep only ANTHROPIC_BASE_URL in settings.json and export ANTHROPIC_API_KEY per terminal session (Step 4).",
      },
      {
        type: "h3",
        id: "step5",
        text: "Step 5 - Run Claude Code through Trim",
      },
      {
        type: "p",
        text: "Where: the same terminal where Step 4 env vars are set (check with echo), or any terminal after Step 4c settings.json. What: use Claude Code exactly as you normally do - Trim is now in the middle. How: Copy a smoke-test block first, then use interactive claude for daily work. Why: this is the real usage path - setup only matters if claude traffic actually flows through Trim. What happens: the request hits Trim → Anthropic → back. What next: verify on the local meter (next section). Inside Claude Code you can also run /status to confirm the Base URL.",
      },
      {
        type: "code",
        language: "bash",
        code: `# Windows / macOS / Linux - one-shot smoke test - Copy this
claude -p "Reply with exactly: claude-code-trim-ok" --model claude-haiku-4-5-20251001`,
      },
      {
        type: "code",
        language: "bash",
        code: `# Windows / macOS / Linux - interactive Claude Code (everyday) - Copy this
claude`,
      },
      {
        type: "h2",
        id: "verify",
        text: "Verify it worked (do not skip)",
      },
      {
        type: "p",
        text: "Where: browser + terminal. When: right after the smoke test or first interactive turn. What you should see: local Requests go up; Claude answered; Base URL still has no /v1.",
      },
      {
        type: "ol",
        items: [
          "Browser: open http://127.0.0.1:8888/dashboard - Requests should increase after the claude command",
          "Terminal: run trim status (proxy healthy) and trim stats (door / savings lines)",
          'Terminal: confirm Base URL - Windows: echo $env:ANTHROPIC_BASE_URL | macOS/Linux/WSL: echo "$ANTHROPIC_BASE_URL" - must be http://127.0.0.1:8888 (or your printed origin) with NO /v1',
          "Inside Claude Code: run /status - Base URL / credential source should point at Trim (especially after Step 4c)",
          "Optional cloud Usage: https://use-trim.com/dashboard",
          "Plans / prices: https://use-trim.com/dashboard → Billing",
        ],
      },
      {
        type: "code",
        language: "bash",
        code: `# Windows / macOS / Linux - Copy this
trim status
trim stats`,
      },
      {
        type: "code",
        language: "powershell",
        code: `# Windows - confirm intercept is still set - Copy this
echo $env:ANTHROPIC_BASE_URL
echo $env:ANTHROPIC_API_KEY.Substring(0,7)   # shows sk-ant- prefix only`,
      },
      {
        type: "code",
        language: "bash",
        code: `# macOS / Linux - confirm intercept is still set - Copy this
echo "$ANTHROPIC_BASE_URL"
echo "$ANTHROPIC_API_KEY" | cut -c1-7   # shows sk-ant- prefix only`,
      },
      {
        type: "h2",
        id: "after",
        text: "Everyday loop (after setup)",
      },
      {
        type: "ol",
        items: [
          "If http://127.0.0.1:8888/dashboard will not load: Terminal → trim start (or rely on trim autostart enable from Step 3)",
          "Open a terminal where ANTHROPIC_BASE_URL is set (Step 4b shell, Step 4c settings.json, or paste Step 4 again)",
          "Run claude as usual - write code, ask questions, use agents",
          "Optional: inside Claude Code, /status to confirm Trim’s origin",
          "Check the local meter when you want proof of savings",
          "You do not need the Trim IDE extension for this Claude Code-only path",
          "You do not set Cursor Override OpenAI Base URL for Claude Code",
          "You do not put /v1 on ANTHROPIC_BASE_URL",
        ],
      },
      {
        type: "code",
        language: "bash",
        code: `# Everyday - if the local meter is down, Copy this first
trim start

# Then (same terminal with env set)
claude`,
      },
      {
        type: "h2",
        id: "do-not",
        text: "What NOT to do on this path",
      },
      {
        type: "ul",
        items: [
          "Do not open Cursor Settings → Models for Claude Code intercept",
          "Do not set Override OpenAI Base URL to …/v1 for this path",
          "Do not set ANTHROPIC_BASE_URL to http://127.0.0.1:8888/v1",
          "Do not expect trim-* custom model ids from Cursor - those are IDE Models names, not Claude Code Anthropic door settings",
          "Do not confuse trim login (Trim Cloud) with ANTHROPIC_API_KEY (Anthropic)",
          "Do not skip trim start - if nothing listens on :8888 (or your port), Claude Code cannot reach Trim",
          "Do not use Windows cmd.exe for the PowerShell blocks - open PowerShell (or use WSL + Linux blocks)",
          "Do not paste macOS/Linux export lines into PowerShell (and the reverse)",
        ],
      },
      {
        type: "h2",
        id: "revert",
        text: "How to turn Trim intercept off (revert)",
      },
      {
        type: "p",
        text: "Where: same places you set the intercept. When: if you want Claude Code to talk to Anthropic directly again (no Trim savings). What: remove or blank ANTHROPIC_BASE_URL. How:",
      },
      {
        type: "ol",
        items: [
          "Session: close the terminal, or unset the variable (Windows: Remove-Item Env:ANTHROPIC_BASE_URL | macOS/Linux: unset ANTHROPIC_BASE_URL)",
          "Step 4b shell profile: delete the export / SetEnvironmentVariable lines you added, then open a new terminal",
          "Step 4c settings.json: remove ANTHROPIC_BASE_URL from the env block (or delete the env block), save, start a new claude session",
          "Optional: trim stop if you also want the local proxy off; trim autostart disable if you do not want it after reboot",
        ],
      },
      {
        type: "code",
        language: "powershell",
        code: `# Windows PowerShell - turn off intercept for this session - Copy this
Remove-Item Env:ANTHROPIC_BASE_URL -ErrorAction SilentlyContinue
# Optional: stop local Trim
trim stop`,
      },
      {
        type: "code",
        language: "bash",
        code: `# macOS / Linux / WSL - turn off intercept for this session - Copy this
unset ANTHROPIC_BASE_URL
# Optional: stop local Trim
trim stop`,
      },
      {
        type: "h2",
        id: "troubleshoot",
        text: "If something fails",
      },
      {
        type: "table",
        headers: ["Symptom", "Likely cause", "Where to fix", "Fix"],
        rows: [
          [
            "model may not exist / path looks like /v1/v1",
            "ANTHROPIC_BASE_URL includes /v1",
            "Terminal env or settings.json",
            "Set origin only: http://127.0.0.1:8888 (no /v1)",
          ],
          [
            "Connection refused / failed to fetch",
            "Trim not running on this PC",
            "Terminal",
            "trim start → open local dashboard",
          ],
          [
            "Auth / 401 from Anthropic",
            "Missing or wrong ANTHROPIC_API_KEY",
            "Terminal env or settings.json",
            "Export / set a valid sk-ant-… key",
          ],
          [
            "Workspace scope error",
            "Org-scoped Anthropic key",
            "User env / shell profile",
            "Prefer workspace-scoped key, or set TRIM_ANTHROPIC_WORKSPACE_ID once then trim start",
          ],
          [
            "No new Requests on local meter",
            "Traffic bypassed Trim",
            "Same terminal as claude /settings",
            "echo Base URL or claude /status - must show 127.0.0.1; re-run Step 4 or 4c",
          ],
          [
            "trim: command not found",
            "Installer PATH not refreshed",
            "New terminal",
            "Close terminal, open new one, trim version",
          ],
          [
            "claude: command not found",
            "Claude Code not installed / PATH",
            "Terminal",
            "Redo Step 0 for your OS; new terminal; claude --version",
          ],
          [
            "Works in one window, fails in another",
            "Session-only export",
            "Shell profile / settings.json",
            "Do Step 4b and/or Step 4c",
          ],
          [
            "Background agent bypasses Trim",
            "Only shell export was set",
            "Claude Code settings.json",
            "Do Step 4c (settings env applies to agents too)",
          ],
          [
            "Wrong port / still 8888 but trim uses another",
            "Hardcoded port mismatch",
            "Terminal",
            "Run trim setup and copy the printed origin",
          ],
          [
            "0% Saved on agent turns",
            "Often normal: tools/system chrome frozen; Segmented Allowlist Deep runs on each prose island in place (order preserved beside paths)",
            "Local dashboard",
            "Open Show savings detail - Deep status + Deep stage (prose islands; order preserved beside paths)",
          ],
        ],
      },
      {
        type: "h2",
        id: "related",
        text: "Related pages",
      },
      {
        type: "note",
        text: "Quickstart (shared install), trim start, CLI overview (trim setup), Connect any IDE (why Cursor uses a different address), Base URL troubleshooting, Always-on with IDE, FAQ (Claude Code only).",
      },
    ],
  ),

  "cli/start": page(
    "trim start",
    "Start the local OpenAI-compatible proxy that runs Fast Mode on every request (and Deep when Preferences Deep is on).",
    [
      {
        type: "code",
        language: "bash",
        code: "trim start",
      },
      {
        type: "p",
        text: "Trim listens on TRIM_PORT (example http://127.0.0.1:8888/v1). Always use the Base URL printed by trim start for on-device clients. Point Continue apiBase at …/v1; VS Code Chat Custom Endpoint at …/v1/chat/completions; Claude Code at the listen origin without /v1. For Cursor Agent, do not use 127.0.0.1 - use Trim Cursor Link (trim cursor-link url) instead. The proxy reports mode, privacy posture, upstream routing, and last door in the terminal and /v1/stats.",
      },
      {
        type: "p",
        text: "For everyday use, prefer Always-on with IDE (extension + trim autostart) so you do not need a manual trim start every morning. Use trim start for one-shot sessions or debugging. If Cursor Link is enabled, trim start also brings up the public HTTPS door and trim stop tears it down with the proxy.",
      },
      { type: "h2", id: "what-you-see", text: "What you see" },
      {
        type: "ul",
        items: [
          "Fresh start: green ● Trim proxy running…, cyan/accent Local Base URL + green Dashboard table rows, gray muted on-device tip + Cursor Agent Cursor Link tip (process stays in the foreground)",
          "Already running (after autostart / another IDE): same Success + Table + Muted design as status/autostart, then exit 0 - never a blank prompt",
        ],
      },
      {
        type: "code",
        language: "text",
        code: `● Trim is already running on http://127.0.0.1:8888
  Local Base URL  http://127.0.0.1:8888/v1
  Dashboard       http://127.0.0.1:8888/dashboard
Attached to the existing local proxy - no second process started.
On-device: Continue / VS Code Chat / Claude Code use 127.0.0.1 as documented.
Cursor Agent: trim cursor-link enable → trim cursor-link url (HTTPS …/v1; not 127.0.0.1).`,
      },
      { type: "h2", id: "behavior", text: "Behavior" },
      {
        type: "ul",
        items: [
          "Fast Mode on live chat completions",
          "Segmented Allowlist Deep on live chat when Preferences Deep is on (each last-user prose island in place, order preserved beside paths; tools/paths frozen)",
          "If a healthy Trim proxy is already up, trim start attaches with a clear success banner instead of double-binding the port",
          "Optional cloud metering when authenticated",
          "Upstream forward to your configured provider after compress",
          "Active-file and stack protection so task-relevant code stays intact",
        ],
      },
      { type: "h2", id: "ops", text: "Operational tips" },
      {
        type: "ul",
        items: [
          "Stop with trim stop (localhost control endpoint)",
          "If the port is busy with a healthy Trim, attach is the expected path",
          "After changing dashboard Preferences (rate, engine, or v2 Base model), run trim config sync; list Base Models with trim config models; enforcers re-read the auto-start preference on their poll interval",
          "After traffic: trim stats (Deep status + Deep stage) or open /dashboard → Show savings detail",
        ],
      },
      {
        type: "callout",
        title: "Local savings meter",
        text: "With trim start running, open http://127.0.0.1:8888/dashboard and use Show savings detail. Or run trim stats for the same Deep status / Deep stage lines in the terminal. Whole-body 0% Saved on Claude Code / Cursor agent chrome is often normal.",
      },
      {
        type: "note",
        text: "Cursor Agent users: enable Trim Cursor Link so trim start prints a https://…/cl/<token>/v1 Base URL Cursor’s cloud can reach. Full guide: Trim Cursor Link + Cursor setup.",
      },
    ],
  ),

  "cli/cursor-link": page(
    "Trim Cursor Link",
    "Public HTTPS door to local Trim for Cursor Agent - lifecycle tied to trim start / trim stop. Where / what / how / why with copy buttons.",
    [
      {
        type: "p",
        text: "Trim Cursor Link productizes the Cloudflare Tunnel workflow inside Trim. You enable it once. While trim start is running, Trim keeps a public https://….trycloudflare.com/cl/<token>/v1 door pointed at your local proxy. Cursor Agent pastes that URL. Compression, adapters, and the local meter still run on your PC. trim stop stops Trim and the tunnel together.",
      },
      {
        type: "callout",
        title: "Why this exists",
        text: "Cursor Agent often runs in Cursor’s cloud. That cloud blocks private IPs (127.0.0.1) with “Access to private networks is forbidden.” A public hostname fixes reachability without moving Trim’s brain to the cloud.",
      },
      {
        type: "h2",
        id: "flow",
        text: "Flow (local E2E preserved)",
      },
      {
        type: "code",
        language: "text",
        code: `trim cursor-link enable
trim start
   → local Trim listens on 127.0.0.1:8888
   → Cursor Link tunnel starts automatically
   → prints Cursor Agent Base URL (https://…/cl/<token>/v1)

Cursor Override OpenAI Base URL = that https URL
   → Cursor cloud → tunnel → YOUR local Trim → provider

trim stop
   → local Trim stops
   → tunnel stops`,
      },
      {
        type: "h2",
        id: "commands",
        text: "Commands (copy)",
      },
      {
        type: "code",
        language: "bash",
        code: `# Enable (once per machine) - Copy this
trim cursor-link enable

# Start Trim + tunnel - Copy this
trim start

# Print Base URL for Cursor - Copy this
trim cursor-link url

# Status table - Copy this
trim cursor-link status

# Rotate access token (revoke old URL path immediately) - Copy this
trim cursor-link rotate
trim cursor-link url
# Paste the new URL into Cursor Override OpenAI Base URL

# Stop Trim + tunnel - Copy this
trim stop

# Disable preference (also stops tunnel + clears token) - Copy this
trim cursor-link disable`,
      },
      {
        type: "h2",
        id: "cloudflared",
        text: "cloudflared helper",
      },
      {
        type: "p",
        text: "Where: PATH, TRIM_CLOUDFLARED, or ~/.trim/bin. What: Cloudflare’s cloudflared binary. How: Trim launches it for you. Preferred: run trim cursor-link enable && trim start and let Trim auto-download into ~/.trim/bin (Windows, macOS, Linux). winget is optional and often missing on Windows. Why: users should never DIY Cloudflare ingress for everyday Cursor Agent use.",
      },
      {
        type: "code",
        language: "powershell",
        code: `# Windows - recommended (no winget required) - Copy this
trim cursor-link enable
trim start
# Trim downloads cloudflared into %USERPROFILE%\\.trim\\bin when missing

# Manual fallback (amd64, no winget) - Copy this
$dir = Join-Path $HOME ".trim\\bin"
New-Item -ItemType Directory -Force -Path $dir | Out-Null
$dest = Join-Path $dir "cloudflared.exe"
Invoke-WebRequest -Uri "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-windows-amd64.exe" -OutFile $dest
& $dest --version
# Optional: $env:TRIM_CLOUDFLARED = $dest
# Windows on ARM: same amd64 .exe (runs under emulation; Cloudflare has no native arm64 Windows build)
# Only if winget exists: winget install --id Cloudflare.cloudflared`,
      },
      {
        type: "code",
        language: "bash",
        code: `# macOS - recommended (no Homebrew required) - Copy this
trim cursor-link enable && trim start
# Trim downloads cloudflared into ~/.trim/bin when missing

# Manual fallback without brew (Apple Silicon) - Copy this
mkdir -p ~/.trim/bin
curl -fsSL -o /tmp/cloudflared.tgz \\
  https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-darwin-arm64.tgz
tar -xzf /tmp/cloudflared.tgz -C ~/.trim/bin
chmod +x ~/.trim/bin/cloudflared
~/.trim/bin/cloudflared --version
# Intel Mac: use cloudflared-darwin-amd64.tgz
# Only if Homebrew exists: brew install cloudflared
# Or: export TRIM_CLOUDFLARED=/opt/homebrew/bin/cloudflared`,
      },
      {
        type: "code",
        language: "bash",
        code: `# Linux - recommended (auto-download) - Copy this
trim cursor-link enable && trim start
# Trim downloads cloudflared into ~/.trim/bin when missing

# Manual fallback (amd64) - Copy this
mkdir -p ~/.trim/bin
curl -fsSL -o ~/.trim/bin/cloudflared \\
  https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64
chmod +x ~/.trim/bin/cloudflared
~/.trim/bin/cloudflared --version
# arm64: use cloudflared-linux-arm64
# Or: export TRIM_CLOUDFLARED=/usr/local/bin/cloudflared`,
      },
      {
        type: "h2",
        id: "privacy",
        text: "Privacy (honest)",
      },
      {
        type: "ul",
        items: [
          "The hostname is public HTTPS so Cursor’s cloud can connect",
          "Fast/Deep compression and metering still run on your machine",
          "This is not “Trim cloud processes your prompts instead of local Trim”",
          "Disable anytime: trim cursor-link disable (clears token)",
          "Rotate token anytime: trim cursor-link rotate then update Cursor (revoke is live - no Trim restart required)",
          "Public path is /cl/<token>/v1 - bare /v1 via Cloudflare edge is rejected",
          "Quick-tunnel hostnames may rotate after restart - re-run trim cursor-link url and update Cursor",
        ],
      },
      {
        type: "h2",
        id: "vs-localhost",
        text: "When to use Cursor Link vs localhost",
      },
      {
        type: "table",
        headers: ["Client", "Base URL", "Guide"],
        rows: [
          [
            "Cursor Agent / cloud-routed Cursor",
            "trim cursor-link url (https://…/cl/<token>/v1)",
            "Cursor setup",
          ],
          ["Claude Code CLI", "http://127.0.0.1:8888 (no /v1)", "Claude Code (CLI only)"],
          [
            "Continue / VS Code Chat / local SDKs",
            "http://127.0.0.1:8888/v1 (or …/chat/completions)",
            "Continue / VS Code setup",
          ],
        ],
      },
      {
        type: "note",
        text: "Related: Cursor setup, trim start, trim stop, Base URL troubleshooting, FAQ.",
      },
    ],
  ),

  "cli/autostart": page(
    "trim autostart",
    "Enable, disable, or check the Always-on with IDE preference from the CLI.",
    [
      {
        type: "p",
        text: "trim autostart mirrors telemetry-style UX for the everyday proxy preference. It syncs auto_start_with_ide with Trim Cloud and best-effort aligns an OS logon hook so Trim can start with your user session when supported.",
      },
      {
        type: "code",
        language: "bash",
        code: `trim autostart status
trim autostart enable
trim autostart disable`,
      },
      { type: "h2", id: "commands", text: "Commands" },
      {
        type: "ul",
        items: [
          "status - show whether auto-start with IDE is on, off, or unset",
          "enable - turn the preference on and best-effort register OS logon when the OS supports it",
          "disable - turn the preference off and uninstall the OS logon hook when supported",
        ],
      },
      { type: "h2", id: "preference-first", text: "Preference first, OS hook second" },
      {
        type: "p",
        text: "The synced preference is the source of truth (dashboard, CLI, IDE extension). The OS logon hook is best-effort: if registration fails, Trim still saves your preference and prints clear next steps - it does not dump raw OS errors as the only message.",
      },
      {
        type: "ul",
        items: [
          "Success (green): Auto-start with IDE enabled - preference is on",
          "Warn (yellow) only if OS logon could not be registered: Preference saved, but OS logon autostart could not be registered",
          "Muted guidance: what happened (for example Access denied) and what to do next",
          "Table next steps: Manual start → trim start; With IDE → keep Start Trim with your IDE on",
          "Details / raw OS text: only with trim --verbose (or -v)",
        ],
      },
      { type: "h2", id: "daemon", text: "Optional trim daemon (OS logon)" },
      {
        type: "p",
        text: "trim daemon install|uninstall|status|run is the advanced OS logon entrypoint. Prefer trim autostart enable for everyday use - it calls the same install path when turning the preference on.",
      },
      {
        type: "ul",
        items: [
          "macOS: launchd LaunchAgent (per-user)",
          "Linux: systemd --user unit (per-user)",
          "Windows: per-user Startup Run key (HKCU CurrentVersion\\Run) → %LOCALAPPDATA%\\Trim\\bin\\trim.exe - industry practice for apps like browsers and editors; no Administrator on a normal PC; npm uninstall never locks the package tree",
        ],
      },
      {
        type: "p",
        text: "daemon run only starts the proxy when the preference is on (or the hosted default when unset), and attaches if a healthy proxy is already listening. On Windows, install copies the CLI into LocalAppData so package managers can remove their trees freely. Older installs that used a scheduled task or ran from node_modules are migrated away on install/uninstall.",
      },
      {
        type: "code",
        language: "bash",
        code: `trim daemon status
trim daemon install
trim daemon uninstall`,
      },
      { type: "h2", id: "verify", text: "Verify on your machine" },
      {
        type: "p",
        text: "Copy these checks after install. Docs code blocks include a copy control in the product docs UI.",
      },
      {
        type: "code",
        language: "bash",
        code: `# Everyday path
trim autostart enable
trim autostart status
trim daemon status

# Soft-fail detail only when needed
trim autostart enable -v

# Cleanup / opt-out
trim autostart disable
trim daemon uninstall`,
      },
      {
        type: "code",
        language: "powershell",
        code: `# Windows: confirm Startup Run → LocalAppData (no admin)
trim daemon install
trim daemon status
Get-ItemProperty HKCU:\\Software\\Microsoft\\Windows\\CurrentVersion\\Run |
  Select-Object TrimProxy
Test-Path \"$env:LOCALAPPDATA\\Trim\\bin\\trim.exe\"
trim daemon uninstall`,
      },
      {
        type: "callout",
        title: "If OS logon registration fails",
        text: "Your preference is still on. Run trim start for a manual session, or open your IDE with the Trim IDE extension / Start Trim with your IDE checked. On locked-down Windows, an organization policy may block Startup programs - that is expected; use manual or IDE start.",
      },
      {
        type: "callout",
        title: "Same preference bit",
        text: "Dashboard checkbox, trim autostart, trim daemon, and the IDE extension share one policy. Uncheck in the dashboard or run trim autostart disable - local enforcers stop auto-starting.",
      },
      {
        type: "note",
        text: "CLI help: trim help autostart and trim help daemon. Managed off: DO_NOT_TRACK=1, TRIM_AUTOSTART_DISABLED=1, or a ~/.config/trim/autostart.off marker.",
      },
    ],
  ),

  "cli/login": page(
    "trim login",
    "Authenticate the CLI to the hosted Service with a device flow.",
    [
      {
        type: "code",
        language: "bash",
        code: "trim login",
      },
      {
        type: "p",
        text: "The CLI opens a browser device-code flow against the hosted app. After success, keys and quotas sync to this machine subject to device-binding rules.",
      },
      { type: "h2", id: "requirements", text: "Requirements" },
      {
        type: "ul",
        items: [
          "Network access to the hosted API and app",
          "An allowed identity provider (for example Google, GitHub, or GitLab)",
          "A recent CLI build if the hosted Service requires an upgrade",
        ],
      },
      { type: "h2", id: "after", text: "After login" },
      {
        type: "ol",
        items: [
          "Keep Trim ready via Always-on with IDE (or trim start for a manual session)",
          "Confirm trim status shows metered events after a request",
          "Open Dashboard → Receipts or plan CTAs for billing details",
        ],
      },
      {
        type: "note",
        text: "API keys issued for IDE or CI may require hardware registration under Settings before they authenticate from a new machine.",
      },
    ],
  ),

  "cli/status": page(
    "trim status",
    "Show Trim cloud auth and quota. For local savings / Deep status, use trim stats and the local dashboard.",
    [
      {
        type: "code",
        language: "bash",
        code: `trim status

# Local savings + Deep status (proxy must be running)
trim stats

# Visual meter
# open http://127.0.0.1:8888/dashboard → Show savings detail`,
      },
      {
        type: "p",
        text: "trim status is your cloud login/quota check. It does not replace the local meter. After trim start and a chat turn, run trim stats for Deep status (applied, no prose islands / chrome frozen, fail-closed) and Deep stage (prose) savings, or open the local dashboard Show savings detail button.",
      },
      { type: "h2", id: "fields", text: "What trim stats shows" },
      {
        type: "ul",
        items: [
          "Deep status: why Deep ran or was skipped on the last request",
          "Deep stage: prose-island tokens before → after (separate from whole-body %)",
          "Last request (whole body): can be ~0% when agent chrome is frozen - often normal",
          "Dashboard tip: link to Show savings detail on the local meter",
        ],
      },
      { type: "h2", id: "empty", text: "Empty or zero readings" },
      {
        type: "ul",
        items: [
          "Confirm the IDE Base URL points at Trim",
          "Confirm trim start is still running (restart after upgrading trim.exe)",
          "Send a fresh request after fixing settings",
          "Open Show savings detail - 0% whole-body Saved on Claude Code / Cursor agent chrome is often normal",
          "Sign in if you expect cloud metering totals (trim status / trim login)",
        ],
      },
    ],
  ),

  "cli/stats": page(
    "trim stats",
    "Everyday local meter: Deep status, Deep stage tokens, whole-body last request, and a link to Show savings detail. Listed next to trim status in trim --help.",
    [
      {
        type: "code",
        language: "bash",
        code: `# From the cli/ folder when using a repo-built binary:
# cd cli
# ..\\trim.exe stats

trim --help
trim help stats
trim help status

trim start
trim stats
trim stats --tui

# Cloud quota (not local savings)
trim status

# Visual meter
# open http://127.0.0.1:8888/dashboard → Show savings detail`,
      },
      {
        type: "p",
        text: "Requires trim start (or Always-on) so the local proxy is listening. trim stats reads GET /v1/stats, prints human-readable Deep lines first, then the raw JSON, plus today SQLite totals when available. After upgrading Trim, restart trim start and run trim config sync so help strings and Deep labels refresh.",
      },
      { type: "h2", id: "everyday", text: "Everyday commands" },
      {
        type: "ol",
        items: [
          "trim start (leave running)",
          "Use Claude Code / Cursor through Trim",
          "trim stats - read Deep status + Deep stage",
          "Open http://127.0.0.1:8888/dashboard → Show savings detail",
          "trim status - only when you need cloud quota",
        ],
      },
      { type: "h2", id: "read-output", text: "How to read the output" },
      {
        type: "ul",
        items: [
          "Deep applied + Deep stage A → B with B < A: live Deep savings on prose islands (order preserved beside paths)",
          "Deep skipped (no prose islands / chrome frozen): tools/paths/schemas protected; whole-body 0% can still be correct",
          "Deep kept Fast (expansion fail-closed): safety gate refused a bad Deep result",
        ],
      },
      {
        type: "callout",
        title: "Not the same as trim status",
        text: "trim status = cloud quota. trim stats = local meter clarity. Use both.",
      },
    ],
  ),

  "cli/compress": page(
    "trim compress",
    "Run Fast or Deep compression on files without starting the full proxy.",
    [
      {
        type: "code",
        language: "bash",
        code: `trim compress path/to/file.go --mode fast
trim compress notes.txt --deep --engine v2 --rate 0.7
trim compress notes.txt --deep --engine long --question "Summarize auth flow" --rate 0.7
trim compress --bootstrap`,
      },
      {
        type: "p",
        text: "Use Fast Mode for interactive IDE chat when Deep is off (Fast-only path). With Preferences Deep on, live proxy runs Fast then Segmented Allowlist Deep on each request (each last-user prose island in place, order preserved beside paths; tools/paths frozen) using your compression rate (keep after/before; default 0.7) and, for engine v2, your synced Base Model. Use trim compress --deep --rate 0.7 for file/batch. Bootstrap may download local Deep dependencies (mbert-base is smaller; xlm-roberta-large is larger and slower). Deep Mode engines are not loaded in Trim cloud.",
      },
      { type: "h2", id: "engines", text: "Deep engines (pick one)" },
      {
        type: "ul",
        items: [
          "v2 / LLMLingua-2 (recommended) - default for new accounts; Base Model catalog applies (trim config models)",
          "long / LongLLMLingua - question-aware; pass --question",
          "v1 / LLMLingua - classic general compression",
        ],
      },
      { type: "h2", id: "rate", text: "Compression rate" },
      {
        type: "p",
        text: "Pass --rate as a Microsoft-style keep-rate (after/before), typically 0.3 to 1.0 step 0.1. Omit --rate to use the synced preference (Dashboard slider or trim config set compression-rate).",
      },
      { type: "h2", id: "base-model", text: "Base Model (LLMLingua-2 v2)" },
      {
        type: "p",
        text: "When engine is v2, trim compress and live Deep use the Base Model from trim config sync (Microsoft Space radio: mbert-base default, optional xlm-roberta-large). There is no --backbone flag; set the catalog id with Dashboard → Settings or trim config set deep-v2-backbone, then sync. List options with trim config models.",
      },
      {
        type: "code",
        language: "bash",
        code: `trim config models
trim config set deep-v2-backbone mbert-base
trim config set deep-v2-backbone xlm-roberta-large
trim compress notes.txt --deep --engine v2 --rate 0.7`,
      },
      { type: "h2", id: "when", text: "When to use compress vs start" },
      {
        type: "ul",
        items: [
          "trim start: interactive IDE chat every turn (Fast always; Deep after Fast when Preferences Deep is on)",
          "trim compress: on-machine file/batch Deep compression (same engines, rate, and v2 Base Model as live proxy Deep)",
          "Deep on live chat is stronger but slower (first turn may load the local model); turn Deep off in Preferences for Fast-only latency",
        ],
      },
    ],
  ),

  "cli/config": page(
    "Configuration",
    "Configure .trimrc rules and sync dashboard preferences to the machine.",
    [
      { type: "h2", id: "trimrc", text: ".trimrc" },
      {
        type: "p",
        text: "Copy .trimrc.example to .trimrc to set mode (mild, balanced, aggressive, custom) and rule knobs for Fast Mode. Keep paths you care about protected so Composer still sees the files that matter.",
      },
      {
        type: "code",
        language: "ini",
        code: `mode=balanced
active_file_protection=true
# Optional: sliding-window chat history (0 or omit = off)
# history_keep_turns=8`,
      },
      {
        type: "p",
        text: "Optional history_keep_turns stubs very old eligible chat turns to save tokens on huge sessions. Default is off. See history_keep_turns for when to enable it, what stays protected, and configuration examples.",
      },
      { type: "h2", id: "commands", text: "trim config" },
      {
        type: "code",
        language: "bash",
        code: `trim config get
trim config get default-tier
trim config set default-tier fast
trim config set default-tier deep
trim config rates
trim config set compression-rate 0.7
trim config models
trim config set deep-v2-backbone mbert-base
trim config sync`,
      },
      {
        type: "p",
        text: "New accounts default to Deep Mode. get shows local preferences under ~/.config/trim/ (default-tier, deep-engine, deep-v2-backbone, compression-rate). rates lists allowed keep-rates and marks billing default plus selected. models lists Microsoft LLMLingua-2 Base Models and marks billing default plus selected. set writes one key locally (when logged in, default-tier, compression-rate, and deep-v2-backbone also update your Trim cloud profile). sync pulls Fast/Deep preferences and provider adapters from your Trim cloud profile (requires trim login).",
      },
      {
        type: "ul",
        items: [
          "fast: live IDE proxy runs Fast Mode only",
          "deep: live IDE proxy runs Fast then Segmented Allowlist Deep (LLMLingua-2 on each last-user prose island in place, order preserved beside paths) at your compression rate and Base Model",
          "compression-rate: keep-rate after/before (Microsoft-aligned; default 0.7; typically 0.3 to 1.0 step 0.1)",
          "deep-v2-backbone: Microsoft Base Model id from Admin catalog (default mbert-base; optional xlm-roberta-large)",
          "After Dashboard → Settings changes (rate slider or Base model): trim config sync, then trim start again",
        ],
      },
      { type: "h2", id: "sync", text: "Config sync" },
      {
        type: "code",
        language: "bash",
        code: "trim config sync",
      },
      {
        type: "p",
        text: "After changing Compression defaults in the dashboard (Deep on/off, engine, compression rate, Base model), sync so the local proxy picks up preferences. New accounts default to Deep (live proxy runs Fast then Segmented Allowlist Deep). Switch to Fast anytime for Fast-only turns.",
      },
      {
        type: "note",
        text: "Compression defaults are per-account Preferences on Dashboard → Settings. There is no Team publish-defaults UI. Each machine still runs trim config sync so local state stays explicit.",
      },
    ],
  ),

  "cli/history-keep-turns": page(
    "history_keep_turns",
    "Optional sliding-window chat history for huge IDE sessions. Default off. Protects tools, files, and agent chrome while stubbing older eligible text turns.",
    [
      {
        type: "p",
        text: "history_keep_turns is an optional Fast Mode knob for very long multi-turn chats. When enabled, Trim keeps the last N eligible conversation turns more fully and may replace older eligible text turns with the synced site_messages stub CLI_PROXY_HISTORY_COMPACTED_STUB so you stop resending ancient prose on every request. If that chrome is missing after trim config sync, history stubbing stays off (fail-closed).",
      },
      {
        type: "callout",
        title: "Default is off (recommended for most users)",
        text: "0 or unset means Trim does not stub old turns this way. Everyday Fast Mode and optional Deep Mode still run. Turn history_keep_turns on only when sessions are huge and you accept that very old text may no longer be fully visible to the model.",
      },
      { type: "h2", id: "what-it-does", text: "What it does" },
      {
        type: "ul",
        items: [
          "Keeps the newest N eligible chat turns (user/assistant text history) more fully",
          "May stub older eligible text turns with a compact history marker",
          "Still forwards a normal multi-turn request structure to the model (not “last message only”)",
          "Does not replace Deep Mode: Deep still compresses each last-user prose island in place (order preserved beside paths) when Deep prefs are on (paths/tools/schemas stay frozen)",
        ],
      },
      { type: "h2", id: "what-stays-protected", text: "What stays protected" },
      {
        type: "p",
        text: "Trim will not stub these turns just because they are old:",
      },
      {
        type: "ul",
        items: [
          "system / developer instructions",
          "tool / function messages and assistant tool_calls turns",
          "multimodal and file parts (images, audio, documents, browser state)",
          "thinking / reasoning_content / thought signatures / thinking_blocks",
          "cache_control / prompt_cache_breakpoint chrome and system-reminder content",
          "assistant prefix:true (Mistral) and similar protocol fields",
        ],
      },
      {
        type: "note",
        text: "When request-level prompt caching is present (top-level cache_control / prompt_cache_* fields), history stubbing is skipped so the reusable prefix stays byte-stable.",
      },
      { type: "h2", id: "when-to-use", text: "When to use it" },
      {
        type: "ul",
        items: [
          "Useful: very long coding sessions, repeated huge pastes, context/cost pressure",
          "Usually unnecessary: short or normal chats - leave off",
          "Tradeoff: older plain-text details can disappear from the model’s view after stubbing",
        ],
      },
      { type: "h2", id: "configure", text: "How to configure" },
      {
        type: "p",
        text: "This is a local project/operator setting. It is not turned on automatically for worldwide users.",
      },
      { type: "h3", id: "trimrc", text: "Project .trimrc (preferred)" },
      {
        type: "p",
        text: "Copy .trimrc.example → .trimrc in your repo (or a parent directory Trim walks up to), then set:",
      },
      {
        type: "code",
        language: "ini",
        code: `# Keep last 8 eligible turns; older eligible text turns may be stubbed
history_keep_turns=8

# Optional: keep last 2 tool outputs; older tool_result / role=tool payloads cleared
# (requires synced CLI_PROXY_TOOL_RESULT_CLEARED_FMT from Trim cloud; 0 = off)
# tool_result_clear_keep_recent=2

# Explicit off
# history_keep_turns=0
# tool_result_clear_keep_recent=0`,
      },
      { type: "h3", id: "tool-result-clearing", text: "Optional tool-result clearing" },
      {
        type: "p",
        text: "tool_result_clear_keep_recent is a separate Fast Mode dial (Claude Code / Cursor / Codex style). When set above 0 and Trim cloud chrome CLI_PROXY_TOOL_RESULT_CLEARED_FMT is synced, older tool_result / role=tool payloads are replaced with that chrome stub while keeping tool ids. The trailing user turn is never cleared. Default is 0 (off). When .trimrc omits the dial, TRIM_TOOL_RESULT_CLEAR_KEEP_RECENT from the process environment applies.",
      },
      { type: "h3", id: "env", text: "Environment when .trimrc omits" },
      {
        type: "p",
        text: "If .trimrc does not set the value, you can use TRIM_HISTORY_KEEP_TURNS (and TRIM_TOOL_RESULT_CLEAR_KEEP_RECENT). Project .trimrc wins when set.",
      },
      {
        type: "code",
        language: "bash",
        code: `# Unix
export TRIM_HISTORY_KEEP_TURNS=8
# export TRIM_TOOL_RESULT_CLEAR_KEEP_RECENT=2

# Windows PowerShell (user scope example)
# [Environment]::SetEnvironmentVariable("TRIM_HISTORY_KEEP_TURNS", "8", "User")`,
      },
      {
        type: "ol",
        items: [
          "Set history_keep_turns (or the env var)",
          "Restart the local proxy (trim stop then trim start, or restart your IDE so Always-on refreshes)",
          "Send a long multi-turn chat and confirm behavior on trim status / traces if needed",
        ],
      },
      { type: "h2", id: "vs-fast-deep", text: "How it relates to Fast and Deep" },
      {
        type: "ul",
        items: [
          "Fast Mode: always runs; can also shorten compressible text without stubbing whole turns",
          "history_keep_turns: optional Fast-path sliding window that stubs older eligible turns",
          "Deep Mode: optional Segmented Allowlist Deep on each last-user prose island in place (order preserved beside paths; does not smash system/tools/paths into LLMLingua). For engine v2, uses your synced Base Model and keep-rate (separate from history_keep_turns stubbing)",
        ],
      },
      {
        type: "callout",
        title: "Best practice",
        text: "Leave history_keep_turns off unless you have a real huge-history problem. Start around 8-30 if you enable it, protect critical files with never_trim / active_file_protection, and turn it back to 0 if the model starts “forgetting” older details you still need.",
      },
    ],
  ),

  "ide/connect": page(
    "Connect any IDE",
    "Pick your tool first, then copy the matching address. Clear where / what / how for each client.",
    [
      {
        type: "p",
        text: "Trim runs on your PC. Your coding tool must send chat to Trim’s address. Different tools use slightly different addresses. Use the Copy button on every block. For friendly IDE model names, see Custom models.",
      },
      {
        type: "callout",
        title: "Claude Code only? Wrong page for full setup",
        text: "If you use Claude Code and not Cursor/VS Code, open Claude Code (CLI only) instead. That guide is terminal-only (ANTHROPIC_BASE_URL origin, no /v1). This page is the hub for IDE-style clients.",
      },
      {
        type: "table",
        headers: ["I use…", "Address to paste", "Where"],
        rows: [
          [
            "Claude Code only",
            "http://127.0.0.1:8888 (no /v1)",
            "Terminal env - full guide: Claude Code (CLI only)",
          ],
          [
            "Cursor Agent",
            "trim cursor-link url (https://…/cl/<token>/v1)",
            "Settings → Models → Override OpenAI Base URL (not 127.0.0.1)",
          ],
          [
            "Continue / Windsurf / Zed / JetBrains / Aider",
            "http://127.0.0.1:8888/v1",
            "That product’s OpenAI Base URL / apiBase setting",
          ],
          [
            "VS Code Chat Custom Endpoint",
            "http://127.0.0.1:8888/v1/chat/completions",
            "Chat: Manage Language Models",
          ],
        ],
      },
      {
        type: "h2",
        id: "steps",
        text: "Steps (every client)",
      },
      {
        type: "ol",
        items: [
          "Install and sign in: trim version → trim login (see Quickstart / Installation)",
          "Keep Trim ready: Dashboard → Preferences → Start Trim with your IDE, then trim autostart enable (or trim start)",
          "Sync adapters + aliases: trim config sync",
          "Pick the correct address: Cursor Agent = trim cursor-link url; on-device IDEs = local …/v1 from the blocks below",
          "Open your editor’s exact settings place (table below) and paste Base URL + model + matching provider key",
          "Send a short chat and confirm http://127.0.0.1:8888/dashboard Requests increase",
          "Plans / prices anytime: https://use-trim.com/dashboard → Billing",
        ],
      },
      {
        type: "h2",
        id: "copy-first",
        text: "Copy these first",
      },
      {
        type: "code",
        language: "text",
        code: `Cursor Agent (REQUIRED) - run: trim cursor-link url
https://….trycloudflare.com/cl/<token>/v1

OpenAI door on-device (Continue, Windsurf, Zed, JetBrains, Aider)
http://127.0.0.1:8888/v1

VS Code Chat Custom Endpoint (full path)
http://127.0.0.1:8888/v1/chat/completions

Claude Code Anthropic door (origin only - no /v1)
http://127.0.0.1:8888

Recommended model ids
trim-claude-sonnet
trim-claude-haiku
trim-claude-opus
trim-gemini-flash
trim-gemini-flash-latest
gpt-4o-mini`,
      },
      { type: "h2", id: "doors", text: "Two doors" },
      {
        type: "ul",
        items: [
          "OpenAI door: POST /v1/chat/completions. DB adapters route by model (openai_compat hosts or anthropic_messages for Claude). When adapters are synced, unmatched models return a clear Trim error (no silent UPSTREAM_OPENAI_URL invent). When adapters are not synced, unmatched traffic uses UPSTREAM_OPENAI_URL.",
          "Anthropic door: POST /v1/messages → UPSTREAM_ANTHROPIC_URL (Claude Code, Anthropic SDK).",
        ],
      },
      {
        type: "h2",
        id: "providers",
        text: "GPT, Gemini, and other providers (DB routes by model)",
      },
      {
        type: "p",
        text: "Trim is not a marketplace and you should not edit cli/.env every day to switch hosts. Enabled rows in provider_adapters match the request model prefix and pick the upstream. dialect openai_compat keeps the OpenAI chat shape and uses upstream_base_url from the database. dialect anthropic_messages translates to Anthropic /v1/messages. After trim config sync, unmatched model prefixes fail closed with a clear Trim error (no silent host invent). UPSTREAM_OPENAI_URL remains the OpenAI-door default only when adapters are not synced.",
      },
      {
        type: "ul",
        items: [
          "GPT / OpenAI: openai_compat row (prefixes gpt-, o1-, o3-, o4-, chatgpt-) → https://api.openai.com. Cursor OpenAI key = OpenAI key for that turn.",
          "Gemini (Google OpenAI-compat): openai_compat row (gemini-, trim-gemini-, …) → Google …/v1beta/openai. Cursor OpenAI key = Google key for that turn.",
          "DeepSeek / Mistral: openai_compat rows with official upstream_base_url values. Same pattern for Groq / Azure / vLLM / Ollama: add or edit a provider_adapters row (Admin → Provider adapters); do not flip UPSTREAM_OPENAI_URL daily.",
          "Claude: anthropic_messages dialect (different /v1/messages API). Routes to UPSTREAM_ANTHROPIC_URL (or optional upstream_base_url).",
          "Amazon Bedrock / Google Vertex (non OpenAI-shaped): not shipped. Same plugin pattern later only if demand is real.",
        ],
      },
      {
        type: "note",
        text: "Cursor has one OpenAI API key field per session. When you switch model family (GPT vs Gemini vs Claude), put that provider’s key in the key field. Trim routes the host from the model id automatically after trim config sync. No OpenRouter.",
      },
      { type: "h2", id: "clients", text: "What to set (by client) - exact places" },
      {
        type: "p",
        text: "Same Trim listen host. Exact menus and files differ. Dedicated pages have full copy-paste blocks.",
      },
      {
        type: "table",
        headers: ["Client", "Exact place", "What to paste"],
        rows: [
          [
            "Cursor Agent",
            "Settings → Models → Override OpenAI Base URL",
            "trim cursor-link url (https://…/cl/<token>/v1) + model id (trim-claude-sonnet, …)",
          ],
          [
            "Continue",
            "~/.continue/config.yaml",
            "provider openai, apiBase …/v1, model trim-claude-sonnet",
          ],
          [
            "VS Code Chat",
            "Chat: Manage Language Models → Custom Endpoint → chatLanguageModels.json",
            "url …/v1/chat/completions + model id",
          ],
          [
            "Windsurf / Zed",
            "User settings.json (see dedicated pages)",
            "Base URL / api_url = …/v1",
          ],
          [
            "JetBrains",
            "AI Assistant OpenAI-compatible endpoint (hint: ~/.config/trim/jetbrains-openai.url)",
            "…/v1",
          ],
          ["Aider / shell", "OPENAI_BASE_URL env", "…/v1 + --model trim-claude-sonnet"],
          ["Claude Code", "ANTHROPIC_BASE_URL env", "http://127.0.0.1:8888 (no /v1)"],
        ],
      },
      {
        type: "ul",
        items: [
          "Continue: modern Continue uses config.yaml (not config.json). Open the Continue sidebar - not VS Code built-in Chat.",
          "VS Code Chat: store the API key via the Add Models wizard (${input:chat.lm.secret…}) - never paste a raw sk- key into chatLanguageModels.json.",
          "trim setup writes common settings when those product folders exist, and prints copy/paste blocks for Continue + VS Code Chat.",
        ],
      },
      {
        type: "h2",
        id: "supported-matrix",
        text: "Supported matrix (honest)",
      },
      {
        type: "ul",
        items: [
          "Continue Chat + Agent through Trim OpenAI door: supported (proven). Use config.yaml + capabilities tool_use for Agent.",
          "VS Code Chat Ask / normal chat through Custom Endpoint → Trim: supported when wizard secret + full …/v1/chat/completions URL + listed model id.",
          "VS Code Chat Agent + Custom Endpoint: not a reliable Trim-supported path yet (tool XML may print as plain text).",
          "Cursor Composer / chat through Trim …/v1: supported (primary path).",
          "Claude Code through Anthropic door: supported (ANTHROPIC_BASE_URL = origin, no /v1).",
          "Windsurf / Zed / Aider / curl OpenAI SDKs: supported when Base URL = Trim …/v1 (exact UI fields vary; trim setup helps when settings files exist).",
          "Dashboard Deep status “chrome frozen / no prose islands” on agent turns: normal - open Show savings detail for Deep stage.",
        ],
      },
      {
        type: "code",
        language: "bash",
        code: `# After trim start / trim setup
export OPENAI_BASE_URL="http://127.0.0.1:8888/v1"
# Claude Code: no /v1 suffix (it appends /v1/messages itself)
export ANTHROPIC_BASE_URL="http://127.0.0.1:8888"
trim config sync
# Confirm model ids your key can use:
curl -s http://127.0.0.1:8888/v1/models -H "Authorization: Bearer $KEY" | head`,
      },
      { type: "h2", id: "keys", text: "Which API key goes in the OpenAI key field" },
      {
        type: "ul",
        items: [
          "Gemini / GPT / other OpenAI-compat models: put that provider’s key (Google key for Gemini OpenAI-compat) in Cursor’s OpenAI API key field",
          "Claude models on the same OpenAI Base URL: put your Anthropic API key in that same OpenAI API key field. Trim’s adapter maps Authorization Bearer → x-api-key for Anthropic",
          "Claude Code (Anthropic door): use ANTHROPIC_API_KEY as usual; only the Base URL points at Trim",
        ],
      },
      {
        type: "h2",
        id: "anthropic-key-types",
        text: "Anthropic key types (workspace vs organization)",
      },
      {
        type: "p",
        text: "Anthropic issues workspace-scoped keys and organization (multi-workspace) keys. Trim supports both. This is not a daily .env flip.",
      },
      {
        type: "ul",
        items: [
          "Preferred for most users worldwide: create the API key inside a workspace in the Anthropic Console (Settings → API keys → scope to a workspace). Workspace-scoped keys need no Trim workspace setting. Cursor and Claude Code work with Base URL + key only.",
          "Organization / multi-workspace keys (Anthropic official): you must send anthropic-workspace-id (wrkspc_…) on every inference request. Find the id in Anthropic Console → Settings → Workspaces.",
          "Trim product path when forwarding (first wins): request header anthropic-workspace-id → process env TRIM_ANTHROPIC_WORKSPACE_ID. Empty is correct for workspace-scoped keys.",
          "anthropic-organization-id is an Anthropic response header only; clients do not send an organization id for Messages inference.",
        ],
      },
      {
        type: "p",
        text: "Installed CLI users (PATH / installer) usually do not edit the developer repo file cli/.env. For an org-scoped key + Cursor, set a one-time user environment variable, then restart trim start.",
      },
      {
        type: "code",
        language: "powershell",
        code: `# Windows (once per user) - then close/reopen the terminal and restart trim start
[System.Environment]::SetEnvironmentVariable(
  "TRIM_ANTHROPIC_WORKSPACE_ID",
  "wrkspc_YOUR_WORKSPACE_ID",
  "User"
)`,
      },
      {
        type: "code",
        language: "bash",
        code: `# macOS / Linux (once) - then open a new shell and restart trim start
echo 'export TRIM_ANTHROPIC_WORKSPACE_ID=wrkspc_YOUR_WORKSPACE_ID' >> ~/.zshrc  # or ~/.bashrc
source ~/.zshrc
trim start`,
      },
      {
        type: "ul",
        items: [
          "Power users / scripts: send header anthropic-workspace-id: wrkspc_… on each request (no Trim env needed).",
          "Developers working inside the Trim repo may instead put TRIM_ANTHROPIC_WORKSPACE_ID in cli/.env (gitignored); same meaning as the OS user env var.",
        ],
      },
      {
        type: "note",
        text: "Never paste https://api.anthropic.com into Cursor Override OpenAI Base URL. Keep Base URL = Trim so compression and the Messages adapter run. See Base URL troubleshooting and Provider adapters.",
      },
      { type: "h2", id: "remote", text: "Remote / Cursor Agent (Trim Cursor Link)" },
      {
        type: "p",
        text: "For Cursor Agent (and any client that cannot call 127.0.0.1), use Trim Cursor Link - an automatic Cloudflare-style tunnel managed by Trim while trim start is running. Point Override OpenAI Base URL at the printed https://…/cl/<token>/v1 URL. Doors, compression, and adapters stay on your PC; only the hostname is public. Do not DIY cloudflared ingress for everyday use. Full guide: Trim Cursor Link + Cursor setup.",
      },
      {
        type: "code",
        language: "bash",
        code: `trim cursor-link enable
trim start
trim cursor-link url
# Paste into Cursor → Settings → Models → Override OpenAI Base URL`,
      },
    ],
  ),

  "ide/custom-models": page(
    "Custom models",
    "Create clear trim-… model names in Cursor, Continue, VS Code Chat, and other IDEs - with copyable examples and exact settings places.",
    [
      {
        type: "p",
        text: "Trim works best when your IDE shows short, memorable model names such as trim-claude-sonnet or trim-gemini-flash-latest. Those names are either (1) Trim cloud aliases that map to a real upstream model, or (2) custom entries you create in the IDE that still send a valid model id through Trim. This page is for everyone: non-technical users can copy-paste; technical users can add their own aliases.",
      },
      {
        type: "callout",
        title: "Two layers (do not confuse them)",
        text: "IDE custom model = a row you add in Cursor Models / Continue config / VS Code Chat so the picker shows a friendly name. Trim alias = a database mapping (trim-claude-sonnet → a real Claude id) maintained under Admin → Billing → Provider adapters (and openai_model_aliases for Gemini). Everyday users pick shipped Trim aliases. Operators create new worldwide aliases.",
      },
      {
        type: "h2",
        id: "recommended",
        text: "Recommended model ids (copy)",
      },
      {
        type: "p",
        text: "Paste one of these into your IDE model field. Prefer Trim aliases over long dated provider ids - aliases stay correct when providers retire versions.",
      },
      {
        type: "code",
        language: "text",
        code: `trim-claude-sonnet
trim-claude-haiku
trim-claude-opus
trim-gemini-flash
trim-gemini-flash-latest
gpt-4o-mini`,
      },
      {
        type: "table",
        headers: ["Model id you type", "What it is for", "Key you put in the IDE"],
        rows: [
          ["trim-claude-sonnet", "Strong Claude coding (alias)", "Anthropic API key"],
          ["trim-claude-haiku", "Fast / cheaper Claude (alias)", "Anthropic API key"],
          ["trim-claude-opus", "Highest Claude quality (alias)", "Anthropic API key"],
          [
            "trim-gemini-flash / trim-gemini-flash-latest",
            "Gemini Flash via Trim (alias)",
            "Google API key",
          ],
          ["gpt-4o-mini (or your gpt-… id)", "OpenAI GPT via Trim", "OpenAI API key"],
        ],
      },
      {
        type: "h2",
        id: "naming",
        text: "Custom naming idea (best practice)",
      },
      {
        type: "p",
        text: "When you add your own IDE entries, use a stable pattern so teammates recognize Trim traffic:",
      },
      {
        type: "code",
        language: "text",
        code: `trim-<provider>-<tier>

Shipped-style examples (prefer these today):
trim-claude-sonnet
trim-claude-haiku
trim-claude-opus
trim-gemini-flash
trim-gemini-flash-latest

Custom naming ideas you can create in the IDE (and ask ops to alias worldwide):
trim-gpt-4o-mini
trim-gpt-4.1
trim-gpt-3.2
trim-gemini-fast
trim-gemini-fast-latest`,
      },
      {
        type: "ul",
        items: [
          "Start with trim- so it is obvious the request should go through Trim, not the vendor directly",
          "Use the provider family next (claude, gpt, gemini)",
          "Use a short tier word (sonnet, haiku, flash, fast, latest, 4o-mini, 3.2) - avoid long dated stamps in the display name",
          "The id you send must still be listed by GET /v1/models or match a Trim alias / adapter prefix - Trim does not invent hosts for unknown names",
          "IDE-only display names that Trim has never aliased will fail closed until an operator adds the alias (Admin → Billing → Provider adapters / openai_model_aliases)",
        ],
      },
      {
        type: "h2",
        id: "list-models",
        text: "See what Trim accepts on your machine",
      },
      {
        type: "code",
        language: "bash",
        code: `trim start
trim config sync
curl -s http://127.0.0.1:8888/v1/models -H "Authorization: Bearer YOUR_PROVIDER_KEY"`,
      },
      {
        type: "h2",
        id: "cursor",
        text: "Cursor - exact place to add a custom model",
      },
      {
        type: "ol",
        items: [
          "Make sure Trim is listening: trim start (or Always-on)",
          "Open Cursor → Settings → Models",
          "Enable Override OpenAI Base URL only (do not also enable Cursor Anthropic BYOK)",
          "Base URL → paste the copy block below",
          "Add / enable a model whose id is trim-claude-sonnet (or another id from the list above)",
          "OpenAI API key field → paste the provider key that matches that model family",
          "Select that model in Chat / Composer and send a short test",
        ],
      },
      {
        type: "code",
        language: "text",
        code: "http://127.0.0.1:8888/v1",
      },
      {
        type: "callout",
        title: "Key must match the model",
        text: "Claude / trim-claude-* → Anthropic key. GPT / gpt-* → OpenAI key. Gemini / trim-gemini-* → Google key. Cursor has one OpenAI key field per session - switch the key when you switch family.",
      },
      {
        type: "h2",
        id: "continue",
        text: "Continue - exact file + copy-paste models",
      },
      {
        type: "p",
        text: "Exact file: Windows C:\\Users\\<you>\\.continue\\config.yaml · macOS/Linux ~/.continue/config.yaml. Open the Continue sidebar (not built-in Chat), then Reload config.",
      },
      {
        type: "code",
        language: "yaml",
        code: `name: Main Config
version: 1.0.0
schema: v1
models:
  - name: Claude Sonnet via Trim
    provider: openai
    model: trim-claude-sonnet
    apiBase: http://127.0.0.1:8888/v1
    apiKey: sk-ant-YOUR_ANTHROPIC_KEY
    roles: [chat, edit, apply]
    capabilities: [tool_use]
    useResponsesApi: false

  - name: Claude Haiku via Trim
    provider: openai
    model: trim-claude-haiku
    apiBase: http://127.0.0.1:8888/v1
    apiKey: sk-ant-YOUR_ANTHROPIC_KEY
    roles: [chat, edit, apply]
    capabilities: [tool_use]
    useResponsesApi: false

  - name: GPT via Trim
    provider: openai
    model: gpt-4o-mini
    apiBase: http://127.0.0.1:8888/v1
    apiKey: sk-YOUR_OPENAI_KEY
    roles: [chat, edit, apply]
    capabilities: [tool_use]
    useResponsesApi: false

  - name: Gemini Flash via Trim
    provider: openai
    model: trim-gemini-flash-latest
    apiBase: http://127.0.0.1:8888/v1
    apiKey: YOUR_GOOGLE_API_KEY
    roles: [chat, edit, apply]
    capabilities: [tool_use]
    useResponsesApi: false`,
      },
      {
        type: "h2",
        id: "vscode",
        text: "VS Code Chat - exact place + copy-paste",
      },
      {
        type: "ol",
        items: [
          "Command Palette → Chat: Manage Language Models → Add Models → Custom Endpoint → Chat Completions",
          "When asked for an API key, paste the provider key (wizard stores ${input:chat.lm.secret…} - never paste raw sk- into the JSON file)",
          "Edit chatLanguageModels.json (Windows %APPDATA%\\Code\\User\\chatLanguageModels.json; macOS ~/Library/Application Support/Code/User/chatLanguageModels.json; Linux ~/.config/Code/User/chatLanguageModels.json)",
          "Set each model id + full url as below → Save → Developer: Reload Window → New Chat → pick the model → Ask mode",
        ],
      },
      {
        type: "code",
        language: "json",
        code: `[
  {
    "name": "Trim",
    "vendor": "customendpoint",
    "apiKey": "\${input:chat.lm.secret.LEAVE_WIZARD_VALUE}",
    "apiType": "chat-completions",
    "models": [
      {
        "id": "trim-claude-sonnet",
        "name": "Claude Sonnet via Trim",
        "url": "http://127.0.0.1:8888/v1/chat/completions",
        "toolCalling": false,
        "vision": false,
        "maxInputTokens": 128000,
        "maxOutputTokens": 16000
      },
      {
        "id": "trim-claude-haiku",
        "name": "Claude Haiku via Trim",
        "url": "http://127.0.0.1:8888/v1/chat/completions",
        "toolCalling": false,
        "vision": false,
        "maxInputTokens": 128000,
        "maxOutputTokens": 16000
      },
      {
        "id": "trim-gemini-flash-latest",
        "name": "Gemini Flash via Trim",
        "url": "http://127.0.0.1:8888/v1/chat/completions",
        "toolCalling": false,
        "vision": false,
        "maxInputTokens": 128000,
        "maxOutputTokens": 16000
      }
    ]
  }
]`,
      },
      {
        type: "h2",
        id: "other-ides",
        text: "Windsurf, Zed, JetBrains, Aider",
      },
      {
        type: "ul",
        items: [
          "Windsurf: User settings.json → openai.baseUrl / windsurf.openai.baseUrl = http://127.0.0.1:8888/v1, then pick/add the model id in Windsurf’s model UI",
          "Zed: settings.json → language_models.openai.api_url = http://127.0.0.1:8888/v1 and list available_models names such as trim-claude-sonnet",
          "JetBrains: Settings → Tools → AI Assistant → OpenAI-compatible endpoint → paste http://127.0.0.1:8888/v1 (hint file from trim setup: ~/.config/trim/jetbrains-openai.url)",
          "Aider / shell: OPENAI_BASE_URL=http://127.0.0.1:8888/v1 and pass --model trim-claude-sonnet (or your id)",
        ],
      },
      {
        type: "h2",
        id: "where-settings",
        text: "Where everything lives (map)",
      },
      {
        type: "table",
        headers: ["What", "Exact place"],
        rows: [
          [
            "Always-on preference",
            "https://use-trim.com/dashboard → Preferences → Start Trim with your IDE",
          ],
          ["Plans / prices / quotas", "https://use-trim.com/dashboard → Billing (or Plans)"],
          [
            "Trim IDE extension API key",
            "Dashboard → Settings → API keys (not your OpenAI/Anthropic/Google key)",
          ],
          [
            "Provider keys for chat",
            "IDE model settings key field (or Continue apiKey / VS Code wizard secret)",
          ],
          ["Cursor Base URL + models", "Cursor → Settings → Models"],
          ["Continue models", "~/.continue/config.yaml"],
          [
            "VS Code Chat models",
            "Code/User/chatLanguageModels.json via Chat: Manage Language Models",
          ],
          [
            "Worldwide Trim aliases (ops)",
            "Admin → Billing → Provider adapters (+ openai_model_aliases)",
          ],
          ["Local adapter sync", "Terminal: trim config sync"],
        ],
      },
      {
        type: "h2",
        id: "ops-aliases",
        text: "Operators: create a new Trim alias for everyone",
      },
      {
        type: "ol",
        items: [
          "Admin console → Billing → Provider adapters",
          "For Claude-style names: add/update model_aliases on the anthropic_messages adapter (example trim-claude-sonnet → current official Claude id)",
          "For Gemini-style names: add a row in openai_model_aliases (client_model → upstream_model)",
          "Optionally advertise the id in provider_discoverable_models so GET /v1/models lists it",
          "Users run trim config sync (or restart trim start) to pull the new alias",
        ],
      },
      {
        type: "note",
        text: "Creating only an IDE display name is not enough if Trim has never seen that model id. Unknown ids fail closed with a clear error after adapters are synced. Prefer shipped trim-… aliases, or ask an operator to add one.",
      },
      {
        type: "note",
        text: "Next: Cursor setup, Continue setup, VS Code setup, Connect any IDE, Provider adapters.",
      },
    ],
  ),

  "ide/cursor": page(
    "Cursor setup",
    "Cursor Agent needs Trim Cursor Link (public HTTPS to local Trim). Localhost alone is blocked by Cursor’s private-network policy. Copy buttons on every block.",
    [
      {
        type: "p",
        text: "Cursor routes many Agent requests through Cursor’s cloud. That cloud cannot call http://127.0.0.1:8888/v1 - you get “Access to private networks is forbidden.” Trim Cursor Link fixes this: an automatic Cloudflare-style tunnel door to Trim still running on your PC. Compression stays local. You never DIY cloudflared config for everyday use.",
      },
      {
        type: "callout",
        title: "Critical - do not paste localhost into Cursor Agent",
        text: "Override OpenAI Base URL for Cursor Agent must be the Trim Cursor Link URL from trim cursor-link url (https://…/cl/<token>/v1), not http://127.0.0.1:8888/v1. Localhost remains correct for Claude Code CLI and other on-device clients. Product language “Connect Cursor (Agent)” = trim cursor-link enable (also shown on the local meter http://127.0.0.1:8888/dashboard with a Copy button).",
      },
      {
        type: "table",
        headers: ["Setting", "Exact place", "What to use"],
        rows: [
          [
            "Cursor Agent Base URL",
            "Settings → Models → Override OpenAI Base URL",
            "trim cursor-link url  (https://…/v1)",
          ],
          [
            "Model id",
            "Settings → Models → model list / add model",
            "trim-claude-sonnet (see Custom models)",
          ],
          ["Provider API key", "Settings → Models → OpenAI API key field", "Matching provider key"],
          ["Local Trim", "Terminal: trim start", "Must be running for the link to work"],
          [
            "Cursor Link preference",
            "Terminal: trim cursor-link enable",
            "Stays on across restarts",
          ],
          ["Plans / prices", "use-trim.com/dashboard → Billing", "Cloud billing"],
        ],
      },
      {
        type: "h2",
        id: "how",
        text: "What happens (plain English)",
      },
      {
        type: "code",
        language: "text",
        code: `Cursor Agent (Cursor cloud)
        │
        │  Override Base URL = https://YOUR_LINK.trycloudflare.com/cl/YOUR_TOKEN/v1
        ▼
Trim Cursor Link (public HTTPS door)
        │
        ▼
Local Trim on YOUR PC  (http://127.0.0.1:8888)
  • Fast Mode (+ optional Segmented Allowlist Deep: each prose island in place, order preserved) on this machine
  • tools/paths/schemas stay frozen; check Deep stage on the local meter
  • local meter: http://127.0.0.1:8888/dashboard
        │
        ▼
Anthropic / OpenAI / Google
        │
        ▼
Reply → local Trim → tunnel → Cursor`,
      },
      {
        type: "h2",
        id: "steps",
        text: "Setup steps (do in order)",
      },
      {
        type: "h3",
        id: "step0-cloudflared",
        text: "Step 0 - cloudflared helper (usually automatic)",
      },
      {
        type: "p",
        text: "Where: Terminal (usually skip). What: Cloudflare’s cloudflared helper. Why: Trim Cursor Link uses it under the hood. How: Preferred path is skip this step and go to Step 1; Trim auto-downloads cloudflared into ~/.trim/bin on first enable/start (Windows, macOS, Linux). winget is not required and is often missing on Windows. Optional manual installs below if auto-download is blocked.",
      },
      {
        type: "code",
        language: "powershell",
        code: `# Windows - preferred (no winget) - Copy this
# Skip to Step 1: trim cursor-link enable && trim start
# Trim downloads into %USERPROFILE%\\.trim\\bin when missing

# Manual fallback if auto-download is blocked (amd64) - Copy this
$dir = Join-Path $HOME ".trim\\bin"
New-Item -ItemType Directory -Force -Path $dir | Out-Null
$dest = Join-Path $dir "cloudflared.exe"
Invoke-WebRequest -Uri "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-windows-amd64.exe" -OutFile $dest
& $dest --version
# Optional: $env:TRIM_CLOUDFLARED = $dest
# Windows on ARM: same amd64 .exe (emulation; no native arm64 Windows build from Cloudflare)
# Only if winget exists: winget install --id Cloudflare.cloudflared`,
      },
      {
        type: "code",
        language: "bash",
        code: `# macOS - preferred (no Homebrew required) - Copy this
# Skip to Step 1: trim cursor-link enable && trim start
# Trim downloads into ~/.trim/bin when missing

# Manual fallback without brew (Apple Silicon) - Copy this
mkdir -p ~/.trim/bin
curl -fsSL -o /tmp/cloudflared.tgz \\
  https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-darwin-arm64.tgz
tar -xzf /tmp/cloudflared.tgz -C ~/.trim/bin
chmod +x ~/.trim/bin/cloudflared
~/.trim/bin/cloudflared --version
# Intel Mac: cloudflared-darwin-amd64.tgz
# Only if Homebrew exists: brew install cloudflared`,
      },
      {
        type: "code",
        language: "bash",
        code: `# Linux - preferred - Copy this
# Skip to Step 1: trim cursor-link enable && trim start
# Trim downloads into ~/.trim/bin when missing

# Manual fallback (amd64) - Copy this
mkdir -p ~/.trim/bin
curl -fsSL -o ~/.trim/bin/cloudflared \\
  https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64
chmod +x ~/.trim/bin/cloudflared
~/.trim/bin/cloudflared --version
# arm64: cloudflared-linux-arm64
# Or: export TRIM_CLOUDFLARED=/usr/local/bin/cloudflared`,
      },
      {
        type: "h3",
        id: "step1",
        text: "Step 1 - Enable Trim Cursor Link",
      },
      {
        type: "p",
        text: "Where: Terminal. What: turn on the preference so every trim start brings the public door up. How:",
      },
      {
        type: "code",
        language: "bash",
        code: `# Windows / macOS / Linux - Copy this
trim cursor-link enable
trim cursor-link status`,
      },
      {
        type: "h3",
        id: "step2",
        text: "Step 2 - Start Trim (tunnel lifecycle starts with it)",
      },
      {
        type: "p",
        text: "Where: Terminal. What: start local Trim; when Cursor Link is enabled, Trim also starts the tunnel and prints Cursor Agent Base URL. Why: tunnel alone does nothing - local Trim must be listening. How:",
      },
      {
        type: "code",
        language: "bash",
        code: `# Windows / macOS / Linux - Copy this
trim start
trim cursor-link url`,
      },
      {
        type: "p",
        text: "What you should see: a https://….trycloudflare.com/cl/<token>/v1 URL (secret path token included). That URL stays up while Trim is running; trim stop tears it down with the proxy.",
      },
      {
        type: "h3",
        id: "step3",
        text: "Step 3 - Point Cursor at the Cursor Link URL",
      },
      {
        type: "ol",
        items: [
          "Cursor → Settings → Models",
          "Turn on Override OpenAI Base URL",
          "Paste the output of trim cursor-link url (must be https://…/v1 - not 127.0.0.1)",
          "Add or pick a model such as trim-claude-haiku / trim-claude-sonnet",
          "Paste the matching provider API key into the OpenAI API key field",
          "Use Agent / Chat / Composer as usual",
        ],
      },
      {
        type: "h2",
        id: "copy-base-url",
        text: "Copy: Base URL shapes (know the difference)",
      },
      {
        type: "code",
        language: "text",
        code: `# Cursor Agent (REQUIRED) - from: trim cursor-link url
# Includes a secret path token (/cl/<token>/) - rotate with: trim cursor-link rotate
https://YOUR_SUBDOMAIN.trycloudflare.com/cl/YOUR_TOKEN/v1

# NOT for Cursor Agent (causes “private networks forbidden”)
http://127.0.0.1:8888/v1

# Still correct for Claude Code CLI / on-device tools (origin, often no /v1 for Anthropic door)
http://127.0.0.1:8888`,
      },
      {
        type: "h2",
        id: "copy-models",
        text: "Copy: model ids to add in Settings → Models",
      },
      {
        type: "code",
        language: "text",
        code: `trim-claude-sonnet
trim-claude-haiku
trim-claude-opus
trim-gemini-flash
trim-gemini-flash-latest
gpt-4o-mini`,
      },
      {
        type: "h2",
        id: "keys-by-model",
        text: "Which key goes in the OpenAI API key field?",
      },
      {
        type: "table",
        headers: ["Model you pick", "Key to paste", "What Trim does"],
        rows: [
          ["gpt-*", "OpenAI API key", "Routes to OpenAI (openai_compat)"],
          [
            "trim-claude-* / claude-* / anthropic/claude-*",
            "Anthropic API key",
            "Translates to Anthropic Messages",
          ],
          ["trim-gemini-* / gemini-*", "Google API key", "Routes to Google OpenAI-compat"],
        ],
      },
      {
        type: "callout",
        title: "Anthropic key tip",
        text: "Prefer a workspace-scoped Anthropic key. Org-scoped keys need TRIM_ANTHROPIC_WORKSPACE_ID once as a user env var - see Connect any IDE → Anthropic key types.",
      },
      {
        type: "h2",
        id: "everyday",
        text: "Everyday loop",
      },
      {
        type: "ol",
        items: [
          "If local dashboard is down: trim start (Cursor Link comes back if enabled)",
          "Confirm URL: trim cursor-link url → still pasted in Cursor",
          "Chat / Agent as usual",
          "Check savings: http://127.0.0.1:8888/dashboard",
          "Stop both Trim + tunnel: trim stop",
        ],
      },
      {
        type: "code",
        language: "bash",
        code: `# Everyday - Copy as needed
trim start
trim cursor-link url
# … use Cursor …
trim stop`,
      },
      { type: "h2", id: "verify", text: "Verify it works" },
      {
        type: "ol",
        items: [
          "trim cursor-link status shows enabled + a https://…/v1 URL",
          "Cursor Override Base URL matches that URL (not 127.0.0.1)",
          "Send Agent/Composer: Reply with exactly: ok",
          "Open http://127.0.0.1:8888/dashboard - Requests increase (proves work is local)",
          "Open Show savings detail - Deep status / Deep stage (prose islands; order preserved beside paths) explain whole-body 0% when chrome dominates",
          "If you still see “private networks forbidden”, the Base URL is still localhost - re-paste trim cursor-link url",
        ],
      },
      {
        type: "h2",
        id: "troubleshoot",
        text: "If something fails",
      },
      {
        type: "table",
        headers: ["Symptom", "Cause", "Fix"],
        rows: [
          [
            "Access to private networks is forbidden",
            "Cursor Base URL is still 127.0.0.1",
            "Paste trim cursor-link url (https://…/v1)",
          ],
          [
            "No public URL",
            "cloudflared missing or Trim down",
            "trim start (auto-downloads cloudflared); or set TRIM_CLOUDFLARED; then trim cursor-link status",
          ],
          [
            "URL changed after restart",
            "Quick tunnel hostname rotates",
            "Run trim cursor-link url and update Cursor Base URL",
          ],
          [
            "Requests not on local meter",
            "Wrong Base URL or Trim stopped",
            "trim start; confirm Override URL from cursor-link url",
          ],
          [
            "Tunnel up but Cursor fails auth",
            "Wrong provider key for model",
            "Paste Anthropic/OpenAI/Google key matching the model id",
          ],
          [
            "0% Saved on Agent turns",
            "Often normal: tools/system chrome frozen; Segmented Allowlist Deep runs on each last-user prose island in place (order preserved beside paths)",
            "Open http://127.0.0.1:8888/dashboard → Show savings detail (Deep status + Deep stage)",
          ],
        ],
      },
      {
        type: "callout",
        title: "After Cursor updates",
        text: "Model settings sometimes reset. Re-check Override Base URL (must stay on Cursor Link) and your trim-… model entries.",
      },
      {
        type: "callout",
        title: "Cursor vs Continue in the same editor",
        text: "Continue is a separate path (~/.continue/config.yaml). Cursor Override does not configure Continue. Prefer one intentional path per chat panel.",
      },
      {
        type: "note",
        text: "Full CLI reference: Trim Cursor Link. Also: Custom models, Always-on with IDE, Connect any IDE, Claude Code (CLI only), Base URL troubleshooting.",
      },
    ],
  ),

  "ide/vscode": page(
    "VS Code setup",
    "Wire VS Code built-in Chat (Custom Endpoint) and point other OpenAI-compatible clients at Trim; install the Trim IDE extension for auto-start.",
    [
      {
        type: "p",
        text: "VS Code has two common Trim paths. (1) Built-in Chat Language Models → Custom Endpoint (BYOK). (2) The Continue extension (separate sidebar - see Continue setup). Do not confuse them: Custom Endpoint labels look like Custom Endpoint/Trim/… with 0 credits; Continue shows the model name from ~/.continue/config.yaml only. Custom trim-… model rows: Custom models.",
      },
      {
        type: "table",
        headers: ["What", "Exact place"],
        rows: [
          [
            "Custom Endpoint models",
            "Command Palette → Chat: Manage Language Models → Add Models → Custom Endpoint",
          ],
          ["JSON file (Windows)", "%APPDATA%\\Code\\User\\chatLanguageModels.json"],
          ["JSON file (macOS)", "~/Library/Application Support/Code/User/chatLanguageModels.json"],
          ["JSON file (Linux)", "~/.config/Code/User/chatLanguageModels.json"],
          ["Plans / prices", "https://use-trim.com/dashboard → Billing"],
        ],
      },
      {
        type: "h2",
        id: "which-path",
        text: "Which path should I use?",
      },
      {
        type: "ul",
        items: [
          "Want agent edits that read/write files reliably in VS Code today? Prefer Continue through Trim (Continue setup).",
          "Want built-in VS Code Chat Ask with your own key through Trim? Use Custom Endpoint below.",
          "VS Code Chat Agent + Custom Endpoint is not a Trim-supported guarantee yet (models may print raw tool tags).",
        ],
      },
      {
        type: "h2",
        id: "custom-endpoint",
        text: "Built-in Chat: Custom Endpoint (official)",
      },
      {
        type: "ol",
        items: [
          "trim start (or Always-on with IDE). Confirm http://127.0.0.1:8888/dashboard loads.",
          "Ctrl+Shift+P (Cmd+Shift+P) → Chat: Manage Language Models",
          "Add Models → Custom Endpoint → Chat Completions",
          "When prompted for an API key, paste the provider key that matches your model (Anthropic sk-ant-… for Claude / trim-claude-*). The wizard stores it as ${input:chat.lm.secret…} - leave that placeholder in the JSON; do not paste the raw key into the file.",
          "VS Code opens chatLanguageModels.json. Set id / name / url as in the copy-paste below (keep the wizard apiKey line).",
          "Save → Developer: Reload Window → New Chat → pick Claude via Trim (or your model name) → Ask mode → send ok",
          "Confirm http://127.0.0.1:8888/dashboard Requests increase and Last door is openai or openai_to_anthropic",
        ],
      },
      {
        type: "p",
        text: "Exact file (Windows): %APPDATA%\\Code\\User\\chatLanguageModels.json. macOS: ~/Library/Application Support/Code/User/chatLanguageModels.json. Linux: ~/.config/Code/User/chatLanguageModels.json.",
      },
      {
        type: "p",
        text: "Use the Copy button on the starter below. Add more model rows using the same pattern (see Custom models). Keep the wizard apiKey line unchanged.",
      },
      {
        type: "code",
        language: "json",
        code: `[
  {
    "name": "Trim",
    "vendor": "customendpoint",
    "apiKey": "\${input:chat.lm.secret.LEAVE_WIZARD_VALUE}",
    "apiType": "chat-completions",
    "models": [
      {
        "id": "trim-claude-sonnet",
        "name": "Claude Sonnet via Trim",
        "url": "http://127.0.0.1:8888/v1/chat/completions",
        "toolCalling": false,
        "vision": false,
        "maxInputTokens": 128000,
        "maxOutputTokens": 16000
      },
      {
        "id": "trim-claude-haiku",
        "name": "Claude Haiku via Trim",
        "url": "http://127.0.0.1:8888/v1/chat/completions",
        "toolCalling": false,
        "vision": false,
        "maxInputTokens": 128000,
        "maxOutputTokens": 16000
      },
      {
        "id": "trim-gemini-flash-latest",
        "name": "Gemini Flash via Trim",
        "url": "http://127.0.0.1:8888/v1/chat/completions",
        "toolCalling": false,
        "vision": false,
        "maxInputTokens": 128000,
        "maxOutputTokens": 16000
      }
    ]
  }
]`,
      },
      {
        type: "code",
        language: "text",
        code: "http://127.0.0.1:8888/v1/chat/completions",
      },
      {
        type: "callout",
        title: "Critical details",
        text: "url must be the full chat completions path (…/v1/chat/completions), not only …/v1. Model id must appear in GET /v1/models for your key (prefer trim-claude-sonnet / trim-claude-haiku / trim-gemini-flash-latest). Raw sk- keys in apiKey are silently ignored by VS Code and cause 401 empty Bearer - apiKey must stay as the wizard ${input:chat.lm.secret…} placeholder. Do not add requestHeaders.Authorization with a pasted sk- either. Prefer workspace-scoped Anthropic keys.",
      },
      {
        type: "h2",
        id: "do-not",
        text: "Do not",
      },
      {
        type: "ul",
        items: [
          "Do not put a raw sk-… string in chatLanguageModels.json apiKey (use the wizard ${input:…} secret).",
          "Do not paste sk- into models[].requestHeaders.Authorization - use the wizard secret only.",
          "Do not use url ending only in /v1 for Custom Endpoint - Chromium clients need …/v1/chat/completions.",
          "Do not pick a model id your key cannot use (example: some accounts reject claude-sonnet-4-5; use trim-claude-sonnet or list /v1/models).",
          "Do not expect Agent mode + Custom Endpoint to match Continue Agent quality yet.",
          "Do not confuse this panel with Continue (Custom Endpoint/… • 0 credits ≠ Claude via Trim from config.yaml).",
        ],
      },
      {
        type: "h2",
        id: "other-clients",
        text: "Other OpenAI-compatible extensions in VS Code",
      },
      {
        type: "ol",
        items: [
          "Ensure the proxy listens via Always-on or trim start",
          "Set that client’s OpenAI-compatible Base URL to http://127.0.0.1:8888/v1",
          "Put the provider key the model needs (Anthropic for Claude on this door)",
          "Send a request and confirm Trim dashboard / trim stats",
        ],
      },
      {
        type: "code",
        language: "text",
        code: `OpenAI-compatible Base URL (most extensions)
http://127.0.0.1:8888/v1

VS Code Chat Custom Endpoint url (full path)
http://127.0.0.1:8888/v1/chat/completions`,
      },
      {
        type: "callout",
        title: "Low or 0% Saved on agent-style turns",
        text: "Whole-request Saved % can look low when tools or long chrome dominate. Live Deep uses Segmented Allowlist Deep: each last-user prose island in place (order preserved beside paths); tools/paths/schemas stay frozen. Open http://127.0.0.1:8888/dashboard → Show savings detail for Deep status and Deep stage.",
      },
      { type: "h2", id: "extension", text: "Install the Trim IDE extension" },
      {
        type: "p",
        text: "The Trim IDE extension works in VS Code and Cursor. It can auto-start the local Fast Mode proxy when the IDE opens (honoring Start Trim with your IDE), and it posts tab acceptance and AI LOC counters so the dashboard stays accurate. Full prompt compression still requires the proxy Base URL / Custom Endpoint override.",
      },
      {
        type: "ol",
        items: [
          "Install Trim IDE (publisher usetrim) from the VS Code Marketplace or Open VSX; sideload a VSIX from extensions/trim-ide only for air-gapped / offline machines - see Trim IDE extension docs",
          "In VS Code: Extensions → search Trim IDE. In Cursor: install usetrim.trim-ide from Open VSX. Air-gapped: … → Install from VSIX",
          "Settings → Trim: Api Url → your API origin (for example https://api.use-trim.com)",
          "Command Palette → Trim: Set API Key (create a key in Dashboard → Settings → API keys)",
          "Leave trim.autoStartWithIde on follow (cloud preference) unless you need a local override",
        ],
      },
      {
        type: "callout",
        title: "Honest limits",
        text: "Not every ghost-text show/accept event is exposed by the editor. The extension counts explicit commands plus optional edit-based estimates. Token savings still come from the local proxy path. Content-Encoding mismatches on Custom Endpoint were fixed in the Trim proxy (rebuffered bodies no longer advertise gzip).",
      },
    ],
  ),

  "ide/continue": page(
    "Continue setup",
    "Configure the Continue.dev extension through Trim with ~/.continue/config.yaml (exact file, copy-paste, Chat vs Agent).",
    [
      {
        type: "p",
        text: "Continue is a first-class Trim client for VS Code and Cursor. It speaks OpenAI chat completions. Point apiBase at Trim …/v1. Prefer modern Continue’s config.yaml (schema v1). Older Continue used config.json; if both exist, edit the file your Continue version actually loads (usually config.yaml on Continue 1.x/2.x). Add as many trim-… models as you want - see Custom models.",
      },
      {
        type: "table",
        headers: ["What", "Exact place"],
        rows: [
          ["Config file (Windows)", "C:\\Users\\<you>\\.continue\\config.yaml"],
          ["Config file (macOS / Linux)", "~/.continue/config.yaml"],
          ["Reload", "Command Palette → Continue: Reload config"],
          ["Chat panel", "Continue sidebar (not VS Code built-in Chat)"],
          ["Plans / prices", "https://use-trim.com/dashboard → Billing"],
        ],
      },
      {
        type: "h2",
        id: "exact-file",
        text: "Exact file",
      },
      {
        type: "ul",
        items: [
          "Windows: C:\\Users\\<you>\\.continue\\config.yaml",
          "macOS / Linux: ~/.continue/config.yaml",
          "Open Continue once if the folder does not exist yet",
          "trim setup updates Continue when ~/.continue exists (prefers config.yaml when present)",
        ],
      },
      {
        type: "h2",
        id: "copy-paste",
        text: "Copy these config blocks (Claude, GPT, Gemini via Trim)",
      },
      {
        type: "p",
        text: "Use the Copy button, then paste into ~/.continue/config.yaml. Replace the apiKey values. Add or remove model blocks as you like - see Custom models for naming ideas (trim-claude-haiku, trim-gemini-flash-latest, …).",
      },
      {
        type: "code",
        language: "yaml",
        code: `name: Main Config
version: 1.0.0
schema: v1
models:
  - name: Claude Sonnet via Trim
    provider: openai
    model: trim-claude-sonnet
    apiBase: http://127.0.0.1:8888/v1
    apiKey: sk-ant-YOUR_ANTHROPIC_KEY
    roles:
      - chat
      - edit
      - apply
    capabilities:
      - tool_use
    useResponsesApi: false

  - name: Claude Haiku via Trim
    provider: openai
    model: trim-claude-haiku
    apiBase: http://127.0.0.1:8888/v1
    apiKey: sk-ant-YOUR_ANTHROPIC_KEY
    roles:
      - chat
      - edit
      - apply
    capabilities:
      - tool_use
    useResponsesApi: false

  - name: GPT via Trim
    provider: openai
    model: gpt-4o-mini
    apiBase: http://127.0.0.1:8888/v1
    apiKey: sk-YOUR_OPENAI_KEY
    roles:
      - chat
      - edit
      - apply
    capabilities:
      - tool_use
    useResponsesApi: false

  - name: Gemini Flash via Trim
    provider: openai
    model: trim-gemini-flash-latest
    apiBase: http://127.0.0.1:8888/v1
    apiKey: YOUR_GOOGLE_API_KEY
    roles:
      - chat
      - edit
      - apply
    capabilities:
      - tool_use
    useResponsesApi: false`,
      },
      {
        type: "ul",
        items: [
          "provider stays openai (Continue’s OpenAI client). Trim’s anthropic_messages adapter translates Claude.",
          "apiKey must match the model family (Anthropic for trim-claude-*, OpenAI for gpt-*, Google for trim-gemini-*).",
          "model must be allowed for your key - run GET /v1/models. Prefer Trim aliases: trim-claude-sonnet, trim-claude-haiku, trim-claude-opus, trim-gemini-flash-latest.",
          "useResponsesApi: false keeps Continue on /v1/chat/completions (required for Trim).",
          "capabilities: tool_use enables Continue Agent tool execution (without it, Agent may print raw invoke XML).",
        ],
      },
      {
        type: "h2",
        id: "steps",
        text: "Steps",
      },
      {
        type: "ol",
        items: [
          "trim start (dashboard http://127.0.0.1:8888/dashboard)",
          "Install the Continue extension in VS Code or Cursor",
          "Edit ~/.continue/config.yaml with the block above (real keys)",
          "Command Palette → Continue: Reload config (or Developer: Reload Window)",
          "Open the Continue sidebar (not VS Code built-in Chat)",
          "Select Claude Sonnet via Trim (or GPT / Gemini / Haiku)",
          "Send: Reply with exactly: ok",
          "Confirm Trim Last door openai_to_anthropic (Claude) or openai / openai_compat_* (GPT/Gemini)",
        ],
      },
      {
        type: "h2",
        id: "chat-vs-agent",
        text: "Chat vs Agent",
      },
      {
        type: "ul",
        items: [
          "Chat: normal Q&A through Trim - always the first smoke test.",
          "Agent: file tools / edits. Requires capabilities tool_use. Low whole-body Saved % is often normal on tool-heavy turns (tools frozen; prose islands may still Deep). Use Show savings detail for Deep stage.",
          "If you see Custom Endpoint/… • 0 credits in the transcript header, you are in VS Code Chat, not Continue.",
        ],
      },
      {
        type: "h2",
        id: "verify",
        text: "Verify",
      },
      {
        type: "code",
        language: "bash",
        code: `trim stats
# or open http://127.0.0.1:8888/dashboard → Show savings detail`,
      },
      {
        type: "callout",
        title: "Low or 0% Saved",
        text: "Whole-request Saved % can look low on Continue Agent turns. Live Deep uses Segmented Allowlist Deep (each prose island in place, order preserved beside paths; tools/paths frozen). Open http://127.0.0.1:8888/dashboard → Show savings detail for Deep status and Deep stage.",
      },
      {
        type: "h2",
        id: "troubleshoot",
        text: "Troubleshoot",
      },
      {
        type: "ul",
        items: [
          "Empty model list: you edited the wrong file, YAML indent broke, or Continue needs Reload config.",
          "401 / invalid key: wrong provider key for the model, or empty Authorization (never use VS Code Custom Endpoint raw-key tricks inside Continue - Continue uses apiKey in config.yaml).",
          "Model not found: pick an id from GET /v1/models (trim-claude-sonnet).",
          "Raw invoke XML in the chat: add capabilities tool_use, reload, use Continue Agent (not VS Code Chat).",
          "Still hitting Anthropic/OpenAI directly: apiBase must be Trim …/v1, not api.anthropic.com / api.openai.com.",
        ],
      },
      {
        type: "note",
        text: "Also see Custom models, Connect any IDE, VS Code setup (Custom Endpoint), Cursor setup, Windsurf / Zed / JetBrains / Aider pages, and Base URL troubleshooting. After shipping new setup chrome (Continue / VS Code Chat blocks), apply the site_messages migration, let the API reload site_messages (restart or periodic reload), then delete ~/.config/trim/cli-chrome-cache.json (or Windows %USERPROFILE%\\.config\\trim\\cli-chrome-cache.json) so trim help setup / trim setup pick up the new copy-paste blocks.",
      },
    ],
  ),

  "ide/windsurf": page(
    "Windsurf setup",
    "Step-by-step: exact settings file, copyable Base URL, Trim model ids, and verify.",
    [
      {
        type: "p",
        text: "Windsurf can override an OpenAI-compatible Base URL. Trim setup writes common keys when Windsurf’s user settings folder exists. Prefer Always-on or trim start first. For creating friendly trim-… model names, also see Custom models.",
      },
      {
        type: "table",
        headers: ["What", "Exact place"],
        rows: [
          ["Base URL keys", "Windsurf User settings.json (paths below)"],
          ["Model picker", "Windsurf model / OpenAI-compatible model UI"],
          ["Provider key", "Windsurf API key field for that OpenAI-compatible provider"],
          ["Plans / prices", "https://use-trim.com/dashboard → Billing"],
        ],
      },
      { type: "h2", id: "exact-file", text: "Exact file" },
      {
        type: "ul",
        items: [
          "Windows: %APPDATA%\\Windsurf\\User\\settings.json",
          "macOS: ~/Library/Application Support/Windsurf/User/settings.json",
          "Linux: ~/.config/Windsurf/User/settings.json (product folder name may vary by build)",
        ],
      },
      { type: "h2", id: "steps", text: "Steps" },
      {
        type: "ol",
        items: [
          "trim start (or Always-on with IDE)",
          "trim setup (updates Windsurf settings when the folder exists)",
          "Or set manually: openai.baseUrl / windsurf.openai.baseUrl / codeium.openai.baseUrl = http://127.0.0.1:8888/v1",
          "In Windsurf’s model UI, pick or add trim-claude-sonnet (or another id from the copy block below)",
          "Put the provider key that matches your model (Anthropic for Claude, OpenAI for GPT, Google for Gemini)",
          "Send a chat and confirm http://127.0.0.1:8888/dashboard Requests increase",
        ],
      },
      {
        type: "h2",
        id: "copy-base-url",
        text: "Copy: Base URL settings",
      },
      {
        type: "code",
        language: "json",
        code: `{
  "openai.baseUrl": "http://127.0.0.1:8888/v1",
  "windsurf.openai.baseUrl": "http://127.0.0.1:8888/v1",
  "codeium.openai.baseUrl": "http://127.0.0.1:8888/v1"
}`,
      },
      {
        type: "h2",
        id: "copy-models",
        text: "Copy: model ids to pick / add",
      },
      {
        type: "code",
        language: "text",
        code: `trim-claude-sonnet
trim-claude-haiku
trim-claude-opus
trim-gemini-flash
trim-gemini-flash-latest
gpt-4o-mini`,
      },
      {
        type: "callout",
        title: "Honest limits",
        text: "Windsurf UI labels change across releases. If a key is ignored, use Windsurf’s current OpenAI-compatible / custom Base URL setting and set it to Trim …/v1. Model ids must still match GET /v1/models / provider adapters. See Custom models for naming ideas such as trim-gpt-4o-mini or trim-gemini-fast-latest.",
      },
      {
        type: "callout",
        title: "Low or 0% Saved",
        text: "Whole-request Saved % can look low on agent turns. Live Deep uses Segmented Allowlist Deep (each prose island in place, order preserved beside paths; tools/paths frozen). Open http://127.0.0.1:8888/dashboard → Show savings detail for Deep status and Deep stage.",
      },
    ],
  ),

  "ide/zed": page(
    "Zed setup",
    "Step-by-step: exact settings file, copyable api_url, multiple trim-… models, and verify.",
    [
      {
        type: "p",
        text: "Zed stores OpenAI-compatible settings under language_models.openai.api_url. trim setup merges that field when Zed’s settings file exists. Add several trim-… names so the picker stays clear - full naming guide: Custom models.",
      },
      {
        type: "table",
        headers: ["What", "Exact place"],
        rows: [
          ["Base URL", "settings.json → language_models.openai.api_url"],
          ["Custom models list", "settings.json → language_models.openai.available_models"],
          ["Provider key", "Zed OpenAI provider key UI (as Zed documents)"],
          ["Plans / prices", "https://use-trim.com/dashboard → Billing"],
        ],
      },
      { type: "h2", id: "exact-file", text: "Exact file" },
      {
        type: "ul",
        items: [
          "Windows: %APPDATA%\\Zed\\settings.json",
          "macOS: ~/Library/Application Support/Zed/settings.json",
          "Linux: ~/.config/zed/settings.json",
        ],
      },
      { type: "h2", id: "copy-paste", text: "Copy-paste (Base URL + custom models)" },
      {
        type: "code",
        language: "json",
        code: `{
  "language_models": {
    "openai": {
      "api_url": "http://127.0.0.1:8888/v1",
      "available_models": [
        { "name": "trim-claude-sonnet", "max_tokens": 16000 },
        { "name": "trim-claude-haiku", "max_tokens": 16000 },
        { "name": "trim-gemini-flash-latest", "max_tokens": 16000 },
        { "name": "gpt-4o-mini", "max_tokens": 16000 }
      ]
    }
  }
}`,
      },
      {
        type: "ol",
        items: [
          "trim start",
          "trim setup (or paste the api_url + available_models above)",
          "Configure your API key in Zed for the OpenAI provider as Zed documents (match the model family)",
          "Pick trim-claude-sonnet (or another listed id) in Zed",
          "Verify on http://127.0.0.1:8888/dashboard",
        ],
      },
      {
        type: "note",
        text: "available_models shape can vary by Zed version - keep api_url = Trim …/v1 as the required Trim field. See Custom models and OpenAI-compatible clients if your Zed build uses different keys.",
      },
      {
        type: "callout",
        title: "Low or 0% Saved",
        text: "Whole-request Saved % can look low on agent turns. Live Deep uses Segmented Allowlist Deep (each prose island in place, order preserved beside paths; tools/paths frozen). Open http://127.0.0.1:8888/dashboard → Show savings detail for Deep status and Deep stage.",
      },
    ],
  ),

  "ide/jetbrains": page(
    "JetBrains setup",
    "Step-by-step: paste Trim …/v1, pick a trim-… model, matching provider key, verify.",
    [
      {
        type: "p",
        text: "JetBrains does not use one portable settings.json for AI Base URL across all IDEs. trim setup writes a small hint file you paste into AI Assistant. Create clear trim-… model choices the same way as other IDEs - see Custom models.",
      },
      {
        type: "table",
        headers: ["What", "Exact place"],
        rows: [
          ["Base URL hint file", "~/.config/trim/jetbrains-openai.url (from trim setup)"],
          [
            "Paste target",
            "Settings → Tools → AI Assistant → OpenAI-compatible / custom OpenAI endpoint",
          ],
          ["Model id", "AI Assistant model field / list"],
          ["Plans / prices", "https://use-trim.com/dashboard → Billing"],
        ],
      },
      { type: "h2", id: "exact-file", text: "Exact hint file" },
      {
        type: "ul",
        items: [
          "Path written by trim setup: ~/.config/trim/jetbrains-openai.url (Windows: %USERPROFILE%\\.config\\trim\\jetbrains-openai.url)",
          "Contents: a single line like http://127.0.0.1:8888/v1",
        ],
      },
      {
        type: "h2",
        id: "copy-values",
        text: "Copy these values",
      },
      {
        type: "code",
        language: "text",
        code: `http://127.0.0.1:8888/v1

trim-claude-sonnet
trim-claude-haiku
trim-claude-opus
trim-gemini-flash-latest
gpt-4o-mini`,
      },
      { type: "h2", id: "steps", text: "Steps" },
      {
        type: "ol",
        items: [
          "trim start",
          "trim setup (creates jetbrains-openai.url)",
          "JetBrains IDE → Settings → Tools → AI Assistant → enable OpenAI-compatible / custom OpenAI endpoint",
          "Paste http://127.0.0.1:8888/v1 (or the hint file contents)",
          "API key = provider key for the model family (Anthropic for Claude on this door)",
          "Pick or type trim-claude-sonnet (or another id from the copy block)",
          "Confirm traffic on http://127.0.0.1:8888/dashboard",
        ],
      },
      {
        type: "callout",
        title: "Honest limits",
        text: "Exact JetBrains menu names differ by IDE (IDEA, PyCharm, …) and plugin version. The Trim contract is the OpenAI-compatible Base URL = Trim …/v1 plus the matching provider key and a Trim-known model id.",
      },
      {
        type: "callout",
        title: "Low or 0% Saved",
        text: "Whole-request Saved % can look low on agent turns. Live Deep uses Segmented Allowlist Deep (each prose island in place, order preserved beside paths; tools/paths frozen). Open http://127.0.0.1:8888/dashboard → Show savings detail for Deep status and Deep stage.",
      },
    ],
  ),

  "ide/aider": page(
    "Aider and shell clients",
    "Copyable env vars, trim-… model ids, curl smoke test, and Claude Code door note.",
    [
      {
        type: "p",
        text: "Many CLI coding agents (Aider and OpenAI SDK scripts) honor OPENAI_BASE_URL or an OpenAI client base_url. That is Trim’s OpenAI door. Claude Code is different - see Claude Code docs (Anthropic door). Model naming guide: Custom models.",
      },
      {
        type: "table",
        headers: ["What", "Exact place"],
        rows: [
          ["OpenAI door Base URL", "Shell env OPENAI_BASE_URL"],
          ["Model id", "Aider --model / SDK model field"],
          ["Provider key", "OPENAI_API_KEY (must match model family)"],
          ["Claude Code Base URL", "ANTHROPIC_BASE_URL (origin only, no /v1)"],
          ["Plans / prices", "https://use-trim.com/dashboard → Billing"],
        ],
      },
      { type: "h2", id: "env", text: "Environment copy-paste" },
      {
        type: "code",
        language: "bash",
        code: `# OpenAI-door clients (Aider, OpenAI Python/JS SDKs, curl)
export OPENAI_BASE_URL="http://127.0.0.1:8888/v1"
# Key must match the model family you request
export OPENAI_API_KEY="sk-…-or-sk-ant-…-for-Claude-via-Trim"

# Claude Code only (Anthropic door) - no /v1 suffix
export ANTHROPIC_BASE_URL="http://127.0.0.1:8888"
export ANTHROPIC_API_KEY="sk-ant-…"`,
      },
      {
        type: "code",
        language: "powershell",
        code: `$env:OPENAI_BASE_URL = "http://127.0.0.1:8888/v1"
$env:OPENAI_API_KEY = "sk-YOUR_KEY_FOR_THIS_MODEL"
# Claude Code:
# $env:ANTHROPIC_BASE_URL = "http://127.0.0.1:8888"
# $env:ANTHROPIC_API_KEY = "sk-ant-YOUR_KEY"`,
      },
      {
        type: "h2",
        id: "copy-models",
        text: "Copy: model ids",
      },
      {
        type: "code",
        language: "text",
        code: `trim-claude-sonnet
trim-claude-haiku
trim-claude-opus
trim-gemini-flash
trim-gemini-flash-latest
gpt-4o-mini`,
      },
      { type: "h2", id: "aider", text: "Aider" },
      {
        type: "ol",
        items: [
          "trim start",
          "export OPENAI_BASE_URL=http://127.0.0.1:8888/v1",
          "export OPENAI_API_KEY to the provider key for your chosen model",
          "Run aider with a model id Trim accepts (example: --model trim-claude-sonnet)",
          "Confirm Last door on the local dashboard",
        ],
      },
      {
        type: "code",
        language: "bash",
        code: `curl http://127.0.0.1:8888/v1/chat/completions \\
  -H "Authorization: Bearer $OPENAI_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"trim-claude-sonnet","messages":[{"role":"user","content":"ping"}]}'`,
      },
      {
        type: "note",
        text: "trim setup prints shell exports for OPENAI_BASE_URL and ANTHROPIC_BASE_URL. Never point OPENAI_BASE_URL at api.anthropic.com - that skips Trim. See Custom models for trim-gpt-… / trim-gemini-fast naming ideas.",
      },
      {
        type: "callout",
        title: "Low or 0% Saved",
        text: "Whole-request Saved % can look low on agent turns. Live Deep uses Segmented Allowlist Deep (each prose island in place, order preserved beside paths; tools/paths frozen). Open http://127.0.0.1:8888/dashboard → Show savings detail for Deep status and Deep stage.",
      },
    ],
  ),

  "ide/extension": page(
    "Trim IDE extension",
    "Install and configure the Trim IDE extension for VS Code and Cursor: auto-start proxy + telemetry.",
    [
      {
        type: "p",
        text: "Trim IDE is not the compressor. It is a local helper with two jobs: (1) when Start Trim with your IDE is on, ensure the local Fast Mode proxy is healthy as the IDE opens (trim start, or attach if already healthy); (2) post tab acceptance and AI LOC metrics to POST /api/v1/me/events so dashboard charts stay database-backed. Prompt compression and routing only happen when Cursor / Continue / VS Code Chat / Claude Code send traffic to the local proxy Base URL (example http://127.0.0.1:8888/v1). That chat path is separate from this extension’s telemetry. Same VSIX runs in VS Code and Cursor.",
      },
      {
        type: "callout",
        title: "Two paths users must not confuse",
        text: "Chat → Trim (Continue apiBase / VS Code Custom Endpoint / Claude Code ANTHROPIC_BASE_URL on local :8888, or Cursor Agent via trim cursor-link url HTTPS) = compression + routing + token savings. Extension → Trim Cloud API (apiUrl + Trim API key) = keep proxy ready + acceptance / LOC charts. Install the extension but never point chat at Trim → auto-start + metrics only, not full product savings.",
      },
      {
        type: "note",
        text: "Brand: the Marketplace / Extensions list icon is media/icon.png (same Trim mark masters as web/admin). Theme pair + banners/screenshots live under extensions/trim-ide/media/. Regenerate icons with python scripts/_gen_site_icons.py; marketplace banners with python scripts/_gen_ext_marketplace_assets.py.",
      },
      { type: "h2", id: "e2e", text: "Worldwide end-to-end flow" },
      {
        type: "ol",
        items: [
          "Sign up at use-trim.com → Dashboard → Settings: leave Start Trim with your IDE checked",
          "Dashboard → Settings → API keys: create a Trim API key (shown once). This is not your OpenAI / Anthropic / Google provider key",
          "Install Trim CLI on the machine (trim on PATH)",
          "Install Trim IDE from the VS Code Marketplace or Open VSX (or sideload VSIX for air-gapped machines)",
          "Editor: set trim.apiUrl (hosted https://api.use-trim.com), Trim: Copy Hardware ID, register that IDE ID on the key (agent IDE), Trim: Set API Key",
          "Point the chat client at the local proxy (Cursor …/v1, Continue apiBase …/v1, VS Code Chat …/v1/chat/completions, Claude Code origin without /v1)",
          "Open the IDE → proxy ready → chat as usual → local savings at http://127.0.0.1:8888/dashboard (or the listen host trim start printed) + cloud charts on use-trim.com",
        ],
      },
      {
        type: "note",
        text: "Everyday loop detail: activate loads ide-chrome (or cache) → Always-on health-check/attach/trim start → chat hits local proxy (Fast, Deep if prefs allow) → optional POST /api/v1/me/events. Uncheck Start Trim with your IDE in the dashboard and the extension’s poll can stop auto-starting (local trim.autoStartWithIde on/off can override).",
      },
      {
        type: "note",
        text: "Optional shared credits: set trim.workspaceId in the editor after your workspace exists. Mental model: Always-on = auto-start preference; Trim API key = login for Trim Cloud; hardware ID = this editor install is allowed to use that key.",
      },
      { type: "h2", id: "install", text: "Install (Marketplace / Open VSX)" },
      {
        type: "ol",
        items: [
          "VS Code: Extensions → search Trim IDE (publisher usetrim) → Install",
          "Cursor / Open VSX clients: install Trim IDE (usetrim.trim-ide) from Open VSX when that registry is used",
          "Complete the Get started with Trim IDE walkthrough (or Configure + hardware steps below)",
          "Privacy: https://use-trim.com/privacy (extension details in extensions/trim-ide/PRIVACY.md)",
        ],
      },
      {
        type: "callout",
        title: "Stores are live (operators)",
        text: "Trim IDE ships as usetrim.trim-ide on the VS Code Marketplace and Open VSX. VS Code: install from Marketplace. Cursor and other Open VSX clients: install from Open VSX. Operators: Marketplace can use manual .vsix upload or optional VSCE_PAT CI; Open VSX uses a one-time PAT bootstrap, then verified namespace ownership + Trusted Publishing (OIDC) via publish-trim-ide.yml and GitHub Environment open-vsx. Do not leave OVSX_PAT set after OIDC works (PAT overrides Trusted Publishing). Full runbook: extensions/trim-ide/PUBLISH.md. Sideload VSIX remains supported for air-gapped installs.",
      },
      { type: "h2", id: "sideload", text: "Sideload VSIX (air-gapped / offline)" },
      {
        type: "code",
        language: "bash",
        code: `cd extensions/trim-ide
npm ci
npm test
npm run package
# then: Extensions → Install from VSIX → trim-ide-*.vsix`,
      },
      {
        type: "h2",
        id: "hardware",
        text: "Register IDE hardware ID (required for device-bound keys)",
      },
      {
        type: "p",
        text: "IDE hardware ID ≠ CLI hardware ID. Same laptop produces two different fingerprints. For the extension, always use Trim: Copy Hardware ID - never a CLI / trim login fingerprint. Details: Device binding.",
      },
      {
        type: "ol",
        items: [
          "Command Palette (Ctrl+Shift+P / Cmd+Shift+P) → Trim: Copy Hardware ID (hex string on clipboard)",
          "Dashboard → Settings → API keys → create/issue a key if needed → copy the Trim secret once",
          "Key row actions → Register device → paste Hardware UUID → agent IDE (dialog defaults to ide) → save",
          "Editor → Trim: Set API Key → paste the Trim key (Secret Storage, never settings.json)",
          "Settings → Trim: Api Url = https://api.use-trim.com (or your self-hosted API origin)",
        ],
      },
      {
        type: "ul",
        items: [
          "Every later cloud call sends Authorization Bearer, X-Hardware-UUID (IDE hash), and X-Trim-Agent-Id: ide",
          "Second PC: run Copy Hardware ID again on that editor and register the new UUID on the key",
          "Common failure: registering the CLI UUID for an IDE key → extension auth rejected",
        ],
      },
      { type: "h2", id: "configure", text: "Configure" },
      {
        type: "ul",
        items: [
          "trim.apiUrl - API base (http/https only; empty keeps the extension off)",
          "trim.autoStartWithIde - follow (cloud preference), on, or off",
          "trim.httpTimeoutSec - fetch timeout (bounds enforced from Trim cloud chrome)",
          "trim.autoFlushSeconds - batch interval (0 = off; bounds from cloud chrome - set explicitly to enable)",
          "trim.trackDocumentEdits - optional multi-line LOC estimate",
          "trim.minLinesForAiHeuristic - min lines per edit to count (must be ≥ 1 when tracking is on)",
          "trim.workspaceId - optional workspace UUID for shared credits (X-Workspace-Id)",
          "Leave trim.autoStartWithIde = follow so Dashboard Start Trim with your IDE controls auto-start",
        ],
      },
      {
        type: "code",
        language: "json",
        code: `{
  "trim.apiUrl": "https://api.use-trim.com",
  "trim.autoStartWithIde": "follow",
  "trim.httpTimeoutSec": 15,
  "trim.autoFlushSeconds": 30,
  "trim.trackDocumentEdits": true,
  "trim.minLinesForAiHeuristic": 3
}`,
      },
      {
        type: "note",
        text: "Setting values such as autoFlushSeconds: 30 are operator-chosen within cloud IDE_AUTO_FLUSH bounds. The extension package default is 0 (off) - it does not invent a flush interval.",
      },
      { type: "h2", id: "wire-chat", text: "Wire the chat client (where savings happen)" },
      {
        type: "p",
        text: "This extension does not set Cursor/Continue/VS Code Chat models for you. After auto-start works, point chat at Trim and pick a trim-… model. Use the Copy button on the block below. Full guide: Custom models.",
      },
      {
        type: "code",
        language: "text",
        code: `http://127.0.0.1:8888/v1
http://127.0.0.1:8888/v1/chat/completions
http://127.0.0.1:8888

trim-claude-sonnet
trim-claude-haiku
trim-claude-opus
trim-gemini-flash-latest
gpt-4o-mini`,
      },
      {
        type: "ul",
        items: [
          "Cursor Agent: Settings → Models → Override OpenAI Base URL → trim cursor-link url (https://…/cl/<token>/v1) + model id (trim-claude-sonnet, …) + provider key - never 127.0.0.1 for Agent",
          "Continue: ~/.continue/config.yaml → provider openai, apiBase …/v1, model trim-claude-sonnet - see Continue setup",
          "VS Code Chat: Custom Endpoint full …/v1/chat/completions + Add Models wizard secret - see VS Code setup",
          "Claude Code: ANTHROPIC_BASE_URL=http://127.0.0.1:8888 (origin only, no /v1)",
          "Provider keys stay in the IDE/agent. Trim: Set API Key is only for Trim Cloud (telemetry / quota / prefs)",
          "Plans / prices: https://use-trim.com/dashboard → Billing (not in the extension settings)",
        ],
      },
      { type: "h2", id: "auto-start", text: "Auto-start with IDE" },
      {
        type: "ul",
        items: [
          "On activate: if preference is on, health-check then start trim if needed",
          "If a healthy proxy already owns the port, attach (no double-bind)",
          "Spawns trim start only - Deep runs inside the proxy when Preferences Deep is on (not a separate compress process on IDE open)",
          "While the IDE stays open, re-checks the preference so a dashboard uncheck can stop auto-start",
          "Fail soft: proxy start failure does not brick the IDE",
          "Local trim.autoStartWithIde on/off overrides the cloud preference on this editor",
        ],
      },
      { type: "h2", id: "activate", text: "What runs on each IDE open" },
      {
        type: "ol",
        items: [
          "Load cached ide-chrome labels; require trim.apiUrl",
          "Sync GET /api/v1/public/ide-chrome (fail-closed if never synced successfully)",
          "Restore pending counters; show status bar",
          "ensureProxyAutostart (attach or trim start); quota check / upgrade prompt when exhausted",
          "Register commands (API key, hardware ID, flush, tab marks); optional LOC tracking and auto-flush",
          "On deactivate: best-effort flush of pending counters",
        ],
      },
      {
        type: "callout",
        title: "Fail-closed chrome",
        text: "UI labels load from GET /api/v1/public/ide-chrome. On first install with no network the extension stays inactive and logs to Output → Trim. After a successful sync, a last-known-good cache keeps labels available offline.",
      },
      { type: "h2", id: "privacy-data", text: "What is sent to Trim Cloud" },
      {
        type: "ul",
        items: [
          "Only when trim.apiUrl and Trim: Set API Key are both set",
          "Sent: hardware UUID hash, agent/version headers, tab shown/accepted counts, optional LOC estimates, preference / quota reads",
          "Not sent by this extension: source files, repo contents, chat prompts, or Composer transcripts",
          "Full privacy: https://use-trim.com/privacy and extensions/trim-ide/PRIVACY.md",
        ],
      },
      { type: "h2", id: "limits", text: "Honest limits" },
      {
        type: "ul",
        items: [
          "Cursor ghost-text accept is not fully exposed via a stable public API - counts use explicit commands plus optional edit heuristics",
          "VS Code Chat Agent + Custom Endpoint is not a Trim-supported guarantee; prefer Continue for agent file edits",
          "Prefer Marketplace / Open VSX for normal installs; VSIX sideload is for air-gapped or offline machines (see PUBLISH.md for operator republish)",
          "Listing banners/screenshots may be brand placeholders until real IDE captures replace them",
        ],
      },
      { type: "h2", id: "remove", text: "Remove the extension" },
      {
        type: "ol",
        items: [
          "Command Palette → Trim: Clear API Key (clears Secret Storage and local extension state)",
          "Command Palette → Extensions → uninstall Trim IDE",
          "On each machine CLI: trim uninstall if you are leaving Trim entirely (see Uninstall)",
        ],
      },
      {
        type: "note",
        text: "Also see Custom models, Always-on with IDE, Settings and API keys, Device binding, Cursor setup, Continue setup, and VS Code setup.",
      },
    ],
  ),

  "ide/openai-compatible": page(
    "OpenAI-compatible clients",
    "Any client that can override an OpenAI Base URL can speak to Trim - with copyable Base URL and trim-… model ids.",
    [
      {
        type: "p",
        text: "Trim exposes an OpenAI-compatible chat completions surface at /v1/chat/completions. Set the client base URL to your Trim listen URL (…/v1 for most SDKs; some BYOK UIs need the full …/v1/chat/completions path - see VS Code setup). Enabled provider_adapters rows route by model prefix (openai_compat hosts or anthropic_messages for Claude). When adapters are synced, models with no matching prefix return a clear Trim error. When adapters are not synced, traffic forwards to UPSTREAM_OPENAI_URL. No OpenRouter required. Friendly IDE names: Custom models.",
      },
      {
        type: "h2",
        id: "steps",
        text: "Steps",
      },
      {
        type: "ol",
        items: [
          "trim start (or Always-on with IDE)",
          "trim config sync",
          "Set the client Base URL to http://127.0.0.1:8888/v1 (or full …/v1/chat/completions if the client requires the chat path)",
          "Pick a model id such as trim-claude-sonnet (see Custom models for naming ideas)",
          "Put the matching provider key in the client (Anthropic / OpenAI / Google)",
          "Send a request and confirm http://127.0.0.1:8888/dashboard",
          "Plans / prices: https://use-trim.com/dashboard → Billing",
        ],
      },
      {
        type: "h2",
        id: "copy-values",
        text: "Copy these values",
      },
      {
        type: "code",
        language: "text",
        code: `http://127.0.0.1:8888/v1
http://127.0.0.1:8888/v1/chat/completions

trim-claude-sonnet
trim-claude-haiku
trim-claude-opus
trim-gemini-flash
trim-gemini-flash-latest
gpt-4o-mini`,
      },
      {
        type: "code",
        language: "bash",
        code: `curl http://127.0.0.1:8888/v1/chat/completions \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer $KEY" \\
  -d '{"model":"trim-claude-sonnet","messages":[{"role":"user","content":"ping"}]}'`,
      },
      { type: "h2", id: "sdk", text: "SDK pattern" },
      {
        type: "p",
        text: "Most OpenAI SDKs accept a base URL override. Point that override at Trim, pick a model id from GET /v1/models, and leave Trim running for the session. Continue uses apiBase: http://127.0.0.1:8888/v1 in ~/.continue/config.yaml.",
      },
      {
        type: "ul",
        items: [
          "Prefer Always-on with IDE or local trim start for interactive work",
          "Run trim config sync so openai_model_aliases + provider_adapters load",
          "Confirm streaming clients still honor the Base URL",
          "OpenAI key for GPT models; Google key for Gemini models; Anthropic key for Claude models on this same OpenAI door (match the key to the model / DB route, not to a daily .env edit)",
          "Prefer Trim aliases (trim-claude-sonnet, trim-gemini-flash-latest) when a dated model id is rejected for your key",
          "Plans / prices: https://use-trim.com/dashboard → Billing",
        ],
      },
      { type: "h2", id: "failover", text: "Same-shape upstream failover" },
      {
        type: "p",
        text: "Optional env only (no invent URLs in code): UPSTREAM_OPENAI_FAILOVER_URL and UPSTREAM_ANTHROPIC_FAILOVER_URL. When set, Trim retries the same request path on transport errors or HTTP 5xx against a backup that speaks the same dialect (OpenAI-compat backup for the OpenAI door, Anthropic-compatible backup for the Anthropic door / adapter target). Adapter translation failures fail closed with a clear error; Trim does not silently invent another provider.",
      },
    ],
  ),

  "ide/claude-code": page(
    "Claude Code (see CLI)",
    "Claude Code is a terminal / CLI path - not an IDE Models menu. Full guide lives under CLI.",
    [
      {
        type: "callout",
        title: "Wrong section?",
        text: "If you only use Claude Code (no Cursor, no VS Code), open the full guide: Claude Code (CLI only). That page has line-by-line where / what / how / why / what happens / verify / everyday steps, plus separate Windows, macOS, and Linux copy blocks (each with a Copy button).",
      },
      {
        type: "p",
        text: "Claude Code uses Trim’s Anthropic door: set ANTHROPIC_BASE_URL to the Trim listen origin only (http://127.0.0.1:8888) with no /v1. Cursor / Continue / VS Code Chat use a different address (…/v1 or …/v1/chat/completions) and are documented on their own IDE pages.",
      },
      {
        type: "code",
        language: "text",
        code: "http://127.0.0.1:8888",
      },
      {
        type: "code",
        language: "bash",
        code: `# Preview only - use the full OS-specific blocks on Claude Code (CLI only)
trim start
trim setup
export ANTHROPIC_BASE_URL="http://127.0.0.1:8888"
export ANTHROPIC_API_KEY="sk-ant-YOUR_KEY"
claude`,
      },
      {
        type: "note",
        text: "Full CLI-only walkthrough: Claude Code (CLI only). For Claude inside Cursor, use Cursor setup + Custom models instead.",
      },
      {
        type: "callout",
        title: "Low or 0% Saved",
        text: "Whole-request Saved % can look low on agent turns. Live Deep uses Segmented Allowlist Deep (each prose island in place, order preserved beside paths; tools/paths frozen). Open http://127.0.0.1:8888/dashboard → Show savings detail for Deep status and Deep stage.",
      },
    ],
  ),

  "ide/provider-adapters": page(
    "Provider adapters",
    "DB-driven OpenAI-compat → provider dialect plugins (Anthropic first).",
    [
      {
        type: "p",
        text: "Trim is not OpenRouter. It stays a compression proxy with pluggable adapters. Rows in public.provider_adapters match model prefixes on /v1/chat/completions. Shipped dialects: anthropic_messages (OpenAI chat ↔ Anthropic /v1/messages) and openai_compat (same OpenAI shape, host from upstream_base_url). GET /v1/models lists public.provider_discoverable_models from the database (Cursor / OpenAI SDKs; optional Claude Code gateway discovery). More dialects can be added later without a marketplace.",
      },
      {
        type: "note",
        text: "Why Messages for Claude, not Anthropic’s OpenAI-compat URL? Anthropic’s own docs say their OpenAI SDK compatibility layer is primarily for testing and is not the long-term production path. Trim translates to native POST /v1/messages so Cursor keeps an OpenAI-shaped client while you get Messages tools/stream/thinking support plus Trim compression. GPT/Gemini/DeepSeek stay openai_compat (no body translate).",
      },
      { type: "h2", id: "when-adapter", text: "openai_compat vs anthropic_messages" },
      {
        type: "ul",
        items: [
          "Same OpenAI chat shape (GPT, Gemini OpenAI-compat, DeepSeek, Mistral, Groq, Azure OpenAI, vLLM, Ollama): dialect openai_compat. Set match_model_prefixes + upstream_base_url in the database. Do not flip UPSTREAM_OPENAI_URL daily.",
          "Different dialect (Claude Messages today; Bedrock/Vertex later if needed): dialect anthropic_messages (or a future dialect). Anthropic is shipped; others are not invented until real demand.",
          "UPSTREAM_OPENAI_URL: OpenAI-door default only when provider adapters are not synced. After sync, unmatched prefixes fail closed (clear Trim error; no silent invent). Optional same-shape failover stays on env.",
        ],
      },
      { type: "h2", id: "flow", text: "Claude-in-Cursor flow" },
      {
        type: "ol",
        items: [
          "Cursor → Override OpenAI Base URL → Trim",
          "Model id matches an enabled adapter prefix (e.g. claude-, anthropic/, trim-claude-)",
          "Trim compresses: Fast every turn; when Deep is on, Segmented Allowlist Deep on each last-user prose island in place (order preserved beside paths; tools/paths frozen)",
          "Adapter translates OpenAI chat → Anthropic /v1/messages (tools + stream)",
          "Auth: Authorization Bearer → x-api-key + anthropic-version from the adapter row",
          "Org/multi-workspace Anthropic keys: Trim also sends anthropic-workspace-id when set (client header → TRIM_ANTHROPIC_WORKSPACE_ID). Workspace-scoped keys omit it.",
          "Response translated back to OpenAI shape for Cursor",
        ],
      },
      { type: "h2", id: "models-discovery", text: "GET /v1/models (gateway discovery)" },
      {
        type: "ul",
        items: [
          "Cursor / OpenAI SDKs: Base URL = Trim …/v1 then GET /v1/models (standard OpenAI list shape).",
          "Claude Code: Base URL = Trim listen origin (no /v1). Optional CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY=1 loads the same catalog (Claude Code keeps ids containing claude or anthropic).",
          "Catalog rows live in provider_discoverable_models (DB only). Ops refresh ids when providers retire models; Trim does not invent models from prefixes.",
          "When provider adapters are synced, a chat model that matches no adapter prefix returns a clear Trim error (no silent host invent / cross-provider swap).",
          "Upstream model_not_found, auth (401), rate-limit/quota (429), and upstream HTML/5xx host outages are remapped to Trim site_messages chrome when synced (honest errors; Trim does not invent entitlement).",
          "OpenAI org/project: send OpenAI-Organization / OpenAI-Project, or set TRIM_OPENAI_ORGANIZATION / TRIM_OPENAI_PROJECT once (headers win).",
          "Optional key-accurate list: GET /v1/models?adapter=openai|gemini|deepseek|mistral with Authorization Bearer. Default GET /v1/models stays the DB catalog.",
          "trim config sync pulls the catalog with provider adapters into local preferences.",
          "trim setup prints Cursor/OpenAI …/v1, Claude Code ANTHROPIC_BASE_URL (origin only, no /v1), Continue ~/.continue/config.yaml hints, and VS Code Chat Custom Endpoint …/v1/chat/completions hints.",
        ],
      },
      { type: "h2", id: "models", text: "Which model id to pick" },
      {
        type: "p",
        text: "Everyday users: prefer Trim aliases and create matching custom entries in the IDE (full guide: Custom models). Operators maintain the aliases in Admin.",
      },
      {
        type: "code",
        language: "text",
        code: `trim-claude-sonnet
trim-claude-haiku
trim-claude-opus
trim-gemini-flash
trim-gemini-flash-latest
gpt-4o-mini`,
      },
      {
        type: "ul",
        items: [
          "Best for Claude: Trim aliases (trim-claude-sonnet / trim-claude-opus / trim-claude-haiku) map via provider_adapters.model_aliases (Admin → Billing → Provider adapters). Aliases are ops-owned and refreshed when Anthropic retires ids.",
          "Also fine: Anthropic’s current official API id (examples: claude-sonnet-5-5, claude-opus-5-5, claude-haiku-4-5-20251001) - prefix match routes them with no invent.",
          "Gemini: trim-gemini-flash / trim-gemini-flash-latest via openai_model_aliases.",
          "Do not rely on retired dated ids unless you intentionally keep them in the DB alias map.",
        ],
      },
      { type: "h2", id: "anthropic-keys", text: "Anthropic workspace vs organization keys" },
      {
        type: "ul",
        items: [
          "Preferred: workspace-scoped Anthropic API key → no Trim workspace id required.",
          "Org / multi-workspace key → set TRIM_ANTHROPIC_WORKSPACE_ID once as a user environment variable on each machine that runs Cursor, or send anthropic-workspace-id on the request. Ids must look like wrkspc_….",
          "Priority when forwarding: client anthropic-workspace-id header → process env TRIM_ANTHROPIC_WORKSPACE_ID. Never a daily host flip.",
        ],
      },
      { type: "h2", id: "ops", text: "Ops / Admin" },
      {
        type: "ul",
        items: [
          "Edit adapters: Admin → Billing → Provider adapters (or SQL on provider_adapters)",
          "Workspace ids for org-scoped Anthropic keys are per-machine (header / TRIM_ANTHROPIC_WORKSPACE_ID), not a shared SaaS Admin default",
          "Edit Gemini Cursor aliases: openai_model_aliases (no hard-coded maps in the proxy)",
          "Users: trim config sync pulls adapters into local preferences",
          "Stats: X-Trim-Door header and /v1/stats last_door (openai | anthropic | openai_to_anthropic | openai_compat_openai | openai_compat_gemini | …)",
          "Adapter door ids come from provider_adapters.door_label in the database (works after adapter sync even if API chrome is offline)",
          "Native door ids (openai / anthropic) and the dashboard “Last door” caption come from site_messages via API chrome (CLI_PROXY_DOOR_* / LOCAL_STATS_LAST_DOOR). Empty chrome fails closed: omit X-Trim-Door / blank caption; Trim does not invent those strings",
          "Optional same-shape failover: UPSTREAM_OPENAI_FAILOVER_URL / UPSTREAM_ANTHROPIC_FAILOVER_URL (see OpenAI-compatible clients). Adapter translate/auth errors stay fail-closed",
        ],
      },
      {
        type: "note",
        text: "If adapters are not synced, the proxy does not invent Claude routing. Requests stay on the OpenAI passthrough door. trim config sync / trim start only overwrite local adapter rows when the API returns provider_adapters_synced=true with non-nil arrays; older API builds that omit those fields leave your last DB-synced adapters intact.",
      },
    ],
  ),

  "ide/troubleshoot-base-url": page(
    "Base URL troubleshooting",
    "Fix the common Cursor + Claude Base URL mistakes - with copyable correct values and model ids.",
    [
      {
        type: "p",
        text: "Do not set Cursor Override OpenAI Base URL to https://api.anthropic.com (or …/v1/). Even though Anthropic publishes a limited OpenAI SDK compatibility layer there, they describe it as not a long-term production path, and pointing Cursor at Anthropic bypasses Trim compression entirely. Trim’s official path is: Base URL = Trim; Claude models on /v1/chat/completions use the DB-driven Anthropic Messages adapter (full Messages features + Fast/Deep). Friendly model names: Custom models.",
      },
      {
        type: "h2",
        id: "copy-correct",
        text: "Copy these correct values",
      },
      {
        type: "p",
        text: "Use the Copy button on this block, then paste into your IDE settings.",
      },
      {
        type: "code",
        language: "text",
        code: `Cursor Agent - run: trim cursor-link url
https://YOUR_SUBDOMAIN.trycloudflare.com/cl/YOUR_TOKEN/v1

Continue / most on-device OpenAI clients
http://127.0.0.1:8888/v1

VS Code Chat Custom Endpoint
http://127.0.0.1:8888/v1/chat/completions

Claude Code (origin only - no /v1)
http://127.0.0.1:8888

Recommended model ids
trim-claude-sonnet
trim-claude-haiku
trim-claude-opus
trim-gemini-flash
trim-gemini-flash-latest
gpt-4o-mini`,
      },
      {
        type: "table",
        headers: ["Symptom", "Likely cause", "Fix"],
        rows: [
          [
            "Access to private networks is forbidden",
            "Cursor Agent Base URL is still 127.0.0.1",
            "Paste trim cursor-link url (https://…/v1) - see Cursor setup",
          ],
          [
            "No savings / traffic missing",
            "Base URL points at vendor, not Trim",
            "Cursor: Cursor Link URL; other IDEs: http://127.0.0.1:8888/v1",
          ],
          [
            "Model not found",
            "Dated id or unknown custom name",
            "Use trim-claude-sonnet / GET /v1/models / Custom models",
          ],
          [
            "401 / empty key",
            "Wrong key family or VS Code raw sk- in JSON",
            "Match key to model; VS Code must use wizard secret",
          ],
          [
            "Claude Code model may not exist",
            "ANTHROPIC_BASE_URL includes /v1",
            "Use origin only: http://127.0.0.1:8888",
          ],
        ],
      },
      { type: "h2", id: "do", text: "Do" },
      {
        type: "ol",
        items: [
          "Always point Cursor OpenAI Base URL at Trim (local or tunnel …/v1)",
          "Claude in Cursor: Anthropic API key in the OpenAI key field + trim-claude-sonnet (or Claude model id) + trim config sync (prefer workspace-scoped Anthropic key)",
          "If Anthropic returns “API key is not scoped to a workspace”: create a workspace-scoped key, or set TRIM_ANTHROPIC_WORKSPACE_ID=wrkspc_… once (see Connect any IDE → Anthropic key types) and restart trim start",
          "Claude Code: ANTHROPIC_BASE_URL=http://127.0.0.1:8888 - no /v1 suffix",
          "GPT in Cursor: OpenAI key + gpt-* model",
          "Gemini in Cursor: Google key + trim-gemini-flash / trim-gemini-flash-latest",
          "Plans / prices: https://use-trim.com/dashboard → Billing",
        ],
      },
      { type: "h2", id: "dont", text: "Don’t" },
      {
        type: "ul",
        items: [
          "Don’t paste api.anthropic.com into Cursor OpenAI Base URL (skips Trim; Anthropic’s OpenAI-compat layer is limited)",
          "Don’t enable Cursor Anthropic BYOK together with Trim’s OpenAI override (fragile / tool breaks)",
          "Do not expect OpenRouter. Trim DB adapters route Claude (Messages) and same-shape hosts (GPT/Gemini/DeepSeek/Mistral)",
          "Do not flip UPSTREAM_OPENAI_URL daily; model prefix + provider_adapters pick the host",
          "Do not treat TRIM_ANTHROPIC_WORKSPACE_ID as a daily setting; use a workspace-scoped Anthropic key when possible, or set the workspace id once for org-scoped keys",
        ],
      },
      {
        type: "note",
        text: "Also see Custom models, Cursor setup, Connect any IDE, and Provider adapters.",
      },
    ],
  ),

  "ide/workflow": page(
    "Day-to-day workflow",
    "Pick your path, then follow only that everyday loop - IDE or Claude Code CLI only.",
    [
      {
        type: "p",
        text: "After first-time setup (Quickstart), your goal is an invisible day. Use only the loop that matches your tool.",
      },
      {
        type: "table",
        headers: ["I use…", "Everyday loop (where / what)"],
        rows: [
          [
            "Cursor / VS Code / Continue",
            "Preferences Always-on on → open editor → chat with Base URL already set → check local dashboard after big tasks",
          ],
          [
            "Claude Code only (no IDE)",
            "If local dashboard is down: trim start → terminal has ANTHROPIC_BASE_URL → run claude → check local dashboard (full guide: Claude Code CLI only)",
          ],
        ],
      },
      {
        type: "h2",
        id: "ide-loop",
        text: "IDE loop (Cursor / Continue / VS Code Chat)",
      },
      {
        type: "ol",
        items: [
          "Leave Start Trim with your IDE checked (https://use-trim.com/dashboard → Preferences) or run trim autostart enable",
          "Open Cursor or VS Code - proxy starts or attaches automatically",
          "Confirm Base URL is still http://127.0.0.1:8888/v1 (or …/v1/chat/completions for VS Code Chat) and a model such as trim-claude-sonnet",
          "Chat in the correct panel (Continue sidebar ≠ VS Code Chat Custom Endpoint/… • 0 credits)",
          "After large tasks: trim stats or http://127.0.0.1:8888/dashboard → Show savings detail",
          "Plans / prices / receipts: Dashboard → Billing / Receipts / Team",
        ],
      },
      {
        type: "h2",
        id: "claude-loop",
        text: "Claude Code loop (CLI only)",
      },
      {
        type: "ol",
        items: [
          "Confirm http://127.0.0.1:8888/dashboard loads (else: trim start)",
          "Open a terminal where ANTHROPIC_BASE_URL=http://127.0.0.1:8888 (no /v1) and ANTHROPIC_API_KEY are set",
          "Run claude as usual",
          "Check the local dashboard Requests / Saved",
          "Plans / prices: https://use-trim.com/dashboard → Billing",
        ],
      },
      { type: "h2", id: "quality", text: "If quality dips" },
      {
        type: "ul",
        items: [
          "Confirm active files you need are still protected",
          "Widen keep rules in .trimrc for that repo",
          "Sync milder Preferences from the dashboard",
          "Compare tokens in vs out so you know Trim is still on the path",
          "Confirm Last door on the local meter matches the client",
        ],
      },
      {
        type: "note",
        text: "See Always-on with IDE, Claude Code (CLI only), Cursor setup, Continue setup, and VS Code setup.",
      },
    ],
  ),

  "dashboard/overview": page(
    "Dashboard overview",
    "Customer dashboard routes: home usage, dedicated Traces / Receipts / Enterprise pages, Team, Settings, shell notifications, and account delete.",
    [
      {
        type: "p",
        text: "Sign in at use-trim.com with an allowed social provider. Nav is chrome-driven: Dashboard (home), Traces, Receipts, Enterprise, Team, Settings. Desktop uses a persistent left sidebar; mobile uses the header menu sheet with the same links. The shell also hosts theme, sign-out, and a notification bell when API chrome provides labels. Shared DataTables use overflow-x so wide columns scroll on small screens.",
      },
      { type: "h2", id: "routes", text: "Routes" },
      {
        type: "ul",
        items: [
          "/dashboard - quota cards, plan/top-up CTAs, MoR customer portal, usage charts, heatmap, avatar sync, delete account (tables live on dedicated nav pages)",
          "/dashboard/traces - event traces DataTable (search, date range, selection, bulk delete, detail dialog); deep link ?id=",
          "/dashboard/receipts - receipts DataTable (search, sync, selection, preview dialog); immutable ledger (no customer delete)",
          "/dashboard/receipts/{id} - receipt detail and PDF download",
          "/dashboard/enterprise - enterprise inquiries DataTable (status filter, search, selection, bulk delete non-activated, Pay with Paddle, detail dialog); deep link ?id=",
          "/dashboard/team - workspaces, members, pending invites (full CRUD + bulk)",
          "/dashboard/settings - identities, Preferences (Always-on + compression), CLI help, API keys",
          "/invite/{token} - public invite preview, sign-in CTA, accept (email must match)",
        ],
      },
      { type: "h2", id: "shell", text: "Shell notifications" },
      {
        type: "ul",
        items: [
          "Bell lists /api/v1/me/notifications with unread count (poll interval from NOTIF_POLL_INTERVAL_MS)",
          "Panel uses skip/limit from billing_settings.default_page_size; scroll loads the next page when meta.has_more (IntersectionObserver + near-bottom fallback)",
          "While the next page loads, chrome.loading_more (NOTIF_LOADING_MORE) shows at the bottom of the list",
          "Clickable rows follow API href and mark the item read: receipts → /dashboard/receipts/{id} (or list ?id=), enterprise offer/activated → /dashboard/enterprise?id=, team → /dashboard/team, admin inquiry/break-glass with id",
          "Mark one read or Mark all read when chrome exposes those actions",
          "Bell fails closed if chrome labels are missing",
        ],
      },
      { type: "h2", id: "seo-404", text: "SEO and not-found" },
      {
        type: "ul",
        items: [
          "Root metadata (title template, description, Open Graph, Twitter card, robots) comes from site_messages via GET /api/v1/public/auth-providers; metadataBase uses NEXT_PUBLIC_APP_URL",
          "Custom app/not-found and app/error pages use PAGE_404_* / PAGE_ERROR_* chrome (migration 209); robots noindex on 404",
          "robots.ts allows public marketing routes and disallows dashboard/auth/cli/invite; sitemap.ts lists home, pricing, login, legal, and docs slugs",
          "Admin console is noindex (ADMIN_SEO_ROBOTS) with its own 404/error chrome",
          "Public /contact page uses COMPANY_SUPPORT_EMAIL plus CONTACT_* mailto subject/body templates (migration 210); Contact appears in marketing header/footer when LANDING_NAV_CONTACT is set",
        ],
      },
      { type: "h2", id: "avatar", text: "Avatar sync" },
      {
        type: "p",
        text: "Home Dashboard can sync your profile avatar from the linked identity when chrome and the API expose the sync action. Labels and success/error toasts are site_messages-backed.",
      },
      { type: "h2", id: "first-login", text: "After first login" },
      {
        type: "ol",
        items: [
          "Confirm plan and remaining quota (or Unlimited when the Service reports it)",
          "Leave Start Trim with your IDE checked under Settings (default on; toggles save immediately)",
          "Issue or register a device-bound API key under Settings",
          "Open Team to create a workspace and invite seats",
          "Save compression Preferences (Deep on/off, engine, compression rate, Base model when engine is v2), then run trim config sync on each machine",
        ],
      },
      { type: "h2", id: "leave", text: "Leaving Trim" },
      {
        type: "p",
        text: "Delete account lives on the home Dashboard (confirm dialog), not under Settings. Hosted data is removed; each machine still needs trim uninstall. See Uninstall.",
      },
    ],
  ),

  "dashboard/usage": page(
    "Usage and traces",
    "Home dashboard metering plus the dedicated Traces page: quota strip, charts, acceptance stats, heatmap, and event traces DataTable.",
    [
      {
        type: "p",
        text: "Open Dashboard for usage charts and quota. Open sidebar Traces (/dashboard/traces) for the event DataTable. Empty charts usually mean the proxy is not logged in or traffic is not flowing through Trim. All labels and empty copy come from site_messages / API.",
      },
      { type: "h2", id: "quota", text: "Quota and plan strip" },
      {
        type: "ul",
        items: [
          "Plan name, credits used / remaining (or Unlimited), tokens-saved style metrics when returned",
          "Exhausted / low quota banners when the subscription payload provides them",
          "Plan and top-up buttons open the shared PlanModal (see Plans and billing)",
          "Merchant of Record customer portal button starts a portal session when allowed",
        ],
      },
      { type: "h2", id: "charts", text: "Charts and filters" },
      {
        type: "ul",
        items: [
          "Cumulative usage chart with API group-by options (model / mode)",
          "Token savings and model / mode / status breakdown charts when event stats return series",
          "Daily tooltips hide zero series; empty days stay visible with a zero total",
          "Acceptance rate plus tab suggestions accepted/shown when acceptance stats exist",
          "AI lines added / lines deleted metric cards when the stats payload includes them",
          "LOC / activity heatmap with scope selector when scopes and days are returned",
        ],
      },
      { type: "h2", id: "traces", text: "Event traces table" },
      {
        type: "ul",
        items: [
          "Dedicated page /dashboard/traces (sidebar Traces); not embedded on home Dashboard",
          "Shared DataTable: when, model, mode, tokens, latency, View, row actions (headers from API)",
          "Search, date-range filters (UTC calendar days on the API), and skip pagination from events meta",
          "Checkbox selection + Delete selected (and per-row Delete) when TABLE_* / EVENTS_DELETE chrome is present; DELETE /me/events",
          "Row opens a detail dialog with the same columns and fail-closed chrome",
          "Notification / deep link ?id= highlights the row and opens the detail dialog",
        ],
      },
      { type: "h2", id: "read", text: "What to look for" },
      {
        type: "ul",
        items: [
          "Tokens in vs tokens out on each event",
          "Percent cut and USD direction when metering is enabled",
          "Gaps that line up with IDE Base URL misconfiguration",
        ],
      },
      {
        type: "callout",
        title: "Tip",
        text: "Run trim start while logged in, send one Composer request, then refresh Traces. If traces stay empty, fix Base URL before debugging billing.",
      },
    ],
  ),

  "dashboard/billing": page(
    "Plans and billing",
    "PlanModal checkout, top-ups, enterprise inquiry, customer portal, dedicated Receipts and Enterprise pages, sync, and receipt detail.",
    [
      {
        type: "p",
        text: "Prices and entitlements live in plan_catalog. Checkout uses the payment partner as Merchant of Record. Dialogs and buttons are backend-labeled (no client invent).",
      },
      { type: "h2", id: "plans", text: "Plans and upgrades (PlanModal)" },
      {
        type: "ul",
        items: [
          "Open in plans mode from Dashboard CTAs",
          "Monthly / annual interval when billing settings return both",
          "Free or expired: new checkout for paid plans",
          "Active paid to higher rank: upgrade preview / confirm when the API requires CONFIRM_REQUIRED",
          "Active paid to lower rank: blocked while the paid period is active",
          "Unlimited metering only when the operator enables that plan dial",
          "Popular badge when plan_catalog.popular is on",
          "Per-seat plans: seat quantity control (min/max/default from billing settings) before checkout",
          "Public /pricing matches PlanModal controls: annual toggle, seat quantity, per-seat price line, change_reason hints, and Unlimited/Popular badges from the same plans API",
          "Enterprise plan kind opens an inquiry dialog (company name, estimated seats, message, submit) instead of immediate checkout",
          "After sales offers seats, Dashboard → Enterprise shows Pay with Paddle (same checkout path as Pro/Team with inquiry_id × offered seats)",
          "Enterprise entitlements and receipts apply only after the Paddle webhook succeeds - not when sales sends the offer",
        ],
      },
      { type: "h2", id: "enterprise", text: "Enterprise (Paddle-only)" },
      {
        type: "p",
        text: "One-line model: inquiry → sales sets offered seats → customer pays Paddle → webhook → Enterprise entitlements + receipt. Admin Send Paddle offer (legacy Activate alias) never unlocks the plan without payment.",
      },
      {
        type: "ul",
        items: [
          "Customer submits inquiry from the Enterprise plan card (PlanModal)",
          "Sales reviews under Admin → Sales and revenue → Enterprise; sets offered seats and notes; status moves contacted → offered (ready for Paddle checkout)",
          "Dedicated /dashboard/enterprise page (sidebar Enterprise) lists the customer's inquiries with shared DataTable chrome",
          "Status filter, search (company/message), checkbox selection, bulk/row delete for non-activated inquiries, View detail dialog, Pay with Paddle when offered",
          "Deep link /dashboard/enterprise?id= (notification kinds user.enterprise_offer_ready / user.enterprise_activated) highlights and opens the inquiry",
          "Pay with Paddle opens the same checkout-session path as Pro/Team with inquiry_id and quantity = offered_seat_quantity",
          "Paid webhook writes subscription, quotas, seats, and receipt; inquiry status becomes activated with paddle_subscription_id / paddle_transaction_id",
          "After payment, Dashboard subscription status matches other paid plans (active, portal, receipts) - never free + exhausted from an unpaid offer; Unlimited Enterprise plans use plan_catalog.unlimited so metering is not 0/0 exhausted",
        ],
      },
      { type: "h2", id: "topups", text: "Top-ups" },
      {
        type: "ul",
        items: [
          "Always a new checkout; tier unchanged",
          "Open PlanModal in top-up mode from usage CTAs",
          "Closing the dialog clears mode so the next open starts clean",
        ],
      },
      { type: "h2", id: "portal", text: "Customer portal" },
      {
        type: "p",
        text: "When chrome and subscription allow, Dashboard starts a MoR portal session so you can manage payment method and cancel renewals before the next billing date.",
      },
      { type: "h2", id: "receipts", text: "Receipts page" },
      {
        type: "ul",
        items: [
          "Dedicated /dashboard/receipts (sidebar Receipts); not embedded on home Dashboard",
          "Shared DataTable: date, invoice, status, total, View, row actions (headers from API)",
          "Search + pagination; Sync from Paddle toolbar when permitted",
          "Checkbox selection + Clear selection (receipts are an immutable MoR ledger - no customer delete)",
          "Row preview dialog; Open full receipt → /dashboard/receipts/{id} for line items and PDF",
          "Notification user.receipt_ready deep-links to receipt detail (APP_PATH_RECEIPTS_PREFIX + entity id); list also accepts ?id=",
          "Receipt detail: download MoR/first-party PDF when URLs exist, plus Print (window.print) when chrome provides print_action_label",
          "PDF filenames and empty states are fail-closed chrome",
        ],
      },
    ],
  ),

  "dashboard/keys": page(
    "Settings and API keys",
    "/dashboard/settings: identities, Preferences, CLI help, device-bound API keys. Account delete is on home Dashboard.",
    [
      {
        type: "p",
        text: "Open Dashboard → Settings. Sections: linked identities, Preferences (Always-on + compression + local agent), CLI help card, API keys DataTable. Labels and confirms are API / site_messages driven.",
      },
      {
        type: "callout",
        title: "Trim API key ≠ provider key",
        text: "Keys on this page authenticate Trim Cloud (extension telemetry, quota, preferences, automation). They are not the OpenAI / Anthropic / Google keys you paste into Cursor, Continue, or VS Code Chat for upstream models. Provider keys stay in the IDE/agent; Trim keys stay in Trim: Set API Key (Secret Storage) or CI env.",
      },
      { type: "h2", id: "account", text: "Account and identities" },
      {
        type: "ul",
        items: [
          "Read linked OAuth identities in a table",
          "Link additional providers from linkable_providers when returned",
          "Unlink from row actions only when can_unlink is true (last identity stays fail-closed)",
        ],
      },
      { type: "h2", id: "cli", text: "CLI help" },
      {
        type: "p",
        text: "When chrome provides cli_title / cli_help_lines, Settings shows a read-only card with install and login hints. It does not invent CLI commands client-side.",
      },
      { type: "h2", id: "keys", text: "API keys" },
      {
        type: "ul",
        items: [
          "Create/issue a key - secret shown once with copy",
          "Read - list with search and pagination",
          "Delete - revoke from row actions",
          "Bulk - checkbox selection + Revoke selected",
          "Register device: hardware UUID plus agent select when the API returns agent options",
          "Bound devices list per key; remove-device from row actions when exposed",
        ],
      },
      { type: "h2", id: "ide-bind", text: "Bind the Trim IDE extension (do this once per PC)" },
      {
        type: "ol",
        items: [
          "Leave Start Trim with your IDE checked under Preferences on this same Settings page",
          "Issue a Trim API key → copy the secret once",
          "In VS Code / Cursor: Command Palette → Trim: Copy Hardware ID",
          "On that key’s row → Register device → paste the UUID → agent IDE → save",
          "Editor → Trim: Set API Key → paste the Trim secret; set trim.apiUrl to https://api.use-trim.com (or your API origin)",
          "Still point Cursor / Continue / VS Code Chat at the local proxy …/8888… for compression savings (see Trim IDE extension)",
        ],
      },
      {
        type: "note",
        text: "IDE hardware ID ≠ CLI fingerprint. Register Trim: Copy Hardware ID for the extension; CLI login binds its own machine id. Same laptop may need both if you use extension and CLI with device-bound keys. See Device binding.",
      },
      { type: "h2", id: "danger", text: "Delete account" },
      {
        type: "p",
        text: "Permanent delete is on the home /dashboard page (confirm dialog), not on Settings. Hosted data is removed; machines still need trim uninstall locally.",
      },
      {
        type: "note",
        text: "Compression Always-on / Deep defaults are edited on this same Settings page; see Preferences for behavior detail.",
      },
    ],
  ),

  "dashboard/team": page(
    "Team and seats",
    "Create and manage workspaces, invites, member roles, and allocated seats on Team plans.",
    [
      {
        type: "p",
        text: "Team is master-detail: pick a workspace on the left, then manage members and pending invites on the right. Labels, confirms, role options, and table chrome come from the API (site_messages) - the browser does not invent copy.",
      },
      { type: "h2", id: "workspaces", text: "Workspaces" },
      {
        type: "ul",
        items: [
          "Create - name a workspace; you become owner",
          "Read - list with search and skip pagination",
          "Update - owners rename from the row actions menu",
          "Delete - owners delete from row actions, or select rows and Delete selected (owned workspaces only)",
          "Row checkboxes and bulk delete use the same DataTable selection pattern as members and invites",
        ],
      },
      { type: "h2", id: "members", text: "Members" },
      {
        type: "ul",
        items: [
          "Create/add - Invite by email with an explicit role (member or admin from API options)",
          "Read - list with search and pagination",
          "Update - owners and admins Change role from row actions (admins cannot promote to owner or edit owners; last owner cannot be demoted)",
          "Delete - Remove member (owners/admins) or Leave (yourself when allowed); bulk Remove selected for removable rows",
        ],
      },
      { type: "h2", id: "invites", text: "Pending invites" },
      {
        type: "ul",
        items: [
          "Create - Invite form (email + role); counts against seats with active members",
          "After create, invite_url is shown once with copy when the API returns it",
          "Read - list with search and skip pagination",
          "Delete - Revoke from row actions or Revoke selected",
          "Invitees must sign in with the invited email on an allowed provider",
        ],
      },
      { type: "h2", id: "accept", text: "Accept invite (/invite/{token})" },
      {
        type: "ul",
        items: [
          "Public preview loads workspace name, invited role, expires, and chrome without inventing copy",
          "Signed-out users get a sign-in CTA that returns to the invite path (?next=)",
          "Accept mutation joins the workspace when the signed-in email matches",
          "When the invite is no longer pending, Open team (or equivalent chrome) links into the dashboard",
          "Mismatched email fails closed; owner must correct or resend the invite",
        ],
      },
      {
        type: "ul",
        items: [
          "Seat counts follow the active plan; invites stop when members plus pending invites reach allocated seats",
          "Do not share one login across a whole team",
        ],
      },
      {
        type: "callout",
        title: "Invite email must match",
        text: "If someone signs in with a different email than the invite, acceptance fails closed. Resend or correct the invite.",
      },
      {
        type: "note",
        text: "API routes: GET/POST /workspaces, PATCH/DELETE /workspaces/{id}, GET members and invites, POST invite, DELETE invite, PATCH/DELETE members/{id}, plus public invite preview and accept. Action labels, confirms, role options, and DataTable selection chrome are site_messages-backed.",
      },
    ],
  ),

  "dashboard/preferences": page(
    "Preferences",
    "Always-on with IDE, compression defaults, and local agent status - edited on /dashboard/settings.",
    [
      {
        type: "p",
        text: "Preferences live on Dashboard → Settings (same page as identities and API keys). Deep Mode on enables live Segmented Allowlist Deep after Fast (each last-user prose island in place, order preserved beside paths; tools/paths/schemas frozen) plus trim compress defaults. Off keeps Fast only on the live proxy.",
      },
      { type: "h2", id: "always-on", text: "Start Trim with your IDE" },
      {
        type: "ul",
        items: [
          "One checkbox (default on) synced as auto_start_with_ide",
          "Means: when you open your editor, keep the local Trim proxy ready - it does not compress chat by itself and does not start Trim inside the browser",
          "Toggle patches immediately (does not wait for the compression Save button)",
          "Uncheck anytime - CLI, daemon, and IDE extension honor it (or set trim.autoStartWithIde = off locally)",
          "The browser cannot start Trim by itself; local enforcers apply the preference on your machine",
          "Pair with API keys + IDE hardware registration when using the Trim IDE extension (see Settings and API keys)",
        ],
      },
      { type: "h2", id: "local-agent", text: "Local agent status" },
      {
        type: "p",
        text: "Settings shows local agent online, offline, or unknown from recent device heartbeats (CLI or IDE). The browser cannot probe localhost; open your IDE or run the proxy on the machine to refresh.",
      },
      { type: "h2", id: "compression", text: "Compression defaults" },
      {
        type: "ol",
        items: [
          "Deep Mode is on by default for new accounts (live proxy runs Fast then Segmented Allowlist Deep); switch to Fast in Settings or with trim config set default-tier fast",
          "Deep engine defaults to LLMLingua-2 (v2); switch to long (needs a question) or v1 if you prefer",
          "Drag Compression rate (keep after/before; default 0.7; typically 0.3 to 1.0 step 0.1), Save, then on each machine: trim config sync (or trim config set compression-rate 0.7 while logged in)",
          "Choose Base model (Microsoft LLMLingua-2: mbert-base default, optional xlm-roberta-large), Save, then trim config sync",
          "Or list and set from CLI: trim config rates / models → trim config set compression-rate 0.7 → trim config set deep-v2-backbone mbert-base → trim config sync",
        ],
      },
      {
        type: "p",
        text: "The rate control matches Microsoft LLMLingua demos: lower keeps less of the prompt; 1.0 keeps almost everything. Bounds and step come from Admin billing (deep_compression_rate_min / max / step). The Base model control matches the Hugging Face Space Base Model radio (mbert-base default, optional xlm-roberta-large); catalog and default come from Admin billing (deep_v2_backbone_options / default_deep_v2_backbone). Each catalog row stores the official Hugging Face id (meetingbank checkpoints).",
      },
      {
        type: "callout",
        title: "Honest UI",
        text: "Copy in the product explains that this starts local Trim when the IDE opens - not that the cloud activates Trim worldwide by itself.",
      },
      {
        type: "note",
        text: "Repo-local .trimrc can still refine rules for a single project without changing team defaults. See Always-on with IDE.",
      },
    ],
  ),

  "admin/overview": page(
    "Admin console overview",
    "Operator console for hosted Trim: home KPIs, grouped Sales and revenue nav, users, catalog, chrome, denylists, compliance, and audit.",
    [
      {
        type: "p",
        text: "The admin app (separate origin / port from the customer dashboard) is RBAC-gated. Nav items and page chrome come from the API. The sidebar is an exclusive accordion (docs-style): sections such as Overview, Customers, Sales and revenue, Catalog and billing config, Access and security, Product and growth, and Platform. Shared DataTable selection powers bulk actions. Toasts portal above all Dialog / AlertDialog / Sheet overlays.",
      },
      { type: "h2", id: "access", text: "Access" },
      {
        type: "ul",
        items: [
          "/login - allow-listed admin identity",
          "Permissions gate each route",
          "Step-up (TOTP / WebAuthn) may be required for sensitive writes (StepUpBar + verify dialog)",
          "Shell notification bell: /api/v1/admin/notifications infinite scroll (skip/limit), loading_more chrome while paging, unread / mark-all when chrome allows",
        ],
      },
      { type: "h2", id: "home", text: "Home (/)" },
      {
        type: "ul",
        items: [
          "KPI cards from the admin dashboard payload (MRR/ARR remain subscription proxies)",
          "Link to Sales and revenue for settled receipt analytics",
          "Health, ops checklist, and alerts DataTables when rows exist",
          "Usage chart + LOC / activity heatmap with scope selector (LocHeatmap - same pattern as customer Dashboard)",
        ],
      },
      { type: "h2", id: "surfaces", text: "Main surfaces" },
      {
        type: "ul",
        items: [
          "Users (+ /users/{id} tabs: Edit, Activity, Credits, Danger)",
          "RBAC - roles, permissions, admin invites",
          "Sales and revenue - /sales/revenue, /sales/receipts, /sales/subscriptions, /sales/credits, /sales/enterprise (legacy redirects: /billing/receipts, /billing/subscriptions, /billing/credits, /enterprise; catalog stays at /billing/plans and /billing/settings)",
          "Catalog - /billing/plans, /billing/settings",
          "Product - defaults, limits, cli, churn, fast (+ edit/runtime)",
          "Chrome - site_messages and legal sections (search/patch UI; SeedAndRefresh is API boot, not a chrome button)",
          "Auth - provider allow-list; OAuth scopes read-only; step-up enroll",
          "Denylist - email / IP / ASN tabs",
          "Compliance - retention + access review",
          "Audit - filters + export",
          "Observability - KPI cards, by-mode/model tables, country heatmap DataTable, webhooks replay, CLI notices",
          "Segments (individuals / teams / enterprises), Email, Distribution, Break-glass",
          "Sidebar menu icons are code-only (Lucide map keyed by admin_nav_items.id). Icons are never stored in the database; labels, hrefs, sections, and permissions stay DB-driven",
        ],
      },
      {
        type: "note",
        text: "Customer-facing docs stay under Dashboard. This section is for operators of a hosted or self-hosted Trim deployment. CRUD toasts use the shared Sonner host (body portal, z-9999) so they stay above Dialog / AlertDialog / Sheet. On viewports ≤600px toasts are full-width with equal side insets (Sonner mobileOffset + globals.css); do not force desktop left:auto / right:1rem on mobile.",
      },
    ],
  ),

  "admin/users": page(
    "Admin users",
    "List filters, user detail tabs (Edit, Activity, Credits, Danger), quota, keys, logout, GDPR.",
    [
      { type: "h2", id: "list", text: "Users list" },
      {
        type: "ul",
        items: [
          "Searchable paginated DataTable",
          "Filters: status, plan, country, provider, created range when chrome/API expose them",
          "Row opens /users/{id}",
        ],
      },
      { type: "h2", id: "edit", text: "Detail → Edit" },
      {
        type: "ul",
        items: [
          "Patch status (suspend/restore) with reason when required",
          "Notes and quota / credit adjustments when permitted",
        ],
      },
      { type: "h2", id: "activity", text: "Detail → Activity" },
      {
        type: "ul",
        items: [
          "Identities, workspaces, and devices (hardware UUID) tables when returned",
          "JA4 fingerprints table",
          "Linked-by-JA4 accounts table",
          "Linked-by-hardware accounts table (separate from JA4-linked)",
        ],
      },
      { type: "h2", id: "credits", text: "Detail → Credits" },
      {
        type: "ul",
        items: [
          "Grant credits dialog",
          "Credit grant history and top-up ledger tables for that user",
        ],
      },
      { type: "h2", id: "danger", text: "Detail → Danger" },
      {
        type: "ul",
        items: ["Revoke all API keys", "Force logout", "GDPR export and erase when Perm* allows"],
      },
    ],
  ),

  "admin/rbac": page(
    "Admin RBAC",
    "Roles with permission checkboxes, permissions catalog, admin invites, bulk remove.",
    [
      {
        type: "ul",
        items: [
          "Roles tab - create / edit / delete; permission checkbox sets; bulk delete when chrome allows",
          "Permissions tab - read-only catalog used when assigning roles",
          "Admins tab - invite / remove admin users; bulk remove when allowed",
          "All writes are permission-gated and may require step-up",
        ],
      },
    ],
  ),

  "admin/revenue": page(
    "Admin sales and revenue",
    "Settled receipt analytics under Sales and revenue: KPIs, range filters, stack chart by plan, by-plan table, receipts drill-out.",
    [
      {
        type: "p",
        text: "Revenue is ledger-backed only (billing_receipts with status completed). Catalog price times seats is never treated as revenue here. Free vs paid account counts are conversion metrics beside the chart, not dollar series. Enterprise appears in revenue only after a completed Paddle receipt (same as Pro/Team). Sales Send Paddle offer never creates a charge; entitlements and receipts arrive only from the paid webhook.",
      },
      { type: "h2", id: "route", text: "/sales/revenue" },
      {
        type: "ul",
        items: [
          "Permission billing.revenue",
          "Range presets from admin_revenue_range_presets (7d, 30d, MTD, YTD, custom calendar); first load omits range and the API resolves the first active preset",
          "KPI strip: settled revenue, completed receipts, refunded receipts, free accounts, paid accounts",
          "Stacked area chart by plan (price_id mapped through plan_catalog paddle price ids)",
          "By-plan DataTable + drill-out link href from admin_nav_items id receipts",
          "Command center revenue link href from admin_nav_items id revenue (chrome labels from site_messages)",
        ],
      },
      { type: "h2", id: "sales-nav", text: "Sales and revenue nav group" },
      {
        type: "ul",
        items: [
          "Revenue, Receipts, Subscriptions, Credits, Enterprise inquiries (hrefs and section labels from admin_nav_items / admin_nav_sections)",
          "Dedicated Lucide icons per menu item in admin app code only (keyed by nav item id; never an icon column in Postgres)",
          "Legacy redirects: /billing/receipts, /billing/subscriptions, /billing/credits, and /enterprise into /sales/* (catalog stays at /billing/plans and /billing/settings)",
        ],
      },
    ],
  ),

  "admin/billing": page(
    "Admin billing and catalog",
    "Catalog config stays under /billing/plans and /billing/settings. Day-to-day sales ops live under /sales/* (see Sales and revenue).",
    [
      { type: "h2", id: "plans", text: "/billing/plans" },
      {
        type: "ul",
        items: [
          "Create and patch plan_catalog rows (cents, Unlimited dial, Popular dial, ranks)",
          "Per-save Sync-to-Paddle flag on the plan dialog (subscription, top-up, and enterprise)",
          "Enterprise plan kind is Paddle-priced like Pro/Team after Sync; sales offer then customer checkout uses those pri_* IDs × offered seats",
          "Toolbar catalog sync when permitted",
          "pricing_bound fail-closed until sellable plans are bound",
        ],
      },
      { type: "h2", id: "settings", text: "/billing/settings" },
      {
        type: "ul",
        items: [
          "Tabs: pricing, deep, lists, policy (labels from chrome)",
          "Pricing - annual_discount_percent, currency, proration / upgrade dials, default_seat_quantity, min_seat_quantity, default_plan_interval when returned",
          "Deep - default/min/max/step compression rate (keep after/before), OOM policy, warm on start, LLMLingua-2 Base Model catalog (deep_v2_backbone_options + default_deep_v2_backbone; seeded mbert-base / optional xlm-roberta-large), read-only deep_v2_model (resolved HF id for the billing default backbone), editable deep_v1_model / deep_long_model, force_tokens JSON, deep_force_reserve_digit, deep_drop_consecutive, deep_use_sentence_level_filter (default off for coding; synced to CLI; not inventable in the optimizer)",
          "Lists - pagination page sizes and chart dials (TTL / days / top-N / date_range_months) - not email/IP denylist (see Denylist)",
          "Policy - cancel / downgrade and related policy dials",
        ],
      },
      { type: "h2", id: "sales-ops", text: "Sales ops (moved)" },
      {
        type: "ul",
        items: [
          "Subscriptions - /sales/subscriptions (legacy /billing/subscriptions redirects)",
          "Receipts and disputes - /sales/receipts",
          "Credits and top-ups - /sales/credits",
          "Settled revenue analytics - /sales/revenue",
        ],
      },
    ],
  ),

  "admin/product": page(
    "Admin product",
    "Operator dials for signup defaults, limits, CLI, churn, and Fast Mode (edit + runtime).",
    [
      {
        type: "p",
        text: "Product settings align new-account defaults with customer Preferences (Deep Mode by default = Fast then Segmented Allowlist Deep; Deep engine v2) without inventing client fallbacks. LLMLingua-2 Base Model catalog and default (mbert-base / optional xlm-roberta-large) live under Admin → Billing → Settings → Deep, not Product tabs.",
      },
      { type: "h2", id: "tabs", text: "Config tabs" },
      {
        type: "ul",
        items: [
          "defaults - compression tier / Deep engine; treesitter, deep_attach, model_routing and related signup dials when present",
          "limits - abuse / rate-limit / PoW / JA4 / hardware cap dials, paddle webhook-queue warn depth, and related product limits",
          "cli - min CLI version / force-upgrade related dials when present",
          "churn - churn / retention usage and idle-day dials when present",
          "fast - Fast Mode product dials",
        ],
      },
      { type: "h2", id: "modes", text: "Edit vs runtime" },
      {
        type: "p",
        text: "Outer tabs switch between editable config and runtime readouts when both chrome sections exist. Patch is permission-gated and may require step-up.",
      },
    ],
  ),

  "admin/chrome": page(
    "Admin chrome",
    "Edit site_messages and legal section bodies that drive web/admin UI copy.",
    [
      {
        type: "ul",
        items: [
          "Messages - search/list codes, patch body (this page is search + patch only)",
          "Legal - Terms/Privacy sections patch; publish / unpublish when exposed",
          "Admin shell nav labels come from API ui-map / nav endpoints (read for the shell; not a free-form editor on this page)",
          "Admin shell nav icons are code-only (map by admin_nav_items.id in the admin app). Do not store icon names in the database",
        ],
      },
      {
        type: "callout",
        title: "SeedAndRefresh vs this UI",
        text: "API boot SeedAndRefresh inserts missing site_messages rows only (insert-on-missing). Prefer migrations or Admin chrome patch for copy changes - there is no SeedAndRefresh button on this page.",
      },
      {
        type: "callout",
        title: "No client invent",
        text: "Empty chrome fails closed in the apps. Prefer migrations or Admin chrome edits over hardcoding strings in React.",
      },
    ],
  ),

  "admin/auth": page(
    "Admin auth settings",
    "Hosted auth provider allow-list, OAuth scope chrome (read), and admin step-up factors.",
    [
      {
        type: "ul",
        items: [
          "Auth settings - read/patch provider allow-lists via edit dialog",
          "OAuth scopes table is read-only display chrome for GitHub/GitLab",
          "Step-up enroll (TOTP / WebAuthn) via StepUpBar and verify dialogs",
        ],
      },
    ],
  ),

  "admin/denylist": page(
    "Admin denylist",
    "Block email domains, IP CIDRs, and ASNs with per-tab CRUD and bulk remove.",
    [
      {
        type: "ul",
        items: [
          "Tabs: email domains, IP CIDRs, ASN",
          "Create via add dialog; delete from row actions",
          "Bulk remove with DataTable selection when chrome allows",
        ],
      },
    ],
  ),

  "admin/compliance": page(
    "Admin compliance",
    "Retention TTL fields + purge; access review list, export, attest dialog.",
    [
      { type: "h2", id: "retention", text: "Retention tab" },
      {
        type: "ul",
        items: [
          "Get/patch retention TTL and related fields from the compliance payload",
          "Purge action with reason when Perm* allows",
        ],
      },
      { type: "h2", id: "review", text: "Access review tab" },
      {
        type: "ul",
        items: [
          "List attestations (period, when, notes)",
          "Export when permitted",
          "Attest dialog (limit, notes, reason) creates a new attestation row",
        ],
      },
    ],
  ),

  "admin/audit": page("Admin audit", "Filterable audit log with export and step-up visibility.", [
    {
      type: "ul",
      items: [
        "Filter bar: action / actor / resource and other options from the API",
        "Paginated log DataTable",
        "Export download when PermAuditExport is granted",
        "Step-up related columns appear when the payload includes them",
      ],
    },
  ]),

  "admin/observability": page(
    "Admin observability",
    "KPI summary cards, event stats tables, country heatmap table, webhook list/replay/payload, CLI min version / force-upgrade.",
    [
      {
        type: "ul",
        items: [
          "KPI summary cards when returned: total events, last-24h success/error, tokens before/after",
          "By-mode / by-model (and related) aggregate tables from observability endpoints",
          "Country heatmap DataTable (not the LocHeatmap scope control - that lives on Admin home)",
          "Webhooks DataTable: search, view payload dialog, replay pending events",
          "CLI min_cli_version and force_upgrade_notice cards when present",
        ],
      },
    ],
  ),

  "admin/segments": page(
    "Admin segments",
    "Read individuals, teams, and enterprises with rich filters.",
    [
      {
        type: "ul",
        items: [
          "Tabs: individuals / teams / enterprises",
          "Filters when API options exist: search (q), plan, status, country, interval, credits_left_max, created range (teams/enterprise variants may differ)",
          "Read-oriented DataTables for operator insight; chrome-driven empty states",
        ],
      },
    ],
  ),

  "admin/enterprise": page(
    "Admin enterprise",
    "Enterprise inquiry queue under /sales/enterprise: sales offer seats, customer pays via Paddle, webhook provisions entitlements.",
    [
      {
        type: "p",
        text: "Flow is Paddle-only: inquiry → sales sets offered seats and sends offer (status offered; terms lock) → customer completes Paddle checkout (enterprise plan price × seats) → webhook writes subscription/quotas/receipt and marks the inquiry activated. Admin Send Paddle offer never grants plan_tier or seats by itself. To revise an offer, move status to Contacted, edit seats, then send again.",
      },
      {
        type: "ul",
        items: [
          "List inquiries with email, company, requested seats, message preview, status (new / contacted / offered / closed / activated), and created time",
          "Detail dialog shows the customer message, seats, user link, and timestamps plus sales fields",
          "Patch status (new / contacted / closed), contract notes, and offered seats before sending an offer",
          "Send Paddle offer saves offered seats then marks status offered, notifies the customer (user.enterprise_offer_ready), and requires an active Enterprise plan with Paddle prices",
          "Once Offered, seat quantity and notes are locked; sales may only move status to Contacted (revise) or Closed (withdraw) before editing terms again - re-offer is blocked until then",
          "Customer Dashboard lists inquiries with Pay with Paddle when status is offered; checkout posts inquiry_id and uses offered_seat_quantity",
          "Paddle webhook provisions plan_tier=enterprise, seats, quotas, and receipts the same way as Pro/Team (resolvePlanFromPrice includes plan_kind enterprise), then marks the inquiry activated (paddle_subscription_id / paddle_transaction_id)",
          "Dashboard subscription status after payment matches other paid plans (active + portal/receipts); no free Enterprise contract path",
          "Legacy /enterprise redirects to /sales/enterprise; POST …/activate aliases …/offer",
        ],
      },
    ],
  ),

  "admin/email": page(
    "Admin email templates",
    "Searchable template list and patch-body dialog for transactional email.",
    [
      {
        type: "ul",
        items: [
          "Search / list templates by code",
          "Edit dialog patches body (and related fields when returned)",
          "Missing templates fail closed at send time",
        ],
      },
    ],
  ),

  "admin/distribution": page(
    "Admin distribution",
    "Release distribution stats table with day/month/year range and sync.",
    [
      {
        type: "ul",
        items: [
          "Range filter: day / month / year when chrome exposes options",
          "Stats DataTable columns from API: day, source, metric, path, country, value",
          "Sync action triggers distribution sync when permitted",
        ],
      },
    ],
  ),

  "admin/break-glass": page(
    "Admin break-glass",
    "Request, approve, deny, revoke (row + bulk), and review emergency access.",
    [
      {
        type: "ul",
        items: [
          "Create break-glass request dialog",
          "Approve / deny / revoke with confirms (step-up may apply)",
          "Bulk revoke selected approved rows via DataTable selection toolbar when chrome allows",
          "One searchable table of requests (statused history in the same list - not a separate history page)",
        ],
      },
    ],
  ),

  "api/overview": page(
    "API overview",
    "The hosted API exposes public chrome, authenticated routes, webhooks, and proxy paths.",
    [
      {
        type: "p",
        text: "Hosted API responsibilities include public auth chrome for the marketing site, quotas, payment webhooks, plans, receipts, admin surfaces for operators, and metering. The local CLI speaks to it after trim login.",
      },
      { type: "h2", id: "surfaces", text: "Main surfaces" },
      {
        type: "code",
        language: "text",
        code: `GET  /api/v1/public/auth-providers
GET  /api/v1/public/plans
POST /webhooks/<payment-partner>
Authenticated product routes under /api/v1/...`,
      },
      { type: "h2", id: "clients", text: "Who calls what" },
      {
        type: "ul",
        items: [
          "Browser app: social login + dashboard JSON APIs",
          "CLI: device login, metering, preference sync",
          "Payment partner: signed webhooks for subscription lifecycle",
          "OpenAI-compatible clients: local proxy Base URL (preferred) or configured gateway paths",
        ],
      },
      {
        type: "callout",
        title: "Base URL for IDEs",
        text: "Day-to-day Composer traffic should hit the local Trim listen URL (Always-on with IDE or trim start), not the marketing site origin.",
      },
    ],
  ),

  "api/auth": page(
    "Authentication",
    "Social OAuth into the dashboard; device-bound keys for CLI and automation.",
    [
      {
        type: "p",
        text: "Web login uses social identity providers allow-listed for the deployment (for example Google, GitHub, GitLab). CLI authentication binds devices and can require hardware registration for keys.",
      },
      { type: "h2", id: "web", text: "Dashboard" },
      {
        type: "ul",
        items: [
          "No email/password sign-up on the hosted product path",
          "Session cookies for the web app after provider login",
          "Disconnecting the last identity is blocked so accounts stay recoverable",
        ],
      },
      { type: "h2", id: "cli", text: "CLI and automation" },
      {
        type: "ul",
        items: [
          "Device-code style login via trim login",
          "API keys for automation subject to device binding rules",
          "Integrity and version checks may reject outdated clients",
        ],
      },
    ],
  ),

  "api/proxy": page(
    "Chat completions proxy",
    "OpenAI-compatible chat completions with Trim compression ahead of upstream.",
    [
      {
        type: "p",
        text: "Clients should target the Trim Base URL printed by trim start. The proxy compresses, then forwards to your configured upstream OpenAI-compatible or Anthropic endpoint depending on deployment settings.",
      },
      {
        type: "code",
        language: "bash",
        code: `export OPENAI_BASE_URL=http://127.0.0.1:8888/v1
# use your normal OpenAI-compatible SDK against OPENAI_BASE_URL`,
      },
      { type: "h2", id: "contract", text: "Compatibility notes" },
      {
        type: "ul",
        items: [
          "Chat completions style requests are the primary path",
          "Keep Authorization headers as your client and upstream expect",
          "Streaming clients should follow the same Base URL override",
        ],
      },
      {
        type: "note",
        text: "If the IDE ignores the Base URL, Trim never sees the request. Confirm Settings → Models after every Cursor update.",
      },
    ],
  ),

  "api/errors": page(
    "Errors and quotas",
    "Understand payment-required, auth, and version errors from the hosted API.",
    [
      {
        type: "p",
        text: "The hosted API rejects requests when quotas, auth, or client integrity checks do not pass. Treat these as actionable signals, not bugs to bypass.",
      },
      {
        type: "ul",
        items: [
          "Payment required when quotas are exhausted - upgrade or buy capacity",
          "No payment-required debit path when your plan has Unlimited metering enabled by the operator",
          "Forbidden on blocked downgrade attempts while a paid period is active",
          "Unauthorized when keys are missing, revoked, or not device-bound",
          "Upgrade CLI when the hosted Service requires a newer build",
        ],
      },
      { type: "h2", id: "debug", text: "Debug checklist" },
      {
        type: "ol",
        items: [
          "Read the exact error body returned to the CLI or dashboard",
          "Confirm plan status and usage in the dashboard",
          "Confirm key and device binding under Settings",
          "Reinstall or upgrade the CLI if version is rejected",
        ],
      },
    ],
  ),

  "security/overview": page(
    "Security overview",
    "Defense in depth across local proxy, device binding, abuse prevention, and billing integrity.",
    [
      {
        type: "p",
        text: "Trim combines local prompt handling, authenticated metering, Merchant of Record billing, and limited security telemetry to protect accounts and free-tier abuse without building advertising profiles on developers.",
      },
      {
        type: "ul",
        items: [
          "Device-bound keys and hardware registration where required",
          "Plan rank and proration rules enforced server-side",
          "Minimum client version requirements for security patches",
          "Operator controls for denylists and runtime settings",
        ],
      },
      { type: "h2", id: "report", text: "Report a problem" },
      {
        type: "p",
        text: "Report suspected vulnerabilities or account takeover to the support email published on Privacy and Terms. Include enough detail to reproduce without sending secrets in clear text when avoidable.",
      },
    ],
  ),

  "security/device-binding": page(
    "Device binding",
    "How hardware registration ties keys and CLI sessions to machines.",
    [
      {
        type: "p",
        text: "When device binding is required, CLI login binds the issuing machine. Dashboard-issued keys for IDE or CI may need hardware registration under Settings before authentication succeeds.",
      },
      {
        type: "ul",
        items: [
          "Reduces casual key sharing across many free accounts",
          "Uses hashed machine identifiers rather than raw serial dumps in product logs",
          "Operators can adjust policy for enterprise deployments",
        ],
      },
      { type: "h2", id: "ide-vs-cli", text: "IDE hardware ID ≠ CLI hardware ID" },
      {
        type: "p",
        text: "The same laptop has two different fingerprints. Mixing them is the most common setup failure for Trim IDE.",
      },
      {
        type: "ul",
        items: [
          'IDE: Trim IDE extension builds SHA256(editor machineId + "|trim-ide") as a hex string. Users never type it - Command Palette → Trim: Copy Hardware ID copies it and shows a confirmation toast. Cloud calls send X-Hardware-UUID plus X-Trim-Agent-Id: ide',
          "CLI: trim login / CLI API uses a separate OS fingerprint (machineid ProtectedID salt TrimCLI, then SHA256). Do not paste that value into an IDE Register device dialog",
          "CI: use the CI agent option and the hardware UUID your pipeline documents - not the IDE command",
        ],
      },
      {
        type: "callout",
        title: "Auth loop (IDE)",
        text: "Copy Hardware ID → Register device on the Trim API key (agent IDE) → Trim: Set API Key + trim.apiUrl. Later flushes/quota/prefs send Bearer + X-Hardware-UUID + X-Trim-Agent-Id: ide. Server allows only if that UUID is registered on the key.",
      },
      { type: "h2", id: "ide-steps", text: "Register an IDE device on an API key" },
      {
        type: "ol",
        items: [
          "Install Trim IDE in VS Code or Cursor",
          "Trim: Copy Hardware ID",
          "Dashboard → Settings → API keys → issue a key if needed",
          "Key row → Register device → paste UUID → agent IDE → save",
          "Editor → Trim: Set API Key with that Trim secret (not a provider key)",
        ],
      },
      {
        type: "callout",
        title: "Common mistakes",
        text: "CLI UUID registered for an IDE key → extension rejected. IDE UUID registered but never Trim: Set API Key → no auth. Key set but device never registered → device-bound auth fails. Second PC without registering its new Copy Hardware ID → second machine fails. Cursor OpenAI/Anthropic key confused with Trim API key → wrong secret entirely. Same machine using both CLI and extension → register both IDs (or separate keys).",
      },
      { type: "h2", id: "laptop-swap", text: "Replacing a machine" },
      {
        type: "ol",
        items: [
          "Sign in on the new machine and complete device registration (new Trim: Copy Hardware ID for the extension; trim login for CLI)",
          "Revoke keys issued for the old machine when you no longer need them",
          "Re-run trim login if the CLI session is stale; re-run Trim: Set API Key in the new editor",
        ],
      },
      {
        type: "note",
        text: "Product walkthrough: Trim IDE extension and Settings and API keys.",
      },
    ],
  ),

  "security/fraud": page(
    "Abuse prevention",
    "Signals used to protect free tiers and paid entitlements.",
    [
      {
        type: "p",
        text: "Hosted Trim may associate limited device, network, and session signals with accounts to detect multi-account abuse and protect paid entitlements. Signals are for security enforcement, not advertising profiles.",
      },
      {
        type: "ul",
        items: [
          "Hashed machine identifiers from authenticated CLI sessions",
          "IP and related edge security signals when available",
          "Velocity checks that stop abuse when quotas or account limits are exceeded",
        ],
      },
      {
        type: "p",
        text: "If you believe your account was restricted in error, contact support with enough detail to verify ownership. Do not attempt to bypass metering or authentication. Enabling Unlimited metering in the plan catalog is operator configuration on the Service, not a client-side quota bypass.",
      },
    ],
  ),

  "security/telemetry": page(
    "Telemetry",
    "Optional anonymous product telemetry and how to disable it.",
    [
      {
        type: "p",
        text: "Optional anonymous product telemetry can be disabled in the product or via supported client settings (including Do Not Track style opt-outs where implemented). Self-hosted operators control whether cloud anti-abuse and billing checks run in their deployment mode.",
      },
      {
        type: "p",
        text: "Official installers (install.sh / install.ps1) may send a soft-fail privacy-light install hit that records only the installer path and CDN country-not your identity or machine fingerprint. Set DO_NOT_TRACK=1 or TRIM_TELEMETRY_DISABLED=1 to skip it; a failed ping never blocks install.",
      },
      {
        type: "ul",
        items: [
          "Telemetry is separate from metering required to bill hosted plans",
          "Disable optional analytics when your organization requires it",
          "Security and auth logs may still exist for operating the Service",
        ],
      },
      {
        type: "note",
        text: "Turning off optional telemetry does not delete historical billing or security records required to run the account.",
      },
    ],
  ),

  "self-hosting/overview": page(
    "Self-hosting overview",
    "Run the API, web app, and data plane on your own infrastructure.",
    [
      {
        type: "p",
        text: "Self-hosting is supported for operators who want their own Trim deployment. You will run a Postgres database, Redis, the API, and the Next.js web app, then point CLI installs at your API.",
      },
      {
        type: "ol",
        items: [
          "Apply database migrations in order from supabase/migrations (include plan_catalog.unlimited 188, Terms/Privacy Unlimited wording 189-190, landing pricing badges 191, Sync-to-Paddle DESC 192, plan_catalog.popular 193, drop obsolete Popular plan-id chrome 194, Sync-to-Paddle modal copy 195, dashboard Usage/activity chrome 196-197, Deep/engine defaults 198-200, workspace CRUD + bulk-delete chrome 201-202, enterprise inquiry detail chrome 203, Sales and revenue nav sections + settled revenue analytics 204, enterprise activate notify/unlimited 205, enterprise Paddle-only offer checkout 206, enterprise Paddle chrome align 207, notification loading-more chrome ensure 208, SEO and 404/error page chrome 209, Contact page chrome 210, notification deep-links 211, dedicated Traces/Receipts/Enterprise dashboard pages 212, DataTable selection + bulk-delete chrome for Traces/Enterprise 213, provider adapters 221, last-door chrome 222, port 8888 align 223, Anthropic OpenAI-compat clarify 224)",
          "Configure server/.env from .env.example (boot must fail if required keys are missing)",
          "Configure apps/web/.env.local from the web example",
          "Boot API and web behind HTTPS",
          "Install CLI and point cloud endpoints at your domain",
          "Configure identity providers and payment webhooks if you offer billing",
          "Manage Unlimited metering and Popular dials in Admin → Plans; annual Switch / Save-% / badge labels via site_messages (no client invent)",
        ],
      },
      { type: "h2", id: "responsibilities", text: "Operator responsibilities" },
      {
        type: "ul",
        items: [
          "Security updates, backups, and secret rotation",
          "Legal notices you publish to your users",
          "Capacity planning for Redis and Postgres",
          "Support for your own customers",
        ],
      },
      {
        type: "note",
        text: "Follow the root README for DNS, payment partner, and provider setup. Hosted use-trim.com terms do not automatically cover your self-hosted users.",
      },
    ],
  ),

  "self-hosting/environment": page(
    "Environment",
    "Required configuration surfaces for API and web.",
    [
      {
        type: "p",
        text: "Cloud-style deployments expect explicit env for database, Redis, JWT or auth secrets, identity provider settings, payment partner keys, CORS, timeouts, and company letterhead fields used on receipts. Missing required keys should prevent boot rather than invent defaults.",
      },
      {
        type: "ul",
        items: [
          "server/.env.example is the source of truth for API variables",
          "apps/web/.env.example covers Next public tokens and API URL",
          "Official hosted CLI needs no env; self-host / local go builds use cli/.env.example (especially TRIM_CLI_HMAC_SECRET matching the server)",
        ],
      },
      { type: "h2", id: "checklist", text: "Before production traffic" },
      {
        type: "ol",
        items: [
          "Confirm migrations applied",
          "Confirm HTTPS and CORS origins",
          "Confirm support email and legal pages",
          "Confirm payment webhooks only accept signed events",
          "Confirm free-tier abuse settings match your risk tolerance",
        ],
      },
    ],
  ),

  "self-hosting/deploy": page(
    "Deploy the API",
    "Typical production shape: containerized API behind HTTPS, web on a frontend host.",
    [
      {
        type: "p",
        text: "A common layout is the API in Docker behind a TLS terminator, the web app on a frontend host, Postgres managed, Redis managed, and DNS with an edge proxy. Wire payment webhooks to your public API webhook path.",
      },
      {
        type: "code",
        language: "text",
        code: `Dockerfile path: server/Dockerfile
Docker context: server
Optional build features: follow README flags for native parsers`,
      },
      { type: "h2", id: "health", text: "After deploy" },
      {
        type: "ul",
        items: [
          "Hit public auth-providers and confirm chrome loads",
          "Complete one social login in a private window",
          "Run trim login against your API",
          "Send one chat completion through trim start",
          "Confirm a test purchase webhook in staging before production",
        ],
      },
    ],
  ),

  "guides/troubleshooting": page(
    "Troubleshooting",
    "Fix common CLI, IDE Base URL, auth, and quota issues when Trim does not behave as expected.",
    [
      {
        type: "p",
        text: "Work through these checks in order. Most production issues are Base URL misconfiguration, an outdated CLI, or a quota that needs an upgrade.",
      },
      { type: "h2", id: "proxy-not-listening", text: "Proxy is not listening" },
      {
        type: "ol",
        items: [
          "Confirm Always-on with IDE is checked, or run trim start",
          "Check trim autostart status and extension Output → Trim",
          "Open the health URL on the port trim start printed (example http://127.0.0.1:8888/health) and confirm status ok",
          "Ensure nothing else bound the same port without a healthy Trim response",
          "Run trim stop then trim start if you need a clean manual session",
        ],
      },
      {
        type: "h2",
        id: "os-logon-blocked",
        text: "OS logon autostart failed (Access denied)",
      },
      {
        type: "ul",
        items: [
          "Your Trim preference can still be on - Always-on with IDE and the Trim IDE extension do not require OS logon",
          "Retry: trim autostart enable (Windows uses Startup Run → %LOCALAPPDATA%\\Trim\\bin; no Administrator on a normal PC)",
          "Manual path: trim start",
          "IDE path: keep Start Trim with your IDE checked and open the editor with the Trim IDE extension",
          "Enterprise PCs: if Startup programs are blocked by policy, use manual or IDE start - that is expected",
          "Need raw OS text: trim autostart enable -v (Details row only)",
        ],
      },
      { type: "h2", id: "cursor-ignores-trim", text: "Cursor still hits the provider directly" },
      {
        type: "ul",
        items: [
          "Open Settings → Models and confirm Override OpenAI Base URL points at Trim Cursor Link (https://…/cl/<token>/v1 from trim cursor-link url) - not api.openai.com and not 127.0.0.1 for Agent",
          "If you see “Access to private networks is forbidden”, you still have localhost pasted - run trim cursor-link enable && trim start && trim cursor-link url and re-paste",
          "Remove a trailing path mistake (Cursor Link URL must end with /v1)",
          "Send a new Agent / Composer request after saving settings",
          "Confirm local Trim is healthy (http://127.0.0.1:8888/dashboard) - the tunnel only forwards to a live proxy",
        ],
      },
      {
        type: "h2",
        id: "private-networks-forbidden",
        text: "Cursor: Access to private networks is forbidden",
      },
      {
        type: "ol",
        items: [
          "This is Cursor blocking 127.0.0.1 from its cloud Agent path - not a Trim crash",
          "Enable Cursor Link: trim cursor-link enable",
          "cloudflared: Trim auto-downloads on enable/start; see Cursor setup / Trim Cursor Link if blocked",
          "trim start → trim cursor-link url → paste https://…/v1 into Override OpenAI Base URL",
          "Keep using localhost only for Claude Code CLI and other on-device clients",
        ],
      },
      {
        type: "h2",
        id: "continue-not-trim",
        text: "Continue / VS Code Chat not hitting Trim",
      },
      {
        type: "ul",
        items: [
          "Continue: confirm ~/.continue/config.yaml apiBase is http://127.0.0.1:8888/v1 and you opened the Continue sidebar (model name from yaml, no 0 credits)",
          "VS Code Chat Custom Endpoint: url must be http://127.0.0.1:8888/v1/chat/completions (full path); apiKey must be wizard ${input:chat.lm.secret…}",
          "Windsurf / Zed / JetBrains: see dedicated IDE docs for exact settings files; Base URL must stay Trim …/v1",
          "Wrong panel: Custom Endpoint/… • 0 credits means built-in Chat, not Continue",
          "Model id: use GET /v1/models - prefer trim-claude-sonnet if a dated Claude id returns not found",
          "401 empty Bearer: raw sk- in chatLanguageModels.json was ignored - re-add via Add Models wizard",
          "ERR_CONTENT_DECODING_FAILED: upgrade Trim proxy (encoding fix) and retry; do not point at api.anthropic.com",
          "trim help setup still shows old Aider-only text: apply site_messages migration 241, reload API chrome, delete ~/.config/trim/cli-chrome-cache.json, then re-run trim help setup",
        ],
      },
      { type: "h2", id: "login-fails", text: "trim login fails" },
      {
        type: "ul",
        items: [
          "Confirm network access to the hosted API",
          "Complete the browser device-code flow with an allowed social provider",
          "Upgrade the CLI if the hosted Service requires a newer build",
          "For device-bound keys, register hardware under Settings while signed in",
        ],
      },
      { type: "h2", id: "ide-hardware", text: "Trim IDE: hardware / API key errors" },
      {
        type: "ul",
        items: [
          "Use Trim: Copy Hardware ID - not the CLI fingerprint (IDE ID ≠ CLI ID)",
          "Dashboard → Settings → API keys → Register device → paste → agent IDE",
          "Trim: Set API Key uses the Trim Cloud key from Settings, not Cursor’s OpenAI/Anthropic provider key",
          "Set trim.apiUrl (hosted https://api.use-trim.com); empty URL keeps the extension inactive",
          "Open Output → Trim for fail-closed chrome / auth messages; see Device binding and Trim IDE extension",
        ],
      },
      { type: "h2", id: "quota-errors", text: "Payment required or quota errors" },
      {
        type: "p",
        text: "Hosted metering returns payment-required style errors when free or plan quotas are exhausted and Unlimited metering is not enabled on your plan. Open Dashboard home (usage) or plan CTAs, review usage, upgrade the plan, or buy a top-up. If the dashboard shows Unlimited, cloud requests are not debited until an operator turns that plan dial off. Reinstalling the CLI does not reset cloud quotas.",
      },
      { type: "h2", id: "quality-regressions", text: "Model quality looks worse after trim" },
      {
        type: "ul",
        items: [
          "Confirm active files and stacks you need are kept (check Fast Mode activity)",
          "Raise Compression rate toward 1.0 (less drop) in Dashboard → Settings, Save, then trim config sync",
          "If Deep engine is v2, try Base model mbert-base first (faster) or xlm-roberta-large when quality needs the heavier backbone (more RAM/slower); list with trim config models",
          "Try Deep Mode: turn Preferences Deep on (live proxy) and/or trim compress --deep for file/batch jobs",
          "If history_keep_turns is set, try 0 temporarily - stubbing old turns can hide details the model still needs",
          "Compare tokens in vs out on trim status or the usage table",
        ],
      },
      { type: "h2", id: "zero-saved-meter", text: "Local meter shows 0% Saved" },
      {
        type: "ul",
        items: [
          "Open http://127.0.0.1:8888/dashboard → Show savings detail",
          "Check Deep status: applied, chrome frozen / no prose islands, or fail-closed kept Fast",
          "Compare Deep stage tokens to whole-request Saved - Segmented Allowlist Deep meters prose islands only (order preserved beside paths); agent chrome stays frozen on purpose",
          "0% whole-request Saved with a correct Claude Code / Cursor answer is often normal, not a broken install",
        ],
      },
      { type: "h2", id: "uninstall-leftovers", text: "Uninstall left files or models behind" },
      {
        type: "ul",
        items: [
          "Run trim uninstall (full purge by default), or curl -fsSL https://use-trim.com/uninstall.sh | sh / irm https://use-trim.com/uninstall.ps1 | iex",
          "On Windows, Settings → Apps → Trim runs the same full purge including %USERPROFILE%\\.trim and Deep Hugging Face hub dirs from site_messages (both mbert and xlm-roberta-large meetingbank caches when configured)",
          "Clear the IDE key and uninstall the Trim IDE extension in the editor",
          "On Windows, trim uninstall renames the running binary; a detached post-exit helper deletes it and removes detected package installs (Windows cannot delete a mapped exe while it runs). A reboot-pending delete is also registered so a killed helper cannot leave vendor\\trim.exe",
          "trim uninstall schedules removal of detected npm / Scoop / Homebrew / winget installs after exit; if one remains, run the matching package-manager uninstall printed as a fallback",
          "Deleting a cloud account does not wipe local machines-uninstall on each device",
        ],
      },
    ],
  ),

  "guides/faq": page(
    "FAQ",
    "Short answers to the questions teams ask before adopting Trim in Cursor, VS Code, and CI.",
    [
      { type: "h2", id: "always-on", text: "Do I need to run trim start every day?" },
      {
        type: "p",
        text: "No. Leave Start Trim with your IDE checked, install the Trim IDE extension and/or trim autostart enable. Open the IDE and Trim stays ready. Use trim start when you want a one-shot manual session. See Always-on with IDE.",
      },
      {
        type: "h2",
        id: "windows-admin",
        text: "Do I need Administrator on Windows for autostart?",
      },
      {
        type: "p",
        text: "No for normal PCs. Trim registers a per-user Startup Run key pointing at %LOCALAPPDATA%\\Trim\\bin\\trim.exe (same pattern as browsers and editors), not an admin Scheduled Task and not a lock on npm’s package tree. If your organization blocks Startup programs, keep the preference on and use trim start or the Trim IDE extension. See trim autostart.",
      },
      {
        type: "h2",
        id: "does-trim-upload-code",
        text: "Does Trim upload my repo to use-trim.com?",
      },
      {
        type: "p",
        text: "Compression runs in the local proxy on your machine (Fast only, or Fast then Segmented Allowlist Deep when Deep Mode is on). The hosted Service stores account, billing, and metering data. Full prompt bodies are not required in Trim cloud for the local proxy path. Upstream model providers still receive the slim prompt you forward.",
      },
      { type: "h2", id: "fast-vs-deep", text: "When should I use Deep Mode?" },
      {
        type: "p",
        text: "Use Fast Mode for interactive IDE chat when Deep Mode is off. When Deep Mode is on in Preferences, trim start runs Fast then Segmented Allowlist Deep (each last-user prose island in place, order preserved beside paths; tools/paths/schemas frozen) at your compression rate (default 0.7) and, for LLMLingua-2 (v2), your Base Model (default mbert-base; optional xlm-roberta-large). Set rate and Base model in Dashboard → Settings, or with trim config rates / models and trim config set compression-rate / deep-v2-backbone, then trim config sync. File/batch: trim compress --deep --rate. Engines are not loaded in Trim cloud.",
      },
      {
        type: "h2",
        id: "history-keep-turns",
        text: "What is history_keep_turns?",
      },
      {
        type: "p",
        text: "An optional sliding-window setting for huge chats. Default off. When set (for example history_keep_turns=8 in .trimrc), Trim may stub older eligible text turns while keeping tools, files, and protocol chrome intact. Most users should leave it off. See history_keep_turns.",
      },
      {
        type: "h2",
        id: "zero-saved-claude-code",
        text: "Why does the local dashboard show 0% Saved on Claude Code or Cursor Agent?",
      },
      {
        type: "p",
        text: "Usually because most of the request is frozen agent chrome (system reminders, tools, git status). Trim protects that on purpose. Eligible last-user prose still runs Segmented Allowlist Deep; paths and schemas stay verbatim. Open Show savings detail for Deep status and Deep stage (prose) savings. Whole-request 0% with a correct answer is not a broken install.",
      },
      { type: "h2", id: "works-with-other-ides", text: "Does Trim work outside Cursor?" },
      {
        type: "p",
        text: "Yes. Documented first-class paths: Cursor, Continue (config.yaml), VS Code Chat Custom Endpoint (Ask), Claude Code CLI only (no IDE - see Claude Code (CLI only)), Windsurf/Zed/OpenAI SDKs, and curl. Install the Trim IDE extension in VS Code or Cursor for acceptance / LOC telemetry when you use those editors. See Connect any IDE for the honest supported matrix (VS Code Chat Agent + Custom Endpoint is not guaranteed).",
      },
      {
        type: "h2",
        id: "private-networks-forbidden",
        text: "Cursor says Access to private networks is forbidden - what now?",
      },
      {
        type: "p",
        text: "Cursor Agent’s cloud cannot call http://127.0.0.1:8888/v1. Enable Trim Cursor Link: trim cursor-link enable → trim start (Trim auto-downloads cloudflared when missing) → paste trim cursor-link url (https://…/cl/<token>/v1) into Cursor Override OpenAI Base URL. Compression still runs on your PC. Full guides: Cursor setup and Trim Cursor Link. Do not keep localhost in Override for Agent.",
      },
      {
        type: "h2",
        id: "claude-code-only",
        text: "I only use Claude Code - no Cursor. What do I do?",
      },
      {
        type: "p",
        text: "Follow Claude Code (CLI only): install Trim → trim login → trim start → set ANTHROPIC_BASE_URL=http://127.0.0.1:8888 (no /v1) and ANTHROPIC_API_KEY → run claude. Do not open Cursor Settings → Models. That page has separate Windows / macOS / Linux copy blocks (Copy button on each), permanent profile steps, verify, and troubleshooting. Optional: trim setup prints the copy/paste block for your listen port. You do not need Cursor Link for Claude Code.",
      },
      {
        type: "h2",
        id: "continue-vs-vscode-chat",
        text: "Continue vs VS Code built-in Chat - which do I use?",
      },
      {
        type: "p",
        text: "Continue = extension sidebar + ~/.continue/config.yaml (best for Agent file edits through Trim today). VS Code Chat Custom Endpoint = Chat: Manage Language Models + chatLanguageModels.json with full …/v1/chat/completions URL and wizard secrets (best for Ask chat). If the footer shows Custom Endpoint/… • 0 credits, you are not in Continue.",
      },
      {
        type: "h2",
        id: "custom-models",
        text: "How do I add custom models like trim-claude-sonnet?",
      },
      {
        type: "p",
        text: "In your IDE model settings (Cursor Settings → Models, Continue ~/.continue/config.yaml, or VS Code chatLanguageModels.json), create entries named trim-claude-sonnet, trim-claude-haiku, trim-gemini-flash-latest, and similar. Point Base URL at Trim, use the matching provider key, and prefer shipped Trim aliases from GET /v1/models. Naming ideas and exact places: Custom models. Ops can add worldwide aliases under Admin → Billing → Provider adapters.",
      },
      {
        type: "h2",
        id: "where-settings",
        text: "Where do I change Always-on, API keys, and plan prices?",
      },
      {
        type: "p",
        text: "Always-on: https://use-trim.com/dashboard → Preferences → Start Trim with your IDE. Trim IDE extension key: Dashboard → Settings → API keys. Plans / prices / quotas: Dashboard → Billing (or Plans). Provider chat keys (OpenAI / Anthropic / Google) stay in the IDE model settings, not in Trim Dashboard.",
      },
      { type: "h2", id: "billing", text: "Who bills me?" },
      {
        type: "p",
        text: "Hosted plans and top-ups are billed through our payment partner as Merchant of Record. Plan prices and Unlimited metering flags live in the product catalog (Unlimited only when an operator enables that plan dial) - see them in Dashboard → Billing. Self-hosted operators manage their own commercial terms.",
      },
      { type: "h2", id: "teams", text: "How do team seats work?" },
      {
        type: "p",
        text: "Team plans allocate seats from the catalog. In Dashboard → Team, owners create, rename, and delete workspaces; owners and admins invite by email with a role (invite_url shown once with copy), change member roles, remove members, and revoke pending invites. Invitees must sign in with the invited email on an allowed provider. Checkbox selection supports bulk delete (owned workspaces), bulk remove (members), and bulk revoke (invites).",
      },
      { type: "h2", id: "uninstall", text: "How do I fully remove Trim from my machine?" },
      {
        type: "p",
        text: "Run trim uninstall (full purge by default). On Windows you can also use Settings → Apps → Trim. That stops Trim processes, clears credentials and setup overrides, deletes local data and Deep Mode caches, removes the CLI binary (scheduled after exit when Windows locks it), wipes %LOCALAPPDATA%\\Trim, and auto-removes detected npm / Scoop / Homebrew / winget installs. Then Clear API Key and uninstall the Trim IDE extension in the editor. Delete the cloud account from the home Dashboard (confirm dialog) only if you also want hosted data removed. See Uninstall.",
      },
      { type: "h2", id: "admin", text: "Where is the Admin console documented?" },
      {
        type: "p",
        text: "Operators use a separate admin app (RBAC + step-up). Docs live under Admin console in the sidebar: overview, users, RBAC, billing catalog, sales and revenue, product, chrome, auth, denylist, compliance, audit, observability, segments, enterprise, email, distribution, and break-glass. Customer FAQ stays customer-focused.",
      },
      {
        type: "callout",
        title: "Still stuck?",
        text: "See Troubleshooting, or open Privacy / Terms for legal commitments. Contact the configured support email for account issues.",
      },
    ],
  ),
};

export function getDocPage(slug: string): DocPage | null {
  return DOC_PAGES[slug] ?? null;
}

export function allDocSlugs(): string[] {
  return Object.keys(DOC_PAGES);
}

export type { ReactNode };
