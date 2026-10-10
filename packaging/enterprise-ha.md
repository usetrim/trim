# Enterprise HA and multi-region (Day-2)

Trim's control plane is designed for a single primary region first (API on Render/Fly, Postgres on Supabase, Redis on Upstash, web on Vercel/Cloudflare). Multi-region is **ops Day-2**, not required for open-source launch.

## Single-region production (supported now)

| Layer | Free / low-cost path | Notes |
|-------|----------------------|--------|
| API | Render / Fly.io Docker (`server/Dockerfile`, `ENABLE_TREESITTER=0` default) | Probe `GET /readyz` (Postgres+Redis **Ping** connectivity: `postgres_write`/`postgres_read`/`redis`) or container `CMD ["/api","-readyz"]` (scratch-safe); `GET /healthz` is liveness only |
| DB | Supabase Postgres | Apply `supabase/migrations` via `supabase db push` through latest (`…00108_*`). `trim_events` is monthly `PARTITION BY RANGE (created_at)`; API boot + DB-interval loop + retention purge + insert retry call `ensure_trim_events_month_partitions`. Optional `DATABASE_READ_URL` for read replica/pooler - omit until you have a real second host |
| Cache | Upstash Redis | TLS URL in `REDIS_URL`; chrome reload pub/sub; chart JSON TTL from `billing_settings.chart_cache_ttl_sec` (admin PATCH) |
| Web | Vercel / Cloudflare Pages | `apps/web` + `apps/admin`; optional `NEXT_PUBLIC_ASSET_PREFIX` when you have a real CDN host (empty = no invent CDN) |
| Edge | Cloudflare DNS + WAF | Optional `TRIM_CF_THREAT_SCORE_MIN` |

Compose helpers:

```bash
docker compose -f docker-compose.prod.yml --env-file server/.env up -d --build
# With local Postgres/Redis sidecars:
docker compose -f docker-compose.prod.yml --profile local-data --env-file server/.env up -d --build
# Two API replicas on one host (Day-2 sketch; not anycast):
docker compose -f docker-compose.multi-region.example.yml --profile demo up -d --build
```

## Multi-region checklist (when you fund it)

1. **Anycast / geo DNS**: Cloudflare or Route 53 latency routing to `api.us`, `api.eu`, `api.ap`.
2. **Stateless API replicas**: Same image in 2+ regions; no local disk state for quotas.
3. **Postgres**: Primary in one region + read replicas; writes stay on primary. Supabase Pro or managed Postgres.
4. **Redis**: Per-region cache for API keys / warm quotas, or Upstash Global. Quotas must remain server-side atomic. `plan_catalog.unlimited` is authoritative; invalidate Redis quota hashes when the admin dial flips (API does this on plan patch).
5. **Paddle webhooks**: Single primary webhook URL (or fan-in to primary) with HMAC + idempotency (already in Go).
6. **CLI kill-switch**: Keep `system:min_cli_version` in Redis replicated or shared.
7. **Observability**: Grafana Cloud free tier or OpenTelemetry exporter to your stack.

## Code signing (paid; CI scaffolding ready)

Release workflow soft-skips until secrets exist:

| Platform | Secrets | Job |
|----------|---------|-----|
| Apple | `APPLE_DEVELOPER_ID_APPLICATION`, `APPLE_ID`, `APPLE_TEAM_ID`, `APPLE_APP_SPECIFIC_PASSWORD` | `sign-darwin` |
| Windows | `WINDOWS_CERT_PFX_BASE64`, `WINDOWS_CERT_PASSWORD` | `sign-windows` |

Without certs, curl / Homebrew installs still work; Gatekeeper and SmartScreen warnings remain until you buy certs.

## Deep Mode and CGO release attach

- Frozen `trim-deep-*` PyInstaller binaries: **default on** every `v*` tag via `deep-attach` (`continue-on-error` if torch OOM). Opt out: `TRIM_SKIP_DEEP_ON_RELEASE=true`. Manual: Actions → **Build trim-deep**.
- Tree-sitter / Zig CGO builds: release jobs on tags + `TRIM_WITH_TREESITTER=1` / `TRIM_WITH_ZIG_TREESITTER=1`.

## Multi-region env sketch (ops; no invent regions in app code)

When you add a second region, keep the same image and fail-closed env keys per replica:

```bash
# api-us / api-eu examples (fill real values; do not invent URLs in code)
DATABASE_URL=postgres://...          # primary write endpoint
# DATABASE_READ_URL=postgres://...   # optional read replica / pooler; omit if none
REDIS_URL=rediss://...               # regional or global Upstash
TRIM_REGION_LABEL=us-east            # optional ops label for logs only
TRIM_PG_MAX_CONNS=20
TRIM_PG_MIN_CONNS=2
# TRIM_PG_READ_MAX_CONNS=20   # required when DATABASE_READ_URL is set
# TRIM_PG_READ_MIN_CONNS=2
TRIM_READYZ_TIMEOUT_SEC=5
```

Paddle webhooks use a durable Redis list (`trim:paddle:webhook:queue`) shared across replicas. Quotas stay Redis/Postgres server-side. Events use `trim_event_outbox` + SKIP LOCKED workers. See README operator checklist for signing and tap tokens.
