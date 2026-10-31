-- Segmented Allowlist Deep: coding-critical force_tokens live in billing_settings only
-- (no invent merge in CLI). Seed/replace with official keepers. Idempotent JSON literal
-- (Postgres parses \n and \\ correctly here; do not rebuild via value::text trimming).

update public.billing_settings
set deep_v2_force_tokens = '["\n","/","\\",".","-","_",":","?","!",",","=","+","#","@"]'::jsonb,
    updated_at = now()
where true;
