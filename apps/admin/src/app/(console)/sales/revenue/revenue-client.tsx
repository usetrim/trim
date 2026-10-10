"use client";

import { AdminFilterBar } from "@/components/admin/admin-filter-bar";
import { RevenueStackedChart } from "@/components/dashboard/revenue-stacked-chart";
import { DataTableSkeleton } from "@/components/skeletons/page-skeletons";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { type ColumnDef, DataTable } from "@/components/ui/data-table";
import type { DateRangeValue } from "@/components/ui/date-range-picker";
import { useAdminRevenue } from "@/hooks/queries/revenue";
import { useAdminToken } from "@/hooks/use-admin-token";
import type { RevenueByPlanRow } from "@/types/admin/revenue";
import Link from "next/link";
import { useEffect, useMemo, useState } from "react";

export function RevenueClient({ initialToken }: { initialToken?: string }) {
  const token = useAdminToken(initialToken);
  const [range, setRange] = useState("");
  const [createdRange, setCreatedRange] = useState<DateRangeValue>({ from: null, to: null });
  const data = useAdminRevenue(token, range, createdRange.from || "", createdRange.to || "");

  const options = data.data?.range_options ?? [];
  useEffect(() => {
    if (range) return;
    // Prefer the range the API actually resolved (empty query → first preset).
    const resolved = data.data?.range?.trim();
    if (resolved) {
      setRange(resolved);
      return;
    }
    const first = options.find((o) => o.id.trim() && o.label.trim());
    if (first?.id) setRange(first.id);
  }, [range, options, data.data?.range]);

  const chrome = data.data?.chrome ?? {};
  const title = chrome.title?.trim() || "";
  const intro = chrome.intro?.trim() || "";
  const chartTitle = chrome.chart_title?.trim() || "";
  const byPlanTitle = chrome.by_plan_title?.trim() || "";
  const drillTitle = chrome.drill_title?.trim() || "";
  const drillLink = chrome.drill_link?.trim() || "";
  const receiptsHref = chrome.receipts_href?.trim() || "";
  const emptyMsg = chrome.empty?.trim() || "";
  const rangeLabel = chrome.range_label?.trim() || "";
  const rangeDesc = chrome.range_desc?.trim() || "";
  const colPlan = chrome.col_plan?.trim() || "";
  const colKind = chrome.col_kind?.trim() || "";
  const colRevenue = chrome.col_revenue?.trim() || "";
  const colReceipts = chrome.col_receipts?.trim() || "";
  const dateRangePlaceholder = chrome.date_range_placeholder?.trim() || "";
  const dateRangeClear = chrome.date_range_clear?.trim() || "";
  const dateRangeApply = chrome.date_range_apply?.trim() || "";
  const htmlLang = chrome.html_lang?.trim() || "";
  const dateRangeMonthsRaw = data.data?.date_range_months;
  const dateRangeMonths =
    typeof dateRangeMonthsRaw === "number" && Number.isFinite(dateRangeMonthsRaw)
      ? dateRangeMonthsRaw
      : 0;
  const kpis = (data.data?.kpi_items ?? []).filter((k) => k.label?.trim());
  const series = data.data?.series ?? [];
  const seriesMeta = data.data?.series_meta ?? [];
  const byPlan = data.data?.by_plan ?? [];

  const columns = useMemo<ColumnDef<RevenueByPlanRow>[]>(
    () => [
      {
        id: "plan",
        header: colPlan,
        cell: ({ row }) => row.original.plan_label,
      },
      {
        id: "kind",
        header: colKind,
        cell: ({ row }) => row.original.plan_kind || "",
      },
      {
        id: "revenue",
        header: colRevenue,
        cell: ({ row }) => <span className="tabular-nums">{row.original.revenue_label}</span>,
      },
      {
        id: "receipts",
        header: colReceipts,
        cell: ({ row }) => <span className="tabular-nums">{row.original.receipt_count}</span>,
      },
    ],
    [colPlan, colKind, colRevenue, colReceipts],
  );

  // isLoading = isPending && isFetching. Never treat a disabled query as a
  // forever-shimmer (TanStack Query v5 keeps isPending true while idle).
  if (Boolean(token) && data.isLoading && !data.data) {
    return (
      <div className="space-y-4">
        {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
        <DataTableSkeleton
          title=""
          columns={[{ label: colPlan }, { label: colRevenue }]}
          rows={6}
        />
      </div>
    );
  }

  if (data.isError && !data.data) {
    return (
      <div className="space-y-4">
        {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
        <p className="text-sm text-destructive">
          {data.error instanceof Error ? data.error.message : ""}
        </p>
      </div>
    );
  }

  const showCustom =
    range === "custom" &&
    Boolean(
      dateRangePlaceholder && dateRangeClear && dateRangeApply && htmlLang && dateRangeMonths > 0,
    );

  return (
    <div className="space-y-4">
      {title ? <h1 className="text-xl font-semibold tracking-tight">{title}</h1> : null}
      {intro ? <p className="text-sm text-muted-foreground">{intro}</p> : null}

      <AdminFilterBar
        filters={[
          {
            kind: "select",
            id: "revenue-range",
            label: rangeLabel,
            description: rangeDesc,
            value: range,
            onChange: setRange,
            options: options.map((o) => ({ value: o.id, label: o.label })),
            required: true,
          },
          ...(showCustom
            ? [
                {
                  kind: "dateRange" as const,
                  id: "revenue-custom-range",
                  label: dateRangePlaceholder,
                  description: rangeDesc,
                  value: createdRange,
                  onChange: setCreatedRange,
                  placeholder: dateRangePlaceholder,
                  clearLabel: dateRangeClear,
                  applyLabel: dateRangeApply,
                  locale: htmlLang,
                  numberOfMonths: dateRangeMonths,
                },
              ]
            : []),
        ]}
      />

      {kpis.length > 0 ? (
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
          {kpis.map((kpi) => (
            <Card key={kpi.id}>
              <CardHeader className="pb-2">
                <CardTitle className="text-xs font-medium text-muted-foreground">
                  {kpi.label}
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-2xl font-semibold tabular-nums tracking-tight">{kpi.value}</p>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : null}

      <Card>
        <CardContent className="pt-6">
          {series.length > 0 && seriesMeta.length > 0 ? (
            <RevenueStackedChart
              title={chartTitle}
              series={series}
              seriesMeta={seriesMeta}
              locale={htmlLang}
            />
          ) : emptyMsg ? (
            <p className="text-sm text-muted-foreground">{emptyMsg}</p>
          ) : null}
        </CardContent>
      </Card>

      {byPlanTitle || byPlan.length > 0 ? (
        <Card>
          <CardHeader>
            {byPlanTitle ? <CardTitle className="text-sm">{byPlanTitle}</CardTitle> : null}
          </CardHeader>
          <CardContent>
            <DataTable
              columns={columns}
              data={byPlan}
              getRowId={(row) => row.plan_id}
              bordered={false}
              isFetching={data.isFetching && !data.isPending}
            />
          </CardContent>
        </Card>
      ) : null}

      {drillTitle && drillLink && receiptsHref ? (
        <p className="text-sm text-muted-foreground">
          {drillTitle}{" "}
          <Link
            href={receiptsHref}
            scroll={false}
            className="text-foreground underline-offset-4 hover:underline"
          >
            {drillLink}
          </Link>
        </p>
      ) : null}
    </div>
  );
}
