"use client";

import { AdminFilterBar } from "@/components/admin/admin-filter-bar";
import { FormPlusTableSkeleton } from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
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
import { Textarea } from "@/components/ui/textarea";
import {
  usePostBreakGlass,
  useResolveBreakGlass,
  useRevokeBreakGlass,
} from "@/hooks/mutations/break-glass";
import { useAdminBreakGlass } from "@/hooks/queries/break-glass";
import { useAdminNavChrome } from "@/hooks/queries/chrome";
import { useAdminToken } from "@/hooks/use-admin-token";
import { useDebouncedValue } from "@/hooks/use-debounced-value";
import { useDefaultPageSize } from "@/hooks/use-default-page-size";
import { queryContentReady, useListBootReady } from "@/hooks/use-list-boot-ready";
import { useRequireStepUp } from "@/hooks/use-require-step-up";
import { chromeFieldDesc } from "@/lib/admin-field-desc";
import { formatChromeField, joinChromeParts } from "@/lib/chrome-join";
import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";

type BreakGlassRow = {
  id: string;
  status: string;
  status_label?: string;
  elevates_permission?: string;
  reason?: string;
  requester_id?: string;
  approver_id?: string;
};

export function BreakGlassClient({ initialToken }: { initialToken?: string }) {
  const token = useAdminToken(initialToken);
  const pageSize = useDefaultPageSize();
  const searchParams = useSearchParams();
  const deepLinkId = (searchParams.get("id") || "").trim();
  const [skip, setSkip] = useState(0);
  const [qInput, setQInput] = useState("");
  const q = useDebouncedValue(qInput.trim(), 250);
  const [rowSel, setRowSel] = useState<RowSelectionState>({});
  const [bulkPending, setBulkPending] = useState(false);
  const chrome = useAdminNavChrome(token);
  const { err: stepErr, requireStepUp } = useRequireStepUp(token);
  const list = useAdminBreakGlass(token, skip, pageSize, q);
  const create = usePostBreakGlass(token);
  const resolve = useResolveBreakGlass(token);
  const revoke = useRevokeBreakGlass(token);
  const [reason, setReason] = useState("");
  const [permission, setPermission] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const title = chrome.data?.ADMIN_NAV_BREAK_GLASS?.trim() || "";
  const pendingCreating = chrome.data?.ADMIN_PENDING_CREATING?.trim() || "";
  const pendingSaving = chrome.data?.ADMIN_PENDING_SAVING?.trim() || "";
  const pendingDelete = chrome.data?.ADMIN_PENDING_DELETING?.trim() || "";
  const pendingBulkRevoke = pendingDelete || pendingSaving;
  const filterSearch = chrome.data?.ADMIN_FILTER_SEARCH?.trim() || "";
  const filterSearchDesc = chrome.data?.ADMIN_FILTER_SEARCH_DESC?.trim() || "";
  const reasonRequired = chrome.data?.ADMIN_REASON_REQUIRED?.trim() || "";
  const approveLabel = chrome.data?.ADMIN_BREAK_GLASS_APPROVE?.trim() || "";
  const denyLabel = chrome.data?.ADMIN_BREAK_GLASS_DENY?.trim() || "";
  const revokeLabel = chrome.data?.ADMIN_BREAK_GLASS_REVOKE?.trim() || "";
  const requesterLabel = chrome.data?.ADMIN_BREAK_GLASS_REQUESTER?.trim() || "";
  const approverLabel = chrome.data?.ADMIN_BREAK_GLASS_APPROVER?.trim() || "";
  const saveLabel = chrome.data?.ADMIN_ACTION_SAVE?.trim() || "";
  const createSectionLabel = chrome.data?.ADMIN_SECTION_CREATE?.trim() || "";
  const dangerSectionLabel = chrome.data?.ADMIN_SECTION_DANGER?.trim() || "";
  const colStatus = chrome.data?.ADMIN_BREAK_GLASS_COL_STATUS?.trim() || "";
  const colPerm = chrome.data?.ADMIN_BREAK_GLASS_COL_PERM?.trim() || "";
  const colReason = chrome.data?.ADMIN_BREAK_GLASS_COL_REASON?.trim() || "";
  const rowActionsLabel = chrome.data?.ADMIN_TABLE_ROW_ACTIONS?.trim() || "";
  const selectAllLabel = chrome.data?.ADMIN_TABLE_SELECT_ALL?.trim() || "";
  const selectRowLabel = chrome.data?.ADMIN_TABLE_SELECT_ROW?.trim() || "";
  const selectedFmt = chrome.data?.ADMIN_TABLE_SELECTED_FMT?.trim() || "";
  const clearSelLabel = chrome.data?.ADMIN_TABLE_CLEAR_SELECTION?.trim() || "";
  const sepDot = chrome.data?.ADMIN_UI_SEP_DOT?.trim() || "";
  const metaFieldFmt = chrome.data?.ADMIN_META_FIELD_FMT?.trim() || "";
  const closeLabel =
    chrome.data?.ADMIN_DIALOG_CLOSE?.trim() || chrome.data?.ADMIN_CLOSE?.trim() || "";
  const selectionChrome =
    selectAllLabel && selectRowLabel
      ? {
          selectAllLabel,
          selectRowLabel,
          selectedCountFmt: selectedFmt || undefined,
        }
      : null;

  const items = (list.data?.items ?? []) as BreakGlassRow[];

  // biome-ignore lint/correctness/useExhaustiveDependencies: re-run when list rows arrive for deep-link scroll
  useEffect(() => {
    if (!deepLinkId) return;
    const el = document.getElementById("break-glass-table");
    if (!el) return;
    el.scrollIntoView({ behavior: "smooth", block: "start" });
  }, [deepLinkId, items.length]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setSkip(0);
  }, [q]);

  const columns = useMemo<ColumnDef<BreakGlassRow>[]>(
    () => [
      {
        id: "status",
        header: colStatus,
        cell: ({ row }) => row.original.status_label || "",
      },
      {
        id: "elevates_permission",
        header: colPerm,
        cell: ({ row }) => (
          <span className="font-mono text-xs">{row.original.elevates_permission || ""}</span>
        ),
      },
      {
        id: "reason",
        header: colReason,
        cell: ({ row }) => (
          <div className="space-y-1">
            <div>{row.original.reason || ""}</div>
            <div className="font-mono text-xs text-muted-foreground">
              {joinChromeParts(
                [
                  formatChromeField(
                    metaFieldFmt,
                    requesterLabel,
                    String(row.original.requester_id || ""),
                  ),
                  formatChromeField(
                    metaFieldFmt,
                    approverLabel,
                    String(row.original.approver_id || ""),
                  ),
                ],
                sepDot,
              )}
            </div>
          </div>
        ),
      },
      {
        id: "_actions",
        header: "",
        cell: ({ row }) => {
          const item = row.original;
          if (!rowActionsLabel) return null;
          const actions = [];
          if (item.status === "pending") {
            if (approveLabel) {
              actions.push({
                id: "approve",
                label: approveLabel,
                isLoading: resolve.isPending,
                pendingLabel: pendingSaving,
                onSelect: () => {
                  if (!requireStepUp()) return;
                  void resolve.mutate({ id: item.id, approve: true });
                },
              });
            }
            if (denyLabel) {
              actions.push({
                id: "deny",
                label: denyLabel,
                isLoading: resolve.isPending,
                pendingLabel: pendingSaving,
                onSelect: () => {
                  if (!requireStepUp()) return;
                  void resolve.mutate({ id: item.id, approve: false });
                },
              });
            }
          }
          if (item.status === "approved" && revokeLabel) {
            actions.push({
              id: "revoke",
              label: revokeLabel,
              destructive: true,
              isLoading: revoke.isPending,
              pendingLabel: pendingDelete || pendingSaving,
              onSelect: () => {
                if (!requireStepUp()) return;
                void revoke.mutate(item.id, {
                  onSuccess: () =>
                    setRowSel((prev) => {
                      const next = { ...prev };
                      delete next[item.id];
                      return next;
                    }),
                });
              },
            });
          }
          if (actions.length === 0) return null;
          return <DataTableRowActions triggerLabel={rowActionsLabel} actions={actions} />;
        },
      },
    ],
    [
      colStatus,
      colPerm,
      colReason,
      requesterLabel,
      approverLabel,
      sepDot,
      metaFieldFmt,
      approveLabel,
      denyLabel,
      revokeLabel,
      rowActionsLabel,
      resolve,
      revoke,
      requireStepUp,
      pendingSaving,
      pendingDelete,
    ],
  );

  async function bulkRevokeApproved() {
    const ids = getSelectedRowIds(rowSel);
    if (!ids.length || !requireStepUp()) return;
    setBulkPending(true);
    try {
      for (const id of ids) {
        const row = items.find((r) => r.id === id);
        if (!row || row.status !== "approved") continue;
        await revoke.mutateAsync(id);
      }
      setRowSel({});
    } finally {
      setBulkPending(false);
    }
  }

  const bootReady = useListBootReady(pageSize, Boolean(chrome.data), queryContentReady(list));
  const showInitialSkeleton = !bootReady;

  if (showInitialSkeleton) {
    return (
      <FormPlusTableSkeleton
        title={title || undefined}
        columns={[colStatus, colPerm, colReason].map((label) => ({ label: label || "" }))}
        rows={pageSize > 0 ? pageSize : 8}
        formFields={0}
        toolbarButtons={saveLabel && pendingCreating ? 1 : 0}
        showFilters
        filterCount={1}
      />
    );
  }

  return (
    <div className="space-y-6">
      {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
      {stepErr ? <p className="text-sm text-destructive">{stepErr}</p> : null}
      {saveLabel && pendingCreating ? (
        <Button type="button" onClick={() => setCreateOpen(true)}>
          {saveLabel}
        </Button>
      ) : null}
      <AdminFilterBar
        filters={[
          {
            kind: "search",
            id: "break-glass-search",
            label: filterSearch,
            description: filterSearchDesc,
            value: qInput,
            onChange: setQInput,
          },
        ]}
      />
      <Card id="break-glass-table" className="scroll-mt-24">
        <CardContent className="pt-6">
          <DataTable
            columns={columns}
            data={items}
            meta={list.data?.meta}
            onPage={(s) => {
              setSkip(s);
              setRowSel({});
            }}
            pageDisabled={list.isFetching}
            isFetching={
              (list.isFetching && !list.isPending) ||
              create.isPending ||
              resolve.isPending ||
              revoke.isPending ||
              bulkPending
            }
            getRowId={(row) => row.id}
            bodyRowClassName={(row) =>
              deepLinkId && row.id === deepLinkId
                ? "bg-accent/50 ring-1 ring-inset ring-border"
                : undefined
            }
            selection={
              selectionChrome
                ? {
                    chrome: selectionChrome,
                    rowSelection: rowSel,
                    onRowSelectionChange: setRowSel,
                    toolbar:
                      revokeLabel && pendingBulkRevoke ? (
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
                            pendingLabel={pendingBulkRevoke}
                            onClick={() => void bulkRevokeApproved()}
                          >
                            {revokeLabel}
                          </Button>
                          {clearSelLabel ? (
                            <Button
                              type="button"
                              size="sm"
                              variant="outline"
                              onClick={() => setRowSel({})}
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
      {saveLabel && pendingCreating ? (
        <Dialog
          open={createOpen}
          onOpenChange={(open) => {
            if (!open) setCreateOpen(false);
            else setCreateOpen(true);
          }}
        >
          <DialogContent closeLabel={closeLabel} className="max-w-2xl">
            <DialogHeader>
              {createSectionLabel ? <DialogTitle>{createSectionLabel}</DialogTitle> : null}
            </DialogHeader>
            <div className="grid gap-4 sm:grid-cols-2">
              {colReason ? (
                <Field
                  id="break-glass-reason"
                  label={colReason}
                  {...chromeFieldDesc(chrome.data, "ADMIN_BREAK_GLASS_REASON_DESC")}
                >
                  <Textarea
                    id="break-glass-reason"
                    value={reason}
                    onChange={(e) => setReason(e.target.value)}
                    placeholder={reasonRequired}
                  />
                </Field>
              ) : null}
              {colPerm ? (
                <Field
                  id="break-glass-perm"
                  label={colPerm}
                  {...chromeFieldDesc(chrome.data, "ADMIN_BREAK_GLASS_PERM_DESC")}
                >
                  <Input
                    id="break-glass-perm"
                    value={permission}
                    onChange={(e) => setPermission(e.target.value)}
                    placeholder={colPerm}
                    className="font-mono text-xs"
                    autoComplete="off"
                  />
                </Field>
              ) : null}
            </div>
            <DialogFooter>
              <Button
                type="button"
                isLoading={create.isPending}
                pendingLabel={pendingCreating}
                onClick={() => {
                  if (!requireStepUp()) return;
                  void create.mutate(
                    {
                      reason,
                      elevates_permission: permission,
                    },
                    {
                      onSuccess: () => {
                        setCreateOpen(false);
                        setReason("");
                        setPermission("");
                      },
                    },
                  );
                }}
              >
                {saveLabel}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      ) : null}
    </div>
  );
}
