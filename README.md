# Trim

Local context optimization for AI coding tools, with an optional cloud control plane for auth, quotas, billing, and receipts.

Point Cursor, Continue, VS Code Chat, Claude Code, and similar clients at a local OpenAI-compatible proxy. Trim reduces prompt size before requests reach your upstream provider. Deep Mode runs on your machine only - never in Trim cloud.

**Website:** [use-trim.com](https://use-trim.com) · **Docs:** [use-trim.com/docs](https://use-trim.com/docs) · **Security:** [SECURITY.md](./SECURITY.md)

[![CI](https://github.com/usetrim/trim/actions/workflows/ci.yml/badge.svg)](https://github.com/usetrim/trim/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-MIT%20%2B%20AGPL--3.0-blue.svg)](./LICENSE)

---

## Why Trim

- **OpenAI-compatible local proxy** - one Base URL (`http://127.0.0.1:8888/v1`) for coding IDEs and CLIs
- **Fast Mode** - AST / heuristic compression on the hot path (`trim start`)
- **Deep Mode** - stronger local compression (LLMLingua family); stays on-device
- **Cloud control plane** (optional) - Google / GitHub / GitLab login, credits, Paddle billing, receipts, workspaces
- **Operator console** - `apps/admin` for plans, billing settings, provider adapters, compliance

---

## Install the CLI

**macOS / Linux**

```bash
curl -fsSL https://use-trim.com/install.sh | sh
```

**Windows (PowerShell)**

```powershell
irm https://use-trim.com/install.ps1 | iex
```

**npm**

```bash
npm install -g @usetrim/trim
```

**Homebrew**

```bash
brew install usetrim/tap/trim
```


**From source (Go)**

```bash
git clone https://github.com/usetrim/trim.git
cd trim/cli && go install ./cmd/trim
```

> `go install github.com/usetrim/trim/cli/cmd/trim@v…` is not supported while `cli/go.mod` contains a local `replace` for the server module. Use a clone, Release binaries, npm, or Homebrew instead.

Optional flags (Unix installer): `TRIM_WITH_TREESITTER=1`, `TRIM_WITH_ZIG_TREESITTER=1`, `TRIM_WITH_DEEP=1`.

Telemetry from installers is privacy-light and soft-fail. Opt out with `DO_NOT_TRACK=1` or `TRIM_TELEMETRY_DISABLED=1`.

Package-manager ops (tokens, taps): [packaging/README.md](./packaging/README.md).
---

## Quick start (end users)

1. Install the CLI (above).
2. Sign in and create an API key:
   ```bash
   trim login
   ```
3. Start the local proxy:
   ```bash
   trim start
   ```
4. In your IDE, set the OpenAI-compatible Base URL to:
   ```text
   http://127.0.0.1:8888/v1
   ```
5. Optional: install **Trim IDE** (publisher `usetrim`) from the VS Code Marketplace or Open VSX for Always-on proxy + dashboard metrics. See [extensions/trim-ide/README.md](./extensions/trim-ide/README.md).

Everyday vs advanced commands: `trim help`.

Uninstall:

```bash
trim uninstall
# keep local data / binary:
trim uninstall --keep-data
```

---

## Repository layout

| Path | Role | License |
| --- | --- | --- |
| `cli/` | Local proxy, compress, setup, daemon, TUI | MIT |
| `server/` | Go API (auth, quotas, webhooks, admin) | AGPL-3.0 |
| `apps/web/` | Next.js product site + dashboard | MIT |
| `apps/admin/` | Operator console | MIT |
| `apps/docs/` | Docs packaging notes (content lives in web `/docs`) | MIT |
| `extensions/trim-ide/` | VS Code / Open VSX extension | MIT |
| `supabase/migrations/` | Postgres schema and seeds | - |
| `packaging/` | Homebrew / Scoop / winget runbooks | - |

---

## Local development

Prerequisites: **Go 1.22+**, **Node.js 20+**, **Docker** (Postgres + Redis), **Git**.

```bash
# Infra
docker compose up -d

# API
cd server
cp .env.example .env
# Fill every required key (cloud and local modes fail closed; no silent defaults)
go run ./cmd/api

# Web dashboard (new terminal)
cd apps/web
cp .env.example .env.local
npm install
npm run dev

# CLI proxy (new terminal)
cd cli
cp .env.example .env   # optional for cloud login / Deep settings
go run ./cmd/trim start
```

- API default: `http://localhost:8080`
- Web default: `http://localhost:3000`
- Local proxy dashboard: `http://localhost:8888/dashboard`

Full contributor setup: [CONTRIBUTING.md](./CONTRIBUTING.md).

### Useful CLI commands

```bash
trim start                          # local OpenAI-compatible proxy
trim compress path/to/file.go --mode fast
trim compress docs.txt --deep --engine v2
trim compress --bootstrap           # first-time Deep dependencies
trim config sync                    # pull cloud preferences
trim setup                          # print IDE / shell wiring
trim tui                            # local stats (+ cloud quota when logged in)
```

Copy `.trimrc.example` → `.trimrc` for compression modes (`mild` | `balanced` | `aggressive` | `custom`).

---

## Self-hosting (cloud stack)

Trim’s hosted product uses:

| Piece | Typical host | Hostname |
| --- | --- | --- |
| Web | Vercel / Cloudflare Pages | `use-trim.com`, `www.use-trim.com` |
| API | Render (Docker: `server/Dockerfile`) | `api.use-trim.com` |
| Admin | Vercel (second project) | `admin.use-trim.com` |
| Database + Auth | Supabase | - |
| Redis | Upstash | - |
| Billing | Paddle Billing (MoR) | webhook → `/webhooks/paddle` |

### Operator checklist (summary)

1. **DNS** - point apex/www, `api`, and `admin` at your hosts (Cloudflare DNS is fine on the Free plan).
2. **Supabase** - apply SQL in `supabase/migrations/` **in filename order**. Enable Google, GitHub, and GitLab Auth; set redirect URLs for web, admin, and localhost. Set API `JWT_SECRET` to the Supabase JWT secret.
3. **Redis** - set `REDIS_URL` (`rediss://…` for Upstash).
4. **Env** - copy and fill:
   - `server/.env.example` → API
   - `apps/web/.env.example` → web
   - `apps/admin/.env.example` → admin
5. **Paddle** - configure plans and sync from **Admin → Plans** (preferred). Webhook: `https://api.use-trim.com/webhooks/paddle`. Prices live in the database (`plan_catalog` + `billing_settings`), not in `.env`.
6. **Deploy** - API from `server/`; web from `apps/web`; admin from `apps/admin`.
7. **Smoke test** - OAuth login → API key → `trim start` → checkout / upgrade → receipt.

Admin console details: [apps/admin/README.md](./apps/admin/README.md).  
Release / taps / signing: [packaging/README.md](./packaging/README.md).

### Subscription behavior (product rules)

| Situation | Behavior |
| --- | --- |
| Free or expired | New checkout for a paid plan |
| Active paid → higher plan | Upgrade with Paddle proration (mode from `billing_settings`) |
| Active paid → lower plan | Blocked while the period is unexpired (no self-serve downgrade) |
| Monthly → annual (same plan) | Treated as upgrade when enabled in settings |
| Annual → monthly (same plan) | Blocked while unexpired |
| Top-up packs | Always a new checkout (tier unchanged) |

Plan order is `plan_catalog.plan_rank`. Optional **Unlimited metering** is a per-plan operator dial (`plan_catalog.unlimited`, default off) - not a client-side flag.

---

## Documentation

Product docs ship with the web app:

- Hosted: [use-trim.com/docs](https://use-trim.com/docs)
- Source: `apps/web/src/lib/docs/`

Topics include IDE setup (Cursor, Continue, VS Code, Claude Code, and others), metering, security (device binding, fraud layers), CLI reference, and dashboard / admin guides.

---

## Security

- Report vulnerabilities privately - see [SECURITY.md](./SECURITY.md)
- Never put the Supabase **service role** key in browser env
- Lock `CORS_ORIGINS` / admin origins to real hosts
- Rotate `JWT_SECRET` and Paddle secrets if leaked

---

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) and [GOVERNANCE.md](./GOVERNANCE.md).

Please open a pull request against `main`. Keep UI copy and public strings backend-driven where the product already expects `site_messages` / API chrome (no hardcoded marketing or error invention in clients).

---

## License

**Dual license** - see [LICENSE](./LICENSE):

| Components | License |
| --- | --- |
| `cli/`, `apps/web/`, `apps/docs/`, `extensions/` | MIT |
| `server/` (hosted control plane) | [AGPL-3.0](./LICENSES/AGPL-3.0.txt) |

If you modify and host the backend as a network service, AGPL-3.0 source disclosure obligations apply.
