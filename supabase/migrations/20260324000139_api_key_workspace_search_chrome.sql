-- Live search chrome for settings API keys and team workspaces lists.
insert into public.site_messages (code, body) values
  (
    'API_KEY_SEARCH',
    'Search API keys'
  ),
  (
    'API_KEY_SEARCH_DESC',
    'Filters by key prefix, id, or status as you type.'
  ),
  (
    'WORKSPACE_SEARCH',
    'Search workspaces'
  ),
  (
    'WORKSPACE_SEARCH_DESC',
    'Filters by name, plan, role, or workspace id as you type.'
  )
on conflict (code) do update set body = excluded.body;
