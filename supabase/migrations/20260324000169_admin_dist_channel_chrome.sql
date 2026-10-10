-- Distribution channel chrome: Homebrew / Scoop / winget / VS Marketplace sync labels.
insert into public.site_messages (code, body) values
  ('ADMIN_DIST_SOURCE_HOMEBREW', 'Homebrew tap'),
  ('ADMIN_DIST_SOURCE_SCOOP', 'Scoop bucket'),
  ('ADMIN_DIST_SOURCE_WINGET', 'winget fork'),
  ('ADMIN_DIST_SOURCE_MARKETPLACE', 'VS Marketplace'),
  ('ADMIN_DIST_METRIC_INSTALLS', 'Installs'),
  ('ADMIN_DIST_HOMEBREW_NOT_CONFIGURED', 'Homebrew tap sync is not configured (TRIM_DIST_HOMEBREW_TAP_REPO).'),
  ('ADMIN_DIST_SCOOP_NOT_CONFIGURED', 'Scoop bucket sync is not configured (TRIM_DIST_SCOOP_BUCKET_REPO).'),
  ('ADMIN_DIST_WINGET_NOT_CONFIGURED', 'winget fork sync is not configured (TRIM_DIST_WINGET_FORK_REPO).'),
  ('ADMIN_DIST_MARKETPLACE_NOT_CONFIGURED', 'VS Marketplace sync is not configured (TRIM_DIST_VSCODE_EXTENSION_ID).'),
  ('ADMIN_DIST_CHANNEL_REPO_REQUIRED', 'Distribution channel repository is required.'),
  ('ADMIN_DIST_MARKETPLACE_ID_REQUIRED', 'VS Marketplace extension id is required (publisher.name).'),
  ('ADMIN_DIST_MARKETPLACE_STATS_MISSING', 'VS Marketplace did not return install statistics for this extension.')
on conflict (code) do update set body = excluded.body, updated_at = now();
