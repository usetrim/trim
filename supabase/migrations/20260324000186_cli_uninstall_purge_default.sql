-- Pro uninstall UX: full purge is the default; --keep-data opts out.
-- HF cache relative path when HF_HOME is unset comes from site_messages (no invent).
insert into public.site_messages (code, body) values
  ('CLI_HELP_UNINSTALL_SHORT', 'Fully remove Trim from this machine (proxy, daemon, data, Deep caches, binary)'),
  ('CLI_HELP_UNINSTALL_LONG', 'Stops the proxy, removes the OS login daemon, clears credentials, reverts IDE Base URL overrides from trim setup, deletes ~/.trim and ~/.config/trim, cleans Deep Mode caches from site_messages lists, removes sidecars, and deletes the CLI binary/PATH when possible. Use --keep-data to skip deleting local data directories and Deep caches.'),
  ('CLI_HELP_UNINSTALL_FLAG_KEEP_DATA', 'Keep ~/.trim, ~/.config/trim, Deep Mode caches, and the CLI binary (only stop/daemon/logout/setup revert)'),
  ('CLI_UNINSTALL_HF_CACHE_REL', '.cache/huggingface'),
  ('CLI_UNINSTALL_ARP_DISPLAY_NAME', 'Trim'),
  ('CLI_UNINSTALL_ARP_PUBLISHER', 'Trim'),
  ('CLI_UNINSTALL_ARP_REG_KEY', 'Trim'),
  ('CLI_UNINSTALL_ARP_REMOVED', 'Removed Trim from Windows Apps & features.')
on conflict (code) do update set body = excluded.body;

-- Retire purge-flag chrome (kept empty so old caches fail closed to blank help).
update public.site_messages
set body = 'Deprecated: uninstall purges by default; use --keep-data to retain local data'
where code = 'CLI_HELP_UNINSTALL_FLAG_PURGE';
