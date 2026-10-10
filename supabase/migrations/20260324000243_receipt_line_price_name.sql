-- Paddle invoice shows product name + price name (e.g. Pro / Pro (monthly)).
alter table public.billing_receipt_line_items
  add column if not exists price_name text;
