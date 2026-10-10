-- Fail-closed checkout chrome when Paddle catalog amount drifts from admin cents.

insert into public.site_messages (code, body) values
  ('PRICE_CATALOG_DRIFT', 'Checkout blocked: Paddle price no longer matches admin catalog amounts. Re-sync plans to Paddle from the admin dashboard.')
on conflict (code) do update set body = excluded.body;
