/** Max distinct series drawn on a stacked area (rest folded into +N). */
export const CHART_STACK_TOP_N = 8;

/** Compact axis tick: 1200 → 1.2k, 1_500_000 → 1.5M. */
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
