-- Enterprise sales inquiries (DB-backed, no mailto-only flow).
-- Also softens catalog copy so SSO / dedicated clusters are sold as scoped, not implied live today.

create table if not exists public.enterprise_inquiries (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references public.profiles(id) on delete cascade,
  email text not null,
  company_name text,
  estimated_seats int,
  message text not null,
  status text not null default 'new'
    check (status in ('new', 'contacted', 'closed')),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists idx_enterprise_inquiries_user
  on public.enterprise_inquiries (user_id, created_at desc);

create index if not exists idx_enterprise_inquiries_status
  on public.enterprise_inquiries (status, created_at desc);

alter table public.enterprise_inquiries enable row level security;

create policy "enterprise_inquiries_select_own" on public.enterprise_inquiries
  for select using (auth.uid() = user_id);

create policy "enterprise_inquiries_insert_own" on public.enterprise_inquiries
  for insert with check (auth.uid() = user_id);

-- Honest catalog features: procurement path is live; SSO / dedicated gateway remain sales-scoped.
update public.plan_catalog
set
  description = 'Custom contracts, pooled seats, and procurement support.',
  features = '["Procurement and Net-30 via sales","Pooled seat and credit planning","Custom SLA negotiation","SSO / SAML (scoped with sales)","Dedicated gateway options (scoped with sales)"]'::jsonb
where id = 'enterprise';
