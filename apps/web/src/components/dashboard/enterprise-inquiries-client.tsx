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
import { useCheckoutSession } from "@/hooks/mutations/billing";
import { useBulkDeleteEnterpriseInquiries } from "@/hooks/mutations/me";
import { useAuthProviders } from "@/hooks/queries/auth";
import { useSubscriptionStatus } from "@/hooks/queries/billing";
import { useMeEnterpriseInquiries } from "@/hooks/queries/enterprise";
import { usePlans } from "@/hooks/queries/plans";
import { useDebouncedValue } from "@/hooks/use-debounced-value";
import { skeletonPageRows, useDefaultPageSize } from "@/hooks/use-default-page-size";
import { useDeferredDialogValue } from "@/hooks/use-deferred-dialog-selection";
import { enterpriseListSkeletonProps } from "@/lib/dedicated-list-skeletons";
import { formatDateTimeFull, formatDateTimeShort } from "@/lib/format-datetime";
import { ensurePaddleSuccessNavigation, openTrimPaddleCheckout } from "@/lib/paddle-checkout";
import { invalidateBilling, refreshBillingUntilSettled } from "@/lib/query-keys";
import { toastApiError } from "@/lib/toast-api";
import type { BillingInterval } from "@/stores/billing-ui";
import { useBillingUI } from "@/stores/billing-ui";
import type { MeEnterpriseInquiry } from "@/types/enterprise";
import { type Paddle, initializePaddle } from "@paddle/paddle-js";
import { useQueryClient } from "@tanstack/react-query";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";

