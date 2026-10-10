"use client";

import { chartTooltipStyle, useTrimChartTheme } from "@/lib/trim-chart-theme";
import type { ReactNode } from "react";

export type TrimChartTooltipRow = {
  name: string;
  value: ReactNode;
  /** Swatch color only - never used as text color. */
  swatch?: string;
};

/**
 * Shared custom tooltip body (same readability pattern as Usage stacked chart).
 * Prefer this over Recharts default content for pie/bar charts in dark mode:
 * Recharts paints item text with the series/slice color, which hides labels
 * on dark greyscale pies.
 */
export function TrimChartTooltipBody({
  title,
  rows,
}: {
  title?: string;
  rows: TrimChartTooltipRow[];
}) {
  const theme = useTrimChartTheme();
  const titleTrim = (title || "").trim();
  const visible = rows.filter(
    (r) => String(r.name || "").trim() && r.value != null && r.value !== "",
  );
  if (!titleTrim && visible.length === 0) return null;
  return (
    <div
      className="min-w-[10rem] max-w-[min(92vw,18rem)] rounded-lg border px-3 py-2 text-xs shadow-[var(--trim-float-shadow)]"
      style={chartTooltipStyle(theme)}
    >
      {titleTrim ? <p className="font-medium text-[var(--trim-fg)]">{titleTrim}</p> : null}
      {visible.length > 0 ? (
        <ul className={titleTrim ? "mt-1.5 space-y-1" : "space-y-1"}>
          {visible.map((r) => (
            <li key={r.name} className="flex items-center gap-2 text-[var(--trim-fg)]">
              {r.swatch ? (
                <span
                  className="h-2 w-2 shrink-0 rounded-full ring-1 ring-[var(--trim-border-strong)]"
                  style={{ background: r.swatch }}
                  aria-hidden
                />
              ) : null}
              <span className="min-w-0 flex-1 truncate text-[var(--trim-muted)]">{r.name}</span>
              <span className="shrink-0 tabular-nums text-[var(--trim-fg)]">{r.value}</span>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}
