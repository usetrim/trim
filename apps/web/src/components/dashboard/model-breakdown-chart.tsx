"use client";

import { ChartLegendList } from "@/components/dashboard/chart-legend";
import { TrimChartTooltipBody } from "@/components/dashboard/trim-chart-tooltip";
import { CHART_PIE_TOP_N, bucketTopN, formatCompactNumber, sharePct } from "@/lib/chart-series";
import { chartPiePalette, useTrimChartTheme } from "@/lib/trim-chart-theme";
import { useMemo } from "react";
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts";
import type { TooltipProps } from "recharts";

type EventPoint = {
  model: string | null;
  tokens_before: number;
  tokens_after: number;
};

type NamedRow = {
  name: string;
  count: number;
  tokens?: number;
  saved?: number;
};

type Slice = {
  id: string;
  name: string;
  value: number;
  tokens: number;
  saved: number;
  count: number;
  fill: string;
};

function ModelTooltip({
  active,
  payload,
  tooltipFmt,
  locale,
}: TooltipProps<number, string> & {
  tooltipFmt?: string;
  locale?: string;
}) {
  if (!active || !payload?.length) return null;
  const item = payload[0];
  const row = item?.payload as Slice | undefined;
  const name = String(row?.name || item?.name || "").trim();
  if (!name) return null;
  const tokensRaw = Number(row?.tokens ?? item?.value ?? 0);
  const tokens = locale ? tokensRaw.toLocaleString(locale) : String(tokensRaw);
  const runs = String(row?.count ?? 0);
  const savedRaw = row?.saved ?? 0;
  const saved = locale ? savedRaw.toLocaleString(locale) : String(savedRaw);
  const value = tooltipFmt
    ? tooltipFmt.replace("%s", tokens).replace("%s", runs).replace("%s", saved)
    : tokens;
  const swatch = String(item?.color || row?.fill || "").trim() || undefined;
  return <TrimChartTooltipBody rows={[{ name, value, swatch }]} />;
}

export function ModelBreakdownChart({
  events,
  breakdown,
  emptyMessage,
  tooltipFmt,
  unknownLabel,
  locale,
}: {
  events?: EventPoint[];
  breakdown?: NamedRow[];
  emptyMessage?: string;
  tooltipFmt?: string;
  unknownLabel?: string;
  locale?: string;
}) {
  const theme = useTrimChartTheme();
  const palette = chartPiePalette(theme);

  const { pieData, legendItems, totalTokens } = useMemo(() => {
    let raw: {
      id: string;
      name: string;
      value: number;
      tokens: number;
      saved: number;
      count: number;
    }[] = [];

    if (breakdown && breakdown.length > 0) {
      raw = breakdown
        .map((r) => {
          const tokens = r.tokens ?? r.count;
          return {
            id: r.name,
            name: r.name,
            value: tokens,
            tokens,
            saved: r.saved ?? 0,
            count: r.count,
          };
        })
        .filter((r) => r.name.trim() && r.value > 0)
        .sort((a, b) => b.value - a.value);
    } else {
      const map = new Map<string, { tokens: number; saved: number; count: number }>();
      for (const e of events ?? []) {
        const trimmed = e.model?.trim() || "";
        const name = trimmed || unknownLabel || "";
        if (!name) continue;
        const cur = map.get(name) ?? { tokens: 0, saved: 0, count: 0 };
        cur.tokens += e.tokens_before;
        cur.saved += Math.max(0, e.tokens_before - e.tokens_after);
        cur.count += 1;
        map.set(name, cur);
      }
      raw = [...map.entries()]
        .map(([name, v]) => ({
          id: name,
          name,
          value: v.tokens,
          tokens: v.tokens,
          saved: v.saved,
          count: v.count,
        }))
        .sort((a, b) => b.value - a.value);
    }

    const total = raw.reduce((s, r) => s + r.value, 0);
    const remainderName = raw.length > CHART_PIE_TOP_N ? `+${raw.length - CHART_PIE_TOP_N}` : "";
    const bucketed = bucketTopN(raw, CHART_PIE_TOP_N, remainderName);

    const pie: Slice[] = bucketed.chart.map((r, i) => {
      const isRemainder = r.id === "__remainder__";
      return {
        id: r.id,
        name: r.name,
        value: r.value,
        tokens: isRemainder ? r.value : r.tokens ?? r.value,
        saved: isRemainder ? 0 : r.saved ?? 0,
        count: isRemainder ? 0 : r.count ?? 0,
        fill: palette[i % palette.length],
      };
    });

    // Color map from pie for legend rows that appear in the chart; remainder rows get subtle.
    const colorById = new Map(pie.map((p) => [p.id, p.fill]));
    const legend = bucketed.legend.map((r, i) => ({
      id: r.id,
      label: r.name,
      color: colorById.get(r.id) || palette[i % palette.length],
      value: r.value,
      valueLabel: locale ? r.value.toLocaleString(locale) : formatCompactNumber(r.value),
      pct: sharePct(r.value, total),
    }));

    return { pieData: pie, legendItems: legend, totalTokens: total };
  }, [events, breakdown, unknownLabel, palette, locale]);

  if (!pieData.length || totalTokens <= 0) {
    return (
      <p className="py-10 text-center text-sm text-[var(--trim-muted)]">{emptyMessage || ""}</p>
    );
  }

  const totalLabel = locale ? totalTokens.toLocaleString(locale) : formatCompactNumber(totalTokens);

  return (
    <div className="flex flex-col gap-4">
      <div className="relative mx-auto h-44 w-full max-w-[16rem] sm:h-48">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={pieData}
              dataKey="value"
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
              content={<ModelTooltip tooltipFmt={tooltipFmt} locale={locale} />}
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
      <ChartLegendList items={legendItems} columns={1} maxHeightClass="max-h-40 sm:max-h-48" />
    </div>
  );
}
