"use client";

import { ChartLegendList } from "@/components/dashboard/chart-legend";
import { FetchProgressBar } from "@/components/ui/fetch-progress";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { CHART_STACK_TOP_N, formatCompactNumber, sharePct } from "@/lib/chart-series";
import { formatDateShort } from "@/lib/format-datetime";
import { chartSeriesPalette, chartTooltipStyle, useTrimChartTheme } from "@/lib/trim-chart-theme";
import type { NamedOption, UsageAxisDay, UsageDaySeries } from "@/types/events";
import { useMemo } from "react";
import {
  Area,
  AreaChart,
  CartesianGrid,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { TooltipProps } from "recharts";

/** Backend marker for day-axis placeholder rows (no stack series). */
const AXIS_ONLY_SERIES_ID = "__axis__";
const REMAINDER_SERIES_ID = "__remainder__";
/** Max rows inside the daily tooltip before folding the rest into a remainder line. */
const TOOLTIP_TOP_N = 8;

type SeriesMeta = { id: string; label: string; color: string };
type ChartRow = {
  day: string;
  day_label: string;
  [k: string]: string | number;
};

function applyShareFmt(fmt: string, pct: string): string {
  const f = fmt.trim();
  if (!f || !f.includes("{pct}")) return "";
  return f.replaceAll("{pct}", pct);
}

function UsageTooltip({
  active,
  payload,
  seriesMeta,
  breakdownLabel,
  dailyTotalLabel,
  cumulativeTotalLabel,
  shareFmt,
}: TooltipProps<number, string> & {
  seriesMeta: SeriesMeta[];
  breakdownLabel: string;
  dailyTotalLabel: string;
  cumulativeTotalLabel: string;
  shareFmt: string;
}) {
  const theme = useTrimChartTheme();
  if (!active || !payload?.length) return null;
  const row = payload[0]?.payload as ChartRow | undefined;
  if (!row) return null;

  const dayLabel = (row.day_label || "").trim();
  if (!dayLabel) return null;

  const dailyTotalDisplay = String(row._daily_total_label ?? "").trim();
  const cumTotalDisplay = String(row._cum_total_label ?? "").trim();
  if (!dailyTotalDisplay || !cumTotalDisplay) return null;

  // Real life: most models are 0 on a given day - hide zeros, rank by daily usage.
  const activeRows = seriesMeta
    .map((s) => {
      const dailyKey = `d_${s.id}`;
      const cumKey = `s_${s.id}`;
      const dailyRaw = Number(row[dailyKey] ?? 0);
      const cumRaw = Number(row[cumKey] ?? 0);
      const dailyLabel = String(row[`${dailyKey}_label`] ?? "").trim();
      const cumLabel = String(row[`${cumKey}_label`] ?? "").trim();
      return { ...s, dailyRaw, cumRaw, dailyLabel, cumLabel };
    })
    .filter((s) => s.dailyRaw > 0 && s.dailyLabel)
    .sort((a, b) => b.dailyRaw - a.dailyRaw);

  const dailyTotal = activeRows.reduce((sum, s) => sum + s.dailyRaw, 0);
  const head = activeRows.slice(0, TOOLTIP_TOP_N);
  const tail = activeRows.slice(TOOLTIP_TOP_N);
  const tailSum = tail.reduce((s, r) => s + r.dailyRaw, 0);

  return (
    <div
      className="max-h-[min(70vh,22rem)] min-w-[11rem] max-w-[min(92vw,20rem)] overflow-y-auto rounded-lg border px-3 py-2 text-xs shadow-[var(--trim-float-shadow)]"
      style={chartTooltipStyle(theme)}
    >
      <p className="font-medium text-[var(--trim-fg)]">{dayLabel}</p>
      {breakdownLabel && head.length > 0 ? (
        <p className="mt-2 text-[10px] font-medium uppercase tracking-wide text-[var(--trim-muted)]">
          {breakdownLabel}
        </p>
      ) : null}
      {head.length > 0 ? (
        <ul className="mt-1.5 space-y-1">
          {head.map((s) => {
            const pct = dailyTotal > 0 ? ((s.dailyRaw / dailyTotal) * 100).toFixed(1) : "0.0";
            const share = applyShareFmt(shareFmt, pct);
            return (
              <li key={s.id} className="flex items-center gap-2 text-[var(--trim-fg)]">
                <span className="h-2 w-2 shrink-0 rounded-full" style={{ background: s.color }} />
                <span className="min-w-0 flex-1 truncate text-[var(--trim-muted)]" title={s.label}>
                  {s.label}
                </span>
                <span className="tabular-nums">{s.dailyLabel}</span>
                {share ? (
                  <span className="tabular-nums text-[var(--trim-muted)]">{share}</span>
                ) : null}
              </li>
            );
          })}
          {tail.length > 0 ? (
            <li className="flex items-center gap-2 text-[var(--trim-muted)]">
              <span className="h-2 w-2 shrink-0 rounded-full bg-[var(--trim-subtle)]" />
              <span className="min-w-0 flex-1 truncate">+{tail.length}</span>
              <span className="tabular-nums">{formatCompactNumber(tailSum)}</span>
            </li>
          ) : null}
        </ul>
      ) : null}
      <div
        className={`space-y-0.5 text-[var(--trim-muted)] ${
          head.length > 0 ? "mt-2 border-t border-[var(--trim-border)] pt-2" : "mt-2"
        }`}
      >
        {dailyTotalLabel ? (
          <p className="flex justify-between gap-3">
            <span>{dailyTotalLabel}</span>
            <span className="tabular-nums text-[var(--trim-fg)]">{dailyTotalDisplay}</span>
          </p>
        ) : null}
        {cumulativeTotalLabel ? (
          <p className="flex justify-between gap-3">
            <span>{cumulativeTotalLabel}</span>
            <span className="tabular-nums text-[var(--trim-fg)]">{cumTotalDisplay}</span>
          </p>
        ) : null}
      </div>
    </div>
  );
}

export function UsageStackedChart({
  title,
  subtitle,
  yAxisLabel,
  todayLabel,
  todayDay,
  groupByPrefix,
  groupByOptions,
  groupBySelected,
  onGroupByChange,
  series,
  days,
  emptyMessage,
  tooltipBreakdown,
  tooltipDailyTotal,
  tooltipCumulativeTotal,
  tooltipShareFmt,
  isFetching = false,
  locale = "",
}: {
  title?: string;
  subtitle?: string;
  yAxisLabel?: string;
  todayLabel?: string;
  todayDay?: string;
  groupByPrefix?: string;
  groupByOptions?: NamedOption[];
  groupBySelected?: string;
  onGroupByChange?: (id: string) => void;
  series?: UsageDaySeries[];
  days?: UsageAxisDay[];
  emptyMessage?: string;
  tooltipBreakdown?: string;
  tooltipDailyTotal?: string;
  tooltipCumulativeTotal?: string;
  tooltipShareFmt?: string;
  isFetching?: boolean;
  /** SITE_HTML_LANG / money_locale - empty skips locale formatting. */
  locale?: string;
}) {
  const theme = useTrimChartTheme();
  const palette = chartSeriesPalette(theme);
  const titleTrim = (title || "").trim();
  const options = (groupByOptions || []).filter((o) => o.id?.trim() && o.label?.trim());
  const points = series || [];
  const axisDays = days || [];
  const emptyTrim = (emptyMessage || "").trim();
  const breakdownTrim = (tooltipBreakdown || "").trim();
  const dailyTotalTrim = (tooltipDailyTotal || "").trim();
  const cumTotalTrim = (tooltipCumulativeTotal || "").trim();
  const shareFmtTrim = (tooltipShareFmt || "").trim();
  const richTooltip = Boolean(breakdownTrim && dailyTotalTrim && cumTotalTrim && shareFmtTrim);

  const { chartData, seriesMeta, plotSeriesMeta, showToday, hasSeries, legendItems } =
    useMemo(() => {
      const order: SeriesMeta[] = [];
      const seen = new Map<string, SeriesMeta>();
      const byDay = new Map<string, ChartRow>();
      const totals = new Map<string, number>();

      const ensureDay = (day: string, dayLabel: string) => {
        let bucket = byDay.get(day);
        if (!bucket) {
          const label = formatDateShort(day, locale) || dayLabel.trim();
          if (!label) return null;
          bucket = { day, day_label: label, _z: 0 };
          byDay.set(day, bucket);
        }
        return bucket;
      };

      for (const axis of axisDays) {
        const day = (axis.day || "").trim();
        if (!day) continue;
        const bucket = ensureDay(day, (axis.day_label || "").trim());
        if (!bucket) continue;
        if ((axis.daily_total_label || "").trim()) {
          bucket._daily_total_label = (axis.daily_total_label || "").trim();
        }
        if ((axis.cum_total_label || "").trim()) {
          bucket._cum_total_label = (axis.cum_total_label || "").trim();
        }
      }

      for (const row of points) {
        const day = (row.day || "").trim();
        if (!day) continue;
        const sid = row.series_id ?? "";
        const seriesLabel = (row.series_label || "").trim();
        const axisOnly = sid === AXIS_ONLY_SERIES_ID || (!seriesLabel && !sid);
        const bucket = ensureDay(day, (row.day_label || "").trim());
        if (!bucket) continue;
        if (axisOnly) continue;
        if (!seriesLabel) continue;
        if (!seen.has(sid)) {
          const meta: SeriesMeta = {
            id: sid,
            label: seriesLabel,
            color: palette[order.length % palette.length],
          };
          seen.set(sid, meta);
          order.push(meta);
        }
        const cumKey = `s_${sid}`;
        const dailyKey = `d_${sid}`;
        const daily = typeof row.daily_tokens === "number" ? row.daily_tokens : 0;
        bucket[cumKey] = row.tokens;
        bucket[dailyKey] = daily;
        totals.set(sid, (totals.get(sid) || 0) + daily);
        if ((row.tokens_label || "").trim())
          bucket[`${cumKey}_label`] = (row.tokens_label || "").trim();
        if ((row.daily_tokens_label || "").trim()) {
          bucket[`${dailyKey}_label`] = (row.daily_tokens_label || "").trim();
        }
      }

      const sortedDays = Array.from(byDay.keys()).sort();
      const data = sortedDays.flatMap((d) => {
        const row = byDay.get(d);
        return row ? [row] : [];
      });
      const today = (todayDay || "").trim();
      const todayInWindow = Boolean(today && byDay.has(today));

      // Legend ranked by period daily total (not discovery order).
      const ranked = [...order].sort((a, b) => (totals.get(b.id) || 0) - (totals.get(a.id) || 0));
      const periodTotal = ranked.reduce((s, m) => s + (totals.get(m.id) || 0), 0);
      const legend = ranked.map((m) => {
        const v = totals.get(m.id) || 0;
        return {
          id: m.id,
          label: m.label,
          color: m.color,
          value: v,
          valueLabel: locale ? v.toLocaleString(locale) : formatCompactNumber(v),
          pct: sharePct(v, periodTotal),
        };
      });

      // Stacked area: top N only - fold the long tail so 20 models don’t mud the chart.
      let plotMeta = ranked;
      if (ranked.length > CHART_STACK_TOP_N) {
        const head = ranked.slice(0, CHART_STACK_TOP_N);
        const tail = ranked.slice(CHART_STACK_TOP_N);
        const remColor = theme.subtle;
        for (const row of data) {
          let dailySum = 0;
          let cumSum = 0;
          for (const s of tail) {
            dailySum += Number(row[`d_${s.id}`] ?? 0);
            cumSum += Number(row[`s_${s.id}`] ?? 0);
          }
          row[`d_${REMAINDER_SERIES_ID}`] = dailySum;
          row[`s_${REMAINDER_SERIES_ID}`] = cumSum;
          if (dailySum > 0) {
            row[`d_${REMAINDER_SERIES_ID}_label`] = locale
              ? dailySum.toLocaleString(locale)
              : formatCompactNumber(dailySum);
          }
          if (cumSum > 0) {
            row[`s_${REMAINDER_SERIES_ID}_label`] = locale
              ? cumSum.toLocaleString(locale)
              : formatCompactNumber(cumSum);
          }
        }
        plotMeta = [
          ...head,
          { id: REMAINDER_SERIES_ID, label: `+${tail.length}`, color: remColor },
        ];
      }

      return {
        chartData: data,
        seriesMeta: ranked,
        plotSeriesMeta: plotMeta,
        showToday: Boolean(todayLabel?.trim() && todayInWindow),
        hasSeries: order.length > 0,
        legendItems: legend,
      };
    }, [points, axisDays, todayDay, todayLabel, palette, locale, theme.subtle]);

  if (!titleTrim || options.length === 0 || chartData.length === 0) {
    return null;
  }

  const selected = (groupBySelected || "").trim();
  const prefix = (groupByPrefix || "").trim();
  const yLabel = (yAxisLabel || "").trim();

  return (
    <div className="space-y-4">
      <FetchProgressBar active={isFetching} />
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <h3 className="text-lg font-semibold tracking-tight text-[var(--trim-fg)] sm:text-xl">
            {titleTrim}
          </h3>
          {subtitle?.trim() ? (
            <p className="mt-1 text-sm text-[var(--trim-muted)]">{subtitle.trim()}</p>
          ) : null}
          {yLabel ? (
            <p className="mt-1 text-[11px] uppercase tracking-wide text-[var(--trim-subtle)]">
              {yLabel}
            </p>
          ) : null}
        </div>
        {prefix && selected && onGroupByChange ? (
          <div className="flex w-full items-center gap-2 sm:w-auto">
            <span className="shrink-0 text-sm text-[var(--trim-muted)]">{prefix}</span>
            <Select value={selected} onValueChange={onGroupByChange}>
              <SelectTrigger className="h-9 w-full min-w-0 sm:w-[140px]" aria-label={prefix}>
                <SelectValue placeholder={prefix} />
              </SelectTrigger>
              <SelectContent>
                {options.map((opt) => (
                  <SelectItem key={opt.id} value={opt.id}>
                    {opt.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        ) : null}
      </div>

      <div className="relative h-52 w-full min-w-0 sm:h-64 md:h-72">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={chartData} margin={{ top: 12, right: 8, left: 0, bottom: 4 }}>
            <CartesianGrid stroke={theme.grid} strokeDasharray="3 3" vertical={false} />
            <XAxis
              dataKey="day_label"
              tick={{ fill: theme.axis, fontSize: 10 }}
              axisLine={{ stroke: theme.axisLine }}
              tickLine={false}
              interval="preserveStartEnd"
              minTickGap={28}
              height={28}
            />
            <YAxis
              tick={{ fill: theme.axis, fontSize: 10 }}
              axisLine={{ stroke: theme.axisLine }}
              tickLine={false}
              width={44}
              domain={hasSeries ? [0, "auto"] : [0, 1]}
              allowDecimals={false}
              tickFormatter={formatCompactNumber}
            />
            {richTooltip ? (
              <Tooltip
                content={
                  <UsageTooltip
                    seriesMeta={seriesMeta}
                    breakdownLabel={breakdownTrim}
                    dailyTotalLabel={dailyTotalTrim}
                    cumulativeTotalLabel={cumTotalTrim}
                    shareFmt={shareFmtTrim}
                  />
                }
                wrapperStyle={{ outline: "none", zIndex: 40 }}
              />
            ) : null}
            {showToday && todayLabel?.trim() ? (
              <ReferenceLine
                x={chartData.find((d) => d.day === todayDay)?.day_label || undefined}
                stroke={theme.subtle}
                strokeDasharray="4 4"
                label={{
                  value: todayLabel.trim(),
                  fill: theme.muted,
                  fontSize: 10,
                  position: "insideTopRight",
                }}
              />
            ) : null}
            {!hasSeries ? (
              <Area
                type="monotone"
                dataKey="_z"
                name={yLabel || " "}
                stroke="transparent"
                fill="transparent"
                fillOpacity={0}
                strokeWidth={0}
                legendType="none"
                activeDot={{ r: 4, fill: theme.muted, stroke: theme.axisLine }}
                isAnimationActive={false}
              />
            ) : null}
            {plotSeriesMeta.map((s) => (
              <Area
                key={s.id}
                type="monotone"
                dataKey={`s_${s.id}`}
                name={s.label}
                stackId="usage"
                stroke={s.color}
                fill={s.color}
                fillOpacity={s.id === REMAINDER_SERIES_ID ? 0.28 : 0.5}
                strokeWidth={1.25}
                isAnimationActive={false}
              />
            ))}
          </AreaChart>
        </ResponsiveContainer>
        {!hasSeries && emptyTrim ? (
          <div className="pointer-events-none absolute inset-y-0 left-11 right-2 flex items-center justify-center px-2 sm:right-3 sm:px-4">
            <p className="w-full max-w-[16rem] text-center text-xs leading-relaxed text-[var(--trim-muted)] sm:max-w-sm sm:text-sm">
              {emptyTrim}
            </p>
          </div>
        ) : null}
      </div>

      {hasSeries ? (
        <ChartLegendList items={legendItems} maxHeightClass="max-h-36 sm:max-h-44" />
      ) : null}
    </div>
  );
}
