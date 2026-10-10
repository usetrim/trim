"use client";

import { FetchProgressBar } from "@/components/ui/fetch-progress";
import {
  formatDateLong,
  formatDateShort,
  formatMonthLong,
  formatMonthNarrow,
} from "@/lib/format-datetime";
import type { LocHeatmapDay, LocHeatmapStat, NamedOption } from "@/types/events";
import { useMemo, useState } from "react";

const INTENSITY_CLASS = [
  "bg-[var(--trim-panel-2)] border border-[var(--trim-border)]",
  "bg-[color-mix(in_srgb,var(--trim-status)_22%,var(--trim-panel-2))]",
  "bg-[color-mix(in_srgb,var(--trim-status)_40%,var(--trim-panel-2))]",
  "bg-[color-mix(in_srgb,var(--trim-status)_62%,var(--trim-panel-2))]",
  "bg-[color-mix(in_srgb,var(--trim-status)_82%,var(--trim-panel-2))] border border-[var(--trim-status)]/40",
];

const CELL = 11;
const GAP = 3;

function applyFmt(template: string, date: string, count: number, level: number): string {
  return template
    .replaceAll("{date}", date)
    .replaceAll("{count}", String(count))
    .replaceAll("{level}", String(level));
}

/** Prefer locale formatting from series days over English UTC server labels. */
function formatHeatmapStatValue(
  stat: LocHeatmapStat,
  days: LocHeatmapDay[],
  locale: string,
): string {
  const fallback = (stat.value || "").trim();
  if (!locale.trim() || days.length === 0) return fallback;

  if (stat.id === "most_active_day") {
    let peak: LocHeatmapDay | null = null;
    for (const d of days) {
      if (!peak || d.value > peak.value) peak = d;
    }
    if (peak?.day) {
      return formatDateShort(peak.day, locale) || formatDateLong(peak.day, locale) || fallback;
    }
  }

  if (stat.id === "most_active_month") {
    const totals = new Map<string, number>();
    for (const d of days) {
      const day = (d.day || "").trim();
      if (day.length < 7) continue;
      const ym = day.slice(0, 7);
      totals.set(ym, (totals.get(ym) || 0) + (typeof d.value === "number" ? d.value : 0));
    }
    let peakYm = "";
    let peakVal = 0;
    for (const [ym, v] of totals) {
      if (v > peakVal || (v === peakVal && (peakYm === "" || ym < peakYm))) {
        peakVal = v;
        peakYm = ym;
      }
    }
    if (peakYm) {
      return formatMonthLong(peakYm, locale) || fallback;
    }
  }

  return fallback;
}