export function EnterpriseInquiriesClient({
  accessToken,
  userId,
  email,
}: {
  accessToken?: string;
  userId?: string;
  email?: string;
}) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const focusId = (searchParams.get("id") || "").trim();
  const deepLinkOpenedFor = useRef<string | null>(null);
  const qc = useQueryClient();
  const [skip, setSkip] = useState(0);
  const [statusFilter, setStatusFilter] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const search = useDebouncedValue(searchInput.trim(), 250);
  const [payingId, setPayingId] = useState("");
  const [paddle, setPaddle] = useState<Paddle | null>(null);
  const [rowSel, setRowSel] = useState<RowSelectionState>({});
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);
  const [bulkDeleteConfirm, setBulkDeleteConfirm] = useState(false);
  const interval = useBillingUI((s) => s.interval);
  const setInterval = useBillingUI((s) => s.setInterval);
  const authProviders = useAuthProviders();
  const subscription = useSubscriptionStatus(accessToken);
  const bulkDelete = useBulkDeleteEnterpriseInquiries(accessToken);
  const dialogCancel = authProviders.data?.site?.dialog_cancel?.trim() || "";
  const htmlLang =
    subscription.data?.money_locale?.trim() || authProviders.data?.site?.html_lang?.trim() || "";
  const authPageSize = useDefaultPageSize();
  const pageSize =
    (subscription.data?.default_page_size && subscription.data.default_page_size > 0
      ? subscription.data.default_page_size
      : 0) || authPageSize;
  const skeletonRows = skeletonPageRows(pageSize);
  const inquiries = useMeEnterpriseInquiries(
    accessToken,
    skip,
    pageSize > 0 ? pageSize : undefined,
    statusFilter || undefined,
    search,
  );
  const plans = usePlans(interval, accessToken);
  const checkout = useCheckoutSession(accessToken);
  const chrome = inquiries.data?.chrome;
  const settings = plans.data?.settings;
  const pageChromeProps = enterpriseListSkeletonProps(authProviders.data?.site, skeletonRows, {
    title: chrome?.title,
    eyebrow: subscription.data?.page_eyebrow,
    colCompany: chrome?.col_company,
    colStatus: chrome?.col_status,
    colRequested: chrome?.col_requested,
    colOffered: chrome?.col_offered,
    colCreated: chrome?.col_created,
    filterStatus: chrome?.filter_status,
    filterStatusDesc: chrome?.filter_status_desc,
    search: chrome?.search_placeholder,
    searchDesc: chrome?.search_description,
  });

  const enterpriseSkeleton = <DedicatedListPageSkeleton {...pageChromeProps} />;
  const {
    open: detailOpen,
    value: selectedInquiry,
    openWith: openDetail,
    onOpenChange: onDetailOpenChangeBase,
  } = useDeferredDialogValue<MeEnterpriseInquiry>();

  const onDetailOpenChange = useCallback(
    (next: boolean) => {
      onDetailOpenChangeBase(next);
      // Clear ?id= on dismiss so the modal stays closed and re-clicks can open again.
      if (!next && focusId) {
        router.replace(pathname, { scroll: false });
      }
    },
    [onDetailOpenChangeBase, focusId, router, pathname],
  );

  useEffect(() => {
    const def = settings?.default_plan_interval?.trim();
    if ((def === "monthly" || def === "annual") && !interval) {
      setInterval(def);
    }
  }, [settings?.default_plan_interval, interval, setInterval]);

  useEffect(() => {
    const token = process.env.NEXT_PUBLIC_PADDLE_CLIENT_TOKEN;
    const env = process.env.NEXT_PUBLIC_PADDLE_ENV;
    if (!token || !env) return;
    void initializePaddle({
      environment: env === "production" ? "production" : "sandbox",
      token,
      eventCallback: (event) => {
        if (event.name === "checkout.completed") {
          void refreshBillingUntilSettled(qc);
          ensurePaddleSuccessNavigation();
          return;
        }
        if (event.name === "checkout.closed") {
          void invalidateBilling(qc);
        }
      },
    }).then((instance) => {
      if (instance) setPaddle(instance);
    });
  }, [qc]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: reset page/selection on filters
  useEffect(() => {
    setSkip(0);
    setRowSel({});
  }, [search, statusFilter]);

  const statusOptions = useMemo(() => {
    return (
      [
        ["", chrome?.filter_all],
        ["new", chrome?.status_new],
        ["contacted", chrome?.status_contacted],
        ["offered", chrome?.status_offered],
        ["closed", chrome?.status_closed],
        ["activated", chrome?.status_activated],
      ] as const
    ).flatMap(([value, label]) => {
      const trimmed = label?.trim() || "";
      return trimmed ? [{ value, label: trimmed }] : [];
    });
  }, [chrome]);

  const payInquiry = useCallback(
    async (row: MeEnterpriseInquiry) => {
      const planId = inquiries.data?.plan_id?.trim() || "";
      const checkoutInterval = (interval || settings?.default_plan_interval?.trim() || "") as
        | BillingInterval
        | "";
      if (!accessToken || !userId || !email) {
        if (settings?.auth_required_plan_message) {
          toast.error(settings.auth_required_plan_message);
        }
        return;
      }
      if (!planId) {
        toast.error(settings?.checkout_unavailable_message || "");
        return;
      }
      if (checkoutInterval !== "monthly" && checkoutInterval !== "annual") {
        toast.error(
          chrome?.checkout_interval_required || settings?.checkout_unavailable_message || "",
        );
        return;
      }
      try {
        setPayingId(row.id);
        const session = await checkout.mutateAsync({
          plan_id: planId,
          interval: checkoutInterval,
          quantity: row.offered_seat_quantity,
          user_id: userId,
          email,
          inquiry_id: row.id,
          confirm: false,
        });
        if (
          session.action === "blocked" ||
          session.action === "same_plan" ||
          session.action === "same"
        ) {
          toast.error(session.message || settings?.change_blocked_default_message || "");
          return;
        }
        if (session.action !== "new_checkout" || !session.price_id) {
          toast.error(session.message || settings?.checkout_unavailable_message || "");
          return;
        }
        if (!paddle) {
          toast.error(session.message || settings?.paddle_js_not_ready_message || "");
          return;
        }
        openTrimPaddleCheckout({
          paddle,
          priceId: session.price_id,
          quantity: session.quantity,
          email,
          customData: session.custom_data ?? {},
          discountId: session.discount_id,
          discountCode: session.discount_code,
          planId: session.plan_id || planId,
        });
      } catch (e) {
        toastApiError(
          e instanceof Error ? e : new Error(settings?.checkout_request_failed_message || ""),
        );
      } finally {
        setPayingId("");
      }
    },
    [
      accessToken,
      checkout,
      chrome?.checkout_interval_required,
      email,
      inquiries.data?.plan_id,
      interval,
      paddle,
      settings,
      userId,
    ],
  );

  const viewLabel = chrome?.view_label?.trim() || "";
  const detailTitle = chrome?.details_label?.trim() || viewLabel;
  const deleteAction = chrome?.delete_action_label?.trim() || "";
  const deletePending = chrome?.delete_pending_label?.trim() || "";
  const deleteConfirm = chrome?.delete_confirm_message?.trim() || "";
  const bulkDeleteConfirmMsg = chrome?.bulk_delete_confirm_message?.trim() || "";
  const tableBulkDelete = chrome?.table_bulk_delete?.trim() || "";
  const tableClearSelection = chrome?.table_clear_selection?.trim() || "";
  const tableRowActions = chrome?.table_row_actions?.trim() || "";
  const selectAll = chrome?.table_select_all?.trim() || "";
  const selectRow = chrome?.table_select_row?.trim() || "";
  const selectedFmt = chrome?.table_selected_fmt?.trim() || "";
  const selectionChrome =
    selectAll && selectRow
      ? {
          selectAllLabel: selectAll,
          selectRowLabel: selectRow,
          selectedCountFmt: selectedFmt || undefined,
        }
      : null;

  const openInquiryDetail = useCallback(
    (row: MeEnterpriseInquiry) => {
      if (!viewLabel && !detailTitle) return;
      openDetail(row);
    },
    [detailTitle, openDetail, viewLabel],
  );

  const canDeleteRow = useCallback((row: MeEnterpriseInquiry) => {
    return (row.status || "").trim().toLowerCase() !== "activated";
  }, []);

  const columns = useMemo<ColumnDef<MeEnterpriseInquiry>[]>(() => {
    const cols: ColumnDef<MeEnterpriseInquiry>[] = [];
    if (chrome?.col_company) {
      cols.push({
        id: "company",
        header: chrome.col_company,
        cell: ({ row }) => row.original.company_name?.trim() || "",
      });
    }
    if (chrome?.col_status) {
      cols.push({
        id: "status",
        header: chrome.col_status,
        cell: ({ row }) => row.original.status_label?.trim() || row.original.status || "",
      });
    }
    if (chrome?.col_requested) {
      cols.push({
        id: "requested",
        header: chrome.col_requested,
        cell: ({ row }) =>
          row.original.estimated_seats && row.original.estimated_seats > 0
            ? String(row.original.estimated_seats)
            : "",
      });
    }
    if (chrome?.col_offered) {
      cols.push({
        id: "offered",
        header: chrome.col_offered,
        cell: ({ row }) =>
          row.original.offered_seat_quantity && row.original.offered_seat_quantity > 0
            ? String(row.original.offered_seat_quantity)
            : "",
      });
    }
    if (chrome?.col_created) {
      cols.push({
        id: "created",
        header: chrome.col_created,
        cell: ({ row }) => (
          <span title={formatDateTimeFull(row.original.created_at, htmlLang) || undefined}>
            {formatDateTimeShort(row.original.created_at, htmlLang)}
          </span>
        ),
      });
    }
    if (viewLabel) {
      cols.push({
        id: "view",
        header: "",
        cell: ({ row }) => (
          <button
            type="button"
            className="text-left text-[var(--trim-fg)] underline-offset-4 hover:underline"
            onClick={() => openInquiryDetail(row.original)}
          >
            {viewLabel}
          </button>
        ),
      });
    }
    if (chrome?.pay_label) {
      cols.push({
        id: "_pay",
        header: "",
        cell: ({ row }) => {
          if (!row.original.can_checkout) return null;
          return (
            <Button
              type="button"
              size="sm"
              isLoading={checkout.isPending && payingId === row.original.id}
              pendingLabel={chrome.pay_pending_label || chrome.pay_label}
              onClick={() => {
                void payInquiry(row.original);
              }}
            >
              {chrome.pay_label}
            </Button>
          );
        },
      });
    }
    if (tableRowActions) {
      cols.push({
        id: "_actions",
        header: "",
        cell: ({ row }) => {
          const id = row.original.id?.trim() || "";
          if (!id) return null;
          const actions = [];
          if (viewLabel || detailTitle) {
            actions.push({
              id: "view",
              label: viewLabel || detailTitle,
              onSelect: () => openInquiryDetail(row.original),
            });
          }
          if (row.original.can_checkout && chrome?.pay_label) {
            actions.push({
              id: "pay",
              label: chrome.pay_label,
              onSelect: () => {
                void payInquiry(row.original);
              },
            });
          }
          if (deleteAction && deleteConfirm && dialogCancel && canDeleteRow(row.original)) {
            actions.push({
              id: "delete",
              label: deleteAction,
              destructive: true,
              onSelect: () => setDeleteConfirmId(id),
            });
          }
          if (!actions.length) return null;
          return <DataTableRowActions triggerLabel={tableRowActions} actions={actions} />;
        },
      });
    }
    return cols;
  }, [
    chrome,
    checkout.isPending,
    payingId,
    payInquiry,
    openInquiryDetail,
    viewLabel,
    detailTitle,
    tableRowActions,
    deleteAction,
    deleteConfirm,
    dialogCancel,
    canDeleteRow,
    htmlLang,
  ]);

  const items = inquiries.data?.items ?? [];
  const selectedIds = getSelectedRowIds(rowSel);
  const deletableSelectedIds = selectedIds.filter((id) => {
    const row = items.find((item) => item.id === id);
    return row ? canDeleteRow(row) : false;
  });
  const showBulkDelete = Boolean(
    tableBulkDelete && bulkDeleteConfirmMsg && deletePending && dialogCancel,
  );

  useEffect(() => {
    if (!focusId) {
      deepLinkOpenedFor.current = null;
      return;
    }
    if (deepLinkOpenedFor.current === focusId) return;
    const row = items.find((item) => item.id === focusId);
    if (!row) return;
    deepLinkOpenedFor.current = focusId;
    if (viewLabel || detailTitle) {
      openDetail(row);
    }
  }, [focusId, items, openDetail, viewLabel, detailTitle]);

  if (!accessToken) {
    return enterpriseSkeleton;
  }

  // Full page shimmer until list + title chrome are ready.
  if (inquiries.isPending && !inquiries.data) {
    return enterpriseSkeleton;
  }

  if (!chrome) {
    if (subscription.isPending || authProviders.isPending) {
      return enterpriseSkeleton;
    }
    return null;
  }

  const pageTitle = chrome.title?.trim() || pageChromeProps.title.trim() || "";
  if (!pageTitle) {
    if (subscription.isPending || authProviders.isPending) {
      return enterpriseSkeleton;
    }
    return null;
  }

  const showEmpty = !inquiries.isPending && items.length === 0;

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
            {chrome.filter_status && statusOptions.length > 1 ? (
              <Field
                id="enterprise-status-filter"
                label={chrome.filter_status}
                description={chrome.filter_status_desc}
                className="w-full sm:w-auto sm:min-w-[12rem]"
              >
                <Select
                  value={statusFilter || "__all__"}
                  onValueChange={(value) => {
                    setStatusFilter(value === "__all__" ? "" : value);
                  }}
                >
                  <SelectTrigger id="enterprise-status-filter" aria-label={chrome.filter_status}>
                    <SelectValue placeholder={chrome.filter_status} />
                  </SelectTrigger>
                  <SelectContent>
                    {statusOptions.map((o) => (
                      <SelectItem key={o.value || "all"} value={o.value || "__all__"}>
                        {o.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>
            ) : null}
          </div>
          {chrome.search_placeholder?.trim() ? (
            <Field
              id="enterprise-search"
              label={chrome.search_placeholder}
              description={chrome.search_description}
              className="w-full sm:max-w-md"
            >
              <Input
                id="enterprise-search"
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                placeholder={chrome.search_placeholder}
                autoComplete="off"
              />
            </Field>
          ) : null}
        </CardHeader>
        <CardContent>
          {showEmpty ? (
            <p className="text-sm text-[var(--trim-muted)]">{chrome.empty || ""}</p>
          ) : (
            <DataTable
              columns={columns}
              data={items}
              meta={inquiries.data?.meta}
              onPage={
                inquiries.data?.meta
                  ? (s) => {
                      setSkip(s);
                      setRowSel({});
                    }
                  : undefined
              }
              pageDisabled={inquiries.isFetching}
              pageInputId="enterprise-inquiries-skip-to"
              isFetching={inquiries.isFetching && !inquiries.isPending}
              getRowId={(row) => row.id}
              bordered={false}
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
                            disabled={!deletableSelectedIds.length || bulkDelete.isPending}
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
          )}
        </CardContent>
      </Card>

      <Dialog open={detailOpen} onOpenChange={onDetailOpenChange}>
        <DialogContent closeLabel={dialogCancel} className="max-w-lg">
          <DialogHeader>
            {detailTitle ? <DialogTitle>{detailTitle}</DialogTitle> : null}
          </DialogHeader>
          <div className="grid gap-3">
            {chrome.col_company ? (
              <Field id="inquiry-detail-company" label={chrome.col_company}>
                <Input
                  id="inquiry-detail-company"
                  value={selectedInquiry?.company_name?.trim() || ""}
                  readOnly
                />
              </Field>
            ) : null}
            {chrome.col_status ? (
              <Field id="inquiry-detail-status" label={chrome.col_status}>
                <Input
                  id="inquiry-detail-status"
                  value={
                    selectedInquiry?.status_label?.trim() || selectedInquiry?.status?.trim() || ""
                  }
                  readOnly
                />
              </Field>
            ) : null}
            {chrome.col_requested ? (
              <Field id="inquiry-detail-requested" label={chrome.col_requested}>
                <Input
                  id="inquiry-detail-requested"
                  value={
                    selectedInquiry?.estimated_seats && selectedInquiry.estimated_seats > 0
                      ? String(selectedInquiry.estimated_seats)
                      : ""
                  }
                  readOnly
                />
              </Field>
            ) : null}
            {chrome.col_offered ? (
              <Field id="inquiry-detail-offered" label={chrome.col_offered}>
                <Input
                  id="inquiry-detail-offered"
                  value={
                    selectedInquiry?.offered_seat_quantity &&
                    selectedInquiry.offered_seat_quantity > 0
                      ? String(selectedInquiry.offered_seat_quantity)
                      : ""
                  }
                  readOnly
                />
              </Field>
            ) : null}
            {chrome.col_message ? (
              <Field id="inquiry-detail-message" label={chrome.col_message}>
                <Input
                  id="inquiry-detail-message"
                  value={selectedInquiry?.message?.trim() || ""}
                  readOnly
                />
              </Field>
            ) : null}
            {chrome.col_created ? (
              <Field id="inquiry-detail-created" label={chrome.col_created}>
                <Input
                  id="inquiry-detail-created"
                  value={
                    formatDateTimeFull(selectedInquiry?.created_at, htmlLang) ||
                    formatDateTimeShort(selectedInquiry?.created_at, htmlLang)
                  }
                  readOnly
                />
              </Field>
            ) : null}
          </div>
          {selectedInquiry?.can_checkout && chrome.pay_label ? (
            <DialogFooter>
              <Button
                type="button"
                isLoading={checkout.isPending && payingId === selectedInquiry.id}
                pendingLabel={chrome.pay_pending_label || chrome.pay_label}
                onClick={() => {
                  void payInquiry(selectedInquiry);
                }}
              >
                {chrome.pay_label}
              </Button>
            </DialogFooter>
          ) : null}
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
          if (!deletableSelectedIds.length) {
            setBulkDeleteConfirm(false);
            return;
          }
          bulkDelete.mutate(deletableSelectedIds, {
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
