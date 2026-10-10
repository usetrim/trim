-- Cursor-like telemetry extensions on trim_events (IDE clients may ingest later).
-- All counters default to 0 so existing proxy events remain valid.

alter table public.trim_events
  add column if not exists tab_suggestions_shown int not null default 0,
  add column if not exists tab_suggestions_accepted int not null default 0,
  add column if not exists ai_lines_added int not null default 0,
  add column if not exists ai_lines_deleted int not null default 0;

comment on column public.trim_events.tab_suggestions_shown is 'Inline tab suggestions shown (IDE telemetry)';
comment on column public.trim_events.tab_suggestions_accepted is 'Inline tab suggestions accepted (IDE telemetry)';
comment on column public.trim_events.ai_lines_added is 'AI-generated lines added (IDE telemetry)';
comment on column public.trim_events.ai_lines_deleted is 'AI-generated lines deleted (IDE telemetry)';
