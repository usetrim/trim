-- Notification bell: scroll load-more chrome + badge overflow (fail closed if unset).

insert into public.site_messages (code, body) values
  ('NOTIF_LOADING_MORE', 'Loading more…'),
  ('NOTIF_MARK_READ_PENDING', 'Updating…'),
  ('NOTIF_BADGE_MAX', '99'),
  ('NOTIF_BADGE_OVERFLOW_FMT', '{max}+'),
  ('NOTIF_BELL_ARIA_COUNT_FMT', '{aria}, {count}')
on conflict (code) do update set body = excluded.body;
