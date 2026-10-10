-- Pending workspace email invites (token accept flow).
-- Invitees do not need an existing profile at invite time.
-- Accept requires signed-in social account whose email matches the invite.

alter table public.billing_settings
  add column if not exists workspace_invite_ttl_hours int not null default 168
    check (workspace_invite_ttl_hours > 0 and workspace_invite_ttl_hours <= 8760);

comment on column public.billing_settings.workspace_invite_ttl_hours is
  'Hours until a pending workspace invite expires. Source of truth for invite TTL (no app invent).';

create table if not exists public.workspace_invites (
  id uuid primary key default gen_random_uuid(),
  workspace_id uuid not null references public.workspaces(id) on delete cascade,
  email text not null,
  role text not null default 'member'
    check (role in ('member', 'admin')),
  token_hash text not null unique,
  invited_by uuid not null references public.profiles(id) on delete cascade,
  status text not null default 'pending'
    check (status in ('pending', 'accepted', 'revoked', 'expired')),
  expires_at timestamptz not null,
  accepted_by uuid references public.profiles(id) on delete set null,
  accepted_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create unique index if not exists idx_workspace_invites_pending_email
  on public.workspace_invites (workspace_id, lower(email))
  where status = 'pending';

create index if not exists idx_workspace_invites_workspace_status
  on public.workspace_invites (workspace_id, status);

create index if not exists idx_workspace_invites_email_pending
  on public.workspace_invites (lower(email))
  where status = 'pending';

alter table public.workspace_invites enable row level security;

create or replace function public.execute_gdpr_deletion(target_user_id uuid)
returns void
language plpgsql
security definer
set search_path = public, auth
as $$
begin
  delete from public.api_keys where user_id = target_user_id;
  delete from public.device_fingerprints where user_id = target_user_id;
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

comment on table public.workspace_invites is
  'Email invites for workspaces. Raw token is shown once; only SHA-256 hash is stored.';
