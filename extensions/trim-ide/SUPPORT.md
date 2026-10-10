# Support - Trim IDE

## Product docs

- Extension guide: [https://use-trim.com/docs/ide/extension](https://use-trim.com/docs/ide/extension)
- Connect any IDE: [https://use-trim.com/docs/ide/connect](https://use-trim.com/docs/ide/connect)
- Install CLI: [https://use-trim.com/docs/installation](https://use-trim.com/docs/installation)
- Privacy: [https://use-trim.com/privacy](https://use-trim.com/privacy) · [PRIVACY.md](./PRIVACY.md)

## Get help

| Channel | Use for |
| --- | --- |
| **support@use-trim.com** | Account, billing, API keys, Always-on preference |
| **security@use-trim.com** | Vulnerability reports ([SECURITY.md](./SECURITY.md)) |
| [GitHub Issues](https://github.com/usetrim/trim/issues) | Non-sensitive bugs and feature requests |
| Editor **Output → Trim** | Local diagnostics (API sync, proxy auto-start, metrics flush) |

## First-run checklist

1. Install **Trim IDE** from the VS Code Marketplace or [Open VSX](https://open-vsx.org/extension/usetrim/trim-ide).
2. Install the `trim` CLI and keep it on `PATH`.
3. Settings → **Trim: Api Url** → `https://api.use-trim.com`.
4. **Trim: Copy Hardware ID** → Dashboard → API keys → **Register device** → agent **IDE**.
5. **Trim: Set API Key** with a **Trim** Cloud key (not a provider key).
6. Leave **Trim: Auto Start With Ide** on `follow` unless you need a local override.
7. Point Cursor / Continue / VS Code Chat at the local proxy Base URL for compression savings - the extension alone is not the compressor.

## Maintainers

Publisher id: **`usetrim`**. Store publishing runbook: **[PUBLISH.md](./PUBLISH.md)** (repo only; not shipped in the VSIX).
