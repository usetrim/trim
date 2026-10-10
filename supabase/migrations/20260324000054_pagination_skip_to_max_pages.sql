-- Skip-to page_skips window from billing_settings (no invent const 500 in Go).

alter table public.billing_settings
  add column if not exists pagination_skip_to_max_pages int;

-- Seed once from prior Go const; operators may change. No column DEFAULT invent thereafter.
update public.billing_settings
set pagination_skip_to_max_pages = coalesce(pagination_skip_to_max_pages, 500)
where id = 'default';

alter table public.billing_settings
  alter column pagination_skip_to_max_pages set not null;

alter table public.billing_settings
  drop constraint if exists billing_settings_pagination_skip_to_max_pages_check;
alter table public.billing_settings
  add constraint billing_settings_pagination_skip_to_max_pages_check
  check (pagination_skip_to_max_pages >= 1 and pagination_skip_to_max_pages <= 10000);

comment on column public.billing_settings.pagination_skip_to_max_pages is
  'Max total_pages for which list APIs include page_skips[]. Beyond this, first/last/prev/next only. Handlers must not invent 500.';

insert into public.site_messages (code, body) values
  ('PAGINATION_SKIP_TO_MAX_INVALID', 'Pagination skip-to window is not configured. Set billing_settings.pagination_skip_to_max_pages.')
on conflict (code) do update set body = excluded.body;
