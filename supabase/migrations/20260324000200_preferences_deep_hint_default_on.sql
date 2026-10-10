-- Clarify Deep Mode hint: lead with On (default), not Off - avoids reading as current state.
-- Product defaults remain deep / v2 / auto_start true (migrations 152, 166, 198).

update public.site_messages
set body = 'On (default) = Deep for trim compress on your machine. Off = Fast for live proxy and local compress defaults. Live IDE proxy always stays Fast.',
    updated_at = now()
where code = 'PREFERENCES_DEEP_HINT';

update public.site_messages
set body = 'Default selected: LLMLingua-2 (v2). Pick long (needs a question) or v1 if you prefer. Only applies when Deep Mode is on.',
    updated_at = now()
where code = 'PREFERENCES_ENGINE_HINT';
