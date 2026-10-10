-- Subscription lifecycle: plan rank, upgrade-only policy, expiration tracking

alter table public.plan_catalog
  add column if not exists plan_rank int not null default 0;

-- Rank drives upgrade vs downgrade decisions (higher = more valuable)
update public.plan_catalog set plan_rank = 0 where id = 'free';
update public.plan_catalog set plan_rank = 10 where id = 'pro';
update public.plan_catalog set plan_rank = 20 where id = 'team';
update public.plan_catalog set plan_rank = 30 where id = 'enterprise';
update public.plan_catalog set plan_rank = 5 where id = 'topup_500';

alter table public.subscriptions
  add column if not exists billing_interval text;

alter table public.subscriptions
  add column if not exists canceled_at timestamptz;

alter table public.subscriptions
  add column if not exists expires_at timestamptz;

-- Mirror current_period_end into expires_at for active rows
update public.subscriptions
set expires_at = current_period_end
where expires_at is null;

create index if not exists idx_subscriptions_user_active
  on public.subscriptions (user_id, status, expires_at desc);

alter table public.billing_settings
  add column if not exists allow_downgrades boolean not null default false;

alter table public.billing_settings
  add column if not exists upgrade_proration_mode text not null default 'prorated_immediately'
    check (upgrade_proration_mode in (
      'prorated_immediately',
      'full_immediately',
      'prorated_next_billing_period',
      'full_next_billing_period',
      'do_not_bill'
    ));

alter table public.billing_settings
  add column if not exists allow_monthly_to_annual_as_upgrade boolean not null default true;

comment on column public.plan_catalog.plan_rank is
  'Relative plan value. Checkout only allows moves to a higher rank (upgrades). Downgrades are blocked when allow_downgrades is false.';

comment on column public.billing_settings.allow_downgrades is
  'When false (default), users with an active unexpired paid plan cannot move to a lower plan_rank.';

comment on column public.billing_settings.upgrade_proration_mode is
  'Passed to Paddle PATCH /subscriptions as proration_billing_mode for upgrades on an active subscription.';
