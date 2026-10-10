# Trim Operator Console

Open-source admin UI for a Trim instance (`apps/admin`). It manages **this** database only - not a multi-tenant SaaS control plane for other customers’ stacks.

## Run locally

1. Apply all SQL files in `supabase/migrations/` **in filename order** to your Supabase project.
2. Configure the API (see `server/.env.example`), including:
   - `TRIM_PLATFORM_OWNER_EMAILS` - comma-separated emails that may use admin (must match a signed-in profile)
   - `TRIM_ADMIN_STEP_UP_TTL_SEC` - step-up session TTL (60–3600)
   - `TRIM_ADMIN_TOTP_KEY` - 64 hex chars for TOTP step-up
   - `ADMIN_ALLOWED_ORIGINS` - e.g. `http://localhost:3001` locally, `https://admin.use-trim.com` in production
   - Optional: `ADMIN_ALLOWED_CIDRS` to restrict admin API by client IP
   - Include the admin origin in `CORS_ORIGINS`
3. Set compliance dials in the database before break-glass / partition ensure (see Compliance UI): e.g. `admin_retention_settings.break_glass_ttl_minutes`, `trim_events_partition_months_ahead`, `trim_events_partition_ensure_sec`.
4. Start the app:

```bash
cd apps/admin
cp .env.example .env.local
npm install
npm run dev
```

5. Open `http://localhost:3001` and sign in with an owner email.

## Plans

### Unlimited metering

Each plan row (Free, Pro, Team, Enterprise, top-ups) has an **Unlimited metering** checkbox (`plan_catalog.unlimited`, default **off**).

- **On** - AuthQuota skips credit debit for that plan tier; public pricing may show the Unlimited feature string; `/me/quota` returns `unlimited` + label.
- **Off** - normal metering. Prefer off for paid plans; use on only as a temporary growth dial.

Saving the checkbox invalidates Redis quota caches for that plan so the next metered request picks up the change. Labels come from `site_messages`.

### Popular badge

Each plan has a **Popular** checkbox (`plan_catalog.popular`, default off). When on, public pricing cards show the Popular badge (label from site chrome). Any combination of plans may be marked popular.

### Sync to Paddle

**Sync to Paddle** is a per-save action flag (not stored on the plan). Check it when monthly/yearly amounts need a Paddle product/price push; leave it unchecked for metering-only edits (Unlimited, Popular). Catalog amounts and `pri_*` / `pro_*` IDs live in the database - not in `.env`.

## Provider adapters

**Billing → Provider adapters** edits `provider_adapters` and `openai_model_aliases`. The CLI loads them via preferences / `trim config sync`.

Dialects:

- `anthropic_messages` - OpenAI chat shape → native Anthropic `/v1/messages`
- `openai_compat` - same-shape OpenAI hosts (GPT, Gemini, DeepSeek, Mistral, …) via `upstream_base_url`

Org-scoped Anthropic keys are configured per machine (request header or `TRIM_ANTHROPIC_WORKSPACE_ID`), not as a global Admin default.

## Admin UX notes

- Success/error toasts use API `message` / `saved_message` via TanStack Query `MutationCache` (no client-invented toast copy).
- After a successful create/update, modals close so the operator can continue; errors leave the modal open for retry.

## Deploy

Host on `admin.use-trim.com` (or your subdomain). Put Cloudflare Access or SSO in front for production. Never embed the Supabase service role key in this app.

## Security

- Platform RBAC is separate from workspace roles.
- Dangerous writes require `X-Trim-Step-Up` from `POST /api/v1/admin/auth/step-up`.
- Suspended or banned accounts are rejected by API auth middleware.
