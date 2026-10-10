"use client";

import { TeamPageSkeleton, WorkspaceListSkeleton } from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  type ColumnDef,
  DataTable,
  type RowSelectionState,
  getSelectedRowIds,
} from "@/components/ui/data-table";
import { DataTableRowActions } from "@/components/ui/data-table-row-actions";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  useCreateWorkspace,
  useDeleteWorkspace,
  useInviteWorkspaceMember,
  useRemoveWorkspaceMember,
  useRenameWorkspace,
  useRevokeWorkspaceInvite,
  useUpdateWorkspaceMemberRole,
} from "@/hooks/mutations/workspaces";
import { useAuthProviders } from "@/hooks/queries/auth";
import {
  useWorkspaceInvites,
  useWorkspaceMembers,
  useWorkspaces,
} from "@/hooks/queries/workspaces";
import { useDebouncedValue } from "@/hooks/use-debounced-value";
import { skeletonPageRows, useDefaultPageSize } from "@/hooks/use-default-page-size";
import { formatProviderPhrase } from "@/lib/auth-providers";
import { formatDateTimeFull, formatDateTimeShort } from "@/lib/format-datetime";
import { teamSkeletonChrome } from "@/lib/skeleton-chrome";
import type { Workspace, WorkspaceInvite, WorkspaceMember } from "@/types/workspace";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

