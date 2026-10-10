"use client";

import { DedicatedListPageSkeleton } from "@/components/skeletons/page-skeletons";
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
import { DateRangePicker, type DateRangeValue } from "@/components/ui/date-range-picker";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { useBulkDeleteEvents } from "@/hooks/mutations/me";
import { useAuthProviders } from "@/hooks/queries/auth";
import { useSubscriptionStatus } from "@/hooks/queries/billing";
import { useEvents } from "@/hooks/queries/me";
import { useDebouncedValue } from "@/hooks/use-debounced-value";
import { skeletonPageRows, useDefaultPageSize } from "@/hooks/use-default-page-size";
import { useDeferredDialogValue } from "@/hooks/use-deferred-dialog-selection";
import { tracesListSkeletonProps } from "@/lib/dedicated-list-skeletons";
import { formatDateTimeFull, formatDateTimeShort } from "@/lib/format-datetime";
import { createClient } from "@/lib/supabase/client";
import { toastApiError } from "@/lib/toast-api";
import type { EventListItem } from "@/types/events";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";

export function TracesClient({ accessToken }: { accessToken?: string }) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const focusId = (searchParams.get("id") || "").trim();
  const deepLinkOpenedFor = useRef<string | null>(null);
  const [skip, setSkip] = useState(0);
  const [searchInput, setSearchInput] = useState("");
  const search = useDebouncedValue(searchInput.trim(), 250);
  const [dateRange, setDateRange] = useState<DateRangeValue>({ from: null, to: null });
  const [rowSel, setRowSel] = useState<RowSelectionState>({});
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);
  const [bulkDeleteConfirm, setBulkDeleteConfirm] = useState(false);
  const authProviders = useAuthProviders();
  const subscription = useSubscriptionStatus(accessToken);
  const bulkDelete = useBulkDeleteEvents(accessToken);
  const dialogCancel = authProviders.data?.site?.dialog_cancel?.trim() || "";
  const htmlLang =
    subscription.data?.money_locale?.trim() || authProviders.data?.site?.html_lang?.trim() || "";
  const authPageSize = useDefaultPageSize();
  const pageSize =
    (subscription.data?.default_page_size && subscription.data.default_page_size > 0
      ? subscription.data.default_page_size
      : 0) || authPageSize;
  const skeletonRows = skeletonPageRows(pageSize);
  const events = useEvents(
    accessToken,
    skip,
    pageSize > 0 ? pageSize : undefined,
    dateRange.from,
    dateRange.to,
    search,
  );
  const {
    open: eventDialogOpen,
    value: selectedEvent,
    openWith: openEventDialog,
    onOpenChange: onEventOpenChangeBase,
  } = useDeferredDialogValue<EventListItem>();

  const onEventOpenChange = useCallback(
    (next: boolean) => {
      onEventOpenChangeBase(next);
      if (!next && focusId) {
        router.replace(pathname, { scroll: false });
      }
    },
    [onEventOpenChangeBase, focusId, router, pathname],
  );

  const pageChromeProps = tracesListSkeletonProps(authProviders.data?.site, skeletonRows, {
    title: subscription.data?.traces_title,
    eyebrow: subscription.data?.page_eyebrow,
    colWhen: subscription.data?.traces_col_when,
    colModel: subscription.data?.traces_col_model,
    colMode: subscription.data?.traces_col_mode,
    colTokens: subscription.data?.traces_col_tokens,
    colLatency: subscription.data?.traces_col_latency,
  });

  const tracesSkeleton = <DedicatedListPageSkeleton {...pageChromeProps} />;

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setSkip(0);
    setRowSel({});
  }, [search, dateRange.from, dateRange.to]);

  type EventRow = NonNullable<typeof events.data>["items"][number];

  const openEvent = useCallback(
    (row: EventRow) => {
      const id = row.id?.trim() || "";
      if (!id) return;
      openEventDialog(row);
    },
    [openEventDialog],
  );

  useEffect(() => {
    if (!focusId) {
      deepLinkOpenedFor.current = null;
      return;
    }
    // Allow a new id (or replay after URL clear→set) to open again.
    if (deepLinkOpenedFor.current === focusId) return;
    const row = (events.data?.items ?? []).find((item) => item.id === focusId);
    if (!row) return;
    deepLinkOpenedFor.current = focusId;
    openEventDialog(row);
  }, [focusId, events.data?.items, openEventDialog]);

  const selectAll = events.data?.table_select_all?.trim() || "";
  const selectRow = events.data?.table_select_row?.trim() || "";
  const selectedFmt = events.data?.table_selected_fmt?.trim() || "";
  const tableBulkDelete = events.data?.table_bulk_delete?.trim() || "";
  const tableClearSelection = events.data?.table_clear_selection?.trim() || "";
  const deleteAction = events.data?.delete_action_label?.trim() || "";
  const deletePending = events.data?.delete_pending_label?.trim() || "";
  const deleteConfirm = events.data?.delete_confirm_message?.trim() || "";
  const bulkDeleteConfirmMsg = events.data?.bulk_delete_confirm_message?.trim() || "";
  const selectionChrome =
    selectAll && selectRow
      ? {
          selectAllLabel: selectAll,
          selectRowLabel: selectRow,
          selectedCountFmt: selectedFmt || undefined,
        }
      : null;
  const selectedIds = getSelectedRowIds(rowSel);
  const showBulkDelete = Boolean(
    tableBulkDelete && bulkDeleteConfirmMsg && deletePending && dialogCancel,
  );

  const eventColumns = useMemo<ColumnDef<EventRow>[]>(
    () => [
      {
        id: "when",
        header: events.data?.col_when || subscription.data?.traces_col_when || "",
        cell: ({ row }) => (
          <span
            className="text-[var(--trim-muted)]"
            title={formatDateTimeFull(row.original.created_at, htmlLang) || undefined}
          >
            {formatDateTimeShort(row.original.created_at, htmlLang)}
          </span>
        ),
      },
      {
        id: "model",
        header: events.data?.col_model || subscription.data?.traces_col_model || "",
        cell: ({ row }) => (
          <span className="text-[var(--trim-fg)]">{row.original.model || ""}</span>
        ),
      },
      {
        id: "mode",
        header: events.data?.col_mode || subscription.data?.traces_col_mode || "",
        cell: ({ row }) => row.original.mode_label || "",
      },
      {
        id: "tokens",
        header: events.data?.col_tokens || subscription.data?.traces_col_tokens || "",
        cell: ({ row }) => (
          <span className="text-[var(--trim-fg)]">
            {row.original.tokens_before}
            {events.data?.tokens_sep || ""}
            {row.original.tokens_after}
          </span>
        ),
      },
      {
        id: "latency",
        header: events.data?.col_latency || subscription.data?.traces_col_latency || "",
        cell: ({ row }) => (
          <span className="text-[var(--trim-fg)]">
            {row.original.latency_ms}
            {events.data?.latency_unit || ""}
          </span>
        ),
      },
      {
        id: "view",
        header: events.data?.col_view || "",
        cell: ({ row }) => {
          const openLabel = events.data?.open_action_label?.trim() || "";
          if (!openLabel) return "";
          return (
            <button
              type="button"
              className="text-left text-[var(--trim-fg)] underline-offset-4 hover:underline"
              onClick={() => openEvent(row.original)}
            >
              {openLabel}
            </button>
          );
        },
      },
      {
        id: "_actions",
        header: "",
        cell: ({ row }) => {
          const id = row.original.id?.trim() || "";
          const openLabel = events.data?.open_action_label?.trim() || "";
          const rowActions = events.data?.table_row_actions?.trim() || "";
          if (!id || !rowActions) return null;
          const actions = [];
          if (openLabel) {
            actions.push({
              id: "view",
              label: openLabel,
              onSelect: () => openEvent(row.original),
            });
          }
          if (deleteAction && deleteConfirm && dialogCancel) {
            actions.push({
              id: "delete",
              label: deleteAction,
              destructive: true,
              onSelect: () => setDeleteConfirmId(id),
            });
          }
          if (!actions.length) return null;
          return <DataTableRowActions triggerLabel={rowActions} actions={actions} />;
        },
      },
    ],
    [
      events.data,
      subscription.data,
      openEvent,
      deleteAction,
      deleteConfirm,
      dialogCancel,
      htmlLang,
    ],
  );

  const loginPath = (
    process.env.NEXT_PUBLIC_APP_PATH_LOGIN ||
    authProviders.data?.site?.path_login ||
    ""
  ).trim();

  if (!accessToken) {
    return tracesSkeleton;
  }

  if (subscription.isError && !subscription.data) {
    const status = (subscription.error as { status?: number } | null)?.status;
    if (status === 401 || status === 403) {
      return tracesSkeleton;
    }
    const errMsg =
      (subscription.error as Error)?.message?.trim() ||
      authProviders.data?.login_failed_message?.trim() ||
      "";
    return (
      <div className="mx-auto flex min-h-[40vh] w-full flex-col items-center justify-center gap-4 px-6 py-16">
        {errMsg ? (
          <p className="max-w-md text-center text-sm text-[var(--trim-muted)]">{errMsg}</p>
        ) : null}
        {loginPath ? (
          <Button
            variant="outline"
            onClick={() => {
              void createClient()
                .auth.signOut()
                .finally(() => router.replace(loginPath));
            }}
          >
            {authProviders.data?.site?.nav_sign_in || ""}
          </Button>
        ) : null}
      </div>
    );
  }

  // Full page shimmer until list + title chrome are ready (Enterprise parity).
  if (events.isPending && !events.data) {
    return tracesSkeleton;
  }
  const pageTitle = subscription.data?.traces_title?.trim() || pageChromeProps.title.trim() || "";
  if (!pageTitle) {
    if (subscription.isPending || authProviders.isPending) {
      return tracesSkeleton;
    }
    return null;
  }

  return (
    <div className="w-full space-y-6">
      <div className="min-w-0">
        {subscription.data?.page_eyebrow ? (
          <p className="text-sm text-[var(--trim-muted)]">{subscription.data.page_eyebrow}</p>
        ) : null}
        <h1 className="text-2xl font-semibold tracking-tight text-[var(--trim-fg)] sm:text-3xl">
          {pageTitle}
        </h1>
      </div>

      <Card className="w-full">
        <CardHeader className="flex flex-col gap-3">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <CardTitle className="text-base">{pageTitle}</CardTitle>
            {events.data?.date_range_placeholder &&
            events.data?.date_range_clear &&
            events.data?.date_range_apply &&
            events.data?.date_range_months &&
            events.data.date_range_months >= 1 ? (
              <Field
                id="traces-date-range"
                label={events.data.date_range_placeholder}
                description={events.data.date_range_description}
                className="w-full sm:w-auto"
              >
                <DateRangePicker
                  id="traces-date-range"
                  className="w-full sm:w-auto"
                  value={dateRange}
                  placeholder={events.data.date_range_placeholder}
                  clearLabel={events.data.date_range_clear}
                  applyLabel={events.data.date_range_apply}
                  locale={subscription.data?.money_locale || ""}
                  numberOfMonths={events.data.date_range_months}
                  onChange={setDateRange}
                />
              </Field>
            ) : null}
          </div>
          {events.data?.search_placeholder?.trim() ? (
            <Field
              id="traces-search"
              label={events.data.search_placeholder}
              description={events.data.search_description}
              className="w-full sm:max-w-md"
            >
              <Input
                id="traces-search"
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                placeholder={events.data.search_placeholder}
                autoComplete="off"
              />
            </Field>
          ) : null}
        </CardHeader>
        <CardContent>
          <DataTable
            columns={eventColumns}
            data={events.data?.items ?? []}
            bordered={false}
            getRowId={(row) => row.id}
            meta={events.data?.meta}
            onPage={
              events.data?.meta
                ? (s) => {
                    setSkip(s);
                    setRowSel({});
                  }
                : undefined
            }
            pageDisabled={events.isFetching}
            pageInputId="traces-skip-to"
            isFetching={events.isFetching && !events.isPending}
            bodyRowClassName={(row) =>
              focusId && row.id === focusId
                ? "bg-[var(--trim-hover)] ring-1 ring-inset ring-[var(--trim-border-strong)]"
                : undefined
            }
            selection={
              selectionChrome
                ? {
                    chrome: selectionChrome,
                    rowSelection: rowSel,
                    onRowSelectionChange: setRowSel,
                    toolbar: showBulkDelete ? (
                      <div className="flex flex-wrap gap-2">
                        <Button
                          type="button"
                          size="sm"
                          variant="destructive"
                          disabled={!selectedIds.length || bulkDelete.isPending}
                          onClick={() => setBulkDeleteConfirm(true)}
                        >
                          {tableBulkDelete}
                        </Button>
                        {tableClearSelection ? (
                          <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            onClick={() => setRowSel({})}
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
          {!events.data?.items.length ? (
            <p className="py-6 text-sm text-[var(--trim-muted)]">
              {events.data?.empty_message || ""}
            </p>
          ) : null}
        </CardContent>
      </Card>

      <Dialog open={eventDialogOpen} onOpenChange={onEventOpenChange}>
        <DialogContent closeLabel={dialogCancel} className="max-w-lg">
          <DialogHeader>
            {events.data?.open_action_label ? (
              <DialogTitle>{events.data.open_action_label}</DialogTitle>
            ) : null}
            {selectedEvent?.model?.trim() || selectedEvent?.request_id?.trim() ? (
              <DialogDescription>
                {selectedEvent.model?.trim() || selectedEvent.request_id?.trim() || ""}
              </DialogDescription>
            ) : null}
          </DialogHeader>
          <div className="grid gap-3">
            {events.data?.col_when || subscription.data?.traces_col_when ? (
              <Field
                id="event-preview-when"
                label={events.data?.col_when || subscription.data?.traces_col_when || ""}
                description={events.data?.preview_field_description}
              >
                <Input
                  id="event-preview-when"
                  value={
                    formatDateTimeFull(selectedEvent?.created_at, htmlLang) ||
                    formatDateTimeShort(selectedEvent?.created_at, htmlLang)
                  }
                  placeholder={events.data?.col_when || subscription.data?.traces_col_when || ""}
                  readOnly
                />
              </Field>
            ) : null}
            {events.data?.col_model || subscription.data?.traces_col_model ? (
              <Field
                id="event-preview-model"
                label={events.data?.col_model || subscription.data?.traces_col_model || ""}
                description={events.data?.preview_field_description}
              >
                <Input
                  id="event-preview-model"
                  value={selectedEvent?.model?.trim() || ""}
                  placeholder={events.data?.col_model || subscription.data?.traces_col_model || ""}
                  readOnly
                />
              </Field>
            ) : null}
            {events.data?.col_mode || subscription.data?.traces_col_mode ? (
              <Field
                id="event-preview-mode"
                label={events.data?.col_mode || subscription.data?.traces_col_mode || ""}
                description={events.data?.preview_field_description}
              >
                <Input
                  id="event-preview-mode"
                  value={selectedEvent?.mode_label?.trim() || selectedEvent?.mode?.trim() || ""}
                  placeholder={events.data?.col_mode || subscription.data?.traces_col_mode || ""}
                  readOnly
                />
              </Field>
            ) : null}
            {events.data?.col_tokens || subscription.data?.traces_col_tokens ? (
              <Field
                id="event-preview-tokens"
                label={events.data?.col_tokens || subscription.data?.traces_col_tokens || ""}
                description={events.data?.preview_field_description}
              >
                <Input
                  id="event-preview-tokens"
                  value={
                    selectedEvent
                      ? `${selectedEvent.tokens_before}${events.data?.tokens_sep || ""}${
                          selectedEvent.tokens_after
                        }`
                      : ""
                  }
                  placeholder={
                    events.data?.col_tokens || subscription.data?.traces_col_tokens || ""
                  }
                  readOnly
                />
              </Field>
            ) : null}
            {events.data?.col_latency || subscription.data?.traces_col_latency ? (
              <Field
                id="event-preview-latency"
                label={events.data?.col_latency || subscription.data?.traces_col_latency || ""}
                description={events.data?.preview_field_description}
              >
                <Input
                  id="event-preview-latency"
                  value={
                    selectedEvent
                      ? `${selectedEvent.latency_ms}${events.data?.latency_unit || ""}`
                      : ""
                  }
                  placeholder={
                    events.data?.col_latency || subscription.data?.traces_col_latency || ""
                  }
                  readOnly
                />
              </Field>
            ) : null}
            {events.data?.col_status ? (
              <Field
                id="event-preview-status"
                label={events.data.col_status}
                description={events.data.preview_field_description}
              >
                <Input
                  id="event-preview-status"
                  value={selectedEvent?.status_label?.trim() || selectedEvent?.status?.trim() || ""}
                  placeholder={events.data.col_status}
                  readOnly
                />
              </Field>
            ) : null}
            {events.data?.col_request_id ? (
              <Field
                id="event-preview-request"
                label={events.data.col_request_id}
                description={events.data.preview_field_description}
              >
                <Input
                  id="event-preview-request"
                  value={selectedEvent?.request_id?.trim() || selectedEvent?.id?.trim() || ""}
                  placeholder={events.data.col_request_id}
                  readOnly
                />
              </Field>
            ) : null}
          </div>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={Boolean(deleteConfirmId)}
        onOpenChange={(open) => {
          if (!open && !bulkDelete.isPending) setDeleteConfirmId(null);
        }}
        description={deleteConfirm}
        cancelLabel={dialogCancel}
        confirmLabel={deleteAction}
        pendingLabel={deletePending || undefined}
        isPending={bulkDelete.isPending}
        destructive
        onConfirm={() => {
          if (!deleteConfirmId) return;
          bulkDelete.mutate([deleteConfirmId], {
            onSuccess: (res) => {
              setDeleteConfirmId(null);
              setRowSel((prev) => {
                const next = { ...prev };
                delete next[deleteConfirmId];
                return next;
              });
              if (res.message?.trim()) toast.success(res.message);
            },
            onError: (e) => toastApiError(e instanceof Error ? e : new Error("")),
          });
        }}
      />

      <ConfirmDialog
        open={bulkDeleteConfirm}
        onOpenChange={(open) => {
          if (!open && !bulkDelete.isPending) setBulkDeleteConfirm(false);
        }}
        description={bulkDeleteConfirmMsg}
        cancelLabel={dialogCancel}
        confirmLabel={tableBulkDelete || deleteAction}
        pendingLabel={deletePending || undefined}
        isPending={bulkDelete.isPending}
        destructive
        onConfirm={() => {
          if (!selectedIds.length) {
            setBulkDeleteConfirm(false);
            return;
          }
          bulkDelete.mutate(selectedIds, {
            onSuccess: (res) => {
              setBulkDeleteConfirm(false);
              setRowSel({});
              if (res.message?.trim()) toast.success(res.message);
            },
            onError: (e) => toastApiError(e instanceof Error ? e : new Error("")),
          });
        }}
      />
    </div>
  );
}
