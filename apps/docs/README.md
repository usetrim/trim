# Trim docs

Product documentation is served by the web app at [use-trim.com/docs](https://use-trim.com/docs).

**Source of truth:** [`apps/web/src/lib/docs/`](../web/src/lib/docs/) (navigation in `nav.ts`, pages in `content.tsx`).

## Features

- Sidebar accordion (one section open at a time)
- Previous / next pager
- On-this-page table of contents
- Theme toggle shared with the marketing site

## Popular entry points

| Topic | Path |
| --- | --- |
| Connect any IDE | `/docs/ide/connect` |
| Cursor | `/docs/ide/cursor` |
| Continue | `/docs/ide/continue` |
| VS Code Chat | `/docs/ide/vscode` |
| Claude Code | `/docs/ide/claude-code` |
| Provider adapters | `/docs/ide/provider-adapters` |
| Base URL troubleshooting | `/docs/ide/troubleshoot-base-url` |
| Trim IDE extension | `/docs/ide/extension` |
| Metering (credits / Unlimited / top-ups) | `/docs/concepts/metering` |

Dashboard and Admin guides live under `/docs/dashboard/*` and `/docs/admin/*`.

## Related READMEs

- Product overview and quick start: [root README](../../README.md)
- Operator console: [apps/admin/README.md](../admin/README.md)
- IDE extension: [extensions/trim-ide/README.md](../../extensions/trim-ide/README.md)
