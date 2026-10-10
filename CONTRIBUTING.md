# Contributing to Trim

Thanks for helping build Trim. This guide gets you from clone to a working local stack.

## Prerequisites

- Go 1.22+
- Node.js 20+
- Docker Desktop (Postgres + Redis via Compose)
- Git

## One-command local stack

```bash
docker compose up -d
```

This starts Postgres on `:5432` and Redis on `:6379`.

## Backend

```bash
cd server
cp .env.example .env
# Required: set TRIM_SAVINGS_USD_PER_MTOK (no silent default)
go mod tidy
go run ./cmd/api
```

API listens on `http://localhost:8080`.

Optional Tree-sitter CGO build (default is CGO-free heuristics + Go stdlib AST):

```bash
# From repo root (requires a C toolchain: gcc / clang / MSVC)
./scripts/enable-treesitter.sh
# or from server/:
cd server && ./scripts/enable-treesitter.sh
go get github.com/smacker/go-tree-sitter@v0.0.0-20240827094217-dd81d9e9be82
CGO_ENABLED=1 go build -tags treesitter -o trim-api ./cmd/api

# Docker / Render (from server/):
cd server && docker build --build-arg ENABLE_TREESITTER=1 -t trim-api .
```

When the `treesitter` tag is off, TypeScript/JavaScript/Python use structural heuristics and Go uses `go/parser`.

## CLI

```bash
cd cli
cp .env.example .env
# Required: TRIM_SAVINGS_USD_PER_MTOK must match ops pricing estimate
go mod tidy
go run ./cmd/trim start --port 8888
```

Proxy listens on `http://localhost:8888`. Use `trim stats --tui` (or `trim tui`) for the live Bubble Tea dashboard.

## Web dashboard

```bash
cd apps/web
cp .env.example .env.local
npm install
npm run dev
```

Dashboard: `http://localhost:3000`.

## Branching

Trim uses **GitHub Flow**. The only long-lived branch is `main` (protected, always releasable).

### Name format

```text
<type>/<short-kebab-description>
```

For monorepo clarity, prefer an area segment:

```text
<type>/<area>-<short-kebab-description>
```

Rules: lowercase, kebab-case, no spaces or underscores, short (about 3–6 words). Do not use personal names, `temp`, `final`, or machine-generated junk names.

### Types (aligned with Conventional Commits)

| Prefix | Use for |
| --- | --- |
| `feat/` | New user-facing feature |
| `fix/` | Bug fix |
| `docs/` | Documentation only |
| `chore/` | Tooling, dependencies, repo hygiene |
| `refactor/` | Internal change with no behavior change |
| `test/` | Tests only |
| `ci/` | GitHub Actions / pipelines |
| `perf/` | Performance |
| `build/` | Build or packaging |
| `revert/` | Revert a previous change |

### Areas (optional)

`cli`, `server`, `web`, `admin`, `ide`, `docs`, `packaging`, `db`

### Examples

```text
feat/web-team-invites
fix/server-paddle-webhook
docs/readme-quick-start
chore/deps-next-security
ci/release-cosign
```

### Workflow

1. Branch from the latest `main`.
2. One concern per branch.
3. Open a pull request into `main`.
4. After merge, delete the branch.

Releases use **git tags** (`v0.1.0`, `v1.0.0`), not long-lived `release/*` branches.

## Commits

Use [Conventional Commits](https://www.conventionalcommits.org/):

```text
feat(web): add workspace invite accept page
fix(server): reject unbound annual checkout
docs: clarify CLI install on Windows
```

Subject in imperative mood, ≤ ~72 characters. Optional body for why / breaking changes.

## Pull requests

1. Fork (if external) and create a branch from `main` using the naming rules above.
2. Keep PRs focused (one concern per PR).
3. Do not introduce em dashes (Unicode U+2014) in user-facing copy or docs.
4. Run lint and tests before opening the PR.
5. Describe the why, not only the what.
6. Acknowledge the CLA (see below).
7. Billing / metering: do not hardcode plan prices, credits, Unlimited, or Popular. Catalog lives in `plan_catalog` (`unlimited` and `popular` default false). User-facing chrome belongs in `site_messages` / migrations. See root README (subscription rules) and `/docs/concepts/metering`.
8. Dashboard Usage + LOC heatmap chrome (tooltips, Tab vs All copy, activity stats) is backend-driven via `site_messages` only. Do not invent client labels or scopes. Tab heatmap is `tab_suggestions_accepted`, not lines edited.
9. Plan vs top-up dialogs: Upgrade opens subscription catalog (`mode: plans`); Buy top-up opens packs only (`mode: topup`). Never expose raw Paddle proration enums in customer UI.
10. Admin/web mutations: success/error toasts via shared MutationCache + API `message` only. Close admin dialogs after successful save. `sync_to_paddle` is a per-request flag, not a plan column.

## Contributor License Agreement (CLA)

By opening a pull request you agree to [CLA.md](./CLA.md).

How to acknowledge:

1. Read [CLA.md](./CLA.md) in full.
2. In your PR description, include this exact line:

   `I have read and agree to the Trim Contributor License Agreement (CLA.md).`

3. Optionally comment the same line on the PR once opened.

Maintainers may request a signed CLA Assistant / GitHub Action acknowledgment when that workflow is enabled on the repository. Until then, the PR description line above is the required acknowledgment.

CLA signature records are stored on the unprotected `cla-signatures` branch (`signatures/version1/cla.json`). Do not add that branch to the `main` protection ruleset.

## Code style

- Go: `gofmt`, idiomatic packages under `internal/` for private logic.
- TypeScript: strict mode, ESLint, Prettier.
- UI: shadcn/ui primitives, zinc/neutral dark theme (no default blue accents).

## Telemetry

Respect `DO_NOT_TRACK=1` and `TRIM_TELEMETRY_DISABLED=1`. Never send emails, tokens, or absolute file paths.
