-- Distribution source/metric labels (no client invent of github/install/clones/...).

insert into public.site_messages (code, body) values
  ('ADMIN_DIST_SOURCE_GITHUB', 'GitHub'),
  ('ADMIN_DIST_SOURCE_INSTALL', 'Install / CDN'),
  ('ADMIN_DIST_METRIC_CLONES', 'Clones'),
  ('ADMIN_DIST_METRIC_VIEWS', 'Visitors'),
  ('ADMIN_DIST_METRIC_RELEASE_DOWNLOADS', 'Release downloads'),
  ('ADMIN_DIST_METRIC_HITS', 'Install hits')
on conflict (code) do update set body = excluded.body;
