-- Replace em dash (U+2014) and en dash (U+2013) with ASCII hyphen in chrome/copy text.
-- Safe, idempotent: only touches rows that still contain those glyphs.

do $$
declare
  em text := U&'\2014';
  en text := U&'\2013';
begin
  if to_regclass('public.site_messages') is not null then
    update public.site_messages
    set body = replace(replace(body, em, '-'), en, '-')
    where body like '%' || em || '%'
       or body like '%' || en || '%';
  end if;

  if to_regclass('public.plan_catalog') is not null then
    update public.plan_catalog
    set
      display_name = replace(replace(coalesce(display_name, ''), em, '-'), en, '-'),
      description = replace(replace(coalesce(description, ''), em, '-'), en, '-')
    where coalesce(display_name, '') like '%' || em || '%'
       or coalesce(display_name, '') like '%' || en || '%'
       or coalesce(description, '') like '%' || em || '%'
       or coalesce(description, '') like '%' || en || '%';
  end if;
end $$;
