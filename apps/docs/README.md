# Trim docs

Product documentation is served by the web app at [use-trim.com/docs](https://use-trim.com/docs).

**Source of truth:** [`apps/web/src/lib/docs/`](../web/src/lib/docs/) (navigation in `nav.ts`, pages in `content.tsx`).

## Features

- Sidebar accordion (one section open at a time)
- Previous / next pager
- On-this-page table of contents
- Theme toggle shared with the marketing site
- Copy button on every docs code block (`CopyableCode`)
- Tables for exact settings places and path pickers

## Recommended reading order (worldwide)

1. [Introduction](https://use-trim.com/docs) - choose your tool
2. [Quickstart](https://use-trim.com/docs/quickstart) - shared install + one Step 4 path
3. Your tool guide only:
   - [Claude Code (CLI only)](https://use-trim.com/docs/cli/claude-code) - no Cursor / no VS Code; Windows / macOS / Linux copy blocks
   - [Cursor](https://use-trim.com/docs/ide/cursor) + [Trim Cursor Link](https://use-trim.com/docs/cli/cursor-link) (Agent; not localhost)
   - [Continue](https://use-trim.com/docs/ide/continue)
   - [VS Code Chat](https://use-trim.com/docs/ide/vscode)
4. Optional: [Custom models](https://use-trim.com/docs/ide/custom-models), [Always-on](https://use-trim.com/docs/concepts/always-on)

## Popular entry points

| Topic | Path |
| --- | --- |
| Introduction (choose your tool) | `/docs` |
| Quickstart | `/docs/quickstart` |
| Installation | `/docs/installation` |
| Claude Code CLI only | `/docs/cli/claude-code` |
| Trim Cursor Link (Cursor Agent) | `/docs/cli/cursor-link` |
| Cursor setup | `/docs/ide/cursor` |
| Custom models | `/docs/ide/custom-models` |
| Always-on | `/docs/concepts/always-on` |
| Cursor | `/docs/ide/cursor` |
| Continue | `/docs/ide/continue` |
| VS Code Chat | `/docs/ide/vscode` |
| Connect any IDE | `/docs/ide/connect` |
| Troubleshooting | `/docs/guides/troubleshooting` |

Dashboard and Admin guides live under `/docs/dashboard/*` and `/docs/admin/*`.

## Related READMEs

- Product overview and quick start: [root README](../../README.md)
- Operator console: [apps/admin/README.md](../admin/README.md)
- IDE extension: [extensions/trim-ide/README.md](../../extensions/trim-ide/README.md)