export function LocHeatmap({
  title,
  total,
  scopes,
  scopeSelected,
  onScopeChange,
  days,
  emptyFmt,
  valueFmt,
  weekdayLabels,
  stats,
  isFetching = false,
  locale = "",
}: {
  title?: string;
  total?: number;
  scopes?: NamedOption[];
  scopeSelected?: string;
  onScopeChange?: (id: string) => void;
  days?: LocHeatmapDay[];
  emptyFmt?: string;
  valueFmt?: string;
  weekdayLabels?: NamedOption[];
  stats?: LocHeatmapStat[];
  isFetching?: boolean;
  /** SITE_HTML_LANG / money_locale - empty skips locale formatting. */
  locale?: string;
}) {
  const titleTrim = (title || "").trim();
  const scopeOpts = (scopes || []).filter((o) => o.id?.trim() && o.label?.trim());
  const cells = days || [];
  const empty = (emptyFmt || "").trim();
  const value = (valueFmt || "").trim();
  const wdLabels = (weekdayLabels || []).filter((o) => o.id?.trim() && o.label?.trim());
  const statRows = (stats || []).filter((s) => s.id?.trim() && s.label?.trim() && s.value?.trim());
  const [hover, setHover] = useState<{
    text: string;
    x: number;
    y: number;
    day: string;
  } | null>(null);

  const calendar = useMemo(() => {
    if (cells.length === 0) {
      return { weeks: [] as (LocHeatmapDay | null)[][], monthLabels: [] as string[] };
    }
    const firstWd =
      typeof cells[0].weekday_mon0 === "number" &&
      cells[0].weekday_mon0 >= 0 &&
      cells[0].weekday_mon0 <= 6
        ? cells[0].weekday_mon0
        : 0;
    const flat: (LocHeatmapDay | null)[] = [];
    for (let i = 0; i < firstWd; i++) flat.push(null);
    for (const d of cells) flat.push(d);
    while (flat.length % 7 !== 0) flat.push(null);

    const weekCount = flat.length / 7;
    const weeks: (LocHeatmapDay | null)[][] = [];
    const monthLabels: string[] = [];
    let prevYm = "";
    for (let w = 0; w < weekCount; w++) {
      const col: (LocHeatmapDay | null)[] = [];
      let month = "";
      let weekYm = "";
      for (let r = 0; r < 7; r++) {
        const cell = flat[w * 7 + r];
        col.push(cell);
        const day = cell?.day?.trim() || "";
        if (!weekYm && day.length >= 7) weekYm = day.slice(0, 7);
      }
      if (weekYm && weekYm !== prevYm) {
        month =
          formatMonthNarrow(`${weekYm}-01`, locale) ||
          // Fall back to first non-empty server month letter in this column.
          col
            .find((c) => c?.month_label?.trim())
            ?.month_label?.trim() ||
          "";
        prevYm = weekYm;
      }
      weeks.push(col);
      monthLabels.push(month);
    }
    return { weeks, monthLabels };
  }, [cells, locale]);

  if (!titleTrim || scopeOpts.length === 0 || cells.length === 0 || (!empty && !value)) {
    return null;
  }

  const selected = (scopeSelected || "").trim();
  const wdByRow = new Map<number, string>();
  for (const opt of wdLabels) {
    const row = Number(opt.id);
    if (Number.isInteger(row) && row >= 0 && row <= 6) {
      wdByRow.set(row, opt.label);
    }
  }

  function intensityOf(d: LocHeatmapDay): number {
    if (typeof d.intensity === "number" && d.intensity >= 0 && d.intensity <= 4) {
      return d.intensity;
    }
    return 0;
  }

  const weekCols = calendar.weeks.length;
  const labelCol = wdByRow.size > 0 ? 28 : 0;
  const gridWidth = weekCols * CELL + Math.max(0, weekCols - 1) * GAP;

  return (
    <div className="space-y-4">
      <FetchProgressBar active={isFetching} />
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="min-w-0">
          <h3 className="text-base font-semibold tracking-tight text-[var(--trim-fg)] sm:text-lg">
            {titleTrim}
          </h3>
          {typeof total === "number" ? (
            <p className="mt-1 text-3xl font-semibold tabular-nums tracking-tight text-[var(--trim-fg)] sm:text-4xl">
              {locale ? total.toLocaleString(locale) : total.toLocaleString()}
            </p>
          ) : null}
        </div>
        {selected && onScopeChange && scopeOpts.length > 0 ? (
          <div
            className="inline-flex w-full flex-wrap gap-1 rounded-lg border border-[var(--trim-border)] bg-[var(--trim-panel-2)] p-1 sm:w-auto"
            role="tablist"
            aria-label={titleTrim}
          >
            {scopeOpts.map((opt) => {
              const active = opt.id === selected;
              return (
                <button
                  key={opt.id}
                  type="button"
                  role="tab"
                  aria-selected={active}
                  onClick={() => onScopeChange(opt.id)}
                  className={
                    active
                      ? "min-w-0 flex-1 rounded-md bg-[var(--trim-panel)] px-3 py-1.5 text-sm font-medium text-[var(--trim-fg)] shadow-[var(--trim-card-shadow)] sm:flex-none"
                      : "min-w-0 flex-1 rounded-md px-3 py-1.5 text-sm text-[var(--trim-muted)] hover:text-[var(--trim-fg)] sm:flex-none"
                  }
                >
                  {opt.label}
                </button>
              );
            })}
          </div>
        ) : null}
      </div>
      <div className="relative overflow-x-auto pb-1">
        <div className="inline-block min-w-0" style={{ width: labelCol + gridWidth }}>
          {calendar.monthLabels.some((m) => m) ? (
            <div
              className="mb-1 grid"
              style={{
                gridTemplateColumns: `${
                  labelCol ? `${labelCol}px ` : ""
                }repeat(${weekCols}, ${CELL}px)`,
                columnGap: `${GAP}px`,
              }}
            >
              {labelCol ? <div /> : null}
              {calendar.monthLabels.map((m, i) => (
                // biome-ignore lint/suspicious/noArrayIndexKey: static month header slots never reorder
                <div key={`m-${i}`} className="text-[10px] leading-none text-[var(--trim-subtle)]">
                  {i === 0 || m !== calendar.monthLabels[i - 1] ? m || "" : ""}
                </div>
              ))}
            </div>
          ) : null}
          <div className="flex" style={{ gap: `${GAP}px` }}>
            {labelCol ? (
              <div className="flex flex-col" style={{ width: labelCol, gap: `${GAP}px` }}>
                {[0, 1, 2, 3, 4, 5, 6].map((row) => (
                  <div
                    key={`wd-${row}`}
                    className="flex items-center justify-end pr-1 text-[10px] leading-none text-[var(--trim-subtle)]"
                    style={{ height: CELL }}
                  >
                    {wdByRow.get(row) || ""}
                  </div>
                ))}
              </div>
            ) : null}
            <div
              className="grid"
              style={{
                width: gridWidth,
                gridTemplateColumns: `repeat(${weekCols}, ${CELL}px)`,
                gridTemplateRows: `repeat(7, ${CELL}px)`,
                gap: `${GAP}px`,
                gridAutoFlow: "column",
              }}
            >
              {calendar.weeks
                .flatMap((col, wi) =>
                  col.map((d, ri) => ({
                    d,
                    key: d?.day ? d.day : `pad-${wi}-${ri}`,
                  })),
                )
                .map(({ d, key }) => {
                  if (!d) {
                    return <div key={key} className="rounded-sm bg-transparent" />;
                  }
                  const level = intensityOf(d);
                  const dateLabel =
                    formatDateLong(d.day, locale) || (d.day_label_long || "").trim();
                  const tip =
                    dateLabel && d.value > 0 && value
                      ? applyFmt(value, dateLabel, d.value, level)
                      : dateLabel && empty
                        ? applyFmt(empty, dateLabel, 0, level)
                        : "";
                  return (
                    <div
                      key={key}
                      className={`rounded-sm ${INTENSITY_CLASS[level] || INTENSITY_CLASS[0]} ${
                        hover?.day === d.day ? "ring-1 ring-[var(--trim-ink)]" : ""
                      }`}
                      onMouseEnter={(e) => {
                        if (!tip) return;
                        const rect = (e.currentTarget as HTMLDivElement).getBoundingClientRect();
                        setHover({
                          text: tip,
                          x: rect.left + rect.width / 2,
                          y: rect.top,
                          day: d.day,
                        });
                      }}
                      onMouseLeave={() => setHover(null)}
                    />
                  );
                })}
            </div>
          </div>
        </div>
        {hover ? (
          <div
            className="pointer-events-none fixed z-[100] -translate-x-1/2 -translate-y-full whitespace-pre-line rounded-md border border-[var(--trim-border-strong)] bg-[var(--trim-panel)] px-2 py-1 text-xs text-[var(--trim-fg)] trim-float"
            style={{ left: hover.x, top: hover.y - 6 }}
          >
            {hover.text}
          </div>
        ) : null}
      </div>
      {statRows.length > 0 ? (
        <div className="grid grid-cols-2 gap-3 border-t border-[var(--trim-border)] pt-4 sm:grid-cols-4">
          {statRows.map((s) => (
            <div key={s.id} className="min-w-0">
              <p className="text-[11px] text-[var(--trim-muted)]">{s.label}</p>
              <p className="mt-0.5 truncate text-sm font-semibold tabular-nums text-[var(--trim-fg)]">
                {formatHeatmapStatValue(s, cells, locale)}
              </p>
            </div>
          ))}
        </div>
      ) : null}
    </div>
  );
}
