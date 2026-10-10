"use client";

import { cn } from "@/lib/utils";

export type ChartLegendItem = {
  id: string;
  label: string;
  color: string;
  value?: number;
  valueLabel?: string;
  pct?: number;
};

/** Ranked, scrollable legend - avoids Recharts Legend overlap with many series. */
export function ChartLegendList({
  items,
  className,
  maxHeightClass = "max-h-40 sm:max-h-48",
  columns = "auto",
}: {
  items: ChartLegendItem[];
  className?: string;
  maxHeightClass?: string;
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
              className="flex min-w-0 items-center gap-2 text-xs leading-snug text-[hsl(var(--foreground))]"
            >
              <span
                className="h-2 w-2 shrink-0 rounded-full ring-1 ring-[hsl(var(--border))]"
                style={{ background: item.color }}
                aria-hidden
              />
              <span
                className="min-w-0 flex-1 truncate text-[hsl(var(--muted-foreground))]"
                title={item.label}
              >
                {item.label}
              </span>
              {valueText ? (
                <span className="shrink-0 tabular-nums text-[hsl(var(--foreground))]">
                  {valueText}
                </span>
              ) : null}
              {pctText ? (
                <span className="w-11 shrink-0 text-right tabular-nums text-[hsl(var(--muted-foreground))]">
                  {pctText}
                </span>
              ) : null}
            </li>
          );
        })}
      </ul>
      {scrollable ? (
        <div
          className="pointer-events-none absolute inset-x-0 bottom-0 h-6 bg-gradient-to-t from-[hsl(var(--card))] to-transparent"
          aria-hidden
        />
      ) : null}
    </div>
  );
}
