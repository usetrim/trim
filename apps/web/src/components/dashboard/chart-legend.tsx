"use client";

import { cn } from "@/lib/utils";

export type ChartLegendItem = {
  id: string;
  label: string;
  color: string;
  /** Primary metric (tokens, runs, …) - already formatted when `valueLabel` is set. */
  value?: number;
  valueLabel?: string;
  /** 0–100 share of the visible total; omit to hide. */
  pct?: number;
};

/**
 * Ranked, scrollable legend for real-world cardinality (20+ models).
 * Replaces Recharts default Legend, which wraps and overlaps under load.
 */
export function ChartLegendList({
  items,
  className,
  maxHeightClass = "max-h-40 sm:max-h-48",
  columns = "auto",
}: {
  items: ChartLegendItem[];
  className?: string;
  maxHeightClass?: string;
  /** auto: 1 col mobile, 2 from sm when many rows */
  columns?: "auto" | 1 | 2;
}) {
  if (items.length === 0) return null;

  const colClass =
    columns === 1
      ? "grid-cols-1"
      : columns === 2
        ? "grid-cols-1 sm:grid-cols-2"
        : items.length > 6
          ? "grid-cols-1 sm:grid-cols-2"
          : "grid-cols-1";

  const scrollable = items.length > 8;

  return (
    <div className="relative min-w-0">
      <ul
        className={cn(
          "grid gap-x-4 gap-y-1.5 overflow-y-auto overscroll-contain pr-1 [scrollbar-width:thin]",
          scrollable && "pb-4",
          maxHeightClass,
          colClass,
          className,
        )}
        aria-label="Legend"
      >
        {items.map((item) => {
          const valueText =
            (item.valueLabel || "").trim() ||
            (typeof item.value === "number" ? String(item.value) : "");
          const pctText =
            typeof item.pct === "number" && Number.isFinite(item.pct)
              ? `${item.pct.toFixed(1)}%`
              : "";
          return (
            <li
              key={item.id}
              className="flex min-w-0 items-center gap-2 text-xs leading-snug text-[var(--trim-fg)]"
            >
              <span
                className="h-2 w-2 shrink-0 rounded-full ring-1 ring-[var(--trim-border-strong)]"
                style={{ background: item.color }}
                aria-hidden
              />
              <span className="min-w-0 flex-1 truncate text-[var(--trim-muted)]" title={item.label}>
                {item.label}
              </span>
              {valueText ? (
                <span className="shrink-0 tabular-nums text-[var(--trim-fg)]">{valueText}</span>
              ) : null}
              {pctText ? (
                <span className="w-11 shrink-0 text-right tabular-nums text-[var(--trim-subtle)]">
                  {pctText}
                </span>
              ) : null}
            </li>
          );
        })}
      </ul>
      {scrollable ? (
        <div
          className="pointer-events-none absolute inset-x-0 bottom-0 h-6 bg-gradient-to-t from-[var(--trim-panel)] to-transparent"
          aria-hidden
        />
      ) : null}
    </div>
  );
}
