"use client";

import { AdminFilterBar } from "@/components/admin/admin-filter-bar";
import { DataTableSkeleton, TabsSkeleton } from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { type ColumnDef, DataTable } from "@/components/ui/data-table";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import { useCreditGrant } from "@/hooks/mutations/users";
import { useAdminNavChrome } from "@/hooks/queries/chrome";
import { useAdminCreditGrants, useAdminTopupLedger } from "@/hooks/queries/users";
import { useAdminToken } from "@/hooks/use-admin-token";
import { useDebouncedValue } from "@/hooks/use-debounced-value";
import { useDefaultPageSize } from "@/hooks/use-default-page-size";
import { queryContentReady, useListBootReady } from "@/hooks/use-list-boot-ready";
import { useRequireStepUp } from "@/hooks/use-require-step-up";
import { chromeFieldDesc } from "@/lib/admin-field-desc";
import { formatDateTimeFull, formatDateTimeShort } from "@/lib/format-datetime";
import { useEffect, useMemo, useState } from "react";

type LedgerRow = Record<string, unknown>;
type Tab = "grants" | "topups";

/** Present API cents as major currency units (fail closed without currency). */
function formatTopupMoney(cents: unknown, currency: string): string {
  const code = currency.trim().toUpperCase();
  if (!code || typeof cents !== "number" || !Number.isFinite(cents)) return "";
  return `${code} ${(cents / 100).toFixed(2)}`;
}

