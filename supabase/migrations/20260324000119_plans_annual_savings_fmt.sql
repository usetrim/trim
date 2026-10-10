-- Savings badge format for annual plans (backend fmt; no client invent).
-- Seeded at API boot via chrome_seed_codes; this migration closes live gaps.

insert into public.site_messages (code, body) values
  ('PLANS_ANNUAL_SAVINGS_FMT', 'Save %d%%')
on conflict (code) do update set body = excluded.body;
