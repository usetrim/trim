"use client";

import { AdminFilterBar } from "@/components/admin/admin-filter-bar";
import { DataTableSkeleton } from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
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
import { useCreatePlan, usePatchPlan, useSyncPaddleCatalog } from "@/hooks/mutations/billing";
import { useAdminBillingSettings, useAdminPlans } from "@/hooks/queries/billing";
import { useAdminNavChrome } from "@/hooks/queries/chrome";
import { useDeferredDialogSelection } from "@/hooks/use-deferred-dialog-selection";
import { useAdminToken } from "@/hooks/use-admin-token";
import { useDebouncedValue } from "@/hooks/use-debounced-value";
import { useDefaultPageSize } from "@/hooks/use-default-page-size";
import { queryContentReady, useListBootReady } from "@/hooks/use-list-boot-ready";
import { useRequireStepUp } from "@/hooks/use-require-step-up";
import { chromeFieldDesc } from "@/lib/admin-field-desc";
import { useEffect, useMemo, useState } from "react";

function planKindLabel(kind: string, chrome: Record<string, string | undefined>): string {
  const key = `ADMIN_PLAN_KIND_${kind.trim().toUpperCase()}`;
  return chrome[key]?.trim() || "";
}

function planKindOptions(chrome: Record<string, string | undefined>) {
  return (
    [
      ["subscription", chrome.ADMIN_PLAN_KIND_SUBSCRIPTION],
      ["topup", chrome.ADMIN_PLAN_KIND_TOPUP],
      ["enterprise", chrome.ADMIN_PLAN_KIND_ENTERPRISE],
    ] as const
  ).filter(([, label]) => Boolean(label?.trim()));
}

/** Present API cents as major currency units (fail closed without currency). */
function formatPlanMoney(cents: unknown, currency: string): string {
  const code = currency.trim().toUpperCase();
  if (!code) return "";
  const n =
    typeof cents === "number"
      ? cents
      : typeof cents === "string" && cents.trim() !== ""
        ? Number(cents)
        : Number.NaN;
  if (!Number.isFinite(n)) return "";
  return `${code} ${(n / 100).toFixed(2)}`;
}

/** Load API cents into major-unit input (empty when missing). */
function centsToMajorInput(cents: unknown): string {
  if (typeof cents !== "number" || !Number.isFinite(cents)) return "";
  return (cents / 100).toFixed(2);
}

/** Parse major-unit input to API cents. */
function majorInputToCents(raw: string): number | null {
  const t = raw.trim();
  if (!t) return null;
  const n = Number(t);
  if (!Number.isFinite(n)) return null;
  return Math.round(n * 100);
}

