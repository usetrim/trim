-- Soften remaining public jargon without inventing entity details.
update public.site_legal_sections
set body = replace(body, 'free-tier eligibility', 'free plan eligibility'),
    updated_at = now()
where body like '%free-tier eligibility%';

update public.site_messages
set body = to_char(now() at time zone 'utc', 'YYYY-MM-DD'),
    updated_at = now()
where code = 'LEGAL_UPDATED_DATE';
