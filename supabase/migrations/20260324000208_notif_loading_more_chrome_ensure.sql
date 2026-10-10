-- Ensure notification panel infinite-scroll chrome is present (idempotent).
-- Panel shows site_messages NOTIF_LOADING_MORE while fetching the next skip page.

insert into public.site_messages (code, body) values
  ('NOTIF_LOADING_MORE', 'Loading more…'),
  ('NOTIF_MARK_READ_PENDING', 'Updating…'),
  ('NOTIF_BELL_ARIA_COUNT_FMT', '{aria}, {count}')
on conflict (code) do update set body = excluded.body, updated_at = now();
