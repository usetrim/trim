-- Plan modal / public plans default interval: Monthly (Annual switch untoggled).
-- Prior seed in 20260324000029 used 'annual'; product default is monthly.

update public.billing_settings
set default_plan_interval = 'monthly'
where id = 'default'
  and default_plan_interval = 'annual';

comment on column public.billing_settings.default_plan_interval is
  'Plan modal default billing interval (monthly|annual). Product default is monthly; frontend must not invent.';
