-- Workspace rename/delete + member role change chrome (DB-driven; no invent).
-- ACTION:/PENDING: labels are also seeded via SeedAndRefresh builtins; this migration
-- ensures live installs get bodies even before next API invent-insert.

insert into public.site_messages (code, body) values
  ('ACTION:WORKSPACE_RENAME', 'Rename'),
  ('PENDING:WORKSPACE_RENAME', 'Renaming...'),
  ('ACTION:WORKSPACE_RENAME_SAVE', 'Save name'),
  ('PENDING:WORKSPACE_RENAME_SAVE', 'Saving...'),
  ('ACTION:WORKSPACE_DELETE', 'Delete workspace'),
  ('PENDING:WORKSPACE_DELETE', 'Deleting...'),
  ('ACTION:WORKSPACE_ROLE_CHANGE', 'Change role'),
  ('PENDING:WORKSPACE_ROLE_CHANGE', 'Updating role...'),
  ('WORKSPACE_RENAME_TITLE', 'Rename workspace'),
  ('WORKSPACE_DELETE_CONFIRM', 'Delete this workspace permanently? Members and pending invites are removed. This cannot be undone.'),
  ('WORKSPACE_ROLE_CHANGE_TITLE', 'Change member role'),
  ('WORKSPACE_ROLE_LABEL', 'Role'),
  ('WORKSPACE_ROLE_DESCRIPTION', 'Owners and admins manage the workspace. Members use shared seats and credits.'),
  ('WORKSPACE_RENAMED', 'Workspace renamed.'),
  ('WORKSPACE_DELETED', 'Workspace deleted.'),
  ('WORKSPACE_MEMBER_ROLE_UPDATED', 'Member role updated.'),
  ('WORKSPACE_NAME_DESC', 'Shown to members and on invite emails.'),
  ('WS_RENAME_FORBIDDEN', 'Only workspace owners can rename this workspace.'),
  ('WS_DELETE_FORBIDDEN', 'Only workspace owners can delete this workspace.'),
  ('WS_ROLE_CHANGE_FORBIDDEN', 'You cannot change this member role.'),
  ('WS_INVALID_MEMBER_ROLE', 'That role is not allowed.'),
  ('WS_CANNOT_DEMOTE_LAST_OWNER', 'Cannot change role: the workspace must keep at least one owner.'),
  ('WS_WORKSPACE_NOT_FOUND', 'Workspace not found.'),
  ('WS_MEMBER_NOT_FOUND', 'Member not found.'),
  ('WS_RENAME_FAILED', 'Could not rename the workspace.'),
  ('WS_DELETE_FAILED', 'Could not delete the workspace.'),
  ('WS_ROLE_CHANGE_FAILED', 'Could not update the member role.')
on conflict (code) do update set body = excluded.body, updated_at = now();
