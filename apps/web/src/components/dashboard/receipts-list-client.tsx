"use client";

import { DedicatedListPageSkeleton } from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { type ColumnDef, DataTable, type RowSelectionState } from "@/components/ui/data-table";
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
import { useDownloadReceiptPdf, useSyncReceipts } from "@/hooks/mutations/billing";
import { useAuthProviders } from "@/hooks/queries/auth";
import { useReceipts, useSubscriptionStatus } from "@/hooks/queries/billing";
import { toast } from "sonner";
import { useDebouncedValue } from "@/hooks/use-debounced-value";
import { skeletonPageRows, useDefaultPageSize } from "@/hooks/use-default-page-size";
import { useDeferredDialogValue } from "@/hooks/use-deferred-dialog-selection";
import { receiptsListSkeletonProps } from "@/lib/dedicated-list-skeletons";
import { formatDateShort, formatDateTimeFull } from "@/lib/format-datetime";
import { createClient } from "@/lib/supabase/client";
import { formatMoney } from "@/lib/utils";
import type { ReceiptSummary } from "@/types/billing";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

function receiptWhenRaw(row: Pick<ReceiptSummary, "paid_at" | "created_at">): string {
  return (row.paid_at || row.created_at || "").trim();
}

export function ReceiptsListClient({ accessToken }: { accessToken?: string }) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const focusId = (searchParams.get("id") || "").trim();
  const deepLinkOpenedFor = useRef<string | null>(null);
  const [skip, setSkip] = useState(0);
  const [searchInput, setSearchInput] = useState("");
  const search = useDebouncedValue(searchInput.trim(), 250);
  const [rowSel, setRowSel] = useState<RowSelectionState>({});
  const [actionNotice, setActionNotice] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const authProviders = useAuthProviders();
  const subscription = useSubscriptionStatus(accessToken);
  const syncReceipts = useSyncReceipts(accessToken);
  const downloadPdf = useDownloadReceiptPdf(accessToken);
  const dialogCancel = authProviders.data?.site?.dialog_cancel?.trim() || "";
  const authPageSize = useDefaultPageSize();
  const pageSize =
    (subscription.data?.default_page_size && subscription.data.default_page_size > 0
      ? subscription.data.default_page_size
      : 0) || authPageSize;
  const skeletonRows = skeletonPageRows(pageSize);
  const receipts = useReceipts(accessToken, skip, pageSize > 0 ? pageSize : undefined, search);
  const htmlLang =
    receipts.data?.money_locale?.trim() ||
    subscription.data?.money_locale?.trim() ||
    authProviders.data?.site?.html_lang?.trim() ||
    "";
  const {
    open: receiptOpen,
    value: selectedReceipt,
    openWith: openReceiptDialog,
    close: closeReceiptDialog,
    onOpenChange: onReceiptOpenChangeBase,
  } = useDeferredDialogValue<ReceiptSummary>();

  const onReceiptOpenChange = useCallback(
    (next: boolean) => {
      onReceiptOpenChangeBase(next);
      if (!next && focusId) {
        router.replace(pathname, { scroll: false });
      }
    },
    [onReceiptOpenChangeBase, focusId, router, pathname],
  );

  const pageChromeProps = receiptsListSkeletonProps(authProviders.data?.site, skeletonRows, {
    title: subscription.data?.receipts_title,
    eyebrow: subscription.data?.page_eyebrow,
    colDate: subscription.data?.receipts_col_date,
    colInvoice: subscription.data?.receipts_col_invoice,
    colStatus: subscription.data?.receipts_col_status,
    colTotal: subscription.data?.receipts_col_total,
    colView: subscription.data?.receipts_col_view,
    syncLabel: subscription.data?.receipts_sync_action_label,
  });

  const receiptsSkeleton = <DedicatedListPageSkeleton {...pageChromeProps} />;

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setSkip(0);
    setRowSel({});
  }, [search]);

  const selectAll = receipts.data?.table_select_all?.trim() || "";
  const selectRow = receipts.data?.table_select_row?.trim() || "";
  const selectedFmt = receipts.data?.table_selected_fmt?.trim() || "";
  const tableClearSelection = receipts.data?.table_clear_selection?.trim() || "";
  const selectionChrome =
    selectAll && selectRow
      ? {
          selectAllLabel: selectAll,
          selectRowLabel: selectRow,
          selectedCountFmt: selectedFmt || undefined,
        }
      : null;

  type ReceiptRow = NonNullable<typeof receipts.data>["items"][number];

  const openReceipt = useCallback(
    (row: ReceiptRow) => {
      const id = row.id?.trim() || "";
      if (!id) return;
      // Prefer full tax-invoice page (print / PDF) over the summary dialog.
      const prefix = (
        subscription.data?.path_receipts_prefix ||
        authProviders.data?.site?.path_receipts_prefix ||
        ""
      ).trim();
      if (prefix) {
        router.push(`${prefix}${id}`);
        return;
      }
      // Last resort if path chrome is missing: keep summary dialog + footer CTA.
      openReceiptDialog(row);
    },
    [
      authProviders.data?.site?.path_receipts_prefix,
      openReceiptDialog,
      router,
      subscription.data?.path_receipts_prefix,
    ],
  );

  const openReceiptPdf = useCallback(
    (row: ReceiptRow) => {
      const id = row.id?.trim() || "";
      if (!id || !accessToken) return;
      const failed = receipts.data?.download_pdf_failed_message || "";
      const done = receipts.data?.download_pdf_done_message || "";
      downloadPdf.mutate(
        {
          href: `/api/v1/billing/receipts/${encodeURIComponent(id)}/pdf`,
          failedMessage: failed,
          filenameFmt: receipts.data?.download_pdf_filename_fmt,
          displayId: row.display_id || row.paddle_invoice_number || id,
        },
        {
          onSuccess: () => {
            if (done) toast.success(done);
          },
          onError: (e) => {
            toast.error(e.message || failed);
          },
        },
      );
    },
    [
      accessToken,
      downloadPdf,
      receipts.data?.download_pdf_done_message,
      receipts.data?.download_pdf_failed_message,
      receipts.data?.download_pdf_filename_fmt,
    ],
  );

  useEffect(() => {
    if (!focusId) {
      deepLinkOpenedFor.current = null;
      return;
    }
    if (deepLinkOpenedFor.current === focusId) return;
    const row = (receipts.data?.items ?? []).find((item) => item.id === focusId);
    if (!row) return;
    deepLinkOpenedFor.current = focusId;
    openReceipt(row);
  }, [focusId, receipts.data?.items, openReceipt]);

  const receiptColumns = useMemo<ColumnDef<ReceiptRow>[]>(
    () => [
      {
        id: "date",
        header: receipts.data?.col_date || subscription.data?.receipts_col_date || "",
        cell: ({ row }) => {
          const when = receiptWhenRaw(row.original);
          return (
            <span
              className="text-[var(--trim-muted)]"
              title={formatDateTimeFull(when, htmlLang) || undefined}
            >
              {formatDateShort(when, htmlLang)}
            </span>
          );
        },
      },
      {
        id: "invoice",
        header: receipts.data?.col_invoice || subscription.data?.receipts_col_invoice || "",
        cell: ({ row }) => (
          <span className="font-mono text-xs text-[var(--trim-muted)]">
            {row.original.display_id || ""}
          </span>
        ),
      },
      {
        id: "status",
        header: receipts.data?.col_status || subscription.data?.receipts_col_status || "",
        cell: ({ row }) => (
          <span className="text-[var(--trim-fg)]">{row.original.status_label || ""}</span>
        ),
      },
      {
        id: "total",
        header: receipts.data?.col_total || subscription.data?.receipts_col_total || "",
        cell: ({ row }) => (
          <span className="text-[var(--trim-fg)]">
            {receipts.data?.money_locale?.trim()
              ? formatMoney(
                  row.original.total_cents,
                  row.original.currency_code,
                  receipts.data.money_locale,
                )
              : ""}
          </span>
        ),
      },
      {
        id: "view",
        header: receipts.data?.col_view || subscription.data?.receipts_col_view || "",
        cell: ({ row }) => {
          const openLabel = receipts.data?.open_action_label?.trim() || "";
          if (!openLabel) return "";
          return (
            <button
              type="button"
              className="text-left text-[var(--trim-fg)] underline-offset-4 hover:underline"
              onClick={() => openReceipt(row.original)}
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
          const openLabel = receipts.data?.open_action_label?.trim() || "";
          const pdfLabel = receipts.data?.download_pdf_action_label?.trim() || "";
          const rowActions = receipts.data?.table_row_actions?.trim() || "";
          if (!id || !rowActions) return null;
          const actions = [];
          if (openLabel) {
            actions.push({
              id: "view",
              label: openLabel,
              onSelect: () => openReceipt(row.original),
            });
          }
          if (pdfLabel && id && accessToken) {
            actions.push({
              id: "pdf",
              label: pdfLabel,
              onSelect: () => openReceiptPdf(row.original),
            });
          }
          if (actions.length === 0) return null;
          return <DataTableRowActions triggerLabel={rowActions} actions={actions} />;
        },
      },
    ],
    [accessToken, receipts.data, subscription.data, openReceipt, openReceiptPdf, htmlLang],
  );

  const loginPath = (
    process.env.NEXT_PUBLIC_APP_PATH_LOGIN ||
    authProviders.data?.site?.path_login ||
    ""
  ).trim();

  if (!accessToken) {
    return receiptsSkeleton;
  }

  if (subscription.isError && !subscription.data) {
    const status = (subscription.error as { status?: number } | null)?.status;
    if (status === 401 || status === 403) {
      return receiptsSkeleton;
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
  if (receipts.isPending && !receipts.data) {
    return receiptsSkeleton;
  }
  const pageTitle = subscription.data?.receipts_title?.trim() || pageChromeProps.title.trim() || "";
  if (!pageTitle) {
    if (subscription.isPending || authProviders.isPending) {
      return receiptsSkeleton;
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
            <div className="flex flex-wrap gap-2">
              {subscription.data?.receipts_sync_action_label ? (
                <Button
                  size="sm"
                  variant="outline"
                  isLoading={syncReceipts.isPending}
                  pendingLabel={subscription.data.receipts_sync_pending_label || undefined}
                  disabled={!accessToken}
                  onClick={() => {
                    setActionNotice(null);
                    setActionError(null);
                    void syncReceipts
                      .mutateAsync()
                      .then((res) => {
                        setActionNotice(res.message || "");
                        setSkip(0);
                      })
                      .catch(() => {
                        setActionError(subscription.data?.receipts_sync_failed_message || "");
                      });
                  }}
                >
                  {subscription.data.receipts_sync_action_label}
                </Button>
              ) : null}
            </div>
          </div>
          {receipts.data?.search_placeholder?.trim() ? (
            <Field
              id="receipts-search"
              label={receipts.data.search_placeholder}
              description={receipts.data.search_description}
              className="w-full sm:max-w-md"
            >
              <Input
                id="receipts-search"
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                placeholder={receipts.data.search_placeholder}
                autoComplete="off"
              />
            </Field>
          ) : null}
        </CardHeader>
        <CardContent>
          {actionNotice || actionError ? (
            <div className="mb-3 space-y-1">
              {actionNotice ? (
                <p className="text-sm text-[var(--trim-fg)]">{actionNotice}</p>
              ) : null}
              {actionError ? <p className="text-sm text-destructive">{actionError}</p> : null}
            </div>
          ) : null}
          <DataTable
            columns={receiptColumns}
            data={receipts.data?.items ?? []}
            bordered={false}
            getRowId={(row) => row.id}
            meta={receipts.data?.meta}
            onPage={
              receipts.data?.meta
                ? (s) => {
                    setSkip(s);
                    setRowSel({});
                  }
                : undefined
            }
            pageDisabled={receipts.isFetching}
            pageInputId="receipts-skip-to"
            isFetching={receipts.isFetching && !receipts.isPending}
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
                    toolbar: tableClearSelection ? (
                      <div className="flex flex-wrap gap-2">
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          onClick={() => setRowSel({})}
                        >
                          {tableClearSelection}
                        </Button>
                      </div>
                    ) : undefined,
                  }
                : undefined
            }
          />
          {!receipts.data?.items.length ? (
            <p className="py-6 text-sm text-[var(--trim-muted)]">
              {receipts.data?.empty_message || ""}
            </p>
          ) : null}
        </CardContent>
      </Card>

      <Dialog open={receiptOpen} onOpenChange={onReceiptOpenChange}>
        <DialogContent closeLabel={dialogCancel} className="max-w-lg">
          <DialogHeader>
            {receipts.data?.open_action_label ? (
              <DialogTitle>{receipts.data.open_action_label}</DialogTitle>
            ) : null}
            {selectedReceipt?.display_id?.trim() ||
            selectedReceipt?.paddle_invoice_number?.trim() ? (
              <DialogDescription>
                {selectedReceipt.display_id?.trim() ||
                  selectedReceipt.paddle_invoice_number?.trim() ||
                  ""}
              </DialogDescription>
            ) : null}
          </DialogHeader>
          <div className="grid gap-3">
            {receipts.data?.col_date || subscription.data?.receipts_col_date ? (
              <Field
                id="receipt-preview-date"
                label={receipts.data?.col_date || subscription.data?.receipts_col_date || ""}
                description={receipts.data?.preview_field_description}
              >
                <Input
                  id="receipt-preview-date"
                  value={
                    selectedReceipt
                      ? formatDateTimeFull(receiptWhenRaw(selectedReceipt), htmlLang) ||
                        formatDateShort(receiptWhenRaw(selectedReceipt), htmlLang)
                      : ""
                  }
                  placeholder={
                    receipts.data?.col_date || subscription.data?.receipts_col_date || ""
                  }
                  readOnly
                />
              </Field>
            ) : null}
            {receipts.data?.col_invoice || subscription.data?.receipts_col_invoice ? (
              <Field
                id="receipt-preview-invoice"
                label={receipts.data?.col_invoice || subscription.data?.receipts_col_invoice || ""}
                description={receipts.data?.preview_field_description}
              >
                <Input
                  id="receipt-preview-invoice"
                  value={
                    selectedReceipt?.display_id?.trim() ||
                    selectedReceipt?.paddle_invoice_number?.trim() ||
                    selectedReceipt?.paddle_transaction_id?.trim() ||
                    ""
                  }
                  placeholder={
                    receipts.data?.col_invoice || subscription.data?.receipts_col_invoice || ""
                  }
                  readOnly
                />
              </Field>
            ) : null}
            {receipts.data?.col_status || subscription.data?.receipts_col_status ? (
              <Field
                id="receipt-preview-status"
                label={receipts.data?.col_status || subscription.data?.receipts_col_status || ""}
                description={receipts.data?.preview_field_description}
              >
                <Input
                  id="receipt-preview-status"
                  value={
                    selectedReceipt?.status_label?.trim() || selectedReceipt?.status?.trim() || ""
                  }
                  placeholder={
                    receipts.data?.col_status || subscription.data?.receipts_col_status || ""
                  }
                  readOnly
                />
              </Field>
            ) : null}
            {receipts.data?.col_total || subscription.data?.receipts_col_total ? (
              <Field
                id="receipt-preview-total"
                label={receipts.data?.col_total || subscription.data?.receipts_col_total || ""}
                description={receipts.data?.preview_field_description}
              >
                <Input
                  id="receipt-preview-total"
                  value={
                    selectedReceipt && receipts.data?.money_locale?.trim()
                      ? formatMoney(
                          selectedReceipt.total_cents,
                          selectedReceipt.currency_code,
                          receipts.data.money_locale,
                        )
                      : ""
                  }
                  placeholder={
                    receipts.data?.col_total || subscription.data?.receipts_col_total || ""
                  }
                  readOnly
                />
              </Field>
            ) : null}
          </div>
          {receipts.data?.open_action_label &&
          selectedReceipt?.id?.trim() &&
          subscription.data?.path_receipts_prefix ? (
            <DialogFooter>
              <Button
                type="button"
                onClick={() => {
                  const id = selectedReceipt.id.trim();
                  const prefix = subscription.data?.path_receipts_prefix || "";
                  closeReceiptDialog();
                  router.push(`${prefix}${id}`);
                }}
              >
                {receipts.data.open_action_label}
              </Button>
            </DialogFooter>
          ) : null}
        </DialogContent>
      </Dialog>
    </div>
  );
}
