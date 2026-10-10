-- Receipts and billing documents (Paddle MoR synced + in-app receipt views)

create table if not exists public.billing_receipts (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references public.profiles(id) on delete cascade,
  workspace_id uuid references public.workspaces(id) on delete set null,

  -- Paddle identifiers (source of truth for legal PDF)
  paddle_transaction_id text unique not null,
  paddle_invoice_number text,
  paddle_customer_id text not null,
  paddle_subscription_id text,
  paddle_invoice_pdf_url text,

  -- Receipt presentation fields (stored from webhook / Paddle API)
  status text not null, -- completed, refunded, past_due
  currency_code text not null,
  subtotal_cents bigint not null default 0,
  tax_cents bigint not null default 0,
  total_cents bigint not null default 0,
  tax_rate_bps int not null default 0,

  bill_to_name text,
  bill_to_email text not null,
  bill_to_company text,
  bill_to_address_line1 text,
  bill_to_address_line2 text,
  bill_to_city text,
  bill_to_region text,
  bill_to_postal_code text,
  bill_to_country text,
  tax_id text,

  period_start timestamptz,
  period_end timestamptz,
  paid_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists idx_billing_receipts_user_paid
  on public.billing_receipts(user_id, paid_at desc nulls last);

create table if not exists public.billing_receipt_line_items (
  id uuid primary key default gen_random_uuid(),
  receipt_id uuid not null references public.billing_receipts(id) on delete cascade,
  position int not null default 0,
  description text not null,
  quantity int not null default 1,
  unit_amount_cents bigint not null default 0,
  amount_cents bigint not null default 0,
  product_sku text,
  price_id text
);

create index if not exists idx_receipt_lines_receipt
  on public.billing_receipt_line_items(receipt_id, position);

-- Plan catalog lives in DB (no hardcoded plan prices in API responses)
create table if not exists public.plan_catalog (
  id text primary key, -- free | pro | team | enterprise
  display_name text not null,
  description text not null,
  price_monthly_cents int,
  price_yearly_cents int,
  credits_monthly int not null,
  per_seat boolean not null default false,
  paddle_price_id_monthly text,
  paddle_price_id_yearly text,
  features jsonb not null default '[]'::jsonb,
  is_public boolean not null default true,
  sort_order int not null default 0,
  updated_at timestamptz not null default now()
);

insert into public.plan_catalog (
  id, display_name, description, price_monthly_cents, price_yearly_cents,
  credits_monthly, per_seat, features, is_public, sort_order
) values
  (
    'free', 'Free', 'Local proxy with limited cloud telemetry sync.',
    0, 0, 50, false,
    '["Local AST proxy","50 cloud credits / mo","Community support"]'::jsonb,
    true, 1
  ),
  (
    'pro', 'Pro', 'For individual developers cutting BYOK API spend.',
    2000, 19200, 500, false,
    '["Everything in Free","500 cloud credits / mo","Usage dashboard","Receipt history","Email support"]'::jsonb,
    true, 2
  ),
  (
    'team', 'Team', 'Shared seats, pooled credits, admin controls.',
    3000, null, 5000, true,
    '["Everything in Pro","Pooled team credits","Admin dashboard","Shared .trimrc rules","Invoice / Net-30 via Paddle"]'::jsonb,
    true, 3
  ),
  (
    'enterprise', 'Enterprise', 'Dedicated gateway, SSO, custom SLA.',
    null, null, 0, true,
    '["Dedicated Go clusters","SSO / SAML","Custom IP allowlists","Custom SLA","Procurement support"]'::jsonb,
    true, 4
  )
on conflict (id) do nothing;

alter table public.billing_receipts enable row level security;
alter table public.billing_receipt_line_items enable row level security;
alter table public.plan_catalog enable row level security;

create policy "receipts_select_own" on public.billing_receipts
  for select using (auth.uid() = user_id);

create policy "receipt_lines_select_own" on public.billing_receipt_line_items
  for select using (
    exists (
      select 1 from public.billing_receipts r
      where r.id = receipt_id and r.user_id = auth.uid()
    )
  );

create policy "plan_catalog_public_read" on public.plan_catalog
  for select using (is_public = true);
