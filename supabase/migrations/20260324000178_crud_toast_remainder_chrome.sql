-- Toast-only success chrome (button labels NOTIF_MARK_* stay unchanged).

insert into public.site_messages (code, body) values
  ('NOTIF_MARK_READ_DONE', 'Notification marked as read.'),
  ('NOTIF_MARK_ALL_READ_DONE', 'All notifications marked as read.'),
  ('PREFERENCES_UNLINK_DONE', 'Sign-in provider disconnected.'),
  ('RECEIPT_PDF_DOWNLOAD_DONE', 'Invoice PDF downloaded.'),
  ('PORTAL_URL_MISSING', 'Billing portal URL was not returned. Try again.'),
  ('AVATAR_SYNC_DONE', 'Avatar synced.'),
  ('ADMIN_TOTP_DISABLED', 'Authenticator app removed.'),
  ('ADMIN_WEBAUTHN_REMOVED', 'Passkey removed.')
on conflict (code) do update set body = excluded.body, updated_at = now();
