# Trim IDE (VS Code / Cursor)

![Trim](https://raw.githubusercontent.com/usetrim/trim/main/extensions/trim-ide/media/banner-dark.png)

**Trim IDE** helps your editor work with [Trim](https://use-trim.com): it can keep the local **Fast Mode** proxy ready when Always-on is enabled, posts tab acceptance / optional AI LOC metrics to Trim Cloud, and helps you point **Cursor Agent** (Cursor Link HTTPS) or local chat clients at Trim.

This extension is **not** the compressor. Token savings happen when your chat tool sends traffic through Trim. **Continue / VS Code Chat / Claude Code** use the **local** proxy (typically `http://127.0.0.1:8888/v1`, or Claude Code origin without `/v1`). **Cursor Agent** must use **Trim Cursor Link** (`trim cursor-link url` → HTTPS `…/cl/<token>/v1`), not `127.0.0.1` (Cursor cloud blocks private networks). That chat path is separate from this extension’s cloud metrics.

| | |
| --- | --- |
| **Install** | Search **Trim IDE** (publisher `usetrim`) in VS Code, or install `usetrim.trim-ide` from [Open VSX](https://open-vsx.org/extension/usetrim/trim-ide) in Cursor |
| **Docs** | [use-trim.com/docs/ide/extension](https://use-trim.com/docs/ide/extension) |
| **Privacy** | [use-trim.com/privacy](https://use-trim.com/privacy) · [PRIVACY.md](https://github.com/usetrim/trim/blob/main/extensions/trim-ide/PRIVACY.md) |
| **Security** | [SECURITY.md](https://github.com/usetrim/trim/blob/main/extensions/trim-ide/SECURITY.md) |
| **Support** | [SUPPORT.md](https://github.com/usetrim/trim/blob/main/extensions/trim-ide/SUPPORT.md) |

![Setup](https://raw.githubusercontent.com/usetrim/trim/main/extensions/trim-ide/media/screenshot-setup.png)

## Two paths

| Path | Who | Purpose |
| --- | --- | --- |
| Chat → Cursor Link (HTTPS `…/cl/<token>/v1`) | **Cursor Agent** | Compression via public door (not `127.0.0.1`) |
| Chat → local proxy (`:8888`) | Continue / VS Code Chat / Claude Code | Compression on-device |
| Extension → Trim Cloud | This extension (`trim.apiUrl` + Trim API key) | Keep proxy ready + acceptance / LOC charts |

If you install the extension but never point your chat Base URL at Trim, you still get auto-start + metrics - **not** full compression savings.

## Install

### VS Code Marketplace

1. Extensions → search **Trim IDE** by **usetrim**
2. Install → reload if prompted
3. Open the **Get started with Trim IDE** walkthrough (Help → Welcome → Walkthroughs), or follow **Configure** below

### Open VSX (Cursor and other Open VSX clients)

Install **Trim IDE** (`usetrim.trim-ide`) from [Open VSX](https://open-vsx.org/extension/usetrim/trim-ide).

### Offline / air-gapped

Build or download a `.vsix`, then **Extensions: Install from VSIX**.  
(From a clone of this repo: `cd extensions/trim-ide && npm ci && npm test && npm run package`.)

Also install the **`trim` CLI** on the machine and keep it on `PATH` ([installation](https://use-trim.com/docs/installation)) so Always-on can start the local proxy.

## First-run checklist

| Step | Where | Action |
| --- | --- | --- |
| 1 | Dashboard → Settings | Leave **Start Trim with your IDE** checked |
| 2 | Dashboard → Settings → API keys | Create a **Trim** API key (not OpenAI / Anthropic / Google) |
| 3 | Editor | **Trim: Copy Hardware ID** |
| 4 | Same key → Register device | Paste the ID → agent **IDE** (not a CLI / `trim login` fingerprint) |
| 5 | Editor | **Trim: Set API Key** → paste the Trim key |
| 6 | Editor settings | `trim.apiUrl` = `https://api.use-trim.com` |
| 7 | Chat Base URL | **Cursor Agent:** `trim cursor-link enable` → `trim start` → **Trim: Copy Cursor Agent Base URL** (or `trim cursor-link url`) → paste HTTPS `…/v1` into Override OpenAI Base URL (not `127.0.0.1`). **Continue / VS Code / etc.:** local `http://127.0.0.1:8888/v1` from `trim start`. Pick a Trim model id such as `trim-claude-sonnet`. |

**Quick model:** Always-on = auto-start preference · Trim API key = Trim Cloud login · Hardware ID = this editor install may use that key.

**Chat models (separate from this extension):** create friendly custom model entries in the editor (`trim-claude-sonnet`, `trim-claude-haiku`, `trim-gemini-flash-latest`, …). Step-by-step with copy buttons: [Custom models](https://use-trim.com/docs/ide/custom-models) · [Cursor](https://use-trim.com/docs/ide/cursor) · [Continue](https://use-trim.com/docs/ide/continue) · [VS Code Chat](https://use-trim.com/docs/ide/vscode).

## Configure

1. Settings → **Trim: Api Url** → `https://api.use-trim.com` (or your API origin)
2. Command Palette → **Trim: Copy Hardware ID** → Dashboard → register as agent **IDE**
3. Command Palette → **Trim: Set API Key**
4. Leave **Trim: Auto Start With Ide** on `follow` (uses the cloud preference), or set `on` / `off` locally
5. Cursor Agent: enable Cursor Link in the CLI, then **Trim: Copy Cursor Agent Base URL** and paste into Override OpenAI Base URL (never auto-enables the tunnel)
6. Optional: auto-flush interval, document LOC tracking, workspace UUID, HTTP timeout

If the API URL is empty or Trim Cloud cannot be reached on first run, the extension stays inactive and logs to **Output → Trim**. After a successful sync, a cache keeps labels working offline.

See [Device binding](https://use-trim.com/docs/security/device-binding).

## Always-on with IDE

- When Always-on is enabled, the extension health-checks the local proxy and runs `trim start` if needed
- If a healthy proxy already owns the port, it attaches (no double-bind)
- It only runs `trim start` (no separate `trim compress` process). Live Segmented Allowlist Deep still runs inside the proxy when Dashboard Preferences set Deep on; file/batch Deep stays `trim compress --deep`
- While the IDE stays open, it re-checks the dashboard preference so turning Always-on off can stop auto-start
- Proxy start failures do **not** block the editor (you’ll see a warning; details in **Output → Trim**)
- The browser cannot start Trim; this extension and/or the CLI handle the local proxy
- Optional OS logon (`trim autostart enable`): macOS launchd, Linux systemd user, Windows Startup Run → `%LOCALAPPDATA%\Trim\bin` (no admin on a normal PC). If OS registration fails, keep Always-on on and use this extension or `trim start` - see [trim autostart](https://use-trim.com/docs/cli/autostart)

## Settings

| Setting | Purpose |
| --- | --- |
| `trim.apiUrl` | API base (`http`/`https` only; empty = off) |
| `trim.autoStartWithIde` | `follow` (cloud), `on`, or `off` |
| `trim.httpTimeoutSec` | HTTP timeout for cloud calls |
| `trim.autoFlushSeconds` | Metrics batch interval (`0` = off; default) |
| `trim.trackDocumentEdits` | Optional multi-line LOC estimate |
| `trim.minLinesForAiHeuristic` | Min lines per edit to count (use ≥ 1 when tracking is on) |
| `trim.workspaceId` | Optional workspace UUID for shared credits |

Example when enabling LOC estimates:

```json
{
  "trim.apiUrl": "https://api.use-trim.com",
  "trim.autoFlushSeconds": 30,
  "trim.trackDocumentEdits": true,
  "trim.minLinesForAiHeuristic": 3
}
```

## Privacy & behaviour (short)

- API key lives in editor **Secret Storage** (never in `settings.json`)
- Cloud calls use your API key, a hashed hardware id, and Trim IDE client headers
- Pending metrics are stored locally until a successful upload
- Diagnostics: **Output → Trim**
- Full details: [PRIVACY.md](https://github.com/usetrim/trim/blob/main/extensions/trim-ide/PRIVACY.md)

## Limitations

- Cursor does not expose every ghost-text show/accept event; counts use explicit commands plus optional edit heuristics
- VS Code Chat Agent + Custom Endpoint is not a guaranteed Trim path; prefer Continue for agent-style edits
- Full chat token savings require pointing chat at Trim (Cursor Agent: Cursor Link HTTPS; Continue / VS Code / Claude Code: local proxy as documented)
- You need the `trim` CLI on `PATH` for Always-on proxy auto-start

## Uninstall

1. Command Palette → **Trim: Clear API Key**
2. Uninstall **Trim IDE** from Extensions
3. Optional: on the machine, `trim uninstall` (see [Uninstall docs](https://use-trim.com/docs/installation))
