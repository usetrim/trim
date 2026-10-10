-- Health age suffix + receipt tax/total format (no client invent).

insert into public.site_messages (code, body) values
  ('ADMIN_HEALTH_AGE_SUFFIX_FMT', ' ({age}s)'),
  ('ADMIN_RECEIPT_TAX_TOTAL_FMT', '{tax} / {total}')
on conflict (code) do update set body = excluded.body;
