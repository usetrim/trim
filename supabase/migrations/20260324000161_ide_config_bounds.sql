-- DB-driven bounds for IDE extension settings (no invent 1-120 / 0-600 in binary).

insert into public.site_messages (code, body) values
  ('IDE_HTTP_TIMEOUT_MIN_SEC', '1'),
  ('IDE_HTTP_TIMEOUT_MAX_SEC', '120'),
  ('IDE_AUTO_FLUSH_MIN_SEC', '0'),
  ('IDE_AUTO_FLUSH_MAX_SEC', '600'),
  ('IDE_AUTO_FLUSH_MIN_ENABLED_SEC', '15'),
  ('IDE_HTTP_TIMEOUT_INVALID', 'trim.httpTimeoutSec outside IDE_HTTP_TIMEOUT_MIN_SEC..MAX_SEC (fail-closed)'),
  ('IDE_AUTO_FLUSH_INVALID', 'trim.autoFlushSeconds outside IDE_AUTO_FLUSH bounds (fail-closed)'),
  ('IDE_CONFIG_TRACK_EDITS_MISSING', 'trim.trackDocumentEdits missing (fail-closed)'),
  ('IDE_CONFIG_MIN_LINES_INVALID', 'trim.minLinesForAiHeuristic invalid (fail-closed)'),
  ('IDE_CONFIG_API_URL_EMPTY', 'trim.apiUrl empty - set Settings → Trim → API URL (fail-closed)')
on conflict (code) do nothing;
