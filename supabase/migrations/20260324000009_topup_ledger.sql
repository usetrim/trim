-- Idempotent one-time credit pack grants keyed by Paddle transaction + price.
-- Prevents double-crediting when transaction.completed (or similar) is retried.

create table if not exists public.billing_topup_ledger (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references public.profiles(id) on delete cascade,
  paddle_transaction_id text not null,
  price_id text not null default '',
  plan_id_hint text,
  credits_granted int not null check (credits_granted > 0),
  quantity int not null default 1 check (quantity > 0),
  created_at timestamptz not null default now(),
  unique (paddle_transaction_id, price_id)
);

create index if not exists idx_billing_topup_ledger_user
  on public.billing_topup_ledger(user_id, created_at desc);

comment on table public.billing_topup_ledger is
  'Server-side ledger for top-up credit packs. One row per Paddle transaction line price; insert-or-skip drives quota increments.';
