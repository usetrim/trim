/**
 * Pure helpers for Trim IDE (unit-tested; no vscode import).
 */

/** Accept only http(s) absolute bases; empty on invalid. No invented host. */
export function parseApiBase(raw: string): string {
  const trimmed = raw.trim().replace(/\/$/, "");
  if (!trimmed) {
    return "";
  }
  let url: URL;
  try {
    url = new URL(trimmed);
  } catch {
    return "";
  }
  if (url.protocol !== "http:" && url.protocol !== "https:") {
    return "";
  }
  const path = url.pathname.replace(/\/$/, "");
  return `${url.origin}${path === "/" ? "" : path}`;
}

/** Join API base + relative path via URL (no string invent). */
export function joinApi(base: string, path: string): string {
  const root = base.endsWith("/") ? base : `${base}/`;
  return new URL(path.replace(/^\//, ""), root).toString();
}

export type CounterSnapshot = {
  tabShown: number;
  tabAccepted: number;
  linesAdded: number;
  linesDeleted: number;
};

export function emptyCounters(): CounterSnapshot {
  return { tabShown: 0, tabAccepted: 0, linesAdded: 0, linesDeleted: 0 };
}

/** Coerce persisted counters; negative / NaN → 0. */
export function normalizeCounters(
  raw: Partial<CounterSnapshot> | undefined | null,
): CounterSnapshot {
  if (!raw || typeof raw !== "object") {
    return emptyCounters();
  }
  const n = (v: unknown): number =>
    typeof v === "number" && Number.isFinite(v) && v > 0 ? Math.floor(v) : 0;
  return {
    tabShown: n(raw.tabShown),
    tabAccepted: n(raw.tabAccepted),
    linesAdded: n(raw.linesAdded),
    linesDeleted: n(raw.linesDeleted),
  };
}

export function pendingTotal(c: CounterSnapshot): number {
  return c.tabShown + c.tabAccepted + c.linesAdded + c.linesDeleted;
}

export function clientVersionHeader(version: string): string {
  return `trim-ide/${version.trim() || "0.0.0"}`;
}

export function userAgentHeader(version: string): string {
  return `TrimIDE/${version.trim() || "0.0.0"}`;
}
