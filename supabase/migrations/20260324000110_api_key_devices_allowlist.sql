-- Multi-device allowlist for API keys (CLI + IDE share a bound key safely).
-- api_key_max_devices is fail-closed: must be set (1-16); default seed 2.

alter table public.admin_product_settings
  add column if not exists api_key_max_devices int;

alter table public.admin_product_settings
  drop constraint if exists admin_product_settings_api_key_max_devices_check;

alter table public.admin_product_settings
  add constraint admin_product_settings_api_key_max_devices_check
  check (api_key_max_devices is null or (api_key_max_devices >= 1 and api_key_max_devices <= 16));

comment on column public.admin_product_settings.api_key_max_devices is
  'Max distinct X-Hardware-UUID values per API key (allowlist). Required for device-bound keys; seed 2 (cli+ide).';

update public.admin_product_settings
set api_key_max_devices = 2
where id = 'default' and api_key_max_devices is null;

create table if not exists public.api_key_devices (
  key_id uuid not null references public.api_keys(id) on delete cascade,
  hardware_uuid text not null,
  agent_id text references public.agent_identity_catalog(id) on delete set null,
  first_seen_at timestamptz not null default now(),
  last_seen_at timestamptz not null default now(),
  primary key (key_id, hardware_uuid)
);

create index if not exists idx_api_key_devices_hw
  on public.api_key_devices (hardware_uuid);

comment on table public.api_key_devices is
  'Allowlisted hardware UUIDs for an API key. Empty + null api_keys.hardware_uuid = unbound key.';

-- Backfill from legacy single-column bind.
insert into public.api_key_devices (key_id, hardware_uuid, agent_id, first_seen_at, last_seen_at)
select k.id, k.hardware_uuid, k.agent_id, k.created_at, now()
from public.api_keys k
where k.hardware_uuid is not null
  and length(trim(k.hardware_uuid)) > 0
on conflict (key_id, hardware_uuid) do nothing;

insert into public.site_messages (code, body) values
  ('HARDWARE_DEVICE_LIMIT', 'This API key already has the maximum number of registered devices. Revoke and re-issue, or remove a device.'),
  ('API_KEY_MAX_DEVICES_MISSING', 'Set admin_product_settings.api_key_max_devices (1-16) before device-bound API keys can enroll.'),
  ('ADMIN_PRODUCT_API_KEY_MAX_DEVICES', 'API key max devices'),
  ('DENYLIST_UNAVAILABLE', 'Abuse denylist unavailable. Try again shortly.')
on conflict (code) do nothing;
