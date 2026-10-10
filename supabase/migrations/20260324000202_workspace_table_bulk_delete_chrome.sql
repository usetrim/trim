-- Workspace list DataTable bulk-delete chrome (DB-driven; no invent).
-- Matches TABLE_BULK_REMOVE / TABLE_BULK_REVOKE pattern used by members and invites.

insert into public.site_messages (code, body) values
  ('TABLE_BULK_DELETE', 'Delete selected'),
  ('WORKSPACE_BULK_DELETE_CONFIRM', 'Delete the selected workspaces permanently? Members and pending invites are removed. This cannot be undone.')
on conflict (code) do update set body = excluded.body, updated_at = now();
