"use client";

import { ChartLegendList } from "@/components/dashboard/chart-legend";
import { TrimChartTooltipBody } from "@/components/dashboard/trim-chart-tooltip";
import { formatCompactNumber, sharePct } from "@/lib/chart-series";
import { chartPiePalette, useTrimChartTheme } from "@/lib/trim-chart-theme";
import { useMemo } from "react";
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts";
import type { TooltipProps } from "recharts";

type NamedRow = {
  name: string;
  count: number;
};

type Slice = {
  id: string;
  name: string;
  count: number;
  fill: string;
};

function ModeTooltip({
  active,
  payload,
  runsFmt,
  locale,
}: TooltipProps<number, string> & {
  runsFmt?: string;
  locale?: string;
}) {
  if (!active || !payload?.length) return null;
  const item = payload[0];
  const row = item?.payload as Slice | undefined;
  const name = String(row?.name || item?.name || "").trim();
  if (!name) return null;
  const nRaw = Number(row?.count ?? item?.value ?? 0);
  const n = locale ? nRaw.toLocaleString(locale) : String(nRaw);
  const value = runsFmt ? runsFmt.replace("%s", n) : n;
  const swatch = String(item?.color || row?.fill || "").trim() || undefined;
  return <TrimChartTooltipBody rows={[{ name, value, swatch }]} />;
}

export function ModeBreakdownChart({
  breakdown,
  emptyMessage,
  runsFmt,
  locale,
}: {
  breakdown?: NamedRow[];
  emptyMessage?: string;
  runsFmt?: string;
  locale?: string;
}) {
  const theme = useTrimChartTheme();
  const palette = chartPiePalette(theme);

  const { pieData, legendItems, total } = useMemo(() => {
    const raw = [...(breakdown ?? [])]
      .map((r) => ({ name: r.name, count: r.count }))
      .filter((r) => r.name.trim() && r.count > 0)
      .sort((a, b) => b.count - a.count);

    const sum = raw.reduce((s, r) => s + r.count, 0);
    const pie: Slice[] = raw.map((r, i) => ({
      id: r.name,
      name: r.name,
      count: r.count,
      fill: palette[i % palette.length],
    }));
    const legend = pie.map((r) => ({
      id: r.id,
      label: r.name,
      color: r.fill,
      value: r.count,
      valueLabel: locale ? r.count.toLocaleString(locale) : formatCompactNumber(r.count),
      pct: sharePct(r.count, sum),
    }));
    return { pieData: pie, legendItems: legend, total: sum };
  }, [breakdown, palette, locale]);

  if (!pieData.length || total <= 0) {
    return (
      <p className="py-10 text-center text-sm text-[var(--trim-muted)]">{emptyMessage || ""}</p>
    );
  }

  const totalLabel = locale ? total.toLocaleString(locale) : formatCompactNumber(total);

  return (
    <div className="flex flex-col gap-4">
      <div className="relative mx-auto h-44 w-full max-w-[16rem] sm:h-48">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={pieData}
              dataKey="count"
              nameKey="name"
              cx="50%"
              cy="50%"
              innerRadius="54%"
              outerRadius="82%"
              paddingAngle={pieData.length > 1 ? 2 : 0}
              stroke={theme.panel}
              strokeWidth={2}
              isAnimationActive={false}
            >
              {pieData.map((d) => (
                <Cell key={d.id} fill={d.fill} />
              ))}
            </Pie>
            <Tooltip
              content={<ModeTooltip runsFmt={runsFmt} locale={locale} />}
              wrapperStyle={{ outline: "none", zIndex: 40 }}
            />
          </PieChart>
        </ResponsiveContainer>
        <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
          <span className="max-w-[5.5rem] truncate text-center text-base font-semibold tabular-nums tracking-tight text-[var(--trim-fg)] sm:text-lg">
            {totalLabel}
          </span>
        </div>
      </div>
      <ChartLegendList items={legendItems} columns={1} maxHeightClass="max-h-32 sm:max-h-40" />
    </div>
  );
}
