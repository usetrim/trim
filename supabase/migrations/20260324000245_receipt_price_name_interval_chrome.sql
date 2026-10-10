-- Paddle invoice product subtitle: "Pro (monthly)" / "Pro (yearly)" from catalog when needed.
insert into public.site_messages (code, body) values
  ('RECEIPT_PRICE_NAME_MONTHLY_FMT', '%s (monthly)'),
  ('RECEIPT_PRICE_NAME_YEARLY_FMT', '%s (yearly)')
on conflict (code) do update set body = excluded.body;

-- Backfill missing line price_name from plan_catalog price id match (DB-driven).
update public.billing_receipt_line_items li
set price_name = pc.display_name || ' (monthly)'
from public.plan_catalog pc
where li.price_id = pc.paddle_price_id_monthly
  and coalesce(nullif(btrim(li.price_name), ''), '') = '';

update public.billing_receipt_line_items li
set price_name = pc.display_name || ' (yearly)'
from public.plan_catalog pc
where li.price_id = pc.paddle_price_id_yearly
  and coalesce(nullif(btrim(li.price_name), ''), '') = '';
