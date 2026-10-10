"use client";

import { AdminFilterBar } from "@/components/admin/admin-filter-bar";
import { DataTableSkeleton } from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { type ColumnDef, DataTable } from "@/components/ui/data-table";
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
import { useOfferEnterpriseInquiry, usePatchEnterpriseInquiry } from "@/hooks/mutations/enterprise";
import { useAdminNavChrome } from "@/hooks/queries/chrome";
import { useAdminEnterprise } from "@/hooks/queries/enterprise";
import { useAdminToken } from "@/hooks/use-admin-token";
import { useDebouncedValue } from "@/hooks/use-debounced-value";
import { useDefaultPageSize } from "@/hooks/use-default-page-size";
import { useDeferredDialogSelection } from "@/hooks/use-deferred-dialog-selection";
import { queryContentReady, useListBootReady } from "@/hooks/use-list-boot-ready";
import { useRequireStepUp } from "@/hooks/use-require-step-up";
import { chromeFieldDesc } from "@/lib/admin-field-desc";
import {
  enterpriseInquiryStatusFilterOptions,
  filterAllSentinel,
  isFilterAll,
} from "@/lib/admin-filter-options";
import { formatDateTimeFull, formatDateTimeShort } from "@/lib/format-datetime";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { type ReactNode, useCallback, useEffect, useMemo, useRef, useState } from "react";

type EntRow = Record<string, unknown>;

function asText(value: unknown): string {
  if (typeof value === "string") return value;
  if (typeof value === "number" && Number.isFinite(value)) return String(value);
  return "";
}

function messagePreview(value: unknown, max = 72): string {
  const text = asText(value).replace(/\s+/g, " ").trim();
  if (!text) return "";
  if (text.length <= max) return text;
  return `${text.slice(0, max)}…`;
}

function DetailItem({
  label,
  children,
  className,
}: {
  label: string;
  children: ReactNode;
  className?: string;
}) {
  if (!label.trim()) return null;
  return (
    <div className={className}>
      <p className="text-xs font-medium text-muted-foreground">{label}</p>
      <div className="mt-1 text-sm text-foreground">{children}</div>
    </div>
  );
}

