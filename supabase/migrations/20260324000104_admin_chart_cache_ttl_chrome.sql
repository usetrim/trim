-- Admin billing settings chrome for chart Redis cache TTL (DB-driven labels only).

insert into public.site_messages (code, body) values
  ('ADMIN_BILLING_CHART_CACHE_TTL', 'Chart cache TTL (seconds)')
on conflict (code) do update set body = excluded.body;
