-- Public domain cutover: usetrim.ai → use-trim.com
-- Idempotent for databases that already applied older seeds containing usetrim.ai.
-- Fresh installs already get use-trim.com from updated historical migrations.
-- Brand/code identifiers (Trim, usetrim GitHub/publisher) are unchanged.

update public.site_messages
set body = replace(body, 'usetrim.ai', 'use-trim.com'),
    updated_at = now()
where body like '%usetrim.ai%';

update public.site_legal_sections
set
  heading = replace(heading, 'usetrim.ai', 'use-trim.com'),
  body = replace(body, 'usetrim.ai', 'use-trim.com'),
  contact_lead = replace(contact_lead, 'usetrim.ai', 'use-trim.com'),
  contact_trail = replace(contact_trail, 'usetrim.ai', 'use-trim.com')
where heading like '%usetrim.ai%'
   or body like '%usetrim.ai%'
   or contact_lead like '%usetrim.ai%'
   or contact_trail like '%usetrim.ai%';
