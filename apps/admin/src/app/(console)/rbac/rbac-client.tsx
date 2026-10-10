"use client";

import { AdminFilterBar } from "@/components/admin/admin-filter-bar";
import { FormPlusTableSkeleton } from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
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
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  useCreateRole,
  useDeleteRole,
  useInviteAdmin,
  usePatchRole,
  useRemoveAdmin,
} from "@/hooks/mutations/rbac";
import { useAdminNavChrome } from "@/hooks/queries/chrome";
import {
  useAdminAdmins,
  useAdminPermissions,
  useAdminRoleOptions,
  useAdminRoles,
} from "@/hooks/queries/rbac";
import { useAdminToken } from "@/hooks/use-admin-token";
import { useDebouncedValue } from "@/hooks/use-debounced-value";
import { useDefaultPageSize } from "@/hooks/use-default-page-size";
import { queryContentReady, useListBootReady } from "@/hooks/use-list-boot-ready";
import { useRequireStepUp } from "@/hooks/use-require-step-up";
import { chromeFieldDesc } from "@/lib/admin-field-desc";
import { joinChromeParts } from "@/lib/chrome-join";
import { useCallback, useEffect, useMemo, useState } from "react";

type RoleRow = {
  id: string;
  name: string;
  slug: string;
  permissions?: string[];
  is_system?: boolean;
  is_owner?: boolean;
};
type PermRow = { code: string; description?: string };
type AdminRow = {
  user_id: string;
  email: string;
  role_slug?: string;
  role_name?: string;
  status?: string;
  status_label?: string;
};

