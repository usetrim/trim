-- Door label chrome for dashboard/TUI (provider adapter path visibility).

insert into public.site_messages (code, body, updated_at)
values
  ('LOCAL_STATS_LAST_DOOR', 'Last door', now()),
  ('LOCAL_TUI_LABEL_LAST_DOOR', 'Last door', now())
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
