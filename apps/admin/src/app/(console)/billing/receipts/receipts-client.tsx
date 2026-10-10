"use client";

import { AdminFilterBar } from "@/components/admin/admin-filter-bar";
import { DualTableSkeleton } from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { type ColumnDef, DataTable, type RowSelectionState } from "@/components/ui/data-table";
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
import { Textarea } from "@/components/ui/textarea";
import {
  usePostDisputeNote,
  useReceiptPDFReissue,
  useReceiptResync,
} from "@/hooks/mutations/billing";
import { useAdminDisputes, useAdminReceipts } from "@/hooks/queries/billing";
import { useAdminNavChrome } from "@/hooks/queries/chrome";
import { useAdminToken } from "@/hooks/use-admin-token";
import { useDebouncedValue } from "@/hooks/use-debounced-value";
import { useDefaultPageSize } from "@/hooks/use-default-page-size";
import { queryContentReady, useListBootReady } from "@/hooks/use-list-boot-ready";
import { chromeFieldDesc } from "@/lib/admin-field-desc";
import { formatDateTimeFull, formatDateTimeShort } from "@/lib/format-datetime";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useEffect, useMemo, useRef, useState } from "react";

type ReceiptRow = Record<string, unknown>;
type DisputeRow = Record<string, unknown>;

/** Present API cents as major currency units (fail closed without currency). */
function formatReceiptMoney(cents: unknown, currency: string): string {
  const code = currency.trim().toUpperCase();
  if (!code || typeof cents !== "number" || !Number.isFinite(cents)) return "";
  return `${code} ${(cents / 100).toFixed(2)}`;
}

