-- Portal cancel (period-end) is independent of SKU downgrade policy.
-- JA4/JA3 multi-account graph persists in Postgres (Redis remains a warm cache).
-- Downgrade policy chrome can include the paid period end date.

alter table public.billing_settings
  add column if not exists allow_cancel_at_period_end boolean not null default true;

comment on column public.billing_settings.allow_cancel_at_period_end is
  'When true, Paddle customer portal cancel_url is returned for active paid subscriptions. SKU downgrades remain blocked via allow_downgrades coercion.';

create table if not exists public.ja4_fingerprints (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references public.profiles(id) on delete cascade,
  ja4_hash text not null,
  last_seen_at timestamptz not null default now(),
  created_at timestamptz not null default now(),
  unique (user_id, ja4_hash)
);

create index if not exists idx_ja4_fingerprints_hash
  on public.ja4_fingerprints (ja4_hash);

alter table public.ja4_fingerprints enable row level security;

comment on table public.ja4_fingerprints is
  'TLS JA4/JA3 fingerprint to user graph for free-tier multi-account limits. Server-side writes only.';

create or replace function public.execute_gdpr_deletion(target_user_id uuid)
returns void
language plpgsql
security definer
set search_path = public, auth
as $$
begin
  delete from public.api_keys where user_id = target_user_id;
  delete from public.device_fingerprints where user_id = target_user_id;
  delete from public.ja4_fingerprints where user_id = target_user_id;
  delete from public.user_quotas where user_id = target_user_id;
  delete from public.workspace_invites where invited_by = target_user_id;
  update public.workspace_invites
    set accepted_by = null
    where accepted_by = target_user_id;
  delete from public.workspace_members where user_id = target_user_id;
  delete from public.subscriptions where user_id = target_user_id;

  update public.billing_receipts
    set user_id = null,
        bill_to_email = 'anonymized@gdpr.deleted',
        bill_to_name = null,
        bill_to_company = null,
        bill_to_address_line1 = null,
        bill_to_address_line2 = null,
        bill_to_city = null,
        bill_to_region = null,
        bill_to_postal_code = null,
        tax_id = null
    where user_id = target_user_id;

  update public.trim_events set user_id = null where user_id = target_user_id;
  delete from public.profiles where id = target_user_id;
  delete from auth.users where id = target_user_id;
end;
$$;

insert into public.site_messages (code, body) values
  ('DOWNGRADE_POLICY_UNTIL_FMT', 'Self-serve plan downgrades are disabled while your current plan is active through %s. Upgrades use Paddle proration on remaining unexpired usage. You can cancel at period end from Manage billing.'),
  ('PORTAL_CANCEL_DISABLED', 'Subscription cancellation from the portal is disabled for this account. Contact support.'),
  ('RECEIPT_PDF_FILENAME_REQUIRED', 'Receipt PDF filename template is not configured.')
on conflict (code) do update set body = excluded.body;
