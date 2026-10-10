"use client";

import { AdminFilterBar } from "@/components/admin/admin-filter-bar";
import { DataTableSkeleton, FilterBarSkeleton } from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { type ColumnDef, DataTable } from "@/components/ui/data-table";
import { useDistributionSync } from "@/hooks/mutations/distribution";
import { useAdminNavChrome } from "@/hooks/queries/chrome";
import { useAdminDistribution } from "@/hooks/queries/distribution";
import { useAdminToken } from "@/hooks/use-admin-token";
import { queryContentReady, useListBootReady } from "@/hooks/use-list-boot-ready";
import { useDefaultPageSize } from "@/hooks/use-default-page-size";
import { useMemo, useState } from "react";

type DistRow = {
  id?: string;
  day?: string;
  source?: string;
  source_label?: string;
  metric?: string;
  metric_label?: string;
  country?: string;
  path?: string;
  value?: number;
};

export function DistributionClient({
  initialToken,
}: {
  initialToken?: string;
}) {
  const token = useAdminToken(initialToken);
  const pageSize = useDefaultPageSize();
  const [skip, setSkip] = useState(0);
  const chrome = useAdminNavChrome(token);
  const rangeDay = chrome.data?.ADMIN_DIST_RANGE_DAY?.trim() || "";
  const rangeMonth = chrome.data?.ADMIN_DIST_RANGE_MONTH?.trim() || "";
  const rangeYear = chrome.data?.ADMIN_DIST_RANGE_YEAR?.trim() || "";
  // Fail closed: no invent default range; operator must pick day/month/year.
  const [range, setRange] = useState("");
  const stats = useAdminDistribution(token, range, skip, pageSize);
  const sync = useDistributionSync(token);
  const title = chrome.data?.ADMIN_NAV_DISTRIBUTION?.trim() || "";
  const syncLabel = chrome.data?.ADMIN_DIST_SYNC?.trim() || "";
  const pending = chrome.data?.ADMIN_PENDING_SYNCING?.trim() || "";
  const colDay = chrome.data?.ADMIN_DIST_COL_DAY?.trim() || "";
  const colSource = chrome.data?.ADMIN_DIST_COL_SOURCE?.trim() || "";
  const colMetric = chrome.data?.ADMIN_DIST_COL_METRIC?.trim() || "";
  const colPath = chrome.data?.ADMIN_DIST_COL_PATH?.trim() || "";
  const colCountry = chrome.data?.ADMIN_DIST_COL_COUNTRY?.trim() || "";
  const colValue = chrome.data?.ADMIN_DIST_COL_VALUE?.trim() || "";
  const filterRange = chrome.data?.ADMIN_DIST_FILTER_RANGE?.trim() || "";
  const filterRangeDesc = chrome.data?.ADMIN_DIST_FILTER_RANGE_DESC?.trim() || "";
  const rangeOptions = (
    [
      rangeDay ? ({ value: "day", label: rangeDay } as const) : null,
      rangeMonth ? ({ value: "month", label: rangeMonth } as const) : null,
      rangeYear ? ({ value: "year", label: rangeYear } as const) : null,
    ] as const
  ).filter((o): o is { value: "day" | "month" | "year"; label: string } => Boolean(o));

  const columns = useMemo<ColumnDef<DistRow>[]>(
    () =>
      (
        [
          {
            id: "day",
            accessorKey: "day",
            header: colDay,
            cell: ({ row }) => row.original.day?.trim() || "",
          },
          {
            id: "source",
            accessorKey: "source_label",
            header: colSource,
            cell: ({ row }) => row.original.source_label?.trim() || "",
          },
          {
            id: "metric",
            accessorKey: "metric_label",
            header: colMetric,
            cell: ({ row }) => row.original.metric_label?.trim() || "",
          },
          colPath
            ? {
                id: "path",
                accessorKey: "path",
                header: colPath,
                cell: ({ row }) => {
                  const p = row.original.path?.trim() || "";
                  return p ? <span className="font-mono text-xs">{p}</span> : "";
                },
              }
            : null,
          {
            id: "country",
            accessorKey: "country",
            header: colCountry,
            cell: ({ row }) => row.original.country?.trim() || "",
          },
          {
            id: "value",
            accessorKey: "value",
            header: colValue,
            cell: ({ row }) =>
              row.original.value != null ? (
                <span className="tabular-nums">{String(row.original.value)}</span>
              ) : (
                ""
              ),
          },
        ] as Array<ColumnDef<DistRow> | null>
      ).filter((c): c is ColumnDef<DistRow> => Boolean(c)),
    [colDay, colSource, colMetric, colPath, colCountry, colValue],
  );

  const bootReady = useListBootReady(
    pageSize,
    Boolean(chrome.data),
    queryContentReady(stats, { enabled: Boolean(range) && pageSize > 0 }),
  );
  const showInitialSkeleton = !bootReady;

  if (showInitialSkeleton) {
    return (
      <div className="space-y-4">
        {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
        <FilterBarSkeleton count={1} leadSearch={false} />
        {range ? (
          <DataTableSkeleton
            title=""
            columns={[colDay, colSource, colMetric, colPath, colCountry, colValue]
              .filter(Boolean)
              .map((label) => ({ label }))}
            rows={pageSize > 0 ? pageSize : 8}
            showFilters={false}
          />
        ) : null}
      </div>
    );
  }

  const items = ((stats.data?.items ?? []) as DistRow[]).filter((row) =>
    Boolean(String(row.id || "").trim()),
  );

  return (
    <div className="space-y-4">
      {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
      <AdminFilterBar
        filters={[
          {
            kind: "select",
            id: "dist-range",
            label: filterRange,
            description: filterRangeDesc,
            value: range,
            onChange: (v) => {
              setRange(v);
              setSkip(0);
            },
            options: rangeOptions,
            required: true,
          },
        ]}
      />
      {syncLabel && pending ? (
        <Button
          type="button"
          isLoading={sync.isPending}
          pendingLabel={pending}
          onClick={() => void sync.mutate()}
        >
          {syncLabel}
        </Button>
      ) : null}
      {sync.data?.github_notice ? (
        <p className="text-sm text-muted-foreground">{sync.data.github_notice}</p>
      ) : null}
      {(sync.data?.channel_notices ?? [])
        .filter((n) => n.trim())
        .map((n) => (
          <p key={n} className="text-sm text-muted-foreground">
            {n}
          </p>
        ))}
      {range && pageSize > 0 ? (
        <Card>
          <CardContent className="pt-6">
            <DataTable
              columns={columns}
              data={items}
              meta={stats.data?.meta}
              onPage={setSkip}
              pageDisabled={stats.isFetching}
              isFetching={stats.isFetching && !stats.isPending}
              getRowId={(row) => String(row.id || "")}
              bordered={false}
            />
          </CardContent>
        </Card>
      ) : null}
    </div>
  );
}
