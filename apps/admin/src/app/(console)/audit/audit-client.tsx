"use client";

import { AdminFilterBar } from "@/components/admin/admin-filter-bar";
import { DataTableSkeleton } from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { type ColumnDef, DataTable } from "@/components/ui/data-table";
import type { DateRangeValue } from "@/components/ui/date-range-picker";
import { Skeleton } from "@/components/ui/skeleton";
import { useExportAudit } from "@/hooks/mutations/audit";
import { useAdminAudit, useAdminAuditFilterOptions } from "@/hooks/queries/audit";
import { useAdminBillingSettings } from "@/hooks/queries/billing";
import { useAdminNavChrome } from "@/hooks/queries/chrome";
import { useAdminToken } from "@/hooks/use-admin-token";
import { useDefaultPageSize } from "@/hooks/use-default-page-size";
import { queryContentReady, useListBootReady } from "@/hooks/use-list-boot-ready";
import { formatDateTimeFull, formatDateTimeShort } from "@/lib/format-datetime";
import { useEffect, useMemo, useState } from "react";

type AuditRow = Record<string, unknown>;

export function AuditClient({ initialToken }: { initialToken?: string }) {
  const token = useAdminToken(initialToken);
  const pageSize = useDefaultPageSize();
  const [skip, setSkip] = useState(0);
  const [action, setAction] = useState("");
  const [actor, setActor] = useState("");
  const [resource, setResource] = useState("");
  const [createdRange, setCreatedRange] = useState<DateRangeValue>({ from: null, to: null });
  const chrome = useAdminNavChrome(token);
  const billing = useAdminBillingSettings(token);
  const filterOptions = useAdminAuditFilterOptions(token);
  const audit = useAdminAudit(token, skip, pageSize, {
    action,
    actor,
    resource,
    created_from: createdRange.from || "",
    created_to: createdRange.to || "",
  });
  const exportMut = useExportAudit(token);
  const title = chrome.data?.ADMIN_NAV_AUDIT?.trim() || "";
  const exportLabel = chrome.data?.ADMIN_ACTION_EXPORT?.trim() || "";
  const pending =
    chrome.data?.ADMIN_PENDING_EXPORTING?.trim() || chrome.data?.ADMIN_PENDING_SAVING?.trim() || "";
  const filterAction = chrome.data?.ADMIN_FILTER_ACTION?.trim() || "";
  const filterActor = chrome.data?.ADMIN_FILTER_ACTOR?.trim() || "";
  const filterResource = chrome.data?.ADMIN_FILTER_RESOURCE?.trim() || "";
  const filterActionDesc = chrome.data?.ADMIN_FILTER_ACTION_DESC?.trim() || "";
  const filterActorDesc = chrome.data?.ADMIN_FILTER_ACTOR_DESC?.trim() || "";
  const filterResourceDesc = chrome.data?.ADMIN_FILTER_RESOURCE_DESC?.trim() || "";
  const filterCreatedFrom = chrome.data?.ADMIN_FILTER_CREATED_FROM?.trim() || "";
  const filterCreatedFromDesc = chrome.data?.ADMIN_FILTER_CREATED_FROM_DESC?.trim() || "";
  const filterAll = chrome.data?.ADMIN_FILTER_ALL?.trim() || "";
  const dateRangePlaceholder = chrome.data?.ADMIN_DATE_RANGE_PLACEHOLDER?.trim() || "";
  const dateRangeClear = chrome.data?.ADMIN_DATE_RANGE_CLEAR?.trim() || "";
  const dateRangeApply = chrome.data?.ADMIN_DATE_RANGE_APPLY?.trim() || "";
  const htmlLang = chrome.data?.SITE_HTML_LANG?.trim() || "";
  const dateRangeMonthsRaw = billing.data?.date_range_months;
  const dateRangeMonths =
    typeof dateRangeMonthsRaw === "number" && Number.isFinite(dateRangeMonthsRaw)
      ? dateRangeMonthsRaw
      : 0;
  const colAction = chrome.data?.ADMIN_AUDIT_COL_ACTION?.trim() || "";
  const colResource = chrome.data?.ADMIN_AUDIT_COL_RESOURCE?.trim() || "";
  const colId = chrome.data?.ADMIN_AUDIT_COL_ID?.trim() || "";
  const colWhen = chrome.data?.ADMIN_AUDIT_COL_WHEN?.trim() || "";
  const colActor = chrome.data?.ADMIN_AUDIT_COL_ACTOR?.trim() || "";
  const colReason = chrome.data?.ADMIN_AUDIT_COL_REASON?.trim() || "";
  const colStepUp = chrome.data?.ADMIN_AUDIT_COL_STEP_UP?.trim() || "";
  const stepUpYes = chrome.data?.ADMIN_AUDIT_STEP_UP_YES?.trim() || "";
  const stepUpNo = chrome.data?.ADMIN_AUDIT_STEP_UP_NO?.trim() || "";

  const actionOptions = useMemo(
    () =>
      (filterOptions.data?.actions || [])
        .map((item) => {
          const value = String(item.value || "").trim();
          const label = String(item.label || item.value || "").trim();
          return value && label ? { value, label } : null;
        })
        .filter((o): o is { value: string; label: string } => Boolean(o)),
    [filterOptions.data?.actions],
  );
  const actorOptions = useMemo(
    () =>
      (filterOptions.data?.actors || [])
        .map((item) => {
          const value = String(item.value || "").trim();
          const label = String(item.label || item.value || "").trim();
          return value && label ? { value, label } : null;
        })
        .filter((o): o is { value: string; label: string } => Boolean(o)),
    [filterOptions.data?.actors],
  );
  const resourceOptions = useMemo(
    () =>
      (filterOptions.data?.resources || [])
        .map((item) => {
          const value = String(item.value || "").trim();
          const label = String(item.label || item.value || "").trim();
          return value && label ? { value, label } : null;
        })
        .filter((o): o is { value: string; label: string } => Boolean(o)),
    [filterOptions.data?.resources],
  );

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setSkip(0);
  }, [action, actor, resource, createdRange.from, createdRange.to]);

  const columns = useMemo<ColumnDef<AuditRow>[]>(() => {
    const cols: ColumnDef<AuditRow>[] = [];
    if (colActor) {
      cols.push({
        id: "actor_user_id",
        header: colActor,
        cell: ({ row }) => String(row.original.actor_user_id || ""),
      });
    }
    if (colAction) {
      cols.push({
        id: "action",
        header: colAction,
        cell: ({ row }) => String(row.original.action || ""),
      });
    }
    if (colResource) {
      cols.push({
        id: "resource_type",
        header: colResource,
        cell: ({ row }) => String(row.original.resource_type || ""),
      });
    }
    if (colId) {
      cols.push({
        id: "resource_id",
        header: colId,
        cell: ({ row }) => (
          <span className="font-mono text-xs">{String(row.original.resource_id || "")}</span>
        ),
      });
    }
    if (colReason) {
      cols.push({
        id: "reason",
        header: colReason,
        cell: ({ row }) => String(row.original.reason || ""),
      });
    }
    if (colStepUp) {
      cols.push({
        id: "step_up_used",
        header: colStepUp,
        cell: ({ row }) => {
          const used = row.original.step_up_used === true;
          if (used) return stepUpYes;
          if (row.original.step_up_used === false) return stepUpNo;
          return "";
        },
      });
    }
    if (colWhen) {
      cols.push({
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
      });
    }
    return cols;
  }, [
    colActor,
    colAction,
    colResource,
    colId,
    colReason,
    colStepUp,
    stepUpYes,
    stepUpNo,
    colWhen,
    htmlLang,
  ]);

  const bootReady = useListBootReady(pageSize, Boolean(chrome.data), queryContentReady(audit));
  const showInitialSkeleton = !bootReady;

  if (showInitialSkeleton) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between gap-3">
          {title ? (
            <h1 className="text-xl font-semibold tracking-tight text-muted-foreground">{title}</h1>
          ) : (
            <Skeleton className="h-7 w-40" />
          )}
          {exportLabel && pending ? (
            <Skeleton className="h-9 w-28" />
          ) : (
            <Skeleton className="h-9 w-28" />
          )}
        </div>
        <DataTableSkeleton
          title=""
          columns={[colActor, colAction, colResource, colId, colReason, colStepUp, colWhen]
            .filter(Boolean)
            .map((label) => ({ label }))}
          rows={pageSize > 0 ? pageSize : 8}
          showFilters
          filterCount={4}
          filterLeadSearch={false}
          filterWideIndexes={[3]}
        />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : <span />}
        {exportLabel && pending ? (
          <Button
            type="button"
            isLoading={exportMut.isPending}
            pendingLabel={pending}
            onClick={() => void exportMut.mutateAsync()}
          >
            {exportLabel}
          </Button>
        ) : null}
      </div>
      <AdminFilterBar
        filters={[
          {
            kind: "select",
            id: "audit-action",
            label: filterAction,
            description: filterActionDesc,
            value: action,
            onChange: setAction,
            options: actionOptions,
            allLabel: filterAll,
          },
          {
            kind: "select",
            id: "audit-actor",
            label: filterActor,
            description: filterActorDesc,
            value: actor,
            onChange: setActor,
            options: actorOptions,
            allLabel: filterAll,
          },
          {
            kind: "select",
            id: "audit-resource",
            label: filterResource,
            description: filterResourceDesc,
            value: resource,
            onChange: setResource,
            options: resourceOptions,
            allLabel: filterAll,
          },
          {
            kind: "dateRange",
            id: "audit-created-range",
            label: filterCreatedFrom || dateRangePlaceholder,
            description: filterCreatedFromDesc,
            value: createdRange,
            onChange: setCreatedRange,
            placeholder: dateRangePlaceholder,
            clearLabel: dateRangeClear,
            applyLabel: dateRangeApply,
            locale: htmlLang,
            numberOfMonths: dateRangeMonths,
          },
        ]}
      />
      <Card>
        <CardContent className="pt-6">
          <DataTable
            columns={columns}
            data={((audit.data?.items as AuditRow[]) ?? []).filter((row) =>
              Boolean(String(row.id || "")),
            )}
            meta={audit.data?.meta}
            onPage={setSkip}
            pageDisabled={audit.isFetching}
            isFetching={audit.isFetching && !audit.isPending}
            getRowId={(row) => String(row.id || "")}
            bordered={false}
          />
        </CardContent>
      </Card>
    </div>
  );
}