export function RbacClient({ initialToken }: { initialToken?: string }) {
  const token = useAdminToken(initialToken);
  const pageSize = useDefaultPageSize();
  const [roleSkip, setRoleSkip] = useState(0);
  const [adminSkip, setAdminSkip] = useState(0);
  const [roleQInput, setRoleQInput] = useState("");
  const [adminQInput, setAdminQInput] = useState("");
  const roleQ = useDebouncedValue(roleQInput.trim(), 250);
  const adminQ = useDebouncedValue(adminQInput.trim(), 250);
  const [roleSel, setRoleSel] = useState<RowSelectionState>({});
  const [adminSel, setAdminSel] = useState<RowSelectionState>({});
  const [bulkPending, setBulkPending] = useState(false);
  const chrome = useAdminNavChrome(token);
  const roles = useAdminRoles(token, roleSkip, pageSize, roleQ);
  const roleOptions = useAdminRoleOptions(token);
  const perms = useAdminPermissions(token);
  const admins = useAdminAdmins(token, adminSkip, pageSize, adminQ);
  const createRole = useCreateRole(token);
  const patchRole = usePatchRole(token);
  const deleteRole = useDeleteRole(token);
  const inviteAdmin = useInviteAdmin(token);
  const removeAdmin = useRemoveAdmin(token);

  const [email, setEmail] = useState("");
  const [roleId, setRoleId] = useState("");
  const [roleName, setRoleName] = useState("");
  const [roleSlug, setRoleSlug] = useState("");
  const [editRoleId, setEditRoleId] = useState("");
  const [editRoleSlug, setEditRoleSlug] = useState("");
  const [selectedPerms, setSelectedPerms] = useState<string[]>([]);
  const [roleDialogOpen, setRoleDialogOpen] = useState(false);
  const [inviteOpen, setInviteOpen] = useState(false);
  const [err, setErr] = useState("");
  const { err: stepErr, requireStepUp, setErr: setStepErr } = useRequireStepUp(token);

  const title = chrome.data?.ADMIN_NAV_RBAC?.trim() || "";
  const tabRoles = chrome.data?.ADMIN_RBAC_TAB_ROLES?.trim() || "";
  const tabPerms = chrome.data?.ADMIN_RBAC_TAB_PERMISSIONS?.trim() || "";
  const tabAdmins = chrome.data?.ADMIN_RBAC_TAB_ADMINS?.trim() || "";
  const createLabel = chrome.data?.ADMIN_RBAC_CREATE_ROLE?.trim() || "";
  const saveRoleLabel = chrome.data?.ADMIN_RBAC_SAVE_ROLE?.trim() || "";
  const selectRoleLabel = chrome.data?.ADMIN_RBAC_SELECT_ROLE?.trim() || "";
  const deleteRoleLabel = chrome.data?.ADMIN_RBAC_DELETE_ROLE?.trim() || "";
  const inviteLabel = chrome.data?.ADMIN_RBAC_INVITE_ADMIN?.trim() || "";
  const removeLabel = chrome.data?.ADMIN_ACTION_REMOVE?.trim() || "";
  const pendingCreating = chrome.data?.ADMIN_PENDING_CREATING?.trim() || "";
  const pendingSaving = chrome.data?.ADMIN_PENDING_SAVING?.trim() || "";
  const pendingUpdating = chrome.data?.ADMIN_PENDING_UPDATING?.trim() || pendingSaving || "";
  const pendingInvite =
    chrome.data?.ADMIN_PENDING_INVITING?.trim() || pendingCreating || pendingSaving || "";
  const pendingDelete = chrome.data?.ADMIN_PENDING_DELETING?.trim() || "";
  const createPendingLabel = pendingCreating || pendingSaving;
  const invitePendingLabel = pendingInvite;
  const bulkRemoveLabel = chrome.data?.ADMIN_TABLE_BULK_REMOVE?.trim() || "";
  const clearSelLabel = chrome.data?.ADMIN_TABLE_CLEAR_SELECTION?.trim() || "";
  const rowActionsLabel = chrome.data?.ADMIN_TABLE_ROW_ACTIONS?.trim() || "";
  const selectAllLabel = chrome.data?.ADMIN_TABLE_SELECT_ALL?.trim() || "";
  const selectRowLabel = chrome.data?.ADMIN_TABLE_SELECT_ROW?.trim() || "";
  const selectedFmt = chrome.data?.ADMIN_TABLE_SELECTED_FMT?.trim() || "";
  const colName = chrome.data?.ADMIN_RBAC_COL_NAME?.trim() || "";
  const filterSearch = chrome.data?.ADMIN_FILTER_SEARCH?.trim() || "";
  const filterSearchDesc = chrome.data?.ADMIN_FILTER_SEARCH_DESC?.trim() || "";
  const createSectionLabel = chrome.data?.ADMIN_SECTION_CREATE?.trim() || "";
  const editSectionLabel = chrome.data?.ADMIN_SECTION_EDIT?.trim() || "";
  const dangerSectionLabel = chrome.data?.ADMIN_SECTION_DANGER?.trim() || "";
  const closeLabel =
    chrome.data?.ADMIN_DIALOG_CLOSE?.trim() || chrome.data?.ADMIN_CLOSE?.trim() || "";

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setRoleSkip(0);
  }, [roleQ]);
  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setAdminSkip(0);
  }, [adminQ]);
  const colSlug = chrome.data?.ADMIN_RBAC_COL_SLUG?.trim() || "";
  const colPerms = chrome.data?.ADMIN_RBAC_COL_PERMS?.trim() || "";
  const colCode = chrome.data?.ADMIN_RBAC_COL_CODE?.trim() || "";
  const colDesc = chrome.data?.ADMIN_RBAC_COL_DESC?.trim() || "";
  const colEmail = chrome.data?.ADMIN_RBAC_COL_EMAIL?.trim() || "";
  const colRole = chrome.data?.ADMIN_RBAC_COL_ROLE?.trim() || "";
  const colStatus = chrome.data?.ADMIN_RBAC_COL_STATUS?.trim() || "";
  const sepComma = chrome.data?.ADMIN_UI_SEP_COMMA?.trim() || "";
  const selectionChrome =
    selectAllLabel && selectRowLabel
      ? {
          selectAllLabel,
          selectRowLabel,
          selectedCountFmt: selectedFmt || undefined,
        }
      : null;

  const withStepUp = useCallback(
    async (run: () => Promise<void>) => {
      setErr("");
      setStepErr("");
      if (!requireStepUp()) return;
      try {
        await run();
      } catch (e) {
        setErr(e instanceof Error ? e.message : "");
      }
    },
    [requireStepUp, setStepErr],
  );

  const clearRoleForm = useCallback(() => {
    setEditRoleId("");
    setEditRoleSlug("");
    setRoleName("");
    setRoleSlug("");
    setSelectedPerms([]);
  }, []);

  const openCreateRole = useCallback(() => {
    clearRoleForm();
    setRoleDialogOpen(true);
  }, [clearRoleForm]);

  const selectRoleForEdit = useCallback((row: RoleRow) => {
    setEditRoleId(row.id);
    setEditRoleSlug(row.slug);
    setSelectedPerms([...(row.permissions || [])]);
    setRoleDialogOpen(true);
  }, []);

  const closeRoleDialog = useCallback(() => {
    setRoleDialogOpen(false);
    clearRoleForm();
  }, [clearRoleForm]);

  const roleItems = (roles.data?.items || []) as RoleRow[];
  const adminItems = (admins.data?.items || []) as AdminRow[];

  const roleColumns = useMemo<ColumnDef<RoleRow>[]>(
    () => [
      {
        id: "name",
        header: colName,
        cell: ({ row }) => (
          <button
            type="button"
            className="font-medium text-foreground hover:underline"
            onClick={() => selectRoleForEdit(row.original)}
          >
            {row.original.name}
          </button>
        ),
      },
      {
        id: "slug",
        header: colSlug,
        cell: ({ row }) => <span className="font-mono text-xs">{row.original.slug}</span>,
      },
      {
        id: "permissions",
        header: colPerms,
        cell: ({ row }) => joinChromeParts(row.original.permissions || [], sepComma),
      },
      {
        id: "_actions",
        header: "",
        cell: ({ row }) => {
          if (row.original.is_system) return null;
          if (!deleteRoleLabel || !rowActionsLabel) return null;
          const id = row.original.id;
          return (
            <DataTableRowActions
              triggerLabel={rowActionsLabel}
              actions={[
                {
                  id: "delete",
                  label: deleteRoleLabel,
                  destructive: true,
                  isLoading: deleteRole.isPending,
                  pendingLabel: pendingDelete,
                  onSelect: () => {
                    void withStepUp(async () => {
                      await deleteRole.mutateAsync(id);
                      setRoleSel((prev) => {
                        const next = { ...prev };
                        delete next[id];
                        return next;
                      });
                      if (editRoleId === id) {
                        closeRoleDialog();
                      }
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
      colName,
      colSlug,
      colPerms,
      sepComma,
      deleteRoleLabel,
      rowActionsLabel,
      deleteRole,
      pendingDelete,
      editRoleId,
      withStepUp,
      selectRoleForEdit,
      closeRoleDialog,
    ],
  );

  const permColumns = useMemo<ColumnDef<PermRow>[]>(
    () => [
      {
        id: "code",
        header: colCode,
        cell: ({ row }) => <span className="font-mono text-xs">{row.original.code}</span>,
      },
      {
        id: "description",
        header: colDesc,
        cell: ({ row }) => row.original.description || "",
      },
    ],
    [colCode, colDesc],
  );

  const adminColumns = useMemo<ColumnDef<AdminRow>[]>(
    () => [
      {
        id: "email",
        header: colEmail,
        cell: ({ row }) => row.original.email,
      },
      {
        id: "role_name",
        header: colRole,
        cell: ({ row }) => row.original.role_name?.trim() || "",
      },
      {
        id: "status",
        header: colStatus,
        cell: ({ row }) => row.original.status_label || "",
      },
      {
        id: "_actions",
        header: "",
        cell: ({ row }) => {
          if (!removeLabel || !rowActionsLabel) return null;
          const userId = row.original.user_id;
          return (
            <DataTableRowActions
              triggerLabel={rowActionsLabel}
              actions={[
                {
                  id: "remove",
                  label: removeLabel,
                  destructive: true,
                  isLoading: removeAdmin.isPending,
                  pendingLabel: pendingDelete,
                  onSelect: () => {
                    void withStepUp(async () => {
                      await removeAdmin.mutateAsync(userId);
                      setAdminSel((prev) => {
                        const next = { ...prev };
                        delete next[userId];
                        return next;
                      });
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
      colEmail,
      colRole,
      colStatus,
      removeLabel,
      rowActionsLabel,
      removeAdmin,
      pendingDelete,
      withStepUp,
    ],
  );

  const pickPermColumns = useMemo<ColumnDef<PermRow>[]>(
    () => [
      {
        id: "_enabled",
        header: selectRowLabel || "",
        cell: ({ row }) => {
          const code = row.original.code;
          const on = selectedPerms.includes(code);
          return (
            <Checkbox
              checked={on}
              aria-label={selectRowLabel || code}
              onCheckedChange={() =>
                setSelectedPerms((prev) => (on ? prev.filter((x) => x !== code) : [...prev, code]))
              }
            />
          );
        },
      },
      {
        id: "code",
        header: colCode,
        cell: ({ row }) => (
          <span className="font-mono text-xs text-foreground">{row.original.code}</span>
        ),
      },
      {
        id: "description",
        header: colDesc,
        cell: ({ row }) => row.original.description || "",
      },
    ],
    [colCode, colDesc, selectedPerms, selectRowLabel],
  );

  async function bulkDeleteRoles() {
    const ids = getSelectedRowIds(roleSel);
    if (!ids.length || !requireStepUp()) return;
    setBulkPending(true);
    try {
      for (const id of ids) {
        const row = roleItems.find((r) => r.id === id);
        if (!row || row.is_system) continue;
        await deleteRole.mutateAsync(id);
        if (editRoleId === id) {
          closeRoleDialog();
        }
      }
      setRoleSel({});
    } finally {
      setBulkPending(false);
    }
  }

  async function bulkRemoveAdmins() {
    const ids = getSelectedRowIds(adminSel);
    if (!ids.length || !requireStepUp()) return;
    setBulkPending(true);
    try {
      for (const id of ids) {
        await removeAdmin.mutateAsync(id);
      }
      setAdminSel({});
    } finally {
      setBulkPending(false);
    }
  }

  // Gate on chrome + primary list only - do not OR every tab query (remounts filters).
  const [tab, setTab] = useState("roles");
  const bootReady = useListBootReady(pageSize, Boolean(chrome.data), queryContentReady(roles));
  const showInitialSkeleton = !bootReady;

  if (showInitialSkeleton) {
    const rbacTabs = [tabRoles, tabPerms, tabAdmins].filter(Boolean).length;
    return (
      <FormPlusTableSkeleton
        title={title || undefined}
        columns={[colName, colSlug, colPerms].map((label) => ({ label: label || "" }))}
        rows={pageSize > 0 ? pageSize : 8}
        formFields={0}
        toolbarButtons={1}
        showTabs
        tabCount={rbacTabs >= 2 ? rbacTabs : 3}
        showFilters
        filterCount={1}
        formFirst
      />
    );
  }

  return (
    <div className="space-y-4">
      {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
      {stepErr || err ? <p className="text-sm text-destructive">{stepErr || err}</p> : null}
      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          {tabRoles ? <TabsTrigger value="roles">{tabRoles}</TabsTrigger> : null}
          {tabPerms ? <TabsTrigger value="permissions">{tabPerms}</TabsTrigger> : null}
          {tabAdmins ? <TabsTrigger value="admins">{tabAdmins}</TabsTrigger> : null}
        </TabsList>
      </Tabs>

      <div className={tab === "roles" ? "space-y-4" : "hidden"}>
        {createLabel && createPendingLabel ? (
          <Button type="button" onClick={openCreateRole}>
            {createLabel}
          </Button>
        ) : null}
        <AdminFilterBar
          filters={[
            {
              kind: "search",
              id: "rbac-roles-search",
              label: filterSearch,
              description: filterSearchDesc,
              value: roleQInput,
              onChange: setRoleQInput,
            },
          ]}
        />
        <Card>
          <CardContent className="pt-6">
            <DataTable
              columns={roleColumns}
              data={roleItems}
              meta={roles.data?.meta}
              onPage={(s) => {
                setRoleSkip(s);
                setRoleSel({});
              }}
              pageDisabled={roles.isFetching}
              isFetching={roles.isFetching && !roles.isPending}
              getRowId={(row) => row.id}
              selection={
                selectionChrome
                  ? {
                      chrome: selectionChrome,
                      rowSelection: roleSel,
                      onRowSelectionChange: setRoleSel,
                      toolbar:
                        bulkRemoveLabel && pendingDelete ? (
                          <div className="flex flex-wrap items-center gap-2">
                            {dangerSectionLabel ? (
                              <p className="w-full text-sm font-medium text-foreground">
                                {dangerSectionLabel}
                              </p>
                            ) : null}
                            <Button
                              type="button"
                              size="sm"
                              variant="destructive"
                              isLoading={bulkPending}
                              pendingLabel={pendingDelete}
                              onClick={() => void bulkDeleteRoles()}
                            >
                              {bulkRemoveLabel}
                            </Button>
                            {clearSelLabel ? (
                              <Button
                                type="button"
                                size="sm"
                                variant="outline"
                                onClick={() => setRoleSel({})}
                              >
                                {clearSelLabel}
                              </Button>
                            ) : null}
                          </div>
                        ) : undefined,
                    }
                  : undefined
              }
              bordered={false}
            />
          </CardContent>
        </Card>
        <Dialog
          open={roleDialogOpen}
          onOpenChange={(open) => {
            if (!open) closeRoleDialog();
            else setRoleDialogOpen(true);
          }}
        >
          <DialogContent closeLabel={closeLabel} className="max-w-3xl">
            <DialogHeader>
              {editRoleId ? (
                editSectionLabel ? (
                  <DialogTitle>{editSectionLabel}</DialogTitle>
                ) : null
              ) : createSectionLabel ? (
                <DialogTitle>{createSectionLabel}</DialogTitle>
              ) : null}
            </DialogHeader>
            {editRoleId && selectRoleLabel ? (
              <p className="text-sm text-muted-foreground">
                {selectRoleLabel}
                {editRoleSlug ? (
                  <span className="ml-2 font-mono text-xs text-foreground">{editRoleSlug}</span>
                ) : null}
              </p>
            ) : null}
            <div className="grid gap-4 sm:grid-cols-2">
              {colName ? (
                <Field
                  id="rbac-role-name"
                  label={colName}
                  {...chromeFieldDesc(chrome.data, "ADMIN_RBAC_ROLE_NAME_DESC")}
                >
                  <Input
                    id="rbac-role-name"
                    value={roleName}
                    onChange={(e) => setRoleName(e.target.value)}
                    placeholder={colName}
                    autoComplete="off"
                  />
                </Field>
              ) : null}
              {colSlug ? (
                <Field
                  id="rbac-role-slug"
                  label={colSlug}
                  {...chromeFieldDesc(chrome.data, "ADMIN_RBAC_ROLE_SLUG_DESC")}
                >
                  <Input
                    id="rbac-role-slug"
                    value={roleSlug}
                    onChange={(e) => setRoleSlug(e.target.value)}
                    placeholder={colSlug}
                    className="font-mono text-xs"
                    autoComplete="off"
                  />
                </Field>
              ) : null}
              {colPerms ? (
                <Field
                  id="rbac-role-perms"
                  label={colPerms}
                  className="sm:col-span-2"
                  {...chromeFieldDesc(chrome.data, "ADMIN_RBAC_PERMS_DESC")}
                >
                  <DataTable
                    columns={pickPermColumns}
                    data={(perms.data?.items || []) as PermRow[]}
                    getRowId={(row) => row.code}
                    isFetching={perms.isFetching && !perms.isPending}
                    bordered={false}
                  />
                </Field>
              ) : (
                <div className="sm:col-span-2">
                  <DataTable
                    columns={pickPermColumns}
                    data={(perms.data?.items || []) as PermRow[]}
                    getRowId={(row) => row.code}
                    isFetching={perms.isFetching && !perms.isPending}
                    bordered={false}
                  />
                </div>
              )}
            </div>
            {(!editRoleId && createLabel && createPendingLabel) || (editRoleId && saveRoleLabel) ? (
              <DialogFooter>
                {!editRoleId && createLabel && createPendingLabel ? (
                  <Button
                    isLoading={createRole.isPending}
                    pendingLabel={createPendingLabel}
                    onClick={() =>
                      void withStepUp(async () => {
                        await createRole.mutateAsync({
                          name: roleName,
                          slug: roleSlug,
                          permissions: selectedPerms,
                        });
                        clearRoleForm();
                        setRoleDialogOpen(false);
                        setRoleSkip(0);
                      })
                    }
                  >
                    {createLabel}
                  </Button>
                ) : null}
                {editRoleId && saveRoleLabel ? (
                  <Button
                    variant="secondary"
                    isLoading={patchRole.isPending}
                    pendingLabel={pendingUpdating || saveRoleLabel}
                    onClick={() =>
                      void withStepUp(async () => {
                        await patchRole.mutateAsync({
                          id: editRoleId,
                          body: { permissions: selectedPerms },
                        });
                        closeRoleDialog();
                      })
                    }
                  >
                    {saveRoleLabel}
                  </Button>
                ) : null}
              </DialogFooter>
            ) : null}
          </DialogContent>
        </Dialog>
      </div>

      <div className={tab === "permissions" ? "space-y-4" : "hidden"}>
        <Card>
          <CardContent className="pt-6">
            <DataTable
              columns={permColumns}
              data={(perms.data?.items || []) as PermRow[]}
              getRowId={(row) => row.code}
              isFetching={perms.isFetching && !perms.isPending}
              bordered={false}
            />
          </CardContent>
        </Card>
      </div>

      <div className={tab === "admins" ? "space-y-4" : "hidden"}>
        {inviteLabel && invitePendingLabel ? (
          <Button type="button" onClick={() => setInviteOpen(true)}>
            {inviteLabel}
          </Button>
        ) : null}
        <AdminFilterBar
          filters={[
            {
              kind: "search",
              id: "rbac-admins-search",
              label: filterSearch,
              description: filterSearchDesc,
              value: adminQInput,
              onChange: setAdminQInput,
            },
          ]}
        />
        <Card>
          <CardContent className="pt-6">
            <DataTable
              columns={adminColumns}
              data={adminItems}
              meta={admins.data?.meta}
              onPage={(s) => {
                setAdminSkip(s);
                setAdminSel({});
              }}
              pageDisabled={admins.isFetching}
              isFetching={admins.isFetching && !admins.isPending}
              getRowId={(row) => row.user_id}
              selection={
                selectionChrome
                  ? {
                      chrome: selectionChrome,
                      rowSelection: adminSel,
                      onRowSelectionChange: setAdminSel,
                      toolbar:
                        bulkRemoveLabel && pendingDelete ? (
                          <div className="flex flex-wrap items-center gap-2">
                            {dangerSectionLabel ? (
                              <p className="w-full text-sm font-medium text-foreground">
                                {dangerSectionLabel}
                              </p>
                            ) : null}
                            <Button
                              type="button"
                              size="sm"
                              variant="destructive"
                              isLoading={bulkPending}
                              pendingLabel={pendingDelete}
                              onClick={() => void bulkRemoveAdmins()}
                            >
                              {bulkRemoveLabel}
                            </Button>
                            {clearSelLabel ? (
                              <Button
                                type="button"
                                size="sm"
                                variant="outline"
                                onClick={() => setAdminSel({})}
                              >
                                {clearSelLabel}
                              </Button>
                            ) : null}
                          </div>
                        ) : undefined,
                    }
                  : undefined
              }
              bordered={false}
            />
          </CardContent>
        </Card>
        {inviteLabel && invitePendingLabel ? (
          <Dialog
            open={inviteOpen}
            onOpenChange={(open) => {
              if (!open) setInviteOpen(false);
              else setInviteOpen(true);
            }}
          >
            <DialogContent closeLabel={closeLabel} className="max-w-2xl">
              <DialogHeader>
                {createSectionLabel ? <DialogTitle>{createSectionLabel}</DialogTitle> : null}
              </DialogHeader>
              <div className="grid gap-4 sm:grid-cols-2">
                {colEmail ? (
                  <Field
                    id="rbac-invite-email"
                    label={colEmail}
                    {...chromeFieldDesc(chrome.data, "ADMIN_RBAC_INVITE_EMAIL_DESC")}
                  >
                    <Input
                      id="rbac-invite-email"
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                      placeholder={colEmail}
                      autoComplete="off"
                    />
                  </Field>
                ) : null}
                {colRole ? (
                  <Field
                    id="rbac-invite-role"
                    label={colRole}
                    {...chromeFieldDesc(chrome.data, "ADMIN_RBAC_INVITE_ROLE_DESC")}
                  >
                    <Select value={roleId || undefined} onValueChange={setRoleId}>
                      <SelectTrigger id="rbac-invite-role" aria-label={colRole}>
                        <SelectValue placeholder={colRole} />
                      </SelectTrigger>
                      <SelectContent>
                        {(roleOptions.data?.items || []).map((r) => (
                          <SelectItem key={r.id} value={r.id}>
                            {r.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </Field>
                ) : null}
              </div>
              <DialogFooter>
                <Button
                  isLoading={inviteAdmin.isPending}
                  pendingLabel={invitePendingLabel}
                  disabled={!email.trim() || !roleId.trim()}
                  onClick={() =>
                    void withStepUp(async () => {
                      if (!email.trim() || !roleId.trim()) return;
                      await inviteAdmin.mutateAsync({
                        email: email.trim(),
                        role_id: roleId,
                      });
                      setEmail("");
                      setRoleId("");
                      setInviteOpen(false);
                      setAdminSkip(0);
                    })
                  }
                >
                  {inviteLabel}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        ) : null}
      </div>
    </div>
  );
}
