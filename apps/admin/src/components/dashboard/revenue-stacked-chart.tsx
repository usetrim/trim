"use client";

import { ChartLegendList } from "@/components/dashboard/chart-legend";
import {
  adminChartSeriesPalette,
  adminChartTooltipStyle,
  useAdminChartTheme,
} from "@/lib/admin-chart-theme";
import { formatCompactNumber, sharePct } from "@/lib/chart-series";
import { formatDateShort } from "@/lib/format-datetime";
import type { RevenueSeriesMeta, RevenueSeriesRow } from "@/types/admin/revenue";
import { useMemo } from "react";
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { TooltipProps } from "recharts";

type SeriesMeta = { id: string; label: string; color: string };

function RevenueTooltip({
  active,
  payload,
  seriesMeta,
}: TooltipProps<number, string> & { seriesMeta: SeriesMeta[] }) {
  const theme = useAdminChartTheme();
  if (!active || !payload?.length) return null;
  const row = payload[0]?.payload as RevenueSeriesRow | undefined;
  if (!row) return null;
  const dayLabel = (row.day_label || "").trim();
  if (!dayLabel) return null;
  const lines = seriesMeta
    .map((s) => {
      const money = String(row[`d_${s.id}_label`] ?? "").trim();
      const raw = Number(row[`d_${s.id}`] ?? 0);
      return { ...s, money, raw };
    })
    .filter((s) => s.raw > 0 && s.money)
    .sort((a, b) => b.raw - a.raw);
  if (!lines.length) return null;
  return (
    <div
      className="max-h-[min(70vh,20rem)] min-w-[11rem] max-w-[min(92vw,18rem)] overflow-y-auto rounded-lg border px-3 py-2 text-xs"
      style={adminChartTooltipStyle(theme)}
    >
      <p className="font-medium text-[hsl(var(--foreground))]">{dayLabel}</p>
      <ul className="mt-1.5 space-y-1">
        {lines.map((s) => (
          <li key={s.id} className="flex items-center justify-between gap-3">
            <span className="flex min-w-0 items-center gap-1.5 text-[hsl(var(--muted-foreground))]">
              <span
                className="inline-block h-2 w-2 shrink-0 rounded-sm"
                style={{ background: s.color }}
              />
              <span className="truncate" title={s.label}>
                {s.label}
              </span>
            </span>
            <span className="shrink-0 tabular-nums text-[hsl(var(--foreground))]">{s.money}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

export function RevenueStackedChart({
  title,
  series,
  seriesMeta,
  locale = "",
}: {
  title: string;
  series: RevenueSeriesRow[];
  seriesMeta: RevenueSeriesMeta[];
  locale?: string;
}) {
  const theme = useAdminChartTheme();
  const palette = adminChartSeriesPalette(theme);
  const meta: SeriesMeta[] = useMemo(
    () =>
      seriesMeta
        .filter((s) => s.id.trim() && s.label.trim())
        .map((s, i) => ({
          id: s.id,
          label: s.label,
          color: palette[i % palette.length] || palette[0],
        })),
    [seriesMeta, palette],
  );
  const chartSeries = useMemo((): RevenueSeriesRow[] => {
    return series.map((row) => ({
      ...row,
      day_label: formatDateShort(row.day, locale) || row.day_label,
    }));
  }, [series, locale]);

  const legendItems = useMemo(() => {
    const totals = new Map<string, number>();
    for (const row of chartSeries) {
      for (const s of meta) {
        const key = `d_${s.id}` as keyof RevenueSeriesRow;
        const raw = Number(row[key] ?? 0);
        totals.set(s.id, (totals.get(s.id) || 0) + raw);
      }
    }
    const period = [...totals.values()].reduce((a, b) => a + b, 0);
    return [...meta]
      .sort((a, b) => (totals.get(b.id) || 0) - (totals.get(a.id) || 0))
      .map((s) => {
        const v = totals.get(s.id) || 0;
        return {
          id: s.id,
          label: s.label,
          color: s.color,
          value: v,
          valueLabel: formatCompactNumber(v),
          pct: sharePct(v, period),
        };
      });
  }, [chartSeries, meta]);

  if (!title.trim() || meta.length === 0 || chartSeries.length === 0) return null;

  return (
    <div className="space-y-3">
      <h2 className="text-sm font-semibold tracking-tight">{title}</h2>
      <div className="h-52 w-full min-w-0 sm:h-64 md:h-72">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={chartSeries} margin={{ top: 8, right: 8, left: 0, bottom: 4 }}>
            <CartesianGrid stroke={theme.grid} strokeDasharray="3 3" vertical={false} />
            <XAxis
              dataKey="day_label"
              tick={{ fill: theme.muted, fontSize: 10 }}
              stroke={theme.axis}
              tickLine={false}
              interval="preserveStartEnd"
              minTickGap={28}
              height={28}
            />
            <YAxis
              tick={{ fill: theme.muted, fontSize: 10 }}
              stroke={theme.axis}
              tickLine={false}
              width={44}
              tickFormatter={formatCompactNumber}
            />
            <Tooltip
              content={<RevenueTooltip seriesMeta={meta} />}
              wrapperStyle={{ outline: "none", zIndex: 40 }}
            />
            {meta.map((s) => (
              <Area
                key={s.id}
                type="monotone"
                dataKey={`d_${s.id}`}
                name={s.label}
                stackId="revenue"
                stroke={s.color}
                fill={s.color}
                fillOpacity={0.35}
                isAnimationActive={false}
              />
            ))}
          </AreaChart>
        </ResponsiveContainer>
      </div>
      <ChartLegendList items={legendItems} maxHeightClass="max-h-32 sm:max-h-40" />
    </div>
  );
}
