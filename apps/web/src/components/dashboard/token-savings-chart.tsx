"use client";

import { TrimChartTooltipBody } from "@/components/dashboard/trim-chart-tooltip";
import { formatCompactNumber } from "@/lib/chart-series";
import { formatDateShort, formatDateTimeShort } from "@/lib/format-datetime";
import { useTrimChartTheme } from "@/lib/trim-chart-theme";
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

type EventPoint = {
  created_at: string;
  tokens_before: number;
  tokens_after: number;
};

type DaySeries = {
  day: string;
  day_label?: string;
  tokens_before: number;
  tokens_after: number;
  tokens_saved?: number;
};

function SavingsTooltip({
  active,
  payload,
  seriesName,
}: TooltipProps<number, string> & { seriesName?: string }) {
  if (!active || !payload?.length) return null;
  const item = payload[0];
  const row = item?.payload as { time?: string; saved?: number };
  const title = String(row?.time || "").trim();
  const name = (seriesName || String(item?.name || "")).trim();
  const value = String(row?.saved ?? item?.value ?? "");
  if (!title && !name) return null;
  return (
    <TrimChartTooltipBody
      title={title || undefined}
      rows={name ? [{ name, value, swatch: String(item?.color || "").trim() || undefined }] : []}
    />
  );
}

export function TokenSavingsChart({
  events,
  series,
  seriesName,
  emptyMessage,
  locale = "",
}: {
  events?: EventPoint[];
  series?: DaySeries[];
  /** Backend-owned legend label (no client invent). */
  seriesName?: string;
  /** Backend-owned empty state (no client invent). */
  emptyMessage?: string;
  /** SITE_HTML_LANG / money_locale - empty skips locale formatting. */
  locale?: string;
}) {
  const theme = useTrimChartTheme();
  const data = useMemo(() => {
    if (series && series.length > 0) {
      return series.map((d) => ({
        time: formatDateShort(d.day, locale) || d.day_label || d.day,
        before: d.tokens_before,
        after: d.tokens_after,
        saved:
          typeof d.tokens_saved === "number"
            ? d.tokens_saved
            : Math.max(0, d.tokens_before - d.tokens_after),
      }));
    }
    const list = events ?? [];
    const sorted = [...list].sort((a, b) =>
      a.created_at < b.created_at ? -1 : a.created_at > b.created_at ? 1 : 0,
    );
    return sorted.map((e) => ({
      time: formatDateTimeShort(e.created_at, locale),
      before: e.tokens_before,
      after: e.tokens_after,
      saved: Math.max(0, e.tokens_before - e.tokens_after),
    }));
  }, [events, series, locale]);

  if (!data.length) {
    return (
      <p className="py-10 text-center text-sm text-[var(--trim-muted)]">{emptyMessage || ""}</p>
    );
  }

  return (
    <div className="h-52 w-full min-w-0 sm:h-64 md:h-72">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 4 }}>
          <defs>
            <linearGradient id="trimSaved" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={theme.muted} stopOpacity={0.45} />
              <stop offset="100%" stopColor={theme.muted} stopOpacity={0.02} />
            </linearGradient>
          </defs>
          <CartesianGrid stroke={theme.grid} strokeDasharray="3 3" vertical={false} />
          <XAxis
            dataKey="time"
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
            tickFormatter={formatCompactNumber}
            allowDecimals={false}
          />
          <Tooltip
            content={<SavingsTooltip seriesName={seriesName} />}
            wrapperStyle={{ outline: "none", zIndex: 40 }}
          />
          <Area
            type="monotone"
            dataKey="saved"
            name={seriesName || ""}
            stroke={theme.ink}
            fill="url(#trimSaved)"
            strokeWidth={1.5}
            isAnimationActive={false}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
