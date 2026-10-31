-- Fix mangled force_tokens from an earlier rebuild that turned \n into two-char \n
-- and \ into \\\\ via value::text trimming. Real JSON escapes only.

update public.billing_settings
set deep_v2_force_tokens = '["\n","/","\\",".","-","_",":","?","!",",","=","+","#","@"]'::jsonb,
    updated_at = now()
where true;
