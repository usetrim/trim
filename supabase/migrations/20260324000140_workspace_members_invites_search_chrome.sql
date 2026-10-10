-- Members / invites live search chrome for team tables.
insert into public.site_messages (code, body) values
  (
    'WORKSPACE_MEMBERS_SEARCH',
    'Search members'
  ),
  (
    'WORKSPACE_MEMBERS_SEARCH_DESC',
    'Filters by email, name, or role as you type.'
  ),
  (
    'WORKSPACE_MEMBERS_EMPTY',
    'No members match this search.'
  ),
  (
    'WORKSPACE_INVITES_SEARCH',
    'Search invites'
  ),
  (
    'WORKSPACE_INVITES_SEARCH_DESC',
    'Filters by email, role, or status as you type.'
  )
on conflict (code) do update set body = excluded.body;
