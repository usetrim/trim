"use client";

import { AdminFilterBar } from "@/components/admin/admin-filter-bar";
import { chromeFieldDesc } from "@/lib/admin-field-desc";
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
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  useAddAsnDenylist,
  useAddEmailDenylist,
  useAddIpDenylist,
  useRemoveAsnDenylist,
  useRemoveEmailDenylist,
  useRemoveIpDenylist,
} from "@/hooks/mutations/denylist";
import { useAdminNavChrome } from "@/hooks/queries/chrome";
import {
  useAdminDenylistASN,
  useAdminDenylistEmail,
  useAdminDenylistIP,
} from "@/hooks/queries/denylist";
import { useAdminToken } from "@/hooks/use-admin-token";
import { queryContentReady, useListBootReady } from "@/hooks/use-list-boot-ready";
import { useDebouncedValue } from "@/hooks/use-debounced-value";
import { useDefaultPageSize } from "@/hooks/use-default-page-size";
import { useRequireStepUp } from "@/hooks/use-require-step-up";
import { useEffect, useMemo, useState } from "react";

type Row = Record<string, unknown>;

export function DenylistClient({ initialToken }: { initialToken?: string }) {
  const token = useAdminToken(initialToken);
  const pageSize = useDefaultPageSize();
  const chrome = useAdminNavChrome(token);
  const { err: stepErr, requireStepUp } = useRequireStepUp(token);
  const [emailSkip, setEmailSkip] = useState(0);
  const [ipSkip, setIpSkip] = useState(0);
  const [asnSkip, setAsnSkip] = useState(0);
  const [emailQInput, setEmailQInput] = useState("");
  const [ipQInput, setIpQInput] = useState("");
  const [asnQInput, setAsnQInput] = useState("");
  const emailQ = useDebouncedValue(emailQInput.trim(), 250);
  const ipQ = useDebouncedValue(ipQInput.trim(), 250);
  const asnQ = useDebouncedValue(asnQInput.trim(), 250);
  const [emailSel, setEmailSel] = useState<RowSelectionState>({});
  const [ipSel, setIpSel] = useState<RowSelectionState>({});
  const [asnSel, setAsnSel] = useState<RowSelectionState>({});
  const [bulkPending, setBulkPending] = useState(false);
  const email = useAdminDenylistEmail(token, emailSkip, pageSize, emailQ);
  const ip = useAdminDenylistIP(token, ipSkip, pageSize, ipQ);
  const asn = useAdminDenylistASN(token, asnSkip, pageSize, asnQ);
  const addEmail = useAddEmailDenylist(token);
  const addIp = useAddIpDenylist(token);
  const addAsn = useAddAsnDenylist(token);
  const removeEmail = useRemoveEmailDenylist(token);
  const removeIp = useRemoveIpDenylist(token);
  const removeAsn = useRemoveAsnDenylist(token);
  const [domain, setDomain] = useState("");
  const [cidr, setCidr] = useState("");
  const [asnValue, setAsnValue] = useState("");
  const [reason, setReason] = useState("");
  const [emailCreateOpen, setEmailCreateOpen] = useState(false);
  const [ipCreateOpen, setIpCreateOpen] = useState(false);
  const [asnCreateOpen, setAsnCreateOpen] = useState(false);
  const pendingCreating = chrome.data?.ADMIN_PENDING_CREATING?.trim() || "";
  const pendingDelete = chrome.data?.ADMIN_PENDING_DELETING?.trim() || "";
  const reasonRequired = chrome.data?.ADMIN_REASON_REQUIRED?.trim() || "";
  const title = chrome.data?.ADMIN_NAV_DENYLIST?.trim() || "";
  const tabEmail = chrome.data?.ADMIN_DENYLIST_TAB_EMAIL?.trim() || "";
  const tabIp = chrome.data?.ADMIN_DENYLIST_TAB_IP?.trim() || "";
  const tabAsn = chrome.data?.ADMIN_DENYLIST_TAB_ASN?.trim() || "";
  const addLabel = chrome.data?.ADMIN_ACTION_ADD?.trim() || "";
  const removeLabel = chrome.data?.ADMIN_ACTION_REMOVE?.trim() || "";
  const bulkRemoveLabel = chrome.data?.ADMIN_TABLE_BULK_REMOVE?.trim() || "";
  const clearSelLabel = chrome.data?.ADMIN_TABLE_CLEAR_SELECTION?.trim() || "";
  const rowActionsLabel = chrome.data?.ADMIN_TABLE_ROW_ACTIONS?.trim() || "";
  const selectAllLabel = chrome.data?.ADMIN_TABLE_SELECT_ALL?.trim() || "";
  const selectRowLabel = chrome.data?.ADMIN_TABLE_SELECT_ROW?.trim() || "";
  const selectedFmt = chrome.data?.ADMIN_TABLE_SELECTED_FMT?.trim() || "";
  const colValue = chrome.data?.ADMIN_DENYLIST_COL_VALUE?.trim() || "";
  const colReason = chrome.data?.ADMIN_DENYLIST_COL_REASON?.trim() || "";
  const filterSearch = chrome.data?.ADMIN_FILTER_SEARCH?.trim() || "";
  const createSectionLabel = chrome.data?.ADMIN_SECTION_CREATE?.trim() || "";
  const dangerSectionLabel = chrome.data?.ADMIN_SECTION_DANGER?.trim() || "";
  const filterSearchDesc = chrome.data?.ADMIN_FILTER_SEARCH_DESC?.trim() || "";
  const closeLabel =
    chrome.data?.ADMIN_DIALOG_CLOSE?.trim() || chrome.data?.ADMIN_CLOSE?.trim() || "";
  const reasonOk = Boolean(reason.trim());

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setEmailSkip(0);
  }, [emailQ]);
  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setIpSkip(0);
  }, [ipQ]);
  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setAsnSkip(0);
  }, [asnQ]);

  const selectionChrome =
    selectAllLabel && selectRowLabel
      ? {
          selectAllLabel,
          selectRowLabel,
          selectedCountFmt: selectedFmt || undefined,
        }
      : null;

  const emailColumns = useMemo<ColumnDef<Row>[]>(
    () => [
      {
        id: "domain",
        header: colValue,
        cell: ({ row }) => String(row.original.domain || "").trim(),
      },
      {
        id: "reason",
        header: colReason,
        cell: ({ row }) => String(row.original.reason || "").trim(),
      },
      {
        id: "_actions",
        header: "",
        cell: ({ row }) => {
          const d = String(row.original.domain || "").trim();
          if (!d || !removeLabel) return null;
          return (
            <DataTableRowActions
              triggerLabel={rowActionsLabel}
              actions={[
                {
                  id: "remove",
                  label: removeLabel,
                  destructive: true,
                  isLoading: removeEmail.isPending,
                  pendingLabel: pendingDelete,
                  onSelect: () => {
                    if (!requireStepUp()) return;
                    void removeEmail.mutate(d, {
                      onSuccess: () =>
                        setEmailSel((prev) => {
                          const next = { ...prev };
                          delete next[d];
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
    [colValue, colReason, removeLabel, rowActionsLabel, removeEmail, requireStepUp, pendingDelete],
  );

  const ipColumns = useMemo<ColumnDef<Row>[]>(
    () => [
      {
        id: "cidr",
        header: colValue,
        cell: ({ row }) => String(row.original.cidr || "").trim(),
      },
      {
        id: "reason",
        header: colReason,
        cell: ({ row }) => String(row.original.reason || "").trim(),
      },
      {
        id: "_actions",
        header: "",
        cell: ({ row }) => {
          const c = String(row.original.cidr || "").trim();
          if (!c || !removeLabel) return null;
          return (
            <DataTableRowActions
              triggerLabel={rowActionsLabel}
              actions={[
                {
                  id: "remove",
                  label: removeLabel,
                  destructive: true,
                  isLoading: removeIp.isPending,
                  pendingLabel: pendingDelete,
                  onSelect: () => {
                    if (!requireStepUp()) return;
                    void removeIp.mutate(c, {
                      onSuccess: () =>
                        setIpSel((prev) => {
                          const next = { ...prev };
                          delete next[c];
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
    [colValue, colReason, removeLabel, rowActionsLabel, removeIp, requireStepUp, pendingDelete],
  );

  const asnColumns = useMemo<ColumnDef<Row>[]>(
    () => [
      {
        id: "asn",
        header: colValue,
        cell: ({ row }) => String(row.original.asn ?? "").trim(),
      },
      {
        id: "reason",
        header: colReason,
        cell: ({ row }) => String(row.original.reason || "").trim(),
      },
      {
        id: "_actions",
        header: "",
        cell: ({ row }) => {
          const a =
            typeof row.original.asn === "number" ? row.original.asn : Number(row.original.asn);
          if (!removeLabel || !Number.isFinite(a) || a <= 0) return null;
          const id = String(a);
          return (
            <DataTableRowActions
              triggerLabel={rowActionsLabel}
              actions={[
                {
                  id: "remove",
                  label: removeLabel,
                  destructive: true,
                  isLoading: removeAsn.isPending,
                  pendingLabel: pendingDelete,
                  onSelect: () => {
                    if (!requireStepUp()) return;
                    void removeAsn.mutate(a, {
                      onSuccess: () =>
                        setAsnSel((prev) => {
                          const next = { ...prev };
                          delete next[id];
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
    [colValue, colReason, removeLabel, rowActionsLabel, removeAsn, requireStepUp, pendingDelete],
  );

  async function bulkRemoveEmail() {
    const ids = getSelectedRowIds(emailSel);
    if (!ids.length || !requireStepUp()) return;
    setBulkPending(true);
    try {
      for (const id of ids) {
        await removeEmail.mutateAsync(id);
      }
      setEmailSel({});
    } finally {
      setBulkPending(false);
    }
  }

  async function bulkRemoveIp() {
    const ids = getSelectedRowIds(ipSel);
    if (!ids.length || !requireStepUp()) return;
    setBulkPending(true);
    try {
      for (const id of ids) {
        await removeIp.mutateAsync(id);
      }
      setIpSel({});
    } finally {
      setBulkPending(false);
    }
  }

  async function bulkRemoveAsn() {
    const ids = getSelectedRowIds(asnSel);
    if (!ids.length || !requireStepUp()) return;
    setBulkPending(true);
    try {
      for (const id of ids) {
        await removeAsn.mutateAsync(Number(id));
      }
      setAsnSel({});
    } finally {
      setBulkPending(false);
    }
  }

  const [tab, setTab] = useState("email");
  const primaryDenylist = tab === "ip" ? ip : tab === "asn" ? asn : email;

  const bootReady = useListBootReady(
    pageSize,
    Boolean(chrome.data),
    queryContentReady(primaryDenylist),
  );
  const showInitialSkeleton = !bootReady;

  if (showInitialSkeleton) {
    const denyTabs = [tabEmail, tabIp, tabAsn].filter(Boolean).length;
    return (
      <FormPlusTableSkeleton
        title={title || undefined}
        columns={[colValue, colReason].map((label) => ({ label: label || "" }))}
        rows={pageSize > 0 ? pageSize : 8}
        formFields={0}
        toolbarButtons={1}
        showTabs
        tabCount={denyTabs >= 2 ? denyTabs : 3}
        showFilters
        filterCount={1}
        filtersBeforeForm
      />
    );
  }

  return (
    <div className="space-y-4">
      {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
      {stepErr ? <p className="text-sm text-destructive">{stepErr}</p> : null}
      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          {tabEmail ? <TabsTrigger value="email">{tabEmail}</TabsTrigger> : null}
          {tabIp ? <TabsTrigger value="ip">{tabIp}</TabsTrigger> : null}
          {tabAsn ? <TabsTrigger value="asn">{tabAsn}</TabsTrigger> : null}
        </TabsList>
      </Tabs>

      <div className={tab === "email" ? "space-y-3" : "hidden"}>
        <AdminFilterBar
          filters={[
            {
              kind: "search",
              id: "denylist-email-search",
              label: filterSearch,
              description: filterSearchDesc,
              value: emailQInput,
              onChange: setEmailQInput,
            },
          ]}
        />
        {addLabel && pendingCreating && reasonRequired ? (
          <Button type="button" onClick={() => setEmailCreateOpen(true)}>
            {addLabel}
          </Button>
        ) : null}
        <Card>
          <CardContent className="pt-6">
            <DataTable
              columns={emailColumns}
              data={((email.data?.items as Row[]) ?? []).filter((item) =>
                Boolean(String(item.domain || "").trim()),
              )}
              meta={email.data?.meta}
              onPage={(s) => {
                setEmailSkip(s);
                setEmailSel({});
              }}
              pageDisabled={email.isFetching}
              isFetching={(email.isFetching && !email.isPending) || addEmail.isPending}
              getRowId={(row) => String(row.domain || "")}
              selection={
                selectionChrome
                  ? {
                      chrome: selectionChrome,
                      rowSelection: emailSel,
                      onRowSelectionChange: setEmailSel,
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
                              onClick={() => void bulkRemoveEmail()}
                            >
                              {bulkRemoveLabel}
                            </Button>
                            {clearSelLabel ? (
                              <Button
                                type="button"
                                size="sm"
                                variant="outline"
                                onClick={() => setEmailSel({})}
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
        {addLabel && pendingCreating && reasonRequired ? (
          <Dialog
            open={emailCreateOpen}
            onOpenChange={(open) => {
              if (!open) setEmailCreateOpen(false);
              else setEmailCreateOpen(true);
            }}
          >
            <DialogContent closeLabel={closeLabel} className="max-w-2xl">
              <DialogHeader>
                {createSectionLabel ? <DialogTitle>{createSectionLabel}</DialogTitle> : null}
              </DialogHeader>
              <div className="grid gap-4 sm:grid-cols-2">
                {colValue ? (
                  <Field
                    id="denylist-email-value"
                    label={colValue}
                    {...chromeFieldDesc(chrome.data, "ADMIN_DENYLIST_VALUE_DESC")}
                  >
                    <Input
                      id="denylist-email-value"
                      value={domain}
                      onChange={(e) => setDomain(e.target.value)}
                      placeholder={colValue}
                      autoComplete="off"
                    />
                  </Field>
                ) : null}
                {colReason ? (
                  <Field
                    id="denylist-email-reason"
                    label={colReason}
                    {...chromeFieldDesc(chrome.data, "ADMIN_DENYLIST_REASON_DESC")}
                  >
                    <Input
                      id="denylist-email-reason"
                      value={reason}
                      onChange={(e) => setReason(e.target.value)}
                      placeholder={reasonRequired}
                      autoComplete="off"
                    />
                  </Field>
                ) : null}
              </div>
              <DialogFooter>
                <Button
                  type="button"
                  isLoading={addEmail.isPending}
                  pendingLabel={pendingCreating}
                  disabled={!domain.trim() || !reasonOk}
                  onClick={() => {
                    if (!requireStepUp()) return;
                    void addEmail.mutate(
                      { domain, reason: reason.trim() },
                      {
                        onSuccess: () => {
                          setDomain("");
                          setReason("");
                          setEmailSkip(0);
                          setEmailCreateOpen(false);
                        },
                      },
                    );
                  }}
                >
                  {addLabel}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        ) : null}
      </div>

      <div className={tab === "ip" ? "space-y-3" : "hidden"}>
        <AdminFilterBar
          filters={[
            {
              kind: "search",
              id: "denylist-ip-search",
              label: filterSearch,
              description: filterSearchDesc,
              value: ipQInput,
              onChange: setIpQInput,
            },
          ]}
        />
        {addLabel && pendingCreating && reasonRequired ? (
          <Button type="button" onClick={() => setIpCreateOpen(true)}>
            {addLabel}
          </Button>
        ) : null}
        <Card>
          <CardContent className="pt-6">
            <DataTable
              columns={ipColumns}
              data={((ip.data?.items as Row[]) ?? []).filter((item) =>
                Boolean(String(item.cidr || "").trim()),
              )}
              meta={ip.data?.meta}
              onPage={(s) => {
                setIpSkip(s);
                setIpSel({});
              }}
              pageDisabled={ip.isFetching}
              isFetching={(ip.isFetching && !ip.isPending) || addIp.isPending}
              getRowId={(row) => String(row.cidr || "")}
              selection={
                selectionChrome
                  ? {
                      chrome: selectionChrome,
                      rowSelection: ipSel,
                      onRowSelectionChange: setIpSel,
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
                              onClick={() => void bulkRemoveIp()}
                            >
                              {bulkRemoveLabel}
                            </Button>
                            {clearSelLabel ? (
                              <Button
                                type="button"
                                size="sm"
                                variant="outline"
                                onClick={() => setIpSel({})}
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
        {addLabel && pendingCreating && reasonRequired ? (
          <Dialog
            open={ipCreateOpen}
            onOpenChange={(open) => {
              if (!open) setIpCreateOpen(false);
              else setIpCreateOpen(true);
            }}
          >
            <DialogContent closeLabel={closeLabel} className="max-w-2xl">
              <DialogHeader>
                {createSectionLabel ? <DialogTitle>{createSectionLabel}</DialogTitle> : null}
              </DialogHeader>
              <div className="grid gap-4 sm:grid-cols-2">
                {colValue ? (
                  <Field
                    id="denylist-ip-value"
                    label={colValue}
                    {...chromeFieldDesc(chrome.data, "ADMIN_DENYLIST_VALUE_DESC")}
                  >
                    <Input
                      id="denylist-ip-value"
                      value={cidr}
                      onChange={(e) => setCidr(e.target.value)}
                      placeholder={colValue}
                      autoComplete="off"
                    />
                  </Field>
                ) : null}
                {colReason ? (
                  <Field
                    id="denylist-ip-reason"
                    label={colReason}
                    {...chromeFieldDesc(chrome.data, "ADMIN_DENYLIST_REASON_DESC")}
                  >
                    <Input
                      id="denylist-ip-reason"
                      value={reason}
                      onChange={(e) => setReason(e.target.value)}
                      placeholder={reasonRequired}
                      autoComplete="off"
                    />
                  </Field>
                ) : null}
              </div>
              <DialogFooter>
                <Button
                  type="button"
                  isLoading={addIp.isPending}
                  pendingLabel={pendingCreating}
                  disabled={!cidr.trim() || !reasonOk}
                  onClick={() => {
                    if (!requireStepUp()) return;
                    void addIp.mutate(
                      { cidr, reason: reason.trim() },
                      {
                        onSuccess: () => {
                          setCidr("");
                          setReason("");
                          setIpSkip(0);
                          setIpCreateOpen(false);
                        },
                      },
                    );
                  }}
                >
                  {addLabel}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        ) : null}
      </div>

      <div className={tab === "asn" ? "space-y-3" : "hidden"}>
        <AdminFilterBar
          filters={[
            {
              kind: "search",
              id: "denylist-asn-search",
              label: filterSearch,
              description: filterSearchDesc,
              value: asnQInput,
              onChange: setAsnQInput,
            },
          ]}
        />
        {addLabel && pendingCreating && reasonRequired ? (
          <Button type="button" onClick={() => setAsnCreateOpen(true)}>
            {addLabel}
          </Button>
        ) : null}
        <Card>
          <CardContent className="pt-6">
            <DataTable
              columns={asnColumns}
              data={((asn.data?.items as Row[]) ?? []).filter((item) => {
                const a = typeof item.asn === "number" ? item.asn : Number(item.asn);
                return Number.isFinite(a) && a > 0;
              })}
              meta={asn.data?.meta}
              onPage={(s) => {
                setAsnSkip(s);
                setAsnSel({});
              }}
              pageDisabled={asn.isFetching}
              isFetching={(asn.isFetching && !asn.isPending) || addAsn.isPending}
              getRowId={(row) => String(row.asn || "")}
              selection={
                selectionChrome
                  ? {
                      chrome: selectionChrome,
                      rowSelection: asnSel,
                      onRowSelectionChange: setAsnSel,
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
                              onClick={() => void bulkRemoveAsn()}
                            >
                              {bulkRemoveLabel}
                            </Button>
                            {clearSelLabel ? (
                              <Button
                                type="button"
                                size="sm"
                                variant="outline"
                                onClick={() => setAsnSel({})}
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
        {addLabel && pendingCreating && reasonRequired ? (
          <Dialog
            open={asnCreateOpen}
            onOpenChange={(open) => {
              if (!open) setAsnCreateOpen(false);
              else setAsnCreateOpen(true);
            }}
          >
            <DialogContent closeLabel={closeLabel} className="max-w-2xl">
              <DialogHeader>
                {createSectionLabel ? <DialogTitle>{createSectionLabel}</DialogTitle> : null}
              </DialogHeader>
              <div className="grid gap-4 sm:grid-cols-2">
                {colValue ? (
                  <Field
                    id="denylist-asn-value"
                    label={colValue}
                    {...chromeFieldDesc(chrome.data, "ADMIN_DENYLIST_VALUE_DESC")}
                  >
                    <Input
                      id="denylist-asn-value"
                      value={asnValue}
                      onChange={(e) => setAsnValue(e.target.value)}
                      placeholder={colValue}
                      autoComplete="off"
                    />
                  </Field>
                ) : null}
                {colReason ? (
                  <Field
                    id="denylist-asn-reason"
                    label={colReason}
                    {...chromeFieldDesc(chrome.data, "ADMIN_DENYLIST_REASON_DESC")}
                  >
                    <Input
                      id="denylist-asn-reason"
                      value={reason}
                      onChange={(e) => setReason(e.target.value)}
                      placeholder={reasonRequired}
                      autoComplete="off"
                    />
                  </Field>
                ) : null}
              </div>
              <DialogFooter>
                <Button
                  type="button"
                  isLoading={addAsn.isPending}
                  pendingLabel={pendingCreating}
                  disabled={!Number(asnValue) || !reasonOk}
                  onClick={() => {
                    if (!requireStepUp()) return;
                    void addAsn.mutate(
                      { asn: Number(asnValue), reason: reason.trim() },
                      {
                        onSuccess: () => {
                          setAsnValue("");
                          setReason("");
                          setAsnSkip(0);
                          setAsnCreateOpen(false);
                        },
                      },
                    );
                  }}
                >
                  {addLabel}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        ) : null}
      </div>
    </div>
  );
}
