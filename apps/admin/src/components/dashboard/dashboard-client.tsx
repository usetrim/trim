"use client";

import { TrimWordmark } from "@/components/brand/trim-wordmark";
import { LocHeatmap } from "@/components/dashboard/loc-heatmap";
import { UsageStackedChart } from "@/components/dashboard/usage-stacked-chart";
import {
  DashboardChartsSkeleton,
  OpsOverviewSkeleton,
} from "@/components/skeletons/page-skeletons";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { type ColumnDef, DataTable } from "@/components/ui/data-table";
import { useAdminNavChrome } from "@/hooks/queries/chrome";
import { useAdminDashboard } from "@/hooks/queries/dashboard";
import { useAdminObservability } from "@/hooks/queries/observability";
import { useAdminToken } from "@/hooks/use-admin-token";
import type {
  DashboardAlertItem,
  DashboardChecklistItem,
  DashboardHealthItem,
} from "@/types/admin/dashboard";
import Link from "next/link";
import { useMemo, useState } from "react";

export function DashboardClient({ initialToken }: { initialToken?: string }) {
  const token = useAdminToken(initialToken);
  const dash = useAdminDashboard(token || "");
  const chrome = useAdminNavChrome(token || "");
  const htmlLang = chrome.data?.SITE_HTML_LANG?.trim() || "";
  const [usageGroupBy, setUsageGroupBy] = useState("");
  const [heatmapScope, setHeatmapScope] = useState("");
  const obs = useAdminObservability(token || "", usageGroupBy, heatmapScope);

  const brand = dash.data?.chrome?.brand?.trim() || "";
  const tagline = dash.data?.chrome?.tagline?.trim() || "";
  const checklistTitle = dash.data?.chrome?.ops_checklist_title?.trim() || "";
  const healthTitle = dash.data?.chrome?.health_title?.trim() || "";
  const alertsTitle = dash.data?.chrome?.alerts_title?.trim() || "";
  const revenueHint = dash.data?.chrome?.revenue_hint?.trim() || "";
  const revenueLink = dash.data?.chrome?.revenue_link?.trim() || "";
  const revenueHref = dash.data?.chrome?.revenue_href?.trim() || "";
  const kpiItems = (dash.data?.kpi_items || []).filter((row) => row.label?.trim());
  const healthItems = (dash.data?.health_items || []).filter((row) => row.label?.trim());
  const checklist = (dash.data?.ops_checklist || []).filter((row) => row.label?.trim());
  const alerts = (dash.data?.alerts || []).filter((row) => row.label?.trim());

  const alertColumns = useMemo<ColumnDef<DashboardAlertItem>[]>(
    () => [
      {
        id: "label",
        header: alertsTitle,
        cell: ({ row }) => row.original.label,
      },
      {
        id: "_count",
        header: "",
        cell: ({ row }) => (
          <span
            className={
              row.original.active
                ? "tabular-nums text-destructive"
                : "tabular-nums text-muted-foreground"
            }
          >
            {row.original.count}
          </span>
        ),
      },
    ],
    [alertsTitle],
  );

  const healthColumns = useMemo<ColumnDef<DashboardHealthItem>[]>(
    () => [
      {
        id: "label",
        header: healthTitle,
        cell: ({ row }) => row.original.label,
      },
      {
        id: "_status",
        header: "",
        cell: ({ row }) => {
          const status = row.original.status_label?.trim() || "";
          if (!status) return null;
          return (
            <span className={row.original.ok ? "text-foreground" : "text-destructive"}>
              {status}
            </span>
          );
        },
      },
    ],
    [healthTitle],
  );

  const checklistColumns = useMemo<ColumnDef<DashboardChecklistItem>[]>(
    () => [
      {
        id: "label",
        header: checklistTitle,
        cell: ({ row }) => row.original.label,
      },
      {
        id: "_status",
        header: "",
        cell: ({ row }) => {
          const status = row.original.status_label?.trim() || "";
          if (!status) return null;
          return (
            <span className={row.original.ok ? "text-foreground" : "text-destructive"}>
              {status}
            </span>
          );
        },
      },
    ],
    [checklistTitle],
  );

  // Full-page boot shimmer only when there is no cached dashboard payload yet.
  const dashLoading = Boolean(token) && dash.isPending && !dash.data;
  const obsLoading = Boolean(token) && obs.isPending && !obs.data;
  const showUsage =
    !obsLoading &&
    Boolean(obs.data?.usage_title?.trim()) &&
    (obs.data?.usage_group_by_options?.length ?? 0) > 0;
  const showHeatmap =
    !obsLoading &&
    Boolean(obs.data?.loc_heatmap_title?.trim()) &&
    (obs.data?.loc_heatmap?.length ?? 0) > 0 &&
    (obs.data?.loc_heatmap_scopes?.length ?? 0) > 0;

  // Full-page boot shimmer matches route loading.tsx (brand + KPIs + charts + ops).
  if (dashLoading) {
    return <OpsOverviewSkeleton metricCount={12} rows={4} />;
  }

  return (
    <div className="space-y-6">
      <div>
        {brand ? <TrimWordmark size="lg" alt={brand} className="mb-1" /> : null}
        {tagline ? <p className="text-sm text-muted-foreground">{tagline}</p> : null}
      </div>
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {kpiItems.map((item) => (
          <Card key={item.id}>
            <CardHeader className="pb-2">
              <CardTitle className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                {item.label}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-2xl font-semibold tabular-nums">{item.value}</p>
            </CardContent>
          </Card>
        ))}
      </div>
      {revenueHint && revenueLink && revenueHref ? (
        <p className="text-sm text-muted-foreground">
          {revenueHint}{" "}
          <Link
            href={revenueHref}
            scroll={false}
            className="text-foreground underline-offset-4 hover:underline"
          >
            {revenueLink}
          </Link>
        </p>
      ) : null}
      {obsLoading ? <DashboardChartsSkeleton /> : null}
      {showUsage && obs.data ? (
        <Card className="min-w-0">
          <CardContent className="min-w-0 overflow-visible pt-6">
            <UsageStackedChart
              title={obs.data.usage_title}
              subtitle={obs.data.usage_subtitle}
              yAxisLabel={obs.data.usage_y_axis}
              todayLabel={obs.data.usage_today_label}
              todayDay={obs.data.today_day}
              groupByPrefix={obs.data.usage_group_by_prefix}
              groupByOptions={obs.data.usage_group_by_options}
              groupBySelected={usageGroupBy || obs.data.usage_group_by_selected}
              onGroupByChange={setUsageGroupBy}
              series={obs.data.usage_series}
              days={obs.data.usage_days}
              emptyMessage={obs.data.usage_empty}
              tooltipBreakdown={obs.data.usage_tooltip_breakdown}
              tooltipDailyTotal={obs.data.usage_tooltip_daily_total}
              tooltipCumulativeTotal={obs.data.usage_tooltip_cumulative_total}
              tooltipShareFmt={obs.data.usage_tooltip_share_fmt}
              isFetching={obs.isFetching && !obs.isPending}
              locale={htmlLang}
            />
          </CardContent>
        </Card>
      ) : null}
      {showHeatmap && obs.data ? (
        <Card className="min-w-0">
          <CardContent className="min-w-0 overflow-visible pt-6">
            <LocHeatmap
              title={obs.data.loc_heatmap_title}
              total={obs.data.loc_heatmap_total}
              scopes={obs.data.loc_heatmap_scopes}
              scopeSelected={heatmapScope || obs.data.loc_heatmap_scope_selected}
              onScopeChange={setHeatmapScope}
              days={obs.data.loc_heatmap}
              emptyFmt={obs.data.loc_heatmap_empty_fmt}
              valueFmt={obs.data.loc_heatmap_value_fmt}
              weekdayLabels={obs.data.loc_heatmap_weekday_labels}
              stats={obs.data.loc_heatmap_stats}
              isFetching={obs.isFetching && !obs.isPending}
              locale={htmlLang}
            />
          </CardContent>
        </Card>
      ) : null}
      {alertsTitle && alerts.length > 0 ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">{alertsTitle}</CardTitle>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={alertColumns}
              data={alerts}
              bordered={false}
              getRowId={(row) => row.id}
              isFetching={dash.isFetching && !dash.isPending}
            />
          </CardContent>
        </Card>
      ) : null}
      {healthTitle && healthItems.length > 0 ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">{healthTitle}</CardTitle>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={healthColumns}
              data={healthItems}
              bordered={false}
              getRowId={(row) => row.id}
              isFetching={dash.isFetching && !dash.isPending}
            />
          </CardContent>
        </Card>
      ) : null}
      {checklistTitle && checklist.length > 0 ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">{checklistTitle}</CardTitle>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={checklistColumns}
              data={checklist}
              bordered={false}
              getRowId={(row) => row.id}
              isFetching={dash.isFetching && !dash.isPending}
            />
          </CardContent>
        </Card>
      ) : null}
    </div>
  );
}
