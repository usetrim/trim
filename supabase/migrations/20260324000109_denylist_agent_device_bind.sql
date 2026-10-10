-- Denylist hot-path + API-key device/agent binding + agent identity catalog.
-- Fail-closed chrome for IP/ASN denylist, hardware bind, and X-Trim-Agent-Id.

create table if not exists public.agent_identity_catalog (
  id text primary key,
  description text not null,
  sort_order int not null default 100,
  created_at timestamptz not null default now(),
  constraint agent_identity_catalog_id_check check (id ~ '^[a-z][a-z0-9_]{0,31}$')
);

comment on table public.agent_identity_catalog is
  'Allowed X-Trim-Agent-Id values for API-key clients (cli, ide, ci). Request header must match a row.';

insert into public.agent_identity_catalog (id, description, sort_order) values
  ('cli', 'Trim CLI (TrimCLI/ User-Agent)', 1),
  ('ide', 'Trim IDE / editor extension', 2),
  ('ci', 'Continuous integration / headless agent', 3)
on conflict (id) do nothing;

alter table public.api_keys
  add column if not exists agent_id text references public.agent_identity_catalog(id) on delete restrict;

comment on column public.api_keys.hardware_uuid is
  'When set, every API-key request must send matching X-Hardware-UUID (device lock).';
comment on column public.api_keys.agent_id is
  'When set, every API-key request must send matching X-Trim-Agent-Id. Null = any catalog agent allowed.';

create index if not exists idx_api_keys_agent_id
  on public.api_keys (agent_id)
  where agent_id is not null;

insert into public.site_messages (code, body) values
  ('IP_DENIED', 'This IP address is denylisted. Contact support.'),
  ('ASN_DENIED', 'This network (ASN) is denylisted. Contact support.'),
  ('HARDWARE_UUID_MISMATCH', 'This API key is locked to a different device. Re-issue the key on this machine.'),
  ('HARDWARE_UUID_BOUND_REQUIRED', 'This API key requires X-Hardware-UUID.'),
  ('AGENT_ID_REQUIRED', 'X-Trim-Agent-Id is required for API key requests.'),
  ('AGENT_ID_UNKNOWN', 'Unknown X-Trim-Agent-Id. Use a value from agent_identity_catalog (cli, ide, ci).'),
  ('AGENT_ID_MISMATCH', 'This API key is locked to a different agent identity.'),
  ('ADMIN_AGENT_CATALOG_TITLE', 'Agent identities'),
  ('ADMIN_AGENT_COL_ID', 'Agent ID'),
  ('ADMIN_AGENT_COL_DESCRIPTION', 'Description')
on conflict (code) do nothing;