export function TeamClient({
  accessToken,
  userId,
}: {
  accessToken?: string;
  userId?: string;
}) {
  const [skip, setSkip] = useState(0);
  const [workspaceSearchInput, setWorkspaceSearchInput] = useState("");
  const workspaceSearch = useDebouncedValue(workspaceSearchInput.trim(), 250);
  const [memberSkip, setMemberSkip] = useState(0);
  const [memberSearchInput, setMemberSearchInput] = useState("");
  const memberSearch = useDebouncedValue(memberSearchInput.trim(), 250);
  const [inviteSkip, setInviteSkip] = useState(0);
  const [inviteSearchInput, setInviteSearchInput] = useState("");
  const inviteSearch = useDebouncedValue(inviteSearchInput.trim(), 250);
  const [workspacesSel, setWorkspacesSel] = useState<RowSelectionState>({});
  const [membersSel, setMembersSel] = useState<RowSelectionState>({});
  const [invitesSel, setInvitesSel] = useState<RowSelectionState>({});
  const [name, setName] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState("");
  const [renameTarget, setRenameTarget] = useState<Workspace | null>(null);
  const [renameName, setRenameName] = useState("");
  const [deleteWorkspaceId, setDeleteWorkspaceId] = useState<string | null>(null);
  const [roleTarget, setRoleTarget] = useState<WorkspaceMember | null>(null);
  const [roleNext, setRoleNext] = useState("");
  const [memberConfirm, setMemberConfirm] = useState<{
    id: string;
    kind: "remove" | "leave";
  } | null>(null);
  const [bulkRemoveConfirm, setBulkRemoveConfirm] = useState(false);
  const [bulkRemovePending, setBulkRemovePending] = useState(false);
  const [bulkRevokePending, setBulkRevokePending] = useState(false);
  const [bulkDeleteConfirm, setBulkDeleteConfirm] = useState(false);
  const [bulkDeletePending, setBulkDeletePending] = useState(false);
  const [revokingId, setRevokingId] = useState<string | null>(null);
  const [lastInviteUrl, setLastInviteUrl] = useState<string | null>(null);
  const [copyLabel, setCopyLabel] = useState<string | null>(null);
  const copyResetRef = useRef<number | null>(null);
  const authProviders = useAuthProviders();
  const dialogCancel = authProviders.data?.site?.dialog_cancel?.trim() || "";
  const htmlLang = authProviders.data?.site?.html_lang?.trim() || "";
  const metaSep = authProviders.data?.site?.meta_sep || "";
  const pageChrome = teamSkeletonChrome(authProviders.data?.site);
  const providerPhrase = formatProviderPhrase(
    (authProviders.data?.items ?? [])
      .map((item) => item.display_name?.trim() || "")
      .filter((name) => name !== ""),
    authProviders.data?.site?.login_phrase_or,
    authProviders.data?.site?.login_phrase_comma,
  );
  const pageSize = useDefaultPageSize();
  const skeletonRows = skeletonPageRows(pageSize);
  const workspaces = useWorkspaces(
    accessToken,
    skip,
    pageSize > 0 ? pageSize : undefined,
    workspaceSearch,
  );
  const members = useWorkspaceMembers(
    accessToken,
    selectedId,
    memberSkip,
    pageSize > 0 ? pageSize : undefined,
    memberSearch,
  );
  const invites = useWorkspaceInvites(
    accessToken,
    selectedId,
    inviteSkip,
    pageSize > 0 ? pageSize : undefined,
    inviteSearch,
  );
  const createWs = useCreateWorkspace(accessToken);
  const renameWs = useRenameWorkspace(accessToken);
  const deleteWs = useDeleteWorkspace(accessToken);
  const invite = useInviteWorkspaceMember(accessToken, selectedId);
  const revokeInvite = useRevokeWorkspaceInvite(accessToken, selectedId);
  const removeMember = useRemoveWorkspaceMember(accessToken, selectedId);
  const updateMemberRole = useUpdateWorkspaceMemberRole(accessToken, selectedId);

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setSkip(0);
    setWorkspacesSel({});
  }, [workspaceSearch]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setMemberSkip(0);
  }, [memberSearch, selectedId]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setInviteSkip(0);
  }, [inviteSearch, selectedId]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setMemberSearchInput("");
    setInviteSearchInput("");
    setInviteRole("");
    setMembersSel({});
    setInvitesSel({});
  }, [selectedId]);

  useEffect(() => {
    return () => {
      if (copyResetRef.current != null) {
        window.clearTimeout(copyResetRef.current);
      }
    };
  }, []);

  const selectedWorkspace = (workspaces.data?.items ?? []).find((ws) => ws.id === selectedId);
  const canManage = selectedWorkspace?.role === "owner" || selectedWorkspace?.role === "admin";
  const ownerCount = members.data?.owner_count ?? 0;

  const tableSelectAll =
    invites.data?.table_select_all?.trim() ||
    members.data?.table_select_all?.trim() ||
    workspaces.data?.table_select_all?.trim() ||
    "";
  const tableSelectRow =
    invites.data?.table_select_row?.trim() ||
    members.data?.table_select_row?.trim() ||
    workspaces.data?.table_select_row?.trim() ||
    "";
  const tableSelectedFmt =
    invites.data?.table_selected_fmt?.trim() ||
    members.data?.table_selected_fmt?.trim() ||
    workspaces.data?.table_selected_fmt?.trim() ||
    "";
  const tableRowActions =
    invites.data?.table_row_actions?.trim() ||
    members.data?.table_row_actions?.trim() ||
    workspaces.data?.table_row_actions?.trim() ||
    "";
  const tableBulkRevoke =
    invites.data?.table_bulk_revoke?.trim() ||
    members.data?.table_bulk_revoke?.trim() ||
    workspaces.data?.table_bulk_revoke?.trim() ||
    "";
  const tableBulkRemove =
    members.data?.table_bulk_remove?.trim() ||
    workspaces.data?.table_bulk_remove?.trim() ||
    invites.data?.table_bulk_remove?.trim() ||
    "";
  const tableBulkDelete = workspaces.data?.table_bulk_delete?.trim() || "";
  const tableClearSelection =
    invites.data?.table_clear_selection?.trim() ||
    members.data?.table_clear_selection?.trim() ||
    workspaces.data?.table_clear_selection?.trim() ||
    "";
  const selectionChrome =
    tableSelectAll && tableSelectRow
      ? {
          selectAllLabel: tableSelectAll,
          selectRowLabel: tableSelectRow,
          selectedCountFmt: tableSelectedFmt || undefined,
        }
      : null;

  const selectWorkspace = useCallback((id: string) => {
    setSelectedId(id);
    setMemberSkip(0);
    setInviteSkip(0);
    setMembersSel({});
    setInvitesSel({});
    setLastInviteUrl(null);
    setCopyLabel(null);
  }, []);

  const memberAction = useCallback(
    (m: WorkspaceMember): "remove" | "leave" | null => {
      const isSelf = Boolean(userId && m.user_id === userId);
      if (isSelf) {
        if (m.role === "owner" && ownerCount <= 1) return null;
        return "leave";
      }
      if (!canManage) return null;
      if (selectedWorkspace?.role === "admin" && m.role === "owner") return null;
      return "remove";
    },
    [userId, ownerCount, canManage, selectedWorkspace?.role],
  );

  async function copyInviteLink(url: string) {
    const copied = (
      members.data?.copied_invite_action_label ||
      workspaces.data?.copied_invite_action_label ||
      ""
    ).trim();
    if (!copied || !url.trim()) return;
    if (copyResetRef.current != null) {
      window.clearTimeout(copyResetRef.current);
      copyResetRef.current = null;
    }
    try {
      await navigator.clipboard.writeText(url);
      setCopyLabel(copied);
      copyResetRef.current = window.setTimeout(() => {
        setCopyLabel(null);
        copyResetRef.current = null;
      }, 1600);
    } catch {
      setCopyLabel(null);
    }
  }

  const removeConfirmMessage =
    members.data?.remove_confirm_message || workspaces.data?.remove_confirm_message || "";
  const removeActionLabel =
    members.data?.remove_action_label || workspaces.data?.remove_action_label || "";
  const leaveConfirmMessage =
    members.data?.leave_confirm_message || workspaces.data?.leave_confirm_message || "";
  const leaveActionLabel =
    members.data?.leave_action_label || workspaces.data?.leave_action_label || "";
  const renameActionLabel = workspaces.data?.rename_action_label?.trim() || "";
  const renameSaveLabel = workspaces.data?.rename_save_action_label?.trim() || "";
  const renameSavePending = workspaces.data?.rename_save_pending_label?.trim() || "";
  const renameTitle = workspaces.data?.rename_title?.trim() || "";
  const deleteActionLabel = workspaces.data?.delete_action_label?.trim() || "";
  const deletePendingLabel = workspaces.data?.delete_pending_label?.trim() || "";
  const deleteConfirmMessage = workspaces.data?.delete_confirm_message?.trim() || "";
  const bulkDeleteConfirmMessage = workspaces.data?.bulk_delete_confirm_message?.trim() || "";
  const roleChangeActionLabel =
    members.data?.role_change_action_label?.trim() ||
    workspaces.data?.role_change_action_label?.trim() ||
    "";
  const roleChangePendingLabel =
    members.data?.role_change_pending_label?.trim() ||
    workspaces.data?.role_change_pending_label?.trim() ||
    "";
  const roleChangeTitle =
    members.data?.role_change_title?.trim() || workspaces.data?.role_change_title?.trim() || "";
  const roleFieldLabel =
    members.data?.role_label?.trim() || workspaces.data?.role_label?.trim() || "";
  const roleFieldDescription =
    members.data?.role_description?.trim() || workspaces.data?.role_description?.trim() || "";
  const inviteRoleOptions = (
    members.data?.invite_roles ||
    workspaces.data?.invite_roles ||
    []
  ).filter((r) => r.id?.trim() && r.label?.trim());
  const memberRoleOptions = useMemo(() => {
    const roles = (members.data?.member_roles || workspaces.data?.member_roles || []).filter(
      (r) => r.id?.trim() && r.label?.trim(),
    );
    if (selectedWorkspace?.role === "admin") {
      return roles.filter((r) => r.id !== "owner");
    }
    return roles;
  }, [members.data?.member_roles, workspaces.data?.member_roles, selectedWorkspace?.role]);
  const revokeActionLabel =
    invites.data?.revoke_action_label ||
    members.data?.revoke_action_label ||
    workspaces.data?.revoke_action_label ||
    "";
  const revokePendingLabel =
    invites.data?.revoke_pending_label ||
    members.data?.revoke_pending_label ||
    workspaces.data?.revoke_pending_label ||
    undefined;

  const selectedWorkspaceIds = getSelectedRowIds(workspacesSel);
  const deletableSelectedIds = selectedWorkspaceIds.filter((id) => {
    const ws = (workspaces.data?.items ?? []).find((row) => row.id === id);
    return ws?.role === "owner";
  });
  const showWorkspacesBulkDelete = Boolean(
    tableBulkDelete && bulkDeleteConfirmMessage && deletePendingLabel && dialogCancel,
  );
  const selectedMemberIds = getSelectedRowIds(membersSel);
  const removableSelectedIds = selectedMemberIds.filter((id) => {
    const m = (members.data?.items ?? []).find((row) => row.id === id);
    return m ? memberAction(m) === "remove" : false;
  });
  const showMembersBulkRemove = Boolean(
    tableBulkRemove && removeConfirmMessage && removeActionLabel && dialogCancel,
  );
  const selectedInviteIds = getSelectedRowIds(invitesSel);
  const showInvitesBulkRevoke = Boolean(tableBulkRevoke && revokeActionLabel);

  async function confirmBulkDeleteWorkspaces() {
    if (!deletableSelectedIds.length) {
      setBulkDeleteConfirm(false);
      return;
    }
    setBulkDeletePending(true);
    try {
      for (const id of deletableSelectedIds) {
        await deleteWs.mutateAsync(id);
        if (selectedId === id) {
          setSelectedId(null);
          setMembersSel({});
          setInvitesSel({});
        }
      }
      setWorkspacesSel({});
      setBulkDeleteConfirm(false);
    } finally {
      setBulkDeletePending(false);
    }
  }

  async function confirmBulkRemove() {
    if (!removableSelectedIds.length) {
      setBulkRemoveConfirm(false);
      return;
    }
    setBulkRemovePending(true);
    try {
      for (const id of removableSelectedIds) {
        await removeMember.mutateAsync(id);
      }
      setMembersSel({});
      setBulkRemoveConfirm(false);
    } finally {
      setBulkRemovePending(false);
    }
  }

  async function bulkRevokeInvites() {
    if (!selectedInviteIds.length) return;
    setBulkRevokePending(true);
    try {
      for (const id of selectedInviteIds) {
        await revokeInvite.mutateAsync(id);
      }
      setInvitesSel({});
    } finally {
      setBulkRevokePending(false);
    }
  }

  const workspaceColumns = useMemo<ColumnDef<Workspace>[]>(
    () => [
      {
        id: "name",
        header: workspaces.data?.name_placeholder || workspaces.data?.list_title || "",
        cell: ({ row }) => {
          const ws = row.original;
          return (
            <button
              type="button"
              onClick={() => selectWorkspace(ws.id)}
              className={`w-full text-left transition hover:bg-[var(--trim-hover)] ${
                selectedId === ws.id
                  ? "text-[var(--trim-fg)]"
                  : "text-[var(--trim-muted)] hover:text-[var(--trim-fg)]"
              }`}
            >
              <p className="font-medium text-[var(--trim-fg)]">{ws.name}</p>
              <p className="mt-1 text-xs text-[var(--trim-muted)]">{ws.summary_line}</p>
            </button>
          );
        },
      },
      {
        id: "_actions",
        header: "",
        cell: ({ row }) => {
          const ws = row.original;
          if (ws.role !== "owner" || !tableRowActions) return null;
          const actions = [];
          if (renameActionLabel && renameTitle && renameSaveLabel && renameSavePending) {
            actions.push({
              id: "rename",
              label: renameActionLabel,
              onSelect: () => {
                setRenameTarget(ws);
                setRenameName(ws.name);
              },
            });
          }
          if (deleteActionLabel && deleteConfirmMessage && deletePendingLabel && dialogCancel) {
            actions.push({
              id: "delete",
              label: deleteActionLabel,
              destructive: true,
              onSelect: () => setDeleteWorkspaceId(ws.id),
            });
          }
          if (actions.length === 0) return null;
          return <DataTableRowActions triggerLabel={tableRowActions} actions={actions} />;
        },
      },
    ],
    [
      workspaces.data,
      selectedId,
      selectWorkspace,
      tableRowActions,
      renameActionLabel,
      renameTitle,
      renameSaveLabel,
      renameSavePending,
      deleteActionLabel,
      deleteConfirmMessage,
      deletePendingLabel,
      dialogCancel,
    ],
  );

  const memberColumns = useMemo<ColumnDef<WorkspaceMember>[]>(
    () => [
      {
        id: "member",
        header: workspaces.data?.members_title || "",
        cell: ({ row }) => {
          const m = row.original;
          return (
            <div className="min-w-0">
              <p className="truncate text-[var(--trim-fg)]">{m.full_name || m.email}</p>
              <p className="truncate text-xs text-[var(--trim-muted)]">{m.email}</p>
            </div>
          );
        },
      },
      {
        id: "_role",
        header: "",
        cell: ({ row }) => (
          <span className="text-[var(--trim-muted)]">{row.original.role_label || ""}</span>
        ),
      },
      {
        id: "_actions",
        header: "",
        cell: ({ row }) => {
          const m = row.original;
          const action = memberAction(m);
          const actions: {
            id: string;
            label: string;
            destructive?: boolean;
            onSelect: () => void;
          }[] = [];
          const canChangeRole =
            canManage &&
            !(selectedWorkspace?.role === "admin" && m.role === "owner") &&
            memberRoleOptions.length > 0 &&
            Boolean(roleChangeActionLabel && roleChangeTitle && roleFieldLabel && dialogCancel);
          if (canChangeRole) {
            actions.push({
              id: "role",
              label: roleChangeActionLabel,
              onSelect: () => {
                setRoleTarget(m);
                setRoleNext(m.role);
              },
            });
          }
          if (action === "remove") {
            if (removeConfirmMessage && dialogCancel && removeActionLabel) {
              actions.push({
                id: "remove",
                label: removeActionLabel,
                destructive: true,
                onSelect: () => setMemberConfirm({ id: m.id, kind: "remove" }),
              });
            }
          }
          if (action === "leave") {
            if (leaveConfirmMessage && dialogCancel && leaveActionLabel) {
              actions.push({
                id: "leave",
                label: leaveActionLabel,
                destructive: true,
                onSelect: () => setMemberConfirm({ id: m.id, kind: "leave" }),
              });
            }
          }
          if (!tableRowActions || actions.length === 0) return null;
          return <DataTableRowActions triggerLabel={tableRowActions} actions={actions} />;
        },
      },
    ],
    [
      workspaces.data,
      dialogCancel,
      memberAction,
      tableRowActions,
      removeConfirmMessage,
      removeActionLabel,
      leaveConfirmMessage,
      leaveActionLabel,
      canManage,
      selectedWorkspace?.role,
      memberRoleOptions,
      roleChangeActionLabel,
      roleChangeTitle,
      roleFieldLabel,
    ],
  );

  const inviteColumns = useMemo<ColumnDef<WorkspaceInvite>[]>(
    () => [
      {
        id: "email",
        header: workspaces.data?.invites_title || "",
        cell: ({ row }) => {
          const inv = row.original;
          const role = inv.role_label?.trim() || "";
          const when = formatDateTimeShort(inv.expires_at, htmlLang);
          const meta = [role, when].filter(Boolean).join(metaSep);
          return (
            <div className="min-w-0">
              <p className="truncate text-[var(--trim-fg)]">{inv.email}</p>
              <p
                className="truncate text-xs text-[var(--trim-muted)]"
                title={formatDateTimeFull(inv.expires_at, htmlLang) || undefined}
              >
                {meta}
              </p>
            </div>
          );
        },
      },
      {
        id: "_actions",
        header: "",
        cell: ({ row }) => {
          const inv = row.original;
          if (!revokeActionLabel) return null;
          return (
            <DataTableRowActions
              triggerLabel={tableRowActions}
              actions={[
                {
                  id: "revoke",
                  label: revokeActionLabel,
                  destructive: true,
                  disabled: revokeInvite.isPending && revokingId === inv.id,
                  onSelect: () => {
                    setRevokingId(inv.id);
                    revokeInvite.mutate(inv.id, {
                      onSettled: () => setRevokingId(null),
                      onSuccess: () =>
                        setInvitesSel((prev) => {
                          const next = { ...prev };
                          delete next[inv.id];
                          return next;
                        }),
                    });
                  },
                },
              ]}
            />
          );
        },
      },
    ],
    [
      workspaces.data,
      revokeActionLabel,
      tableRowActions,
      revokeInvite,
      revokingId,
      htmlLang,
      metaSep,
    ],
  );

  const workspacesPending = workspaces.isPending;
  const workspacesData = workspaces.data;
  const workspacesError = workspaces.error;

  if (!workspacesData && !workspacesError) {
    return (
      <TeamPageSkeleton
        chrome={pageChrome}
        workspaceRows={skeletonRows}
        memberRows={skeletonRows}
        embedded
      />
    );
  }

  return (
    <div className="w-full space-y-6">
      <div className="min-w-0">
        <p className="text-sm text-[var(--trim-muted)]">{workspaces.data?.page_eyebrow}</p>
        <h1 className="text-2xl font-semibold tracking-tight text-[var(--trim-fg)] sm:text-3xl">
          {workspaces.data?.page_title}
        </h1>
        <p className="mt-1 text-sm text-[var(--trim-muted)]">
          {workspaces.data?.page_description}
          {providerPhrase ? ` (${providerPhrase})` : ""}
        </p>
      </div>

      <div className="w-full">
        <div className="grid gap-6 lg:grid-cols-[1fr_1.2fr]">
          <Card>
            <CardHeader>
              <CardTitle className="text-base text-[var(--trim-fg)]">
                {workspaces.data?.list_title}
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <form
                className="flex flex-col gap-3 sm:flex-row sm:items-end"
                onSubmit={(e) => {
                  e.preventDefault();
                  if (!name.trim()) return;
                  createWs.mutate(
                    { name: name.trim() },
                    {
                      onSuccess: () => {
                        setName("");
                        workspaces.refetch();
                      },
                    },
                  );
                }}
              >
                <Field
                  id="team-workspace-name"
                  label={workspaces.data?.name_label || workspaces.data?.list_title || ""}
                  description={workspaces.data?.name_description}
                  className="min-w-0 flex-1"
                >
                  <Input
                    id="team-workspace-name"
                    placeholder={workspaces.data?.name_placeholder || ""}
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                  />
                </Field>
                <Button
                  type="submit"
                  disabled={!workspaces.data?.create_action_label}
                  isLoading={createWs.isPending}
                  pendingLabel={workspaces.data?.create_pending_label || undefined}
                >
                  {workspaces.data?.create_action_label}
                </Button>
              </form>

              {workspacesData?.search_placeholder?.trim() ? (
                <Field
                  id="team-workspaces-search"
                  label={workspacesData.search_placeholder}
                  description={workspacesData.search_description}
                  className="w-full"
                >
                  <Input
                    id="team-workspaces-search"
                    value={workspaceSearchInput}
                    onChange={(e) => setWorkspaceSearchInput(e.target.value)}
                    placeholder={workspacesData.search_placeholder}
                    autoComplete="off"
                  />
                </Field>
              ) : null}

              {workspacesPending && workspacesData == null ? (
                <WorkspaceListSkeleton rows={skeletonRows} />
              ) : null}
              {workspacesError ? (
                <p className="text-sm text-destructive">
                  {(workspacesError instanceof Error && workspacesError.message) || ""}
                </p>
              ) : null}

              {workspacesData ? (
                <>
                  <DataTable
                    columns={workspaceColumns}
                    data={workspacesData.items ?? []}
                    bordered={false}
                    getRowId={(row) => row.id}
                    meta={workspacesData.meta}
                    onPage={
                      workspacesData.meta
                        ? (s) => {
                            setSkip(s);
                            setWorkspacesSel({});
                          }
                        : undefined
                    }
                    pageDisabled={workspaces.isFetching}
                    pageInputId="team-workspaces-skip-to"
                    isFetching={workspaces.isFetching && !workspacesPending}
                    bodyRowClassName={(row) =>
                      selectedId === row.id ? "bg-[var(--trim-hover)]/60" : undefined
                    }
                    selection={
                      selectionChrome
                        ? {
                            chrome: selectionChrome,
                            rowSelection: workspacesSel,
                            onRowSelectionChange: setWorkspacesSel,
                            toolbar: showWorkspacesBulkDelete ? (
                              <div className="flex flex-wrap gap-2">
                                <Button
                                  type="button"
                                  size="sm"
                                  variant="destructive"
                                  disabled={!deletableSelectedIds.length}
                                  onClick={() => setBulkDeleteConfirm(true)}
                                >
                                  {tableBulkDelete}
                                </Button>
                                {tableClearSelection ? (
                                  <Button
                                    type="button"
                                    size="sm"
                                    variant="outline"
                                    onClick={() => setWorkspacesSel({})}
                                  >
                                    {tableClearSelection}
                                  </Button>
                                ) : null}
                              </div>
                            ) : undefined,
                          }
                        : undefined
                    }
                  />
                  {(workspacesData.items ?? []).length === 0 ? (
                    <p className="py-3 text-sm text-[var(--trim-muted)]">
                      {workspacesData.empty_message}
                    </p>
                  ) : null}
                </>
              ) : null}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base text-[var(--trim-fg)]">
                {workspaces.data?.members_title}
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {!selectedId ? (
                <p className="text-sm text-[var(--trim-muted)]">{workspaces.data?.select_hint}</p>
              ) : (
                <>
                  {canManage ? (
                    <form
                      className="flex flex-col gap-3 sm:flex-row sm:items-end"
                      onSubmit={(e) => {
                        e.preventDefault();
                        if (!inviteEmail.trim() || !inviteRole.trim()) return;
                        invite.mutate(
                          { email: inviteEmail.trim(), role: inviteRole.trim() },
                          {
                            onSuccess: (data) => {
                              setInviteEmail("");
                              setInviteRole("");
                              setLastInviteUrl(data.invite_url);
                              setCopyLabel(null);
                              members.refetch();
                              invites.refetch();
                            },
                          },
                        );
                      }}
                    >
                      <Field
                        id="team-invite-email"
                        label={
                          workspaces.data?.invite_email_label ||
                          members.data?.invite_action_label ||
                          workspaces.data?.invite_action_label ||
                          ""
                        }
                        description={workspaces.data?.invite_email_description}
                        className="min-w-0 flex-1"
                      >
                        <Input
                          id="team-invite-email"
                          placeholder={workspaces.data?.invite_email_placeholder || ""}
                          type="email"
                          value={inviteEmail}
                          onChange={(e) => setInviteEmail(e.target.value)}
                        />
                      </Field>
                      {roleFieldLabel && inviteRoleOptions.length > 0 ? (
                        <Field
                          id="team-invite-role"
                          label={roleFieldLabel}
                          description={roleFieldDescription || undefined}
                          className="w-full sm:w-40"
                        >
                          <Select value={inviteRole || undefined} onValueChange={setInviteRole}>
                            <SelectTrigger id="team-invite-role" aria-label={roleFieldLabel}>
                              <SelectValue placeholder={roleFieldLabel} />
                            </SelectTrigger>
                            <SelectContent>
                              {inviteRoleOptions.map((opt) => (
                                <SelectItem key={opt.id} value={opt.id}>
                                  {opt.label}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        </Field>
                      ) : null}
                      <Button
                        type="submit"
                        disabled={
                          !(
                            members.data?.invite_action_label ||
                            workspaces.data?.invite_action_label
                          ) ||
                          inviteRoleOptions.length === 0 ||
                          !inviteRole.trim()
                        }
                        isLoading={invite.isPending}
                        pendingLabel={
                          members.data?.invite_pending_label ||
                          workspaces.data?.invite_pending_label ||
                          undefined
                        }
                      >
                        {members.data?.invite_action_label || workspaces.data?.invite_action_label}
                      </Button>
                    </form>
                  ) : null}
                  {invite.error ? (
                    <p className="text-sm text-destructive">
                      {(invite.error instanceof Error && invite.error.message) || ""}
                    </p>
                  ) : null}
                  {lastInviteUrl ? (
                    <div className="rounded-md border border-[var(--trim-border)] bg-[var(--trim-bg)] p-3">
                      <p className="text-xs text-[var(--trim-muted)]">
                        {workspaces.data?.invite_url_label}
                      </p>
                      <p className="mt-1 break-all text-xs text-[var(--trim-muted)]">
                        {lastInviteUrl}
                      </p>
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        className="mt-2 min-w-[5.5rem]"
                        disabled={
                          !(
                            members.data?.copy_invite_action_label ||
                            workspaces.data?.copy_invite_action_label
                          )
                        }
                        onClick={() => void copyInviteLink(lastInviteUrl)}
                      >
                        {copyLabel ||
                          members.data?.copy_invite_action_label ||
                          workspaces.data?.copy_invite_action_label}
                      </Button>
                    </div>
                  ) : null}
                  {removeMember.error ? (
                    <p className="text-sm text-destructive">
                      {(removeMember.error instanceof Error && removeMember.error.message) || ""}
                    </p>
                  ) : null}
                  {members.data?.search_placeholder?.trim() ? (
                    <Field
                      id="team-members-search"
                      label={members.data.search_placeholder}
                      description={members.data.search_description}
                      className="w-full"
                    >
                      <Input
                        id="team-members-search"
                        value={memberSearchInput}
                        onChange={(e) => setMemberSearchInput(e.target.value)}
                        placeholder={members.data.search_placeholder}
                        autoComplete="off"
                      />
                    </Field>
                  ) : null}
                  <>
                    <DataTable
                      columns={memberColumns}
                      data={members.data?.items ?? []}
                      bordered={false}
                      getRowId={(row) => row.id}
                      meta={members.data?.meta}
                      onPage={
                        members.data?.meta
                          ? (s) => {
                              setMemberSkip(s);
                              setMembersSel({});
                            }
                          : undefined
                      }
                      pageDisabled={members.isFetching}
                      pageInputId="team-members-skip-to"
                      isFetching={members.isFetching}
                      selection={
                        selectionChrome
                          ? {
                              chrome: selectionChrome,
                              rowSelection: membersSel,
                              onRowSelectionChange: setMembersSel,
                              toolbar: showMembersBulkRemove ? (
                                <div className="flex flex-wrap gap-2">
                                  <Button
                                    type="button"
                                    size="sm"
                                    variant="destructive"
                                    disabled={!removableSelectedIds.length}
                                    onClick={() => setBulkRemoveConfirm(true)}
                                  >
                                    {tableBulkRemove}
                                  </Button>
                                  {tableClearSelection ? (
                                    <Button
                                      type="button"
                                      size="sm"
                                      variant="outline"
                                      onClick={() => setMembersSel({})}
                                    >
                                      {tableClearSelection}
                                    </Button>
                                  ) : null}
                                </div>
                              ) : undefined,
                            }
                          : undefined
                      }
                    />
                    {!members.isFetching && (members.data?.items ?? []).length === 0 ? (
                      <p className="py-3 text-sm text-[var(--trim-muted)]">
                        {members.data?.empty_message || ""}
                      </p>
                    ) : null}
                  </>

                  {canManage ? (
                    <div className="border-t border-[var(--trim-border)] pt-4">
                      <p className="mb-2 text-sm font-medium text-[var(--trim-fg)]">
                        {workspaces.data?.invites_title}
                      </p>
                      {invites.data?.search_placeholder?.trim() ? (
                        <Field
                          id="team-invites-search"
                          label={invites.data.search_placeholder}
                          description={invites.data.search_description}
                          className="mb-3 w-full"
                        >
                          <Input
                            id="team-invites-search"
                            value={inviteSearchInput}
                            onChange={(e) => setInviteSearchInput(e.target.value)}
                            placeholder={invites.data.search_placeholder}
                            autoComplete="off"
                          />
                        </Field>
                      ) : null}
                      {invites.error ? (
                        <p className="text-sm text-destructive">
                          {(invites.error instanceof Error && invites.error.message) || ""}
                        </p>
                      ) : null}
                      <>
                        <DataTable
                          columns={inviteColumns}
                          data={invites.data?.items ?? []}
                          bordered={false}
                          getRowId={(row) => row.id}
                          meta={invites.data?.meta}
                          onPage={
                            invites.data?.meta
                              ? (s) => {
                                  setInviteSkip(s);
                                  setInvitesSel({});
                                }
                              : undefined
                          }
                          pageDisabled={invites.isFetching}
                          pageInputId="team-invites-skip-to"
                          isFetching={invites.isFetching}
                          selection={
                            selectionChrome
                              ? {
                                  chrome: selectionChrome,
                                  rowSelection: invitesSel,
                                  onRowSelectionChange: setInvitesSel,
                                  toolbar: showInvitesBulkRevoke ? (
                                    <div className="flex flex-wrap gap-2">
                                      <Button
                                        type="button"
                                        size="sm"
                                        variant="destructive"
                                        isLoading={bulkRevokePending}
                                        pendingLabel={revokePendingLabel}
                                        disabled={!selectedInviteIds.length}
                                        onClick={() => void bulkRevokeInvites()}
                                      >
                                        {tableBulkRevoke}
                                      </Button>
                                      {tableClearSelection ? (
                                        <Button
                                          type="button"
                                          size="sm"
                                          variant="outline"
                                          onClick={() => setInvitesSel({})}
                                        >
                                          {tableClearSelection}
                                        </Button>
                                      ) : null}
                                    </div>
                                  ) : undefined,
                                }
                              : undefined
                          }
                        />
                        {!invites.isFetching && (invites.data?.items ?? []).length === 0 ? (
                          <p className="py-2 text-sm text-[var(--trim-muted)]">
                            {workspaces.data?.invites_empty_message}
                          </p>
                        ) : null}
                      </>
                    </div>
                  ) : null}
                </>
              )}
            </CardContent>
          </Card>
        </div>
        <ConfirmDialog
          open={Boolean(memberConfirm)}
          onOpenChange={(open) => {
            if (!open && !removeMember.isPending) setMemberConfirm(null);
          }}
          description={memberConfirm?.kind === "leave" ? leaveConfirmMessage : removeConfirmMessage}
          cancelLabel={dialogCancel}
          confirmLabel={memberConfirm?.kind === "leave" ? leaveActionLabel : removeActionLabel}
          pendingLabel={
            memberConfirm?.kind === "leave"
              ? members.data?.leave_pending_label ||
                workspaces.data?.leave_pending_label ||
                undefined
              : members.data?.remove_pending_label ||
                workspaces.data?.remove_pending_label ||
                undefined
          }
          isPending={removeMember.isPending}
          destructive
          onConfirm={() => {
            if (!memberConfirm) return;
            const kind = memberConfirm.kind;
            const id = memberConfirm.id;
            removeMember.mutate(id, {
              onSettled: () => setMemberConfirm(null),
              onSuccess: () => {
                setMembersSel((prev) => {
                  const next = { ...prev };
                  delete next[id];
                  return next;
                });
                if (kind === "leave") {
                  setSelectedId(null);
                  setMembersSel({});
                  setInvitesSel({});
                  void workspaces.refetch();
                }
              },
            });
          }}
        />
        <ConfirmDialog
          open={bulkRemoveConfirm}
          onOpenChange={(open) => {
            if (!open && !bulkRemovePending) setBulkRemoveConfirm(false);
          }}
          description={removeConfirmMessage}
          cancelLabel={dialogCancel}
          confirmLabel={tableBulkRemove}
          pendingLabel={
            members.data?.remove_pending_label || workspaces.data?.remove_pending_label || undefined
          }
          isPending={bulkRemovePending}
          destructive
          onConfirm={() => {
            void confirmBulkRemove();
          }}
        />
        <ConfirmDialog
          open={Boolean(deleteWorkspaceId)}
          onOpenChange={(open) => {
            if (!open && !deleteWs.isPending) setDeleteWorkspaceId(null);
          }}
          description={deleteConfirmMessage}
          cancelLabel={dialogCancel}
          confirmLabel={deleteActionLabel}
          pendingLabel={deletePendingLabel || undefined}
          isPending={deleteWs.isPending}
          destructive
          onConfirm={() => {
            if (!deleteWorkspaceId) return;
            const id = deleteWorkspaceId;
            deleteWs.mutate(id, {
              onSettled: () => setDeleteWorkspaceId(null),
              onSuccess: () => {
                setWorkspacesSel((prev) => {
                  const next = { ...prev };
                  delete next[id];
                  return next;
                });
                if (selectedId === id) {
                  setSelectedId(null);
                  setMembersSel({});
                  setInvitesSel({});
                }
              },
            });
          }}
        />
        <ConfirmDialog
          open={bulkDeleteConfirm}
          onOpenChange={(open) => {
            if (!open && !bulkDeletePending) setBulkDeleteConfirm(false);
          }}
          description={bulkDeleteConfirmMessage}
          cancelLabel={dialogCancel}
          confirmLabel={tableBulkDelete}
          pendingLabel={deletePendingLabel || undefined}
          isPending={bulkDeletePending}
          destructive
          onConfirm={() => {
            void confirmBulkDeleteWorkspaces();
          }}
        />
        <Dialog
          open={Boolean(renameTarget)}
          onOpenChange={(open) => {
            if (!open && !renameWs.isPending) {
              setRenameTarget(null);
              setRenameName("");
            }
          }}
        >
          <DialogContent closeLabel={dialogCancel}>
            <DialogHeader>
              <DialogTitle>{renameTitle}</DialogTitle>
              {workspaces.data?.name_description ? (
                <DialogDescription>{workspaces.data.name_description}</DialogDescription>
              ) : null}
            </DialogHeader>
            <Field
              id="team-rename-workspace-name"
              label={workspaces.data?.name_label || workspaces.data?.list_title || ""}
              description={workspaces.data?.name_description}
            >
              <Input
                id="team-rename-workspace-name"
                value={renameName}
                onChange={(e) => setRenameName(e.target.value)}
                placeholder={workspaces.data?.name_placeholder || ""}
              />
            </Field>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                disabled={renameWs.isPending || !dialogCancel}
                onClick={() => {
                  setRenameTarget(null);
                  setRenameName("");
                }}
              >
                {dialogCancel}
              </Button>
              <Button
                type="button"
                disabled={!renameTarget || !renameName.trim() || !renameSaveLabel}
                isLoading={renameWs.isPending}
                pendingLabel={renameSavePending || undefined}
                onClick={() => {
                  if (!renameTarget || !renameName.trim()) return;
                  renameWs.mutate(
                    { workspaceId: renameTarget.id, name: renameName.trim() },
                    {
                      onSuccess: () => {
                        setRenameTarget(null);
                        setRenameName("");
                      },
                    },
                  );
                }}
              >
                {renameSaveLabel}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
        <Dialog
          open={Boolean(roleTarget)}
          onOpenChange={(open) => {
            if (!open && !updateMemberRole.isPending) {
              setRoleTarget(null);
              setRoleNext("");
            }
          }}
        >
          <DialogContent closeLabel={dialogCancel}>
            <DialogHeader>
              <DialogTitle>{roleChangeTitle}</DialogTitle>
              {roleFieldDescription ? (
                <DialogDescription>{roleFieldDescription}</DialogDescription>
              ) : null}
            </DialogHeader>
            {roleFieldLabel && memberRoleOptions.length > 0 ? (
              <Field
                id="team-member-role"
                label={roleFieldLabel}
                description={roleFieldDescription}
              >
                <Select value={roleNext || undefined} onValueChange={setRoleNext}>
                  <SelectTrigger id="team-member-role" aria-label={roleFieldLabel}>
                    <SelectValue placeholder={roleFieldLabel} />
                  </SelectTrigger>
                  <SelectContent>
                    {memberRoleOptions.map((opt) => (
                      <SelectItem key={opt.id} value={opt.id}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>
            ) : null}
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                disabled={updateMemberRole.isPending || !dialogCancel}
                onClick={() => {
                  setRoleTarget(null);
                  setRoleNext("");
                }}
              >
                {dialogCancel}
              </Button>
              <Button
                type="button"
                disabled={
                  !roleTarget ||
                  !roleNext.trim() ||
                  !roleChangeActionLabel ||
                  memberRoleOptions.length === 0
                }
                isLoading={updateMemberRole.isPending}
                pendingLabel={roleChangePendingLabel || undefined}
                onClick={() => {
                  if (!roleTarget || !roleNext.trim()) return;
                  updateMemberRole.mutate(
                    { memberId: roleTarget.id, role: roleNext.trim() },
                    {
                      onSuccess: () => {
                        setRoleTarget(null);
                        setRoleNext("");
                        void members.refetch();
                      },
                    },
                  );
                }}
              >
                {roleChangeActionLabel}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </div>
  );
}
