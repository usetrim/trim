-- Persist Paddle seat quantity on subscriptions so new workspaces inherit Team seats.
-- Store optional payment method summary on receipts (from Paddle payments array).

alter table public.subscriptions
  add column if not exists seat_quantity int not null default 1
  check (seat_quantity >= 1);

comment on column public.subscriptions.seat_quantity is
  'Seat count from Paddle subscription items[0].quantity. Used when creating workspaces under team/enterprise.';

-- Backfill from owned workspaces where owner already has a higher allocated_seats (best effort).
update public.subscriptions s
set seat_quantity = greatest(s.seat_quantity, coalesce((
  select max(w.allocated_seats)
  from public.workspaces w
  where w.owner_id = s.user_id
), 1))
where s.plan_tier in ('team', 'enterprise')
  and s.status in ('active', 'trialing', 'past_due');

alter table public.billing_receipts
  add column if not exists payment_method_summary text;

comment on column public.billing_receipts.payment_method_summary is
  'Human-readable payment method from Paddle (e.g. Visa ending 4242). Null when Paddle omits payments.';
