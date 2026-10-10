-- Uninstall path lists and HF hub subdir from site_messages (no invent in CLI).
-- Clear-API-Key chrome covers extension local state cleanup.
insert into public.site_messages (code, body) values
  ('CLI_UNINSTALL_HOME_DIRS_REL', '.trim,.config/trim'),
  ('CLI_UNINSTALL_DARWIN_LOG_RELS', 'Library/Logs/trim.log,Library/Logs/trim.err.log'),
  ('CLI_UNINSTALL_HF_HUB_SUBDIR', 'hub'),
  ('CLI_UNINSTALL_HOME_DIRS_MISSING', 'Uninstall home dirs skipped: CLI_UNINSTALL_HOME_DIRS_REL not configured in site_messages.'),
  ('CLI_UNINSTALL_PIP_BINS', 'pip,pip3'),
  ('CLI_UNINSTALL_PYTHON_BINS', 'python,python3'),
  ('CLI_UNINSTALL_PIP_MODULE', 'pip'),
  ('IDE_API_KEY_CLEARED', 'Trim API key and local extension state cleared.'),
  ('CLI_UNINSTALL_NEXT_EXT', 'IDE extension: Trim: Clear API Key (clears Secret Storage + local extension state), then Command Palette → Extensions → uninstall Trim IDE.')
on conflict (code) do update set body = excluded.body;
