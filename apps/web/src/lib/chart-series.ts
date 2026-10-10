/**
 * Helpers for dashboard charts under real cardinality (many models / zero days).
 */

/** Max distinct series drawn on a stacked area (rest folded into +N). */
export const CHART_STACK_TOP_N = 8;

/** Max pie slices before remainder fold. */
export const CHART_PIE_TOP_N = 7;

export type NamedMetric = {
  id: string;
  name: string;
  value: number;
  /** Extra fields preserved on the primary rows (e.g. saved, count). */
  meta?: Record<string, number>;
};

export type BucketResult<T extends NamedMetric> = {
  /** Slices for the pie/bar - top N + optional remainder. */
  chart: T[];
  /** Full ranked list for the legend (includes every original row). */
  legend: T[];
  remainderValue: number;
};

/**
 * Keep the top `limit` rows for the visual; fold the rest into one remainder row
 * when `remainderName` is non-empty. Legend always lists every original row.
 */
export function bucketTopN<T extends NamedMetric>(
  rows: T[],
  limit: number,
  remainderName: string,
): BucketResult<T> {
  const sorted = [...rows].sort((a, b) => b.value - a.value);
  if (sorted.length <= limit || limit < 1) {
    return { chart: sorted, legend: sorted, remainderValue: 0 };
  }
  const head = sorted.slice(0, limit);
  const tail = sorted.slice(limit);
  const remainderValue = tail.reduce((s, r) => s + r.value, 0);
  const name = remainderName.trim();
  if (!name || remainderValue <= 0) {
    return { chart: sorted, legend: sorted, remainderValue: 0 };
  }
  const remainder = {
    ...head[0],
    id: "__remainder__",
    name,
    value: remainderValue,
    meta: undefined,
  } as T;
  return { chart: [...head, remainder], legend: sorted, remainderValue };
}

/** Compact axis tick: 1200 → 1.2k, 1_500_000 → 1.5M (locale-agnostic ASCII). */
export function formatCompactNumber(n: number): string {
  if (!Number.isFinite(n)) return "";
  const abs = Math.abs(n);
  if (abs >= 1_000_000) {
    const v = n / 1_000_000;
    return `${trimNum(v)}M`;
  }
  if (abs >= 1_000) {
    const v = n / 1_000;
    return `${trimNum(v)}k`;
  }
  return String(Math.round(n));
}

function trimNum(v: number): string {
  const t = Math.abs(v) >= 10 ? v.toFixed(0) : v.toFixed(1);
  return t.replace(/\.0$/, "");
}

export function sharePct(part: number, total: number): number | undefined {
  if (!(total > 0) || !Number.isFinite(part)) return undefined;
  return Math.round((part / total) * 1000) / 10;
}
