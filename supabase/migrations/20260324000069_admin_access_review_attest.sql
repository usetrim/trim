-- Access review attestations, compliance + break-glass chrome, plan features editor labels.
create table if not exists public.admin_access_review_attestations (
  id uuid primary key default gen_random_uuid(),
  attested_by uuid not null references public.profiles(id) on delete restrict,
  period_label text not null,
  notes text not null default '',
  roster_snapshot jsonb not null,
  created_at timestamptz not null default now(),
  constraint admin_access_review_period_nonempty check (char_length(trim(period_label)) > 0)
);

create index if not exists idx_admin_access_review_attestations_created
  on public.admin_access_review_attestations (created_at desc);

comment on table public.admin_access_review_attestations is
  'Quarterly (or periodic) platform-admin roster attestations with immutable snapshot.';

insert into public.site_messages (code, body) values
  ('ADMIN_ACCESS_REVIEW_EXPORT', 'Export roster'),
  ('ADMIN_ACCESS_REVIEW_ATTEST', 'Attest this review'),
  ('ADMIN_ACCESS_REVIEW_PERIOD', 'Review period label'),
  ('ADMIN_ACCESS_REVIEW_NOTES', 'Attestation notes'),
  ('ADMIN_ACCESS_REVIEW_DONE', 'Access review attestation recorded.'),
  ('ADMIN_ACCESS_REVIEW_HISTORY', 'Prior attestations'),
  ('ADMIN_PLAN_FEATURES', 'Features (JSON array)'),
  ('ADMIN_BREAK_GLASS_REVOKE', 'Revoke'),
  ('ADMIN_BREAK_GLASS_REQUESTER', 'Requester'),
  ('ADMIN_BREAK_GLASS_APPROVER', 'Approver'),
  ('ADMIN_BREAK_GLASS_PERM_INVALID', 'elevates_permission must be a catalog permission code.')
on conflict (code) do nothing;
