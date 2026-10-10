-- Enterprise catalog row must not be treated as a self-serve subscription.
-- Contact-sales plans use plan_kind=enterprise (no Paddle price sync / checkout).

update public.plan_catalog
set plan_kind = 'enterprise',
    updated_at = now()
where id = 'enterprise'
  and plan_kind = 'subscription';

comment on column public.plan_catalog.plan_kind is
  'subscription | topup | enterprise. enterprise is contact-sales only (no Paddle catalog sync).';