export function EnterpriseClient({ initialToken }: { initialToken?: string }) {
  const token = useAdminToken(initialToken);
  const pageSize = useDefaultPageSize();
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const deepLinkId = (searchParams.get("id") || "").trim();
  const deepLinkOpenedFor = useRef<string | null>(null);
  const [skip, setSkip] = useState(0);
  const [qInput, setQInput] = useState("");
  const [statusFilter, setStatusFilter] = useState(filterAllSentinel());
  const q = useDebouncedValue(qInput.trim(), 250);
  const {
    open: dialogOpen,
    selectedId,
    openWith,
    close: closeDialog,
    onOpenChange: onDialogOpenChange,
  } = useDeferredDialogSelection();
  const [status, setStatus] = useState("");
  const [notes, setNotes] = useState("");
  const [offeredSeats, setOfferedSeats] = useState("");
  const chrome = useAdminNavChrome(token);
  const statusQuery = isFilterAll(statusFilter) ? "" : statusFilter;
  const data = useAdminEnterprise(token, skip, pageSize, q, statusQuery);
  const patch = usePatchEnterpriseInquiry(token);
  const offer = useOfferEnterpriseInquiry(token);
  const { err: stepErr, requireStepUp } = useRequireStepUp(token);
  const [actionErr, setActionErr] = useState("");
  /** Which footer CTA is in-flight - never spin sibling buttons on shared mutate pending. */
  const [footerAction, setFooterAction] = useState<"save" | "offer" | null>(null);
  const title = chrome.data?.ADMIN_NAV_ENTERPRISE?.trim() || "";
  const pending =
    chrome.data?.ADMIN_PENDING_UPDATING?.trim() || chrome.data?.ADMIN_PENDING_SAVING?.trim() || "";
  const saveLabel = chrome.data?.ADMIN_ACTION_SAVE?.trim() || "";
  const activateLabel =
    chrome.data?.ADMIN_ENTERPRISE_OFFER?.trim() ||
    chrome.data?.ADMIN_ENTERPRISE_ACTIVATE?.trim() ||
    "";
  const activatePending =
    chrome.data?.ADMIN_ENTERPRISE_OFFER_PENDING?.trim() ||
    chrome.data?.ADMIN_ENTERPRISE_ACTIVATE_PENDING?.trim() ||
    "";
  const activateDesc =
    chrome.data?.ADMIN_ENTERPRISE_OFFER_DESC?.trim() ||
    chrome.data?.ADMIN_ENTERPRISE_ACTIVATE_DESC?.trim() ||
    "";
  const offerLockedDesc = chrome.data?.ADMIN_ENTERPRISE_OFFER_LOCKED_DESC?.trim() || "";
  const viewLabel = chrome.data?.ADMIN_ACTION_VIEW?.trim() || "";
  const rowActionsLabel = chrome.data?.ADMIN_TABLE_ROW_ACTIONS?.trim() || "";
  const notesLabel = chrome.data?.ADMIN_ENTERPRISE_CONTRACT_NOTES?.trim() || "";
  const seatsLabel = chrome.data?.ADMIN_ENTERPRISE_OFFERED_SEATS?.trim() || "";
  const colEmail = chrome.data?.ADMIN_ENTERPRISE_COL_EMAIL?.trim() || "";
  const colCompany = chrome.data?.ADMIN_ENTERPRISE_COL_COMPANY?.trim() || "";
  const colStatus = chrome.data?.ADMIN_ENTERPRISE_COL_STATUS?.trim() || "";
  const colSeats = chrome.data?.ADMIN_ENTERPRISE_COL_SEATS?.trim() || "";
  const colCreated = chrome.data?.ADMIN_ENTERPRISE_COL_CREATED?.trim() || "";
  const htmlLang = chrome.data?.SITE_HTML_LANG?.trim() || "";
  const colMessage = chrome.data?.ADMIN_ENTERPRISE_COL_MESSAGE?.trim() || "";
  const detailsLabel = chrome.data?.ADMIN_ENTERPRISE_DETAILS?.trim() || "";
  const inquiryMessageLabel = chrome.data?.ADMIN_ENTERPRISE_INQUIRY_MESSAGE?.trim() || "";
  const estimatedSeatsLabel = chrome.data?.ADMIN_ENTERPRISE_ESTIMATED_SEATS?.trim() || "";
  const userLabel = chrome.data?.ADMIN_ENTERPRISE_USER?.trim() || "";
  const createdLabel = chrome.data?.ADMIN_ENTERPRISE_CREATED?.trim() || "";
  const updatedLabel = chrome.data?.ADMIN_ENTERPRISE_UPDATED?.trim() || "";
  const activatedStatusLabel = chrome.data?.ADMIN_ENTERPRISE_STATUS_ACTIVATED?.trim() || "";
  const filterSearch = chrome.data?.ADMIN_FILTER_SEARCH?.trim() || "";
  const filterSearchDesc = chrome.data?.ADMIN_FILTER_SEARCH_DESC?.trim() || "";
  const filterStatus = chrome.data?.ADMIN_FILTER_STATUS?.trim() || "";
  const filterStatusDesc = chrome.data?.ADMIN_FILTER_STATUS_DESC?.trim() || "";
  const filterAll = chrome.data?.ADMIN_FILTER_ALL?.trim() || "";
  const closeLabel =
    chrome.data?.ADMIN_DIALOG_CLOSE?.trim() || chrome.data?.ADMIN_CLOSE?.trim() || "";
  const editSectionLabel =
    chrome.data?.ADMIN_ENTERPRISE_EDIT?.trim() || chrome.data?.ADMIN_SECTION_EDIT?.trim() || "";
  const listStatusOptions = enterpriseInquiryStatusFilterOptions(chrome.data ?? {});

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setSkip(0);
  }, [q, statusQuery]);
  const statusOptions = (
    [
      ["new", chrome.data?.ADMIN_ENTERPRISE_STATUS_NEW],
      ["contacted", chrome.data?.ADMIN_ENTERPRISE_STATUS_CONTACTED],
      ["closed", chrome.data?.ADMIN_ENTERPRISE_STATUS_CLOSED],
    ] as const
  ).filter(([, label]) => Boolean(label?.trim()));
  /** After Send Paddle offer: withdraw (closed) or revise (contacted) only. */
  const offeredStatusOptions = (
    [
      ["offered", chrome.data?.ADMIN_ENTERPRISE_STATUS_OFFERED],
      ["contacted", chrome.data?.ADMIN_ENTERPRISE_STATUS_CONTACTED],
      ["closed", chrome.data?.ADMIN_ENTERPRISE_STATUS_CLOSED],
    ] as const
  ).filter(([, label]) => Boolean(label?.trim()));

  const selectInquiry = useCallback(
    (row: EntRow) => {
      const id = asText(row.id);
      if (!id) return;
      openWith(id);
      setStatus(asText(row.status));
      setNotes(asText(row.contract_notes));
      setOfferedSeats(asText(row.offered_seat_quantity));
      setActionErr("");
    },
    [openWith],
  );

  const columns = useMemo<ColumnDef<EntRow>[]>(
    () => [
      {
        id: "email",
        header: colEmail,
        cell: ({ row }) => {
          const id = asText(row.original.id);
          const email = asText(row.original.email);
          if (!id || !email) return "";
          return (
            <button
              type="button"
              className="text-left text-foreground hover:underline"
              onClick={() => selectInquiry(row.original)}
            >
              {email}
            </button>
          );
        },
      },
      {
        id: "company_name",
        header: colCompany,
        cell: ({ row }) => asText(row.original.company_name),
      },
      {
        id: "estimated_seats",
        header: colSeats,
        cell: ({ row }) => {
          const seats = asText(row.original.estimated_seats);
          return seats && seats !== "0" ? seats : "";
        },
      },
      {
        id: "message",
        header: colMessage,
        cell: ({ row }) => (
          <span className="line-clamp-2 max-w-[18rem] text-muted-foreground">
            {messagePreview(row.original.message)}
          </span>
        ),
      },
      {
        id: "status",
        header: colStatus,
        cell: ({ row }) => asText(row.original.status_label),
      },
      {
        id: "created_at",
        header: colCreated,
        cell: ({ row }) => (
          <span
            className="whitespace-nowrap text-xs text-muted-foreground"
            title={formatDateTimeFull(asText(row.original.created_at), htmlLang) || undefined}
          >
            {formatDateTimeShort(asText(row.original.created_at), htmlLang)}
          </span>
        ),
      },
      {
        id: "_actions",
        header: "",
        cell: ({ row }) => {
          const id = asText(row.original.id);
          if (!id || !viewLabel || !rowActionsLabel) return null;
          return (
            <DataTableRowActions
              triggerLabel={rowActionsLabel}
              actions={[
                {
                  id: "view",
                  label: viewLabel,
                  onSelect: () => selectInquiry(row.original),
                },
              ]}
            />
          );
        },
      },
    ],
    [
      colEmail,
      colCompany,
      colSeats,
      colMessage,
      colStatus,
      colCreated,
      viewLabel,
      rowActionsLabel,
      selectInquiry,
      htmlLang,
    ],
  );

  const bootReady = useListBootReady(pageSize, Boolean(chrome.data), queryContentReady(data));
  const showInitialSkeleton = !bootReady;

  const items = useMemo(
    () =>
      ((data.data?.items as EntRow[]) ?? []).filter(
        (item) => asText(item.id) && asText(item.email),
      ),
    [data.data?.items],
  );
  const selected = useMemo(
    () => items.find((item) => asText(item.id) === selectedId) ?? null,
    [items, selectedId],
  );

  // Deep-link ?id= must run unconditionally (hooks cannot follow early return).
  // Once opened for an id, do not re-open on dialog close while ?id= is still present.
  useEffect(() => {
    if (showInitialSkeleton) return;
    if (!deepLinkId) {
      deepLinkOpenedFor.current = null;
      return;
    }
    if (deepLinkOpenedFor.current === deepLinkId) return;
    const row = items.find((item) => asText(item.id) === deepLinkId);
    if (!row) return;
    deepLinkOpenedFor.current = deepLinkId;
    selectInquiry(row);
  }, [showInitialSkeleton, deepLinkId, items, selectInquiry]);

  const handleDialogOpenChange = useCallback(
    (next: boolean) => {
      onDialogOpenChange(next);
      // Drop ?id= on dismiss so close sticks and notification re-clicks can open again.
      if (!next) {
        setFooterAction(null);
        if (deepLinkId) {
          router.replace(pathname, { scroll: false });
        }
      }
    },
    [onDialogOpenChange, deepLinkId, router, pathname],
  );

  if (showInitialSkeleton) {
    return (
      <div className="space-y-4">
        {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
        <DataTableSkeleton
          title=""
          columns={[colEmail, colCompany, colSeats, colMessage, colStatus, colCreated].map(
            (label) => ({
              label: label || "",
            }),
          )}
          rows={pageSize > 0 ? pageSize : 8}
          showFilters
          filterCount={1}
        />
      </div>
    );
  }

  const persistedStatus = selected ? asText(selected.status).trim().toLowerCase() : "";
  const isActivated = persistedStatus === "activated";
  const isOfferedPersisted = persistedStatus === "offered";
  const isClosedPersisted = persistedStatus === "closed";
  /** Paid row is fully immutable. Offered locks seats/notes until status is moved away. */
  const offerTermsLocked = isActivated || isOfferedPersisted;
  const statusReadonly = isActivated;
  const statusReadonlyLabel = activatedStatusLabel || asText(selected?.status_label) || status;
  const statusSelectOptions = isOfferedPersisted ? offeredStatusOptions : statusOptions;
  const canSendOffer =
    Boolean(activateLabel && activatePending) &&
    !isActivated &&
    !isOfferedPersisted &&
    !isClosedPersisted;
  /** While offered, Save only commits Contacted (revise) or Closed (withdraw). */
  const canSave = Boolean(saveLabel) && !isActivated;
  const userId = selected ? asText(selected.user_id) : "";
  const fullName = selected ? asText(selected.full_name) : "";

  return (
    <div className="space-y-4">
      {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
      {stepErr ? <p className="text-sm text-destructive">{stepErr}</p> : null}
      {actionErr ? <p className="text-sm text-destructive">{actionErr}</p> : null}
      <AdminFilterBar
        filters={[
          {
            kind: "search",
            id: "enterprise-search",
            label: filterSearch,
            description: filterSearchDesc,
            value: qInput,
            onChange: setQInput,
          },
          ...(filterStatus && listStatusOptions.length > 0
            ? [
                {
                  kind: "select" as const,
                  id: "enterprise-status-filter",
                  label: filterStatus,
                  description: filterStatusDesc,
                  value: statusFilter,
                  onChange: setStatusFilter,
                  options: [
                    ...(filterAll ? [{ value: filterAllSentinel(), label: filterAll }] : []),
                    ...listStatusOptions,
                  ],
                },
              ]
            : []),
        ]}
      />
      <Card>
        <CardContent className="pt-6">
          <DataTable
            columns={columns}
            data={items}
            meta={data.data?.meta}
            onPage={setSkip}
            pageDisabled={data.isFetching}
            isFetching={(data.isFetching && !data.isPending) || patch.isPending}
            getRowId={(row) => asText(row.id)}
            bordered={false}
          />
        </CardContent>
      </Card>
      <Dialog open={dialogOpen} onOpenChange={handleDialogOpenChange}>
        <DialogContent closeLabel={closeLabel} className="max-w-2xl">
          <DialogHeader>
            {editSectionLabel || detailsLabel ? (
              <DialogTitle>{editSectionLabel || detailsLabel}</DialogTitle>
            ) : null}
          </DialogHeader>
          <div className="space-y-6">
            {selected ? (
              <section className="space-y-3 rounded-md border border-border/80 bg-muted/20 p-4">
                {detailsLabel ? (
                  <h3 className="text-sm font-semibold text-foreground">{detailsLabel}</h3>
                ) : null}
                <div className="grid gap-4 sm:grid-cols-2">
                  <DetailItem label={colEmail}>{asText(selected.email)}</DetailItem>
                  <DetailItem label={colCompany}>{asText(selected.company_name) || "-"}</DetailItem>
                  <DetailItem label={estimatedSeatsLabel || colSeats}>
                    {asText(selected.estimated_seats) || "-"}
                  </DetailItem>
                  <DetailItem label={colStatus}>
                    {asText(selected.status_label) || asText(selected.status) || "-"}
                  </DetailItem>
                  <DetailItem label={userLabel} className="sm:col-span-2">
                    {userId ? (
                      <Link
                        href={`/users/${userId}`}
                        className="text-foreground underline-offset-4 hover:underline"
                      >
                        {fullName
                          ? `${fullName} · ${asText(selected.email)}`
                          : asText(selected.email)}
                      </Link>
                    ) : (
                      "-"
                    )}
                  </DetailItem>
                  <DetailItem label={inquiryMessageLabel || colMessage} className="sm:col-span-2">
                    <p className="whitespace-pre-wrap break-words rounded-md border border-border bg-background/60 px-3 py-2 text-sm">
                      {asText(selected.message) || "-"}
                    </p>
                  </DetailItem>
                  <DetailItem label={createdLabel || colCreated}>
                    {formatDateTimeFull(asText(selected.created_at), htmlLang) ||
                      formatDateTimeShort(asText(selected.created_at), htmlLang) ||
                      "-"}
                  </DetailItem>
                  <DetailItem label={updatedLabel}>
                    {formatDateTimeFull(asText(selected.updated_at), htmlLang) ||
                      formatDateTimeShort(asText(selected.updated_at), htmlLang) ||
                      "-"}
                  </DetailItem>
                </div>
              </section>
            ) : null}

            <div className="grid gap-4 sm:grid-cols-2">
              {colStatus && (statusReadonly || statusSelectOptions.length > 0) ? (
                <Field
                  id="enterprise-status"
                  label={colStatus}
                  {...chromeFieldDesc(chrome.data, "ADMIN_ENTERPRISE_STATUS_DESC")}
                >
                  {statusReadonly ? (
                    <Input id="enterprise-status" value={statusReadonlyLabel} readOnly disabled />
                  ) : (
                    <Select value={status} onValueChange={setStatus}>
                      <SelectTrigger id="enterprise-status" aria-label={colStatus}>
                        <SelectValue placeholder={colStatus} />
                      </SelectTrigger>
                      <SelectContent>
                        {statusSelectOptions.map(([value, label]) => (
                          <SelectItem key={value} value={value}>
                            {label?.trim()}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  )}
                </Field>
              ) : null}
              {seatsLabel ? (
                <Field
                  id="enterprise-seats"
                  label={seatsLabel}
                  {...chromeFieldDesc(chrome.data, "ADMIN_ENTERPRISE_OFFERED_SEATS_DESC")}
                >
                  <Input
                    id="enterprise-seats"
                    value={offeredSeats}
                    onChange={(e) => setOfferedSeats(e.target.value)}
                    placeholder={seatsLabel}
                    disabled={offerTermsLocked}
                    readOnly={offerTermsLocked}
                    inputMode="numeric"
                  />
                </Field>
              ) : null}
              {notesLabel ? (
                <Field
                  id="enterprise-notes"
                  label={notesLabel}
                  className="sm:col-span-2"
                  {...chromeFieldDesc(chrome.data, "ADMIN_ENTERPRISE_CONTRACT_NOTES_DESC")}
                >
                  <Textarea
                    id="enterprise-notes"
                    value={notes}
                    onChange={(e) => setNotes(e.target.value)}
                    placeholder={notesLabel}
                    disabled={offerTermsLocked}
                    readOnly={offerTermsLocked}
                    rows={4}
                  />
                </Field>
              ) : null}
              {isOfferedPersisted && offerLockedDesc ? (
                <p className="sm:col-span-2 text-xs text-muted-foreground">{offerLockedDesc}</p>
              ) : null}
              {activateDesc && canSendOffer ? (
                <p className="sm:col-span-2 text-xs text-muted-foreground">{activateDesc}</p>
              ) : null}
            </div>
          </div>
          {canSendOffer || canSave ? (
            <DialogFooter className="flex flex-wrap gap-2">
              {canSendOffer ? (
                <Button
                  type="button"
                  variant="secondary"
                  isLoading={footerAction === "offer"}
                  pendingLabel={activatePending}
                  disabled={
                    !offeredSeats.trim() || Number(offeredSeats) < 1 || footerAction !== null
                  }
                  onClick={() => {
                    if (!requireStepUp()) return;
                    setActionErr("");
                    const seats = Number(offeredSeats);
                    if (!Number.isFinite(seats) || seats < 1) return;
                    setFooterAction("offer");
                    void (async () => {
                      try {
                        await patch.mutateAsync({
                          id: selectedId,
                          status,
                          contract_notes: notes,
                          offered_seat_quantity: seats,
                        });
                        await offer.mutateAsync(selectedId);
                        closeDialog();
                        setActionErr("");
                      } catch (e: unknown) {
                        setActionErr(e instanceof Error ? e.message : "");
                      } finally {
                        setFooterAction(null);
                      }
                    })();
                  }}
                >
                  {activateLabel}
                </Button>
              ) : null}
              {canSave ? (
                <Button
                  type="button"
                  isLoading={footerAction === "save"}
                  pendingLabel={pending || saveLabel}
                  disabled={
                    footerAction !== null ||
                    (isOfferedPersisted
                      ? status !== "contacted" && status !== "closed"
                      : !status.trim())
                  }
                  onClick={() => {
                    if (!requireStepUp()) return;
                    setActionErr("");
                    setFooterAction("save");
                    void patch
                      .mutateAsync(
                        isOfferedPersisted
                          ? {
                              id: selectedId,
                              status,
                            }
                          : {
                              id: selectedId,
                              status,
                              contract_notes: notes,
                              offered_seat_quantity: offeredSeats
                                ? Number(offeredSeats)
                                : undefined,
                            },
                      )
                      .then(() => {
                        closeDialog();
                        setActionErr("");
                      })
                      .catch((e: unknown) => {
                        setActionErr(e instanceof Error ? e.message : "");
                      })
                      .finally(() => {
                        setFooterAction(null);
                      });
                  }}
                >
                  {saveLabel}
                </Button>
              ) : null}
            </DialogFooter>
          ) : null}
        </DialogContent>
      </Dialog>
    </div>
  );
}
