-- Align admin Product dials + chrome with signup defaults (site_messages SoT unchanged).
-- Signup still reads DEFAULT_* from site_messages; this keeps Admin → Product from looking blank/Fast.

update public.site_messages
set body = 'Default compression tier for new accounts when unset in preferences (deep = Deep Mode on).',
    updated_at = now()
where code = 'ADMIN_PRODUCT_MODE_DESC'
  and body is distinct from 'Default compression tier for new accounts when unset in preferences (deep = Deep Mode on).';

update public.admin_product_settings
set
  default_compression_mode = coalesce(nullif(btrim(default_compression_mode), ''), 'deep'),
  default_deep_engine = coalesce(nullif(btrim(default_deep_engine), ''), 'v2'),
  updated_at = now()
where id = 'default';