export function ReceiptsClient({ initialToken }: { initialToken?: string }) {
  const token = useAdminToken(initialToken);
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();
  const listBase = pathname.startsWith("/sales/receipts") ? "/sales/receipts" : "/billing/receipts";
  const deepLinkId = (searchParams.get("id") || "").trim();
  const deepLinkOpenedFor = useRef<string | null>(null);
  const pageSize = useDefaultPageSize();
  const [skip, setSkip] = useState(0);
  const [disputeSkip, setDisputeSkip] = useState(0);
  const [rowSel, setRowSel] = useState<RowSelectionState>({});
  const [disputeSel, setDisputeSel] = useState<RowSelectionState>({});
  const [qInput, setQInput] = useState("");
  const q = useDebouncedValue(qInput.trim(), 250);
  const [disputeQInput, setDisputeQInput] = useState("");
  const disputeQ = useDebouncedValue(disputeQInput.trim(), 250);
  const [txId, setTxId] = useState("");
  const [userId, setUserId] = useState("");
  const [note, setNote] = useState("");
  const [disputeStatus, setDisputeStatus] = useState("");
  const [disputeOpen, setDisputeOpen] = useState(false);
  const chrome = useAdminNavChrome(token);
  const receipts = useAdminReceipts(token, skip, pageSize, q);
  const disputes = useAdminDisputes(token, disputeSkip, pageSize, disputeQ);
  const dispute = usePostDisputeNote(token);
  const resync = useReceiptResync(token);
  const pdfReissue = useReceiptPDFReissue(token);
  const title = chrome.data?.ADMIN_NAV_RECEIPTS?.trim() || "";
  const searchLabel = chrome.data?.ADMIN_RECEIPTS_SEARCH?.trim() || "";
  const filterSearch = chrome.data?.ADMIN_FILTER_SEARCH?.trim() || "";
  const filterSearchDesc = chrome.data?.ADMIN_FILTER_SEARCH_DESC?.trim() || "";
  const disputeTitle = chrome.data?.ADMIN_DISPUTE_TITLE?.trim() || "";
  const disputeListTitle = chrome.data?.ADMIN_DISPUTE_LIST_TITLE?.trim() || "";
  const txLabel = chrome.data?.ADMIN_DISPUTE_TX_LABEL?.trim() || "";
  const userLabel = chrome.data?.ADMIN_DISPUTE_USER_LABEL?.trim() || "";
  const noteLabel = chrome.data?.ADMIN_DISPUTE_NOTE_LABEL?.trim() || "";
  const statusLabel =
    chrome.data?.ADMIN_RECEIPT_COL_STATUS?.trim() ||
    chrome.data?.ADMIN_DISPUTE_STATUS_LABEL?.trim() ||
    "";
  const disputeWhenLabel = chrome.data?.ADMIN_DISPUTE_COL_WHEN?.trim() || "";
  const whenLabel = chrome.data?.ADMIN_RECEIPT_COL_WHEN?.trim() || disputeWhenLabel;
  const invoiceLabel = chrome.data?.ADMIN_RECEIPT_COL_INVOICE?.trim() || "";
  const customerLabel = chrome.data?.ADMIN_RECEIPT_COL_CUSTOMER?.trim() || "";
  const totalLabel = chrome.data?.ADMIN_RECEIPT_COL_TOTAL?.trim() || "";
  const viewDetailLabel =
    chrome.data?.ADMIN_RECEIPT_VIEW_DETAIL?.trim() ||
    chrome.data?.ADMIN_RECEIPT_PDF_OPEN?.trim() ||
    "";
  const htmlLang = chrome.data?.SITE_HTML_LANG?.trim() || "";
  const disputeByLabel = chrome.data?.ADMIN_DISPUTE_COL_CREATED_BY?.trim() || "";
  const pendingCreating = chrome.data?.ADMIN_PENDING_CREATING?.trim() || "";
  const pendingUpdating =
    chrome.data?.ADMIN_PENDING_UPDATING?.trim() || chrome.data?.ADMIN_PENDING_SAVING?.trim() || "";
  const saveLabel = chrome.data?.ADMIN_ACTION_SAVE?.trim() || "";
  const resyncLabel = chrome.data?.ADMIN_ACTION_RESYNC?.trim() || "";
  const pdfReissueLabel = chrome.data?.ADMIN_ACTION_PDF_REISSUE?.trim() || "";
  const rowActionsLabel = chrome.data?.ADMIN_TABLE_ROW_ACTIONS?.trim() || "";
  const selectAllLabel = chrome.data?.ADMIN_TABLE_SELECT_ALL?.trim() || "";
  const selectRowLabel = chrome.data?.ADMIN_TABLE_SELECT_ROW?.trim() || "";
  const selectedFmt = chrome.data?.ADMIN_TABLE_SELECTED_FMT?.trim() || "";
  const clearSelLabel = chrome.data?.ADMIN_TABLE_CLEAR_SELECTION?.trim() || "";
  const selectionChrome =
    selectAllLabel && selectRowLabel
      ? {
          selectAllLabel,
          selectRowLabel,
          selectedCountFmt: selectedFmt || undefined,
        }
      : null;
  const closeLabel =
    chrome.data?.ADMIN_DIALOG_CLOSE?.trim() || chrome.data?.ADMIN_CLOSE?.trim() || "";
  const statusOptions = (
    [
      ["open", chrome.data?.ADMIN_DISPUTE_STATUS_OPEN],
      ["watching", chrome.data?.ADMIN_DISPUTE_STATUS_WATCHING],
      ["closed", chrome.data?.ADMIN_DISPUTE_STATUS_CLOSED],
    ] as const
  ).filter(([, label]) => Boolean(label?.trim()));

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setSkip(0);
    setRowSel({});
  }, [q]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setDisputeSkip(0);
    setDisputeSel({});
  }, [disputeQ]);

  // Notification / drill-down ?id= → open tax-invoice detail (same pattern as enterprise).
  useEffect(() => {
    if (!deepLinkId) {
      deepLinkOpenedFor.current = null;
      return;
    }
    if (deepLinkOpenedFor.current === deepLinkId) return;
    deepLinkOpenedFor.current = deepLinkId;
    router.push(`${listBase}/${encodeURIComponent(deepLinkId)}`);
  }, [deepLinkId, listBase, router]);

  const columns = useMemo<ColumnDef<ReceiptRow>[]>(() => {
    const cols: ColumnDef<ReceiptRow>[] = [];
    if (whenLabel) {
      cols.push({
        id: "when",
        header: whenLabel,
        cell: ({ row }) => {
          const raw = String(row.original.paid_at || row.original.created_at || "");
          return (
            <span title={formatDateTimeFull(raw, htmlLang) || undefined}>
              {formatDateTimeShort(raw, htmlLang)}
            </span>
          );
        },
      });
    }
    if (invoiceLabel) {
      cols.push({
        id: "invoice",
        header: invoiceLabel,
        cell: ({ row }) => {
          const id = String(row.original.id || "");
          const label = String(row.original.display_id || row.original.paddle_invoice_number || "");
          if (!id || !label) return label;
          return (
            <Link
              href={`${listBase}/${encodeURIComponent(id)}`}
              className="font-medium text-foreground underline-offset-4 hover:underline"
            >
              {label}
            </Link>
          );
        },
      });
    }
    if (customerLabel) {
      cols.push({
        id: "customer",
        header: customerLabel,
        cell: ({ row }) => String(row.original.bill_to_email || ""),
      });
    }
    if (statusLabel) {
      cols.push({
        id: "status",
        header: statusLabel,
        cell: ({ row }) => String(row.original.status_label || row.original.status || ""),
      });
    }
    if (totalLabel) {
      cols.push({
        id: "total",
        header: totalLabel,
        cell: ({ row }) => {
          const total = formatReceiptMoney(
            row.original.total_cents,
            String(row.original.currency_code || ""),
          );
          return total ? <span className="tabular-nums">{total}</span> : "";
        },
      });
    }

    const showView = Boolean(viewDetailLabel);
    const showResync = Boolean(resyncLabel && pendingUpdating);
    const showPdf = Boolean(pdfReissueLabel && pendingUpdating);
    if (rowActionsLabel && (showView || showResync || showPdf)) {
      cols.push({
        id: "_actions",
        header: "",
        cell: ({ row }) => {
          const id = String(row.original.id || "");
          if (!id) return null;
          const menuActions = [];
          if (showView) {
            menuActions.push({
              id: "view",
              label: viewDetailLabel,
              onSelect: () => {
                router.push(`${listBase}/${encodeURIComponent(id)}`);
              },
            });
          }
          if (showResync) {
            menuActions.push({
              id: "resync",
              label: resyncLabel,
              onSelect: () => {
                void resync.mutate(id);
              },
            });
          }
          if (showPdf) {
            menuActions.push({
              id: "pdf-reissue",
              label: pdfReissueLabel,
              onSelect: () => {
                void pdfReissue.mutate(id);
              },
            });
          }
          if (menuActions.length === 0) return null;
          return <DataTableRowActions triggerLabel={rowActionsLabel} actions={menuActions} />;
        },
      });
    }
    return cols;
  }, [
    whenLabel,
    invoiceLabel,
    customerLabel,
    statusLabel,
    totalLabel,
    viewDetailLabel,
    htmlLang,
    listBase,
    resyncLabel,
    pdfReissueLabel,
    pendingUpdating,
    rowActionsLabel,
    resync,
    pdfReissue,
    router,
  ]);

  const disputeColumns = useMemo<ColumnDef<DisputeRow>[]>(() => {
    const cols: ColumnDef<DisputeRow>[] = [];
    if (txLabel) {
      cols.push({
        id: "paddle_transaction_id",
        header: txLabel,
        cell: ({ row }) => (
          <span className="font-mono text-xs">
            {String(row.original.paddle_transaction_id || "")}
          </span>
        ),
      });
    }
    if (userLabel) {
      cols.push({
        id: "user_id",
        header: userLabel,
        cell: ({ row }) => (
          <span className="font-mono text-xs">{String(row.original.user_id || "")}</span>
        ),
      });
    }
    if (noteLabel) {
      cols.push({
        id: "note",
        header: noteLabel,
        cell: ({ row }) => String(row.original.note || ""),
      });
    }
    if (statusLabel) {
      cols.push({
        id: "status",
        header: statusLabel,
        cell: ({ row }) => String(row.original.status_label || ""),
      });
    }
    if (disputeByLabel) {
      cols.push({
        id: "created_by",
        header: disputeByLabel,
        cell: ({ row }) => (
          <span className="font-mono text-xs">{String(row.original.created_by || "")}</span>
        ),
      });
    }
    if (disputeWhenLabel) {
      cols.push({
        id: "created_at",
        header: disputeWhenLabel,
        cell: ({ row }) => {
          const raw = String(row.original.created_at || "");
          return (
            <span title={formatDateTimeFull(raw, htmlLang) || undefined}>
              {formatDateTimeShort(raw, htmlLang)}
            </span>
          );
        },
      });
    }
    return cols;
  }, [txLabel, userLabel, noteLabel, statusLabel, disputeByLabel, disputeWhenLabel, htmlLang]);

  const bootReady = useListBootReady(pageSize, Boolean(chrome.data), queryContentReady(receipts));
  const showInitialSkeleton = !bootReady;

  if (showInitialSkeleton) {
    const primary = [whenLabel, invoiceLabel, customerLabel, statusLabel, totalLabel]
      .filter(Boolean)
      .map((label) => ({ label }));
    const secondary = [txLabel, userLabel, noteLabel, statusLabel]
      .filter(Boolean)
      .map((label) => ({ label }));
    return (
      <div className="space-y-4">
        <DualTableSkeleton
          title={title || undefined}
          primaryColumns={
            primary.length > 0
              ? primary
              : [{ label: "" }, { label: "" }, { label: "" }, { label: "" }, { label: "" }]
          }
          secondaryColumns={
            secondary.length > 0 ? secondary : [{ label: "" }, { label: "" }, { label: "" }]
          }
          rows={pageSize > 0 ? pageSize : 8}
          showFilters
          filterCount={1}
          midToolbarButtons={1}
        />
      </div>
    );
  }

  const items = ((receipts.data?.items as ReceiptRow[]) ?? []).filter((item) =>
    Boolean(String(item.id || "")),
  );
  const disputeItems = ((disputes.data?.items as DisputeRow[]) ?? []).filter((item) =>
    Boolean(String(item.id || "")),
  );

  return (
    <div className="space-y-6">
      {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
      <AdminFilterBar
        filters={[
          {
            kind: "search",
            id: "receipts-search",
            label: searchLabel,
            description: filterSearchDesc,
            value: qInput,
            onChange: setQInput,
          },
        ]}
      />
      <Card>
        <CardContent className="pt-6">
          <DataTable
            columns={columns}
            data={items}
            meta={receipts.data?.meta}
            onPage={(s) => {
              setSkip(s);
              setRowSel({});
            }}
            pageDisabled={receipts.isFetching}
            pageInputId="admin-receipts-skip-to"
            isFetching={
              (receipts.isFetching && !receipts.isPending) ||
              resync.isPending ||
              pdfReissue.isPending
            }
            getRowId={(row) => String(row.id)}
            bordered={false}
            selection={
              selectionChrome
                ? {
                    chrome: selectionChrome,
                    rowSelection: rowSel,
                    onRowSelectionChange: setRowSel,
                    toolbar: clearSelLabel ? (
                      <div className="flex flex-wrap gap-2">
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          onClick={() => setRowSel({})}
                        >
                          {clearSelLabel}
                        </Button>
                      </div>
                    ) : undefined,
                  }
                : undefined
            }
          />
        </CardContent>
      </Card>
      {disputeTitle && statusOptions.length > 0 && saveLabel && pendingCreating ? (
        <Button type="button" onClick={() => setDisputeOpen(true)}>
          {disputeTitle}
        </Button>
      ) : null}
      {disputeListTitle && disputeColumns.length > 0 ? (
        <div className="space-y-3">
          <h2 className="text-sm font-medium text-foreground">{disputeListTitle}</h2>
          <AdminFilterBar
            filters={[
              {
                kind: "search",
                id: "disputes-search",
                label: filterSearch,
                description: filterSearchDesc,
                value: disputeQInput,
                onChange: setDisputeQInput,
              },
            ]}
          />
          <Card>
            <CardContent className="pt-6">
              <DataTable
                columns={disputeColumns}
                data={disputeItems}
                meta={disputes.data?.meta}
                onPage={(s) => {
                  setDisputeSkip(s);
                  setDisputeSel({});
                }}
                pageDisabled={disputes.isFetching}
                pageInputId="admin-disputes-skip-to"
                isFetching={(disputes.isFetching && !disputes.isPending) || dispute.isPending}
                getRowId={(row) => String(row.id || "")}
                bordered={false}
                selection={
                  selectionChrome
                    ? {
                        chrome: selectionChrome,
                        rowSelection: disputeSel,
                        onRowSelectionChange: setDisputeSel,
                        toolbar: clearSelLabel ? (
                          <div className="flex flex-wrap gap-2">
                            <Button
                              type="button"
                              size="sm"
                              variant="outline"
                              onClick={() => setDisputeSel({})}
                            >
                              {clearSelLabel}
                            </Button>
                          </div>
                        ) : undefined,
                      }
                    : undefined
                }
              />
            </CardContent>
          </Card>
        </div>
      ) : null}
      {disputeTitle && statusOptions.length > 0 ? (
        <Dialog
          open={disputeOpen}
          onOpenChange={(open) => {
            if (!open) setDisputeOpen(false);
            else setDisputeOpen(true);
          }}
        >
          <DialogContent closeLabel={closeLabel} className="max-w-2xl">
            <DialogHeader>
              <DialogTitle>{disputeTitle}</DialogTitle>
            </DialogHeader>
            <div className="grid gap-4 sm:grid-cols-2">
              {txLabel ? (
                <Field
                  id="dispute-tx"
                  label={txLabel}
                  {...chromeFieldDesc(chrome.data, "ADMIN_DISPUTE_TX_DESC")}
                >
                  <Input
                    id="dispute-tx"
                    value={txId}
                    onChange={(e) => setTxId(e.target.value)}
                    placeholder={txLabel}
                  />
                </Field>
              ) : null}
              {userLabel ? (
                <Field
                  id="dispute-user"
                  label={userLabel}
                  {...chromeFieldDesc(chrome.data, "ADMIN_DISPUTE_USER_DESC")}
                >
                  <Input
                    id="dispute-user"
                    value={userId}
                    onChange={(e) => setUserId(e.target.value)}
                    placeholder={userLabel}
                  />
                </Field>
              ) : null}
              {noteLabel ? (
                <Field
                  id="dispute-note"
                  label={noteLabel}
                  {...chromeFieldDesc(chrome.data, "ADMIN_RECEIPT_DISPUTE_REASON_DESC")}
                  className="sm:col-span-2"
                >
                  <Textarea
                    id="dispute-note"
                    value={note}
                    onChange={(e) => setNote(e.target.value)}
                    placeholder={noteLabel}
                  />
                </Field>
              ) : null}
              {statusLabel ? (
                <Field
                  id="dispute-status"
                  label={statusLabel}
                  {...chromeFieldDesc(chrome.data, "ADMIN_DISPUTE_STATUS_DESC")}
                >
                  <Select value={disputeStatus || undefined} onValueChange={setDisputeStatus}>
                    <SelectTrigger id="dispute-status" aria-label={statusLabel}>
                      <SelectValue placeholder={statusLabel} />
                    </SelectTrigger>
                    <SelectContent>
                      {statusOptions.map(([value, label]) => (
                        <SelectItem key={value} value={value}>
                          {label?.trim()}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </Field>
              ) : null}
            </div>
            {saveLabel ? (
              <DialogFooter>
                <Button
                  type="button"
                  isLoading={dispute.isPending}
                  pendingLabel={pendingCreating || saveLabel}
                  disabled={!note.trim() || !disputeStatus}
                  onClick={() =>
                    void dispute.mutate(
                      {
                        paddle_transaction_id: txId.trim() || undefined,
                        user_id: userId.trim() || undefined,
                        note: note.trim(),
                        status: disputeStatus,
                      },
                      {
                        onSuccess: () => {
                          setTxId("");
                          setUserId("");
                          setNote("");
                          setDisputeStatus("");
                          setDisputeOpen(false);
                        },
                      },
                    )
                  }
                >
                  {saveLabel}
                </Button>
              </DialogFooter>
            ) : null}
          </DialogContent>
        </Dialog>
      ) : null}
    </div>
  );
}
