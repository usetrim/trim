-- Align leftover installer/Cursor/IDE chrome that still said :8000 with TRIM_PORT examples.

update public.site_messages
set body = replace(body, 'http://127.0.0.1:8000', 'http://127.0.0.1:8888'),
    updated_at = now()
where body like '%http://127.0.0.1:8000%';

update public.site_messages
set body = replace(body, 'http://localhost:8000', 'http://localhost:8888'),
    updated_at = now()
where body like '%http://localhost:8000%';
