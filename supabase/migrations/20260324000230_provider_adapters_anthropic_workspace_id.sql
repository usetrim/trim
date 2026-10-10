-- Schema column for Anthropic workspace id (compat). Product path does NOT inject this column.
-- Org/multi-workspace keys: client anthropic-workspace-id header or TRIM_ANTHROPIC_WORKSPACE_ID.
-- Prefer workspace-scoped Anthropic keys (omit header/env). Anthropic returns 400 if an org-scoped
-- key is used with neither.

alter table public.provider_adapters
  add column if not exists anthropic_workspace_id text;

alter table public.provider_adapters
  drop constraint if exists provider_adapters_anthropic_workspace_id_check;

alter table public.provider_adapters
  add constraint provider_adapters_anthropic_workspace_id_check
  check (
    anthropic_workspace_id is null
    or length(btrim(anthropic_workspace_id)) between 1 and 128
  );

comment on column public.provider_adapters.anthropic_workspace_id is
  'Optional Anthropic workspace id column (schema compat). Product path for org/multi-workspace keys: client anthropic-workspace-id header or TRIM_ANTHROPIC_WORKSPACE_ID. Prefer workspace-scoped keys (omit). Proxy does not inject this column as a SaaS default.';
