# Privacy - Trim IDE extension

This document describes what the **Trim IDE** VS Code / Cursor extension collects and sends when you use it with Trim Cloud (`https://use-trim.com` / your configured `trim.apiUrl`).

Product privacy policy (hosted account, website, billing): **[https://use-trim.com/privacy](https://use-trim.com/privacy)**

## What this extension does

1. Optionally starts or attaches to the **local** Trim Fast Mode proxy on your machine (`trim start`) when Always-on with IDE is enabled.
2. Posts **aggregated IDE metrics** to Trim Cloud so your dashboard charts stay accurate.

## Data sent to Trim Cloud (when configured)

Only when **both** are set:

- `trim.apiUrl` (HTTPS/HTTP API origin), and
- an API key stored in the editor **Secret Storage** (never in `settings.json`)

| Data | Purpose |
| --- | --- |
| API key (`Authorization: Bearer`) | Authenticate as your Trim account / workspace |
| `X-Hardware-UUID` | Stable hashed IDE device id (not your raw disk serial); bind keys / fraud controls |
| `User-Agent: TrimIDE/<version>`, `X-Client-Version`, `X-Trim-Agent-Id: ide` | Client identification / compatibility |
| Optional `X-Workspace-Id` | Shared workspace credits when you set `trim.workspaceId` |
| Tab shown / accepted counts | Dashboard acceptance charts |
| Optional AI LOC estimates (lines added/deleted) | Dashboard LOC charts when you enable document-edit tracking |
| Preference / quota reads | Honor Always-on preference; show upgrade when quota is exhausted |

**Not uploaded by this extension:** your source files, repo contents, chat prompts, or Composer transcripts. Prompt compression happens on the **local** proxy when you point the IDE Base URL at Trim; that path is separate from this extension’s telemetry flush.

## Local-only data

- Pending counters in extension `globalState` until a successful flush
- Cached UI labels from Trim Cloud (public ide-chrome endpoint)
- API key in VS Code / Cursor Secret Storage

Clear with **Trim: Clear API Key**, then uninstall the extension.

## Your choices

- Leave `trim.apiUrl` empty → extension stays inactive (no cloud calls).
- Set `trim.autoFlushSeconds` to `0` → no automatic flushes (manual **Trim: Flush Telemetry Now** only).
- Disable `trim.trackDocumentEdits` → no edit-based LOC estimates.
- Use a self-hosted API URL → data goes to **your** deployment under **your** policies.

## Contact

Privacy / DPO questions: see [https://use-trim.com/privacy](https://use-trim.com/privacy) and support channels in [SUPPORT.md](./SUPPORT.md).
