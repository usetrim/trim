-- IDE extension: fail-closed copy when trim.apiUrl is not a valid http(s) URL.
insert into public.site_messages (code, body) values
  ('IDE_CONFIG_API_URL_INVALID', 'trim.apiUrl must be an absolute http(s) URL (fail-closed)')
on conflict (code) do nothing;
