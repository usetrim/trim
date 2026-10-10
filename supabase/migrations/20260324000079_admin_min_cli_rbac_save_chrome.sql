-- DB-driven MIN_CLI_VERSION override (empty = use process env).
-- Chrome for RBAC role save.

alter table public.admin_product_settings
  add column if not exists min_cli_version text;

comment on column public.admin_product_settings.min_cli_version is
  'Optional override of MIN_CLI_VERSION. Null or empty means use process env (fail closed to env).';

insert into public.site_messages (code, body) values
  ('ADMIN_PRODUCT_MIN_CLI', 'Minimum CLI version'),
  ('ADMIN_RBAC_SAVE_ROLE', 'Save role permissions'),
  ('ADMIN_RBAC_SELECT_ROLE', 'Select a role to edit permissions'),
  ('ADMIN_CHECKLIST_MIGRATIONS_MIN_CLI', 'Product min CLI column migration applied')
on conflict (code) do update set body = excluded.body;
