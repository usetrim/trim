-- CLI / IDE conversion chrome when cloud credits are exhausted (402 / remaining <= 0).
-- Bodies are operator-editable via site_messages; clients must not invent copy.

insert into public.site_messages (code, body) values
  ('CLI_QUOTA_EXHAUSTED_TITLE', 'Cloud credits exhausted'),
  ('CLI_QUOTA_EXHAUSTED_BODY', 'Upgrade your plan or buy a top-up to keep using Trim cloud. Local Fast Mode still works on your machine.'),
  ('CLI_QUOTA_UPGRADE_HINT_FMT', 'Upgrade: %s'),
  ('CLI_QUOTA_UPGRADE_OPENING', 'Opening upgrade in your browser…'),
  ('CLI_QUOTA_UPGRADE_URL_MISSING', 'Upgrade URL unavailable. Open the Trim dashboard to upgrade or buy a top-up.'),
  ('CLI_HELP_UPGRADE_SHORT', 'Open upgrade / top-up in the browser when cloud credits are exhausted'),
  ('IDE_QUOTA_EXHAUSTED_TITLE', 'Cloud credits exhausted'),
  ('IDE_QUOTA_EXHAUSTED_BODY', 'Upgrade your plan or buy a top-up to keep using Trim cloud. Local Fast Mode still works on your machine.'),
  ('IDE_QUOTA_UPGRADE_ACTION', 'Upgrade')
on conflict (code) do update set body = excluded.body, updated_at = now();
