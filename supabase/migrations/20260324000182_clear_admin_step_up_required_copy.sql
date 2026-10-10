-- Obsolete per-action step-up copy. Enrolled MFA is enough for admin CRUD.
-- Clears hostile "Confirm this action..." toast/dialog text if any client still requests it.

update public.site_messages
set body = '',
    updated_at = now()
where code = 'ADMIN_STEP_UP_REQUIRED'
  and body is distinct from '';
