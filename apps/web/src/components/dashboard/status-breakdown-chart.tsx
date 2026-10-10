"use client";

import { TrimChartTooltipBody } from "@/components/dashboard/trim-chart-tooltip";
import { formatCompactNumber } from "@/lib/chart-series";
import { chartOutcomeColors, useTrimChartTheme } from "@/lib/trim-chart-theme";
import { useMemo } from "react";
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { TooltipProps } from "recharts";

type EventPoint = {
  status: string;
};

type NamedRow = {
  name: string;
  code?: string;
  count: number;
};

type StatusChrome = {
  empty_message?: string;
  success_rate_prefix?: string;
  runs_series_name?: string;
  scope_full_label?: string;
  scope_page_label?: string;
  traces_unit?: string;
  status_success_label?: string;
  status_error_label?: string;
  status_success_code?: string;
  meta_sep?: string;
};

type BarRow = { name: string; count: number; fill: string; kind: "success" | "error" | "other" };

function StatusTooltip({
  active,
  payload,
  seriesName,
}: TooltipProps<number, string> & { seriesName?: string }) {
  if (!active || !payload?.length) return null;
  const item = payload[0];
  const row = item?.payload as BarRow | undefined;
  const name = String(row?.name || item?.name || "").trim();
  if (!name) return null;
  const value = String(row?.count ?? item?.value ?? "");
  const series = (seriesName || "").trim();
  const swatch = String(item?.color || row?.fill || "").trim() || undefined;
  return (
    <TrimChartTooltipBody
      title={series ? name : undefined}
      rows={[{ name: series || name, value, swatch }]}
    />
  );
}

function classifyKind(
  name: string,
  code: string | undefined,
  successLabel: string,
  errorLabel: string,
  successCode: string,
): BarRow["kind"] {
  const codeLc = (code || "").toLowerCase();
  if (successCode && codeLc === successCode) return "success";
  if (successLabel && name === successLabel) return "success";
  if (errorLabel && name === errorLabel) return "error";
  if (codeLc.includes("error") || codeLc.includes("fail")) return "error";
  if (codeLc.includes("success") || codeLc.includes("ok")) return "success";
  return "other";
}

export function StatusBreakdownChart({
  events,
  breakdown,
  successRate,
  successCount,
  chrome,
}: {
  events?: EventPoint[];
  breakdown?: NamedRow[];
  /** Server-computed success rate (0-100). Prefer over client math. */
  successRate?: number;
  successCount?: number;
  chrome?: StatusChrome;
}) {
  const theme = useTrimChartTheme();
  const outcome = chartOutcomeColors(theme);
  const successLabel = chrome?.status_success_label || "";
  const errorLabel = chrome?.status_error_label || "";
  const successCode = (chrome?.status_success_code || "").toLowerCase();
  const scopeFull = chrome?.scope_full_label || "";
  const scopePage = chrome?.scope_page_label || "";

  const data = useMemo(() => {
    if (breakdown && breakdown.length > 0) {
      const bars: BarRow[] = breakdown.map((r) => {
        const kind = classifyKind(r.name, r.code, successLabel, errorLabel, successCode);
        const fill =
          kind === "success" ? outcome.success : kind === "error" ? outcome.error : theme.ink;
        return { name: r.name, count: r.count, fill, kind };
      });
      const total = bars.reduce((s, b) => s + b.count, 0);
      const rate =
        typeof successRate === "number"
          ? successRate
          : total === 0
            ? 0
            : Math.round(
                ((typeof successCount === "number"
                  ? successCount
                  : breakdown
                      .filter((b) =>
                        successCode ? String(b.code || "").toLowerCase() === successCode : false,
                      )
                      .reduce((s, b) => s + b.count, 0)) /
                  total) *
                  1000,
              ) / 10;
      return { bars, rate, total, scope: scopeFull };
    }
    let success = 0;
    let error = 0;
    for (const e of events ?? []) {
      if (successCode && String(e.status).toLowerCase() === successCode) {
        success += 1;
      } else {
        error += 1;
      }
    }
    const total = success + error;
    const rate =
      typeof successRate === "number"
        ? successRate
        : total === 0
          ? 0
          : Math.round((success / total) * 1000) / 10;
    return {
      bars: [
        { name: errorLabel, count: error, fill: outcome.error, kind: "error" as const },
        { name: successLabel, count: success, fill: outcome.success, kind: "success" as const },
      ].filter((b) => b.name),
      rate,
      total,
      scope: scopePage,
    };
  }, [
    events,
    breakdown,
    successRate,
    successCount,
    successLabel,
    errorLabel,
    successCode,
    scopeFull,
    scopePage,
    outcome.success,
    outcome.error,
    theme.ink,
  ]);

  if (!data.total) {
    return (
      <p className="py-10 text-center text-sm text-[var(--trim-muted)]">
        {chrome?.empty_message || ""}
      </p>
    );
  }

  const rateTone =
    data.rate >= 80
      ? "text-[var(--trim-status)]"
      : data.rate >= 50
        ? "text-[var(--trim-fg)]"
        : "text-destructive";

  return (
    <div className="space-y-3">
      <p className="text-sm leading-relaxed text-[var(--trim-muted)]">
        {chrome?.success_rate_prefix || ""}{" "}
        <span className={`font-semibold tabular-nums ${rateTone}`}>{data.rate}%</span>
        <span className="text-[var(--trim-subtle)]">
          {" "}
          {chrome?.meta_sep || ""}
          {data.total} {chrome?.traces_unit || ""}
          {data.scope ? ` (${data.scope})` : ""}
        </span>
      </p>

      {/* Mobile-friendly summary chips */}
      <div className="flex flex-wrap gap-2 sm:hidden">
        {data.bars.map((b) => (
          <div
            key={b.name}
            className="inline-flex items-center gap-2 rounded-full border border-[var(--trim-border)] bg-[var(--trim-panel-2)] px-2.5 py-1 text-xs"
          >
            <span className="h-2 w-2 rounded-full" style={{ background: b.fill }} aria-hidden />
            <span className="text-[var(--trim-muted)]">{b.name}</span>
            <span className="font-medium tabular-nums text-[var(--trim-fg)]">
              {formatCompactNumber(b.count)}
            </span>
          </div>
        ))}
      </div>

      <div className="h-48 w-full min-w-0 sm:h-56 md:h-64">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart
            data={data.bars}
            margin={{ top: 8, right: 8, left: 0, bottom: 4 }}
            barCategoryGap="28%"
          >
            <CartesianGrid stroke={theme.grid} strokeDasharray="3 3" vertical={false} />
            <XAxis
              dataKey="name"
              tick={{ fill: theme.axis, fontSize: 11 }}
              axisLine={{ stroke: theme.axisLine }}
              tickLine={false}
              interval={0}
            />
            <YAxis
              allowDecimals={false}
              tick={{ fill: theme.axis, fontSize: 10 }}
              axisLine={{ stroke: theme.axisLine }}
              tickLine={false}
              width={40}
              tickFormatter={formatCompactNumber}
            />
            <Tooltip
              content={<StatusTooltip seriesName={chrome?.runs_series_name} />}
              wrapperStyle={{ outline: "none", zIndex: 40 }}
              cursor={{ fill: theme.isDark ? "rgba(255,255,255,0.04)" : "rgba(23,23,23,0.04)" }}
            />
            <Bar
              dataKey="count"
              name={chrome?.runs_series_name || ""}
              radius={[6, 6, 0, 0]}
              maxBarSize={72}
              isAnimationActive={false}
            >
              {data.bars.map((b) => (
                <Cell key={b.name} fill={b.fill} />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