export function CreditsClient({ initialToken }: { initialToken?: string }) {
  const token = useAdminToken(initialToken);
  const pageSize = useDefaultPageSize();
  const [tab, setTab] = useState<Tab>("grants");
  const [skip, setSkip] = useState(0);
  const [userInput, setUserInput] = useState("");
  const userFilter = useDebouncedValue(userInput.trim(), 250);
  const [grantUserId, setGrantUserId] = useState("");
  const [grantCredits, setGrantCredits] = useState("");
  const [grantReason, setGrantReason] = useState("");
  const [grantErr, setGrantErr] = useState("");
  const [grantOpen, setGrantOpen] = useState(false);
  const chrome = useAdminNavChrome(token);
  const grants = useAdminCreditGrants(token, userFilter, skip, pageSize);
  const topups = useAdminTopupLedger(token, userFilter, skip, pageSize);
  const creditMut = useCreditGrant(token);
  const { err: stepErr, requireStepUp, setErr: setStepErr } = useRequireStepUp(token);

  const title = chrome.data?.ADMIN_NAV_CREDITS?.trim() || "";
  const tabGrants = chrome.data?.ADMIN_CREDITS_TAB_GRANTS?.trim() || "";
  const tabTopups = chrome.data?.ADMIN_CREDITS_TAB_TOPUPS?.trim() || "";
  const filterUser = chrome.data?.ADMIN_CREDITS_FILTER_USER?.trim() || "";
  const filterUserDesc = chrome.data?.ADMIN_CREDITS_FILTER_USER_DESC?.trim() || "";
  const grantTitle = chrome.data?.ADMIN_CREDITS_GRANT_TITLE?.trim() || "";
  const grantUserLabel = chrome.data?.ADMIN_CREDITS_GRANT_USER?.trim() || "";
  const grantAmountLabel = chrome.data?.ADMIN_CREDITS_GRANT_AMOUNT?.trim() || "";
  const grantAction = chrome.data?.ADMIN_CREDITS_GRANT_ACTION?.trim() || "";
  const reasonRequired = chrome.data?.ADMIN_REASON_REQUIRED?.trim() || "";
  const pendingCreating = chrome.data?.ADMIN_PENDING_CREATING?.trim() || "";
  const colUser = chrome.data?.ADMIN_CREDITS_COL_USER?.trim() || "";
  const colCredits = chrome.data?.ADMIN_CREDITS_COL_CREDITS?.trim() || "";
  const colReason = chrome.data?.ADMIN_CREDITS_COL_REASON?.trim() || "";
  const colBy = chrome.data?.ADMIN_CREDITS_COL_BY?.trim() || "";
  const colWhen = chrome.data?.ADMIN_CREDITS_COL_WHEN?.trim() || "";
  const htmlLang = chrome.data?.SITE_HTML_LANG?.trim() || "";
  const colTopUser = chrome.data?.ADMIN_TOPUP_COL_USER?.trim() || "";
  const colTx = chrome.data?.ADMIN_TOPUP_COL_TX?.trim() || "";
  const colPrice = chrome.data?.ADMIN_TOPUP_COL_PRICE?.trim() || "";
  const colPlan = chrome.data?.ADMIN_TOPUP_COL_PLAN?.trim() || "";
  const colTopCredits = chrome.data?.ADMIN_TOPUP_COL_CREDITS?.trim() || "";
  const colQty = chrome.data?.ADMIN_TOPUP_COL_QTY?.trim() || "";
  const colTopWhen = chrome.data?.ADMIN_TOPUP_COL_WHEN?.trim() || "";
  const closeLabel =
    chrome.data?.ADMIN_DIALOG_CLOSE?.trim() || chrome.data?.ADMIN_CLOSE?.trim() || "";

  const active = tab === "grants" ? grants : topups;

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setSkip(0);
  }, [userFilter, tab]);

  const grantColumns = useMemo<ColumnDef<LedgerRow>[]>(
    () => [
      {
        id: "user_id",
        header: colUser,
        cell: ({ row }) => (
          <span className="font-mono text-xs">{String(row.original.user_id || "")}</span>
        ),
      },
      {
        id: "credits",
        header: colCredits,
        cell: ({ row }) => String(row.original.credits ?? ""),
      },
      {
        id: "reason",
        header: colReason,
        cell: ({ row }) => String(row.original.reason || ""),
      },
      {
        id: "created_by",
        header: colBy,
        cell: ({ row }) => (
          <span className="font-mono text-xs">{String(row.original.created_by || "")}</span>
        ),
      },
      {
        id: "created_at",
        header: colWhen,
        cell: ({ row }) => {
          const raw = String(row.original.created_at || "");
          return (
            <span title={formatDateTimeFull(raw, htmlLang) || undefined}>
              {formatDateTimeShort(raw, htmlLang)}
            </span>
          );
        },
      },
    ],
    [colUser, colCredits, colReason, colBy, colWhen, htmlLang],
  );

  const topupColumns = useMemo<ColumnDef<LedgerRow>[]>(
    () => [
      {
        id: "user_id",
        header: colTopUser,
        cell: ({ row }) => (
          <span className="font-mono text-xs">{String(row.original.user_id || "")}</span>
        ),
      },
      {
        id: "paddle_transaction_id",
        header: colTx,
        cell: ({ row }) => (
          <span className="font-mono text-xs">
            {String(row.original.paddle_transaction_id || "")}
          </span>
        ),
      },
      {
        id: "price_cents",
        header: colPrice,
        cell: ({ row }) => {
          const currency = typeof row.original.currency === "string" ? row.original.currency : "";
          return formatTopupMoney(row.original.price_cents, currency);
        },
      },
      {
        id: "plan_id",
        header: colPlan,
        cell: ({ row }) => String(row.original.plan_id || ""),
      },
      {
        id: "credits_granted",
        header: colTopCredits,
        cell: ({ row }) => String(row.original.credits_granted ?? ""),
      },
      {
        id: "quantity",
        header: colQty,
        cell: ({ row }) => String(row.original.quantity ?? ""),
      },
      {
        id: "created_at",
        header: colTopWhen,
        cell: ({ row }) => {
          const raw = String(row.original.created_at || "");
          return (
            <span title={formatDateTimeFull(raw, htmlLang) || undefined}>
              {formatDateTimeShort(raw, htmlLang)}
            </span>
          );
        },
      },
    ],
    [colTopUser, colTx, colPrice, colPlan, colTopCredits, colQty, colTopWhen, htmlLang],
  );

  const columns = tab === "grants" ? grantColumns : topupColumns;
  const skeletonCols =
    tab === "grants"
      ? [colUser, colCredits, colReason, colBy, colWhen]
      : [colTopUser, colTx, colPrice, colPlan, colTopCredits, colQty, colTopWhen];

  const bootReady = useListBootReady(pageSize, Boolean(chrome.data), queryContentReady(active));
  const showInitialSkeleton = !bootReady;

  if (showInitialSkeleton) {
    return (
      <div className="space-y-6">
        {title ? (
          <h1 className="text-xl font-semibold tracking-tight text-muted-foreground">{title}</h1>
        ) : null}
        <div className="flex flex-wrap gap-2">
          <Skeleton className="h-9 w-28" />
        </div>
        <TabsSkeleton count={2} />
        <DataTableSkeleton
          title=""
          columns={skeletonCols.map((label) => ({ label: label || "" }))}
          rows={pageSize > 0 ? pageSize : 8}
          showFilters
          filterCount={1}
        />
      </div>
    );
  }

  const items = ((active.data?.items as LedgerRow[]) ?? []).filter((row) =>
    Boolean(String(row.id || "")),
  );

  return (
    <div className="space-y-6">
      {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
      {stepErr || grantErr ? (
        <p className="text-sm text-destructive">{stepErr || grantErr}</p>
      ) : null}

      {grantTitle &&
      grantUserLabel &&
      grantAmountLabel &&
      grantAction &&
      reasonRequired &&
      pendingCreating ? (
        <Button type="button" onClick={() => setGrantOpen(true)}>
          {grantAction}
        </Button>
      ) : null}

      <div className="flex flex-wrap gap-2">
        {tabGrants ? (
          <Button
            type="button"
            size="sm"
            variant={tab === "grants" ? "default" : "outline"}
            onClick={() => {
              setTab("grants");
              setSkip(0);
            }}
          >
            {tabGrants}
          </Button>
        ) : null}
        {tabTopups ? (
          <Button
            type="button"
            size="sm"
            variant={tab === "topups" ? "default" : "outline"}
            onClick={() => {
              setTab("topups");
              setSkip(0);
            }}
          >
            {tabTopups}
          </Button>
        ) : null}
      </div>

      <AdminFilterBar
        filters={[
          {
            kind: "search",
            id: "credits-user-filter",
            label: filterUser,
            description: filterUserDesc,
            value: userInput,
            onChange: setUserInput,
          },
        ]}
      />

      <Card>
        <CardContent className="pt-6">
          <DataTable
            columns={columns}
            data={items}
            meta={active.data?.meta}
            onPage={setSkip}
            pageDisabled={active.isFetching}
            isFetching={
              (active.isFetching && !active.isPending) || (tab === "grants" && creditMut.isPending)
            }
            getRowId={(row) => String(row.id || "")}
            bordered={false}
          />
        </CardContent>
      </Card>

      {grantTitle &&
      grantUserLabel &&
      grantAmountLabel &&
      grantAction &&
      reasonRequired &&
      pendingCreating ? (
        <Dialog
          open={grantOpen}
          onOpenChange={(open) => {
            if (!open) setGrantOpen(false);
            else setGrantOpen(true);
          }}
        >
          <DialogContent closeLabel={closeLabel} className="max-w-2xl">
            <DialogHeader>
              <DialogTitle>{grantTitle}</DialogTitle>
            </DialogHeader>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field
                id="credits-grant-user"
                label={grantUserLabel}
                {...chromeFieldDesc(chrome.data, "ADMIN_CREDITS_USER_ID_DESC")}
              >
                <Input
                  id="credits-grant-user"
                  value={grantUserId}
                  onChange={(e) => setGrantUserId(e.target.value)}
                  placeholder={grantUserLabel}
                  className="font-mono text-xs"
                  autoComplete="off"
                />
              </Field>
              <Field
                id="credits-grant-amount"
                label={grantAmountLabel}
                {...chromeFieldDesc(chrome.data, "ADMIN_CREDITS_AMOUNT_DESC")}
              >
                <Input
                  id="credits-grant-amount"
                  value={grantCredits}
                  onChange={(e) => setGrantCredits(e.target.value)}
                  placeholder={grantAmountLabel}
                  autoComplete="off"
                />
              </Field>
              <Field
                id="credits-grant-reason"
                label={reasonRequired}
                {...chromeFieldDesc(chrome.data, "ADMIN_CREDITS_REASON_DESC")}
                className="sm:col-span-2"
              >
                <Textarea
                  id="credits-grant-reason"
                  value={grantReason}
                  onChange={(e) => setGrantReason(e.target.value)}
                  placeholder={reasonRequired}
                />
              </Field>
            </div>
            <DialogFooter>
              <Button
                type="button"
                isLoading={creditMut.isPending}
                pendingLabel={pendingCreating}
                onClick={() => {
                  setGrantErr("");
                  setStepErr("");
                  if (!requireStepUp()) return;
                  const credits = Number(grantCredits);
                  if (!grantUserId.trim() || !Number.isFinite(credits) || !grantReason.trim()) {
                    setGrantErr(reasonRequired);
                    return;
                  }
                  void creditMut
                    .mutateAsync({
                      user_id: grantUserId.trim(),
                      credits,
                      reason: grantReason.trim(),
                    })
                    .then(() => {
                      setGrantUserId("");
                      setGrantCredits("");
                      setGrantReason("");
                      setGrantOpen(false);
                      setTab("grants");
                      setSkip(0);
                    })
                    .catch((e) => {
                      setGrantErr(e instanceof Error ? e.message : "");
                    });
                }}
              >
                {grantAction}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      ) : null}
    </div>
  );
}