export function PlansClient({ initialToken }: { initialToken?: string }) {
  const token = useAdminToken(initialToken);
  const pageSize = useDefaultPageSize();
  const [skip, setSkip] = useState(0);
  const [qInput, setQInput] = useState("");
  const q = useDebouncedValue(qInput.trim(), 250);
  const chrome = useAdminNavChrome(token);
  const plans = useAdminPlans(token, skip, pageSize, q);
  const billing = useAdminBillingSettings(token);
  const patch = usePatchPlan(token);
  const createPlan = useCreatePlan(token);
  const syncCatalog = useSyncPaddleCatalog(token);
  const { err: stepErr, requireStepUp } = useRequireStepUp(token);
  const {
    open: editOpen,
    selectedId,
    openWith: openEdit,
    close: closeEdit,
    onOpenChange: onEditOpenChange,
  } = useDeferredDialogSelection();
  const [displayName, setDisplayName] = useState("");
  const [description, setDescription] = useState("");
  const [planRank, setPlanRank] = useState("");
  const [sortOrder, setSortOrder] = useState("");
  const [monthly, setMonthly] = useState("");
  const [yearly, setYearly] = useState("");
  const [creditsMonthly, setCreditsMonthly] = useState("");
  const [perSeat, setPerSeat] = useState(false);
  const [unlimited, setUnlimited] = useState(false);
  const [popular, setPopular] = useState(false);
  const [featuresText, setFeaturesText] = useState("");
  const [isActive, setIsActive] = useState(false);
  const [isPublic, setIsPublic] = useState(false);
  const [actionErr, setActionErr] = useState("");
  const [newId, setNewId] = useState("");
  const [newName, setNewName] = useState("");
  const [newKind, setNewKind] = useState("");
  const [newMonthly, setNewMonthly] = useState("");
  const [newYearly, setNewYearly] = useState("");
  const [newCredits, setNewCredits] = useState("");
  const [newPerSeat, setNewPerSeat] = useState(false);
  const [newUnlimited, setNewUnlimited] = useState(false);
  const [newPopular, setNewPopular] = useState(false);
  const [newRank, setNewRank] = useState("");
  const [newSort, setNewSort] = useState("");
  const [newDesc, setNewDesc] = useState("");
  const [newActive, setNewActive] = useState(false);
  const [newPublic, setNewPublic] = useState(false);
  const [newSyncPaddle, setNewSyncPaddle] = useState(false);
  const [editSyncPaddle, setEditSyncPaddle] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);

  const labels = chrome.data ?? {};
  const title = labels.ADMIN_NAV_PLANS?.trim() || "";
  const pendingUpdating =
    labels.ADMIN_PENDING_UPDATING?.trim() || labels.ADMIN_PENDING_SAVING?.trim() || "";
  const pendingCreating = labels.ADMIN_PENDING_CREATING?.trim() || "";
  const filterSearch = labels.ADMIN_FILTER_SEARCH?.trim() || "";
  const filterSearchDesc =
    labels.ADMIN_PLANS_SEARCH_DESC?.trim() || labels.ADMIN_FILTER_SEARCH_DESC?.trim() || "";
  const saveLabel = labels.ADMIN_ACTION_SAVE?.trim() || "";
  const syncLabel = labels.ADMIN_PLAN_SYNC_PADDLE?.trim() || "";
  const syncPending = labels.ADMIN_PLAN_SYNC_PENDING?.trim() || "";
  const createLabel = labels.ADMIN_PLAN_CREATE?.trim() || "";
  const editSectionLabel =
    labels.ADMIN_PLAN_EDIT?.trim() || labels.ADMIN_SECTION_EDIT?.trim() || "";
  const createSectionLabel = labels.ADMIN_SECTION_CREATE?.trim() || "";
  const planIdLabel = labels.ADMIN_PLAN_ID?.trim() || "";
  const amountMonthlyLabel = labels.ADMIN_PLAN_AMOUNT_MONTHLY?.trim() || "";
  const amountYearlyLabel = labels.ADMIN_PLAN_AMOUNT_YEARLY?.trim() || "";
  const creditsLabel = labels.ADMIN_PLAN_CREDITS?.trim() || "";
  const perSeatLabel = labels.ADMIN_PLAN_PER_SEAT?.trim() || "";
  const unlimitedLabel = labels.ADMIN_PLAN_UNLIMITED?.trim() || "";
  const unlimitedColLabel = labels.ADMIN_PLAN_UNLIMITED_COL?.trim() || unlimitedLabel;
  const popularLabel = labels.ADMIN_PLAN_POPULAR?.trim() || "";
  const popularColLabel = labels.ADMIN_PLAN_POPULAR_COL?.trim() || popularLabel;
  const paddleIdsLabel = labels.ADMIN_PLAN_PADDLE_IDS?.trim() || "";
  const productIdLabel = labels.ADMIN_PLAN_PRODUCT_ID?.trim() || "";
  const activeLabel = labels.ADMIN_PLAN_ACTIVE?.trim() || "";
  const publicLabel = labels.ADMIN_PLAN_PUBLIC?.trim() || "";
  const descLabel = labels.ADMIN_PLAN_DESCRIPTION?.trim() || "";
  const rankLabel = labels.ADMIN_PLAN_RANK?.trim() || "";
  const sortLabel = labels.ADMIN_PLAN_SORT_ORDER?.trim() || "";
  const kindLabel = labels.ADMIN_PLAN_KIND?.trim() || "";
  const priMonthlyLabel = labels.ADMIN_PLAN_PRICE_MONTHLY?.trim() || "";
  const priYearlyLabel = labels.ADMIN_PLAN_PRICE_YEARLY?.trim() || "";
  const priTopupLabel = labels.ADMIN_PLAN_PRICE_TOPUP?.trim() || "";
  const featuresLabel = labels.ADMIN_PLAN_FEATURES?.trim() || "";
  const unboundLabel = labels.ADMIN_PRICING_UNBOUND?.trim() || "";
  const pricingBound = Boolean(billing.data?.pricing_bound);
  const currency = String(billing.data?.default_currency || "").trim();
  const colName = labels.ADMIN_PLAN_COL_NAME?.trim() || "";
  const colKind = labels.ADMIN_PLAN_COL_KIND?.trim() || "";
  const viewLabel = labels.ADMIN_ACTION_VIEW?.trim() || "";
  const rowActionsLabel = labels.ADMIN_TABLE_ROW_ACTIONS?.trim() || "";
  const closeLabel = labels.ADMIN_DIALOG_CLOSE?.trim() || labels.ADMIN_CLOSE?.trim() || "";
  const clearSelLabel = labels.ADMIN_TABLE_CLEAR_SELECTION?.trim() || closeLabel;

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setSkip(0);
  }, [q]);

  const items = plans.data?.items || [];
  const selected = items.find((p) => String(p.id || "") === selectedId);
  const selectedKind = typeof selected?.plan_kind === "string" ? selected.plan_kind : "";
  const selectedKindLabel = planKindLabel(selectedKind, labels);
  const productId = String(selected?.paddle_product_id || "");
  const priMonthly = String(selected?.paddle_price_id_monthly || "");
  const priYearly = String(selected?.paddle_price_id_yearly || "");
  const priTopup = String(selected?.paddle_price_id_topup || "");

  const planColumns = useMemo<ColumnDef<Record<string, unknown>>[]>(
    () => [
      {
        id: "display_name",
        header: colName,
        cell: ({ row }) => {
          const id = String(row.original.id || "");
          const name = String(row.original.display_name || "");
          if (!id || !name) return "";
          return (
            <button
              type="button"
              className="text-left text-foreground hover:underline"
              onClick={() => openEdit(id)}
            >
              {name}
            </button>
          );
        },
      },
      {
        id: "plan_kind",
        header: colKind,
        cell: ({ row }) => {
          const kind = typeof row.original.plan_kind === "string" ? row.original.plan_kind : "";
          return planKindLabel(kind, labels);
        },
      },
      {
        id: "price_monthly_cents",
        header: amountMonthlyLabel,
        cell: ({ row }) => formatPlanMoney(row.original.price_monthly_cents, currency),
      },
      {
        id: "credits_monthly",
        header: creditsLabel,
        cell: ({ row }) => {
          const n = row.original.credits_monthly;
          return typeof n === "number" && Number.isFinite(n) ? String(n) : "";
        },
      },
      {
        id: "unlimited",
        header: unlimitedColLabel,
        cell: ({ row }) => (row.original.unlimited ? "✓" : ""),
      },
      {
        id: "popular",
        header: popularColLabel,
        cell: ({ row }) => (row.original.popular ? "✓" : ""),
      },
      {
        id: "price_yearly_cents",
        header: amountYearlyLabel,
        cell: ({ row }) => formatPlanMoney(row.original.price_yearly_cents, currency),
      },
      {
        id: "_actions",
        header: "",
        cell: ({ row }) => {
          const id = String(row.original.id || "");
          if (!id || !viewLabel || !rowActionsLabel) return null;
          return (
            <DataTableRowActions
              triggerLabel={rowActionsLabel}
              actions={[
                {
                  id: "view",
                  label: viewLabel,
                  onSelect: () => openEdit(id),
                },
              ]}
            />
          );
        },
      },
    ],
    [
      colName,
      colKind,
      amountMonthlyLabel,
      amountYearlyLabel,
      creditsLabel,
      unlimitedColLabel,
      popularColLabel,
      currency,
      labels,
      viewLabel,
      rowActionsLabel,
      openEdit,
    ],
  );

  // Sync-to-Paddle is a per-save action flag (not stored on the plan). Reset only when
  // opening a different plan - never on list refetch after PATCH, or the checkbox flickers off.
  // biome-ignore lint/correctness/useExhaustiveDependencies: reset ephemeral flag when selection changes
  useEffect(() => {
    setEditSyncPaddle(false);
  }, [selectedId]);

  // Seed persisted plan fields only when the operator opens a row. List refetch must not
  // overwrite in-progress edits (Unlimited, prices, etc.).
  // biome-ignore lint/correctness/useExhaustiveDependencies: items read at selection time only
  useEffect(() => {
    if (!selectedId) return;
    const plan = items.find((p) => String(p.id || "") === selectedId);
    if (!plan) return;
    setDisplayName(String(plan.display_name || ""));
    setDescription(String(plan.description || ""));
    setPlanRank(plan.plan_rank != null ? String(plan.plan_rank) : "");
    setSortOrder(plan.sort_order != null ? String(plan.sort_order) : "");
    setMonthly(centsToMajorInput(plan.price_monthly_cents));
    setYearly(centsToMajorInput(plan.price_yearly_cents));
    setCreditsMonthly(
      typeof plan.credits_monthly === "number" && Number.isFinite(plan.credits_monthly)
        ? String(plan.credits_monthly)
        : "",
    );
    setPerSeat(Boolean(plan.per_seat));
    setUnlimited(Boolean(plan.unlimited));
    setPopular(Boolean(plan.popular));
    setFeaturesText(plan.features != null ? JSON.stringify(plan.features, null, 2) : "");
    setIsActive(Boolean(plan.is_active));
    setIsPublic(Boolean(plan.is_public));
  }, [selectedId]);

  // Billing settings are for dialog defaults only - never block the whole page on them.
  const bootReady = useListBootReady(pageSize, Boolean(chrome.data), queryContentReady(plans));
  const showInitialSkeleton = !bootReady;

  if (showInitialSkeleton) {
    return (
      <div className="space-y-4">
        {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
        <DataTableSkeleton
          title=""
          columns={[colName, colKind, amountMonthlyLabel, creditsLabel, amountYearlyLabel].map(
            (label) => ({
              label: label || "",
            }),
          )}
          rows={pageSize > 0 ? pageSize : 8}
          filterCount={1}
        />
      </div>
    );
  }

  return (
    <div className="space-y-4" data-ui-rev="plans-20260930c">
      <div className="flex flex-wrap items-center justify-between gap-3">
        {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : <span />}
        {(syncLabel && syncPending) || (createLabel && pendingCreating && planIdLabel) ? (
          <div
            role="group"
            aria-label={`${syncLabel} ${createLabel}`.trim()}
            data-plans-toolbar="sync-create"
            data-ui-rev="plans-toolbar-20260930c"
            style={{
              display: "flex",
              flexDirection: "row",
              flexWrap: "wrap",
              alignItems: "center",
              gap: 16,
            }}
          >
            {syncLabel && syncPending ? (
              <Button
                type="button"
                className="shrink-0"
                isLoading={syncCatalog.isPending}
                pendingLabel={syncPending}
                onClick={() => {
                  if (!requireStepUp()) return;
                  setActionErr("");
                  void syncCatalog.mutateAsync().catch((e: unknown) => {
                    setActionErr(e instanceof Error ? e.message : "");
                  });
                }}
              >
                {syncLabel}
              </Button>
            ) : null}
            {syncLabel && syncPending && createLabel && pendingCreating && planIdLabel ? (
              <span
                aria-hidden
                data-plans-gap
                style={{
                  display: "inline-block",
                  width: 16,
                  minWidth: 16,
                  height: 8,
                  flex: "0 0 16px",
                }}
              />
            ) : null}
            {createLabel && pendingCreating && planIdLabel ? (
              <Button
                type="button"
                variant="outline"
                className="shrink-0"
                onClick={() => setCreateOpen(true)}
              >
                {createLabel}
              </Button>
            ) : null}
          </div>
        ) : null}
      </div>
      {stepErr ? <p className="text-sm text-destructive">{stepErr}</p> : null}
      {actionErr ? <p className="text-sm text-destructive">{actionErr}</p> : null}
      {!pricingBound && unboundLabel ? (
        <p className="rounded-md border border-border bg-card px-4 py-3 text-sm text-foreground">
          {unboundLabel}
        </p>
      ) : null}
      <AdminFilterBar
        filters={[
          {
            kind: "search",
            id: "plans-search",
            label: filterSearch,
            description: filterSearchDesc,
            value: qInput,
            onChange: setQInput,
          },
        ]}
      />
      <Card className="bg-background">
        <CardContent className="pt-6">
          <DataTable
            columns={planColumns}
            data={(items as Array<Record<string, unknown>>).filter(
              (plan) => Boolean(String(plan.id || "")) && Boolean(String(plan.display_name || "")),
            )}
            meta={plans.data?.meta}
            onPage={setSkip}
            pageDisabled={plans.isFetching}
            isFetching={plans.isFetching && !plans.isPending}
            getRowId={(row) => String(row.id)}
            bordered={false}
          />
        </CardContent>
      </Card>
      {createLabel && pendingCreating && planIdLabel ? (
        <Dialog
          open={createOpen}
          onOpenChange={(open) => {
            if (!open) setCreateOpen(false);
            else setCreateOpen(true);
          }}
        >
          <DialogContent closeLabel={closeLabel} className="max-w-3xl">
            <DialogHeader>
              {createSectionLabel ? <DialogTitle>{createSectionLabel}</DialogTitle> : null}
            </DialogHeader>
            <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
              <Field
                id="plan-new-id"
                label={planIdLabel}
                {...chromeFieldDesc(labels, "ADMIN_PLAN_ID_DESC")}
              >
                <Input
                  id="plan-new-id"
                  value={newId}
                  onChange={(e) => setNewId(e.target.value)}
                  placeholder={planIdLabel}
                  className="font-mono text-xs"
                />
              </Field>
              {colName ? (
                <Field
                  id="plan-new-name"
                  label={colName}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_NAME_DESC")}
                >
                  <Input
                    id="plan-new-name"
                    value={newName}
                    onChange={(e) => setNewName(e.target.value)}
                    placeholder={colName}
                  />
                </Field>
              ) : null}
              {kindLabel && planKindOptions(labels).length > 0 ? (
                <Field
                  id="plan-new-kind"
                  label={kindLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_KIND_DESC")}
                >
                  <Select value={newKind || undefined} onValueChange={setNewKind}>
                    <SelectTrigger id="plan-new-kind" aria-label={kindLabel}>
                      <SelectValue placeholder={kindLabel} />
                    </SelectTrigger>
                    <SelectContent>
                      {planKindOptions(labels).map(([value, label]) => (
                        <SelectItem key={value} value={value}>
                          {label?.trim()}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </Field>
              ) : null}
              {amountMonthlyLabel ? (
                <Field
                  id="plan-new-monthly"
                  label={amountMonthlyLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_PRICE_MONTHLY_DESC")}
                >
                  <Input
                    id="plan-new-monthly"
                    value={newMonthly}
                    onChange={(e) => setNewMonthly(e.target.value)}
                    placeholder={amountMonthlyLabel}
                  />
                </Field>
              ) : null}
              {amountYearlyLabel ? (
                <Field
                  id="plan-new-yearly"
                  label={amountYearlyLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_PRICE_YEARLY_DESC")}
                >
                  <Input
                    id="plan-new-yearly"
                    value={newYearly}
                    onChange={(e) => setNewYearly(e.target.value)}
                    placeholder={amountYearlyLabel}
                  />
                </Field>
              ) : null}
              {creditsLabel ? (
                <Field
                  id="plan-new-credits"
                  label={creditsLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_CREDITS_DESC")}
                >
                  <Input
                    id="plan-new-credits"
                    value={newCredits}
                    onChange={(e) => setNewCredits(e.target.value)}
                    placeholder={creditsLabel}
                    inputMode="numeric"
                  />
                </Field>
              ) : null}
              {perSeatLabel ? (
                <Field
                  id="plan-new-per-seat"
                  label={perSeatLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_PER_SEAT_DESC")}
                >
                  <div className="flex items-center gap-2 pt-1">
                    <Checkbox
                      id="plan-new-per-seat"
                      checked={newPerSeat}
                      onCheckedChange={(v) => setNewPerSeat(v === true)}
                    />
                  </div>
                </Field>
              ) : null}
              {unlimitedLabel ? (
                <Field
                  id="plan-new-unlimited"
                  label={unlimitedLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_UNLIMITED_DESC")}
                >
                  <div className="flex items-center gap-2 pt-1">
                    <Checkbox
                      id="plan-new-unlimited"
                      checked={newUnlimited}
                      onCheckedChange={(v) => setNewUnlimited(v === true)}
                    />
                  </div>
                </Field>
              ) : null}
              {popularLabel ? (
                <Field
                  id="plan-new-popular"
                  label={popularLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_POPULAR_DESC")}
                >
                  <div className="flex items-center gap-2 pt-1">
                    <Checkbox
                      id="plan-new-popular"
                      checked={newPopular}
                      onCheckedChange={(v) => setNewPopular(v === true)}
                    />
                  </div>
                </Field>
              ) : null}
              {descLabel ? (
                <Field
                  id="plan-new-desc"
                  label={descLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_DESCRIPTION_DESC")}
                  className="sm:col-span-2 xl:col-span-3"
                >
                  <Textarea
                    id="plan-new-desc"
                    value={newDesc}
                    onChange={(e) => setNewDesc(e.target.value)}
                    placeholder={descLabel}
                  />
                </Field>
              ) : null}
              {rankLabel ? (
                <Field
                  id="plan-new-rank"
                  label={rankLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_RANK_DESC")}
                >
                  <Input
                    id="plan-new-rank"
                    value={newRank}
                    onChange={(e) => setNewRank(e.target.value)}
                    placeholder={rankLabel}
                  />
                </Field>
              ) : null}
              {sortLabel ? (
                <Field
                  id="plan-new-sort"
                  label={sortLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_SORT_ORDER_DESC")}
                >
                  <Input
                    id="plan-new-sort"
                    value={newSort}
                    onChange={(e) => setNewSort(e.target.value)}
                    placeholder={sortLabel}
                  />
                </Field>
              ) : null}
              {activeLabel ? (
                <Field
                  id="plan-new-active"
                  label={activeLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_ACTIVE_DESC")}
                >
                  <div className="flex items-center gap-2 pt-1">
                    <Checkbox
                      id="plan-new-active"
                      checked={newActive}
                      onCheckedChange={(v) => setNewActive(v === true)}
                    />
                  </div>
                </Field>
              ) : null}
              {publicLabel ? (
                <Field
                  id="plan-new-public"
                  label={publicLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_PUBLIC_DESC")}
                >
                  <div className="flex items-center gap-2 pt-1">
                    <Checkbox
                      id="plan-new-public"
                      checked={newPublic}
                      onCheckedChange={(v) => setNewPublic(v === true)}
                    />
                  </div>
                </Field>
              ) : null}
              {syncLabel ? (
                <Field
                  id="plan-new-sync"
                  label={syncLabel}
                  descriptionWrap
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_SYNC_PADDLE_DESC")}
                >
                  <div className="flex items-center gap-2 pt-1">
                    <Checkbox
                      id="plan-new-sync"
                      checked={newSyncPaddle}
                      onCheckedChange={(v) => setNewSyncPaddle(v === true)}
                    />
                  </div>
                </Field>
              ) : null}
            </div>
            <DialogFooter>
              <Button
                type="button"
                isLoading={createPlan.isPending}
                pendingLabel={pendingCreating}
                onClick={() => {
                  if (!requireStepUp()) return;
                  if (!newId.trim() || !newName.trim() || !newKind.trim()) return;
                  if (rankLabel && !newRank.trim()) return;
                  if (sortLabel && !newSort.trim()) return;
                  setActionErr("");
                  void createPlan
                    .mutateAsync({
                      id: newId.trim(),
                      display_name: newName.trim(),
                      description: newDesc.trim(),
                      plan_kind: newKind,
                      plan_rank: newRank ? Number(newRank) : undefined,
                      price_monthly_cents: majorInputToCents(newMonthly),
                      price_yearly_cents: majorInputToCents(newYearly),
                      credits_monthly: newCredits.trim() ? Number(newCredits) : 0,
                      per_seat: newPerSeat,
                      unlimited: newUnlimited,
                      popular: newPopular,
                      is_public: newPublic,
                      is_active: newActive,
                      sort_order: newSort ? Number(newSort) : undefined,
                      sync_to_paddle: newSyncPaddle,
                    })
                    .then(() => {
                      setCreateOpen(false);
                      setNewId("");
                      setNewName("");
                      setNewKind("");
                      setNewMonthly("");
                      setNewYearly("");
                      setNewCredits("");
                      setNewPerSeat(false);
                      setNewUnlimited(false);
                      setNewPopular(false);
                      setNewRank("");
                      setNewSort("");
                      setNewDesc("");
                      setNewActive(false);
                      setNewPublic(false);
                      setNewSyncPaddle(false);
                    })
                    .catch((e: unknown) => {
                      setActionErr(e instanceof Error ? e.message : "");
                    });
                }}
              >
                {createLabel}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      ) : null}
      {selectedId ? (
        <Dialog open={editOpen} onOpenChange={onEditOpenChange}>
          <DialogContent closeLabel={closeLabel} className="max-w-3xl">
            <DialogHeader>
              {editSectionLabel ? <DialogTitle>{editSectionLabel}</DialogTitle> : null}
            </DialogHeader>
            <div className="grid gap-4 sm:grid-cols-2">
              {colName ? (
                <Field
                  id="plan-display-name"
                  label={colName}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_NAME_DESC")}
                >
                  <Input
                    id="plan-display-name"
                    value={displayName}
                    onChange={(e) => setDisplayName(e.target.value)}
                    placeholder={colName}
                  />
                </Field>
              ) : null}
              {kindLabel && selectedKindLabel ? (
                <p className="self-end pb-7 text-sm text-muted-foreground">
                  {kindLabel}: {selectedKindLabel}
                </p>
              ) : null}
              {descLabel ? (
                <Field
                  id="plan-description"
                  label={descLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_DESCRIPTION_DESC")}
                  className="sm:col-span-2"
                >
                  <Textarea
                    id="plan-description"
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                    placeholder={descLabel}
                  />
                </Field>
              ) : null}
              {rankLabel ? (
                <Field
                  id="plan-rank"
                  label={rankLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_RANK_DESC")}
                >
                  <Input
                    id="plan-rank"
                    value={planRank}
                    onChange={(e) => setPlanRank(e.target.value)}
                    placeholder={rankLabel}
                  />
                </Field>
              ) : null}
              {sortLabel ? (
                <Field
                  id="plan-sort"
                  label={sortLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_SORT_ORDER_DESC")}
                >
                  <Input
                    id="plan-sort"
                    value={sortOrder}
                    onChange={(e) => setSortOrder(e.target.value)}
                    placeholder={sortLabel}
                  />
                </Field>
              ) : null}
              {amountMonthlyLabel ? (
                <Field
                  id="plan-monthly"
                  label={amountMonthlyLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_PRICE_MONTHLY_DESC")}
                >
                  <Input
                    id="plan-monthly"
                    value={monthly}
                    onChange={(e) => setMonthly(e.target.value)}
                    placeholder={amountMonthlyLabel}
                  />
                </Field>
              ) : null}
              {amountYearlyLabel ? (
                <Field
                  id="plan-yearly"
                  label={amountYearlyLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_PRICE_YEARLY_DESC")}
                >
                  <Input
                    id="plan-yearly"
                    value={yearly}
                    onChange={(e) => setYearly(e.target.value)}
                    placeholder={amountYearlyLabel}
                  />
                </Field>
              ) : null}
              {creditsLabel ? (
                <Field
                  id="plan-credits"
                  label={creditsLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_CREDITS_DESC")}
                >
                  <Input
                    id="plan-credits"
                    value={creditsMonthly}
                    onChange={(e) => setCreditsMonthly(e.target.value)}
                    placeholder={creditsLabel}
                    inputMode="numeric"
                  />
                </Field>
              ) : null}
              {perSeatLabel ? (
                <Field
                  id="plan-per-seat"
                  label={perSeatLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_PER_SEAT_DESC")}
                >
                  <div className="flex min-h-10 items-center gap-2">
                    <Checkbox
                      id="plan-per-seat"
                      checked={perSeat}
                      onCheckedChange={(v) => setPerSeat(v === true)}
                    />
                  </div>
                </Field>
              ) : null}
              {unlimitedLabel ? (
                <Field
                  id="plan-unlimited"
                  label={unlimitedLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_UNLIMITED_DESC")}
                >
                  <div className="flex min-h-10 items-center gap-2">
                    <Checkbox
                      id="plan-unlimited"
                      checked={unlimited}
                      onCheckedChange={(v) => setUnlimited(v === true)}
                    />
                  </div>
                </Field>
              ) : null}
              {popularLabel ? (
                <Field
                  id="plan-popular"
                  label={popularLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_POPULAR_DESC")}
                >
                  <div className="flex min-h-10 items-center gap-2">
                    <Checkbox
                      id="plan-popular"
                      checked={popular}
                      onCheckedChange={(v) => setPopular(v === true)}
                    />
                  </div>
                </Field>
              ) : null}
              {paddleIdsLabel ? (
                <div className="space-y-1 rounded-md border border-border bg-background p-3 text-xs text-muted-foreground sm:col-span-2">
                  <p className="font-medium text-foreground">{paddleIdsLabel}</p>
                  {productIdLabel && productId ? (
                    <p className="break-all font-mono">
                      {productIdLabel}: {productId}
                    </p>
                  ) : null}
                  {priMonthlyLabel && priMonthly ? (
                    <p className="break-all font-mono">
                      {priMonthlyLabel}: {priMonthly}
                    </p>
                  ) : null}
                  {priYearlyLabel && priYearly ? (
                    <p className="break-all font-mono">
                      {priYearlyLabel}: {priYearly}
                    </p>
                  ) : null}
                  {priTopupLabel && priTopup ? (
                    <p className="break-all font-mono">
                      {priTopupLabel}: {priTopup}
                    </p>
                  ) : null}
                </div>
              ) : null}
              {featuresLabel ? (
                <Field
                  id="plan-features"
                  label={featuresLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_FEATURES_DESC")}
                  className="sm:col-span-2"
                >
                  <Textarea
                    id="plan-features"
                    value={featuresText}
                    onChange={(e) => setFeaturesText(e.target.value)}
                    placeholder={featuresLabel}
                    className="font-mono text-xs"
                  />
                </Field>
              ) : null}
              {activeLabel ? (
                <Field
                  id="plan-active"
                  label={activeLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_ACTIVE_DESC")}
                >
                  <div className="flex min-h-10 items-center gap-2">
                    <Checkbox
                      id="plan-active"
                      checked={isActive}
                      onCheckedChange={(v) => setIsActive(v === true)}
                    />
                  </div>
                </Field>
              ) : null}
              {publicLabel ? (
                <Field
                  id="plan-public"
                  label={publicLabel}
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_PUBLIC_DESC")}
                >
                  <div className="flex min-h-10 items-center gap-2">
                    <Checkbox
                      id="plan-public"
                      checked={isPublic}
                      onCheckedChange={(v) => setIsPublic(v === true)}
                    />
                  </div>
                </Field>
              ) : null}
              {syncLabel ? (
                <Field
                  id="plan-edit-sync"
                  label={syncLabel}
                  descriptionWrap
                  {...chromeFieldDesc(labels, "ADMIN_PLAN_SYNC_PADDLE_DESC")}
                >
                  <div className="flex min-h-10 items-center gap-2">
                    <Checkbox
                      id="plan-edit-sync"
                      checked={editSyncPaddle}
                      onCheckedChange={(v) => setEditSyncPaddle(v === true)}
                    />
                  </div>
                </Field>
              ) : null}
            </div>
            <DialogFooter>
              {clearSelLabel ? (
                <Button type="button" variant="outline" onClick={closeEdit}>
                  {clearSelLabel}
                </Button>
              ) : closeLabel ? (
                <Button type="button" variant="outline" onClick={closeEdit}>
                  {closeLabel}
                </Button>
              ) : null}
              {saveLabel ? (
                <Button
                  type="button"
                  isLoading={patch.isPending}
                  pendingLabel={pendingUpdating || saveLabel}
                  onClick={() => {
                    if (!requireStepUp()) return;
                    setActionErr("");
                    let features: unknown = undefined;
                    if (featuresLabel && featuresText.trim()) {
                      try {
                        features = JSON.parse(featuresText) as unknown;
                      } catch {
                        return;
                      }
                    }
                    void patch
                      .mutateAsync({
                        id: selectedId,
                        body: {
                          display_name: displayName,
                          description,
                          plan_rank: planRank ? Number(planRank) : undefined,
                          sort_order: sortOrder ? Number(sortOrder) : undefined,
                          price_monthly_cents: majorInputToCents(monthly) ?? 0,
                          price_yearly_cents: majorInputToCents(yearly) ?? 0,
                          credits_monthly: creditsMonthly.trim()
                            ? Number(creditsMonthly)
                            : undefined,
                          per_seat: perSeat,
                          unlimited,
                          popular,
                          features,
                          is_active: isActive,
                          is_public: isPublic,
                          sync_to_paddle: editSyncPaddle,
                        },
                      })
                      .then(() => {
                        closeEdit();
                      })
                      .catch((e: unknown) => {
                        setActionErr(e instanceof Error ? e.message : "");
                      });
                  }}
                >
                  {saveLabel}
                </Button>
              ) : null}
            </DialogFooter>
          </DialogContent>
        </Dialog>
      ) : null}
    </div>
  );
}
