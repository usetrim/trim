"use client";

import { useSyncExternalStore } from "react";
import type { CSSProperties } from "react";

export type TrimChartTheme = {
  grid: string;
  axis: string;
  axisLine: string;
  muted: string;
  fg: string;
  panel: string;
  /** Elevated surface for floating tooltips (contrasts with chart panel). */
  tooltipBg: string;
  border: string;
  borderStrong: string;
  ink: string;
  subtle: string;
  bg: string;
  floatShadow: string;
  isDark: boolean;
};

function readVar(name: string, fallback: string): string {
  if (typeof window === "undefined") return fallback;
  const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
  return v || fallback;
}

function readTheme(): TrimChartTheme {
  const isDark =
    typeof document !== "undefined" && document.documentElement.classList.contains("dark");
  const panel = readVar("--trim-panel", isDark ? "#0f1011" : "#ffffff");
  // Elevate above chart cards so dark popovers contrast with --trim-panel.
  const tooltipBg = isDark
    ? "color-mix(in srgb, var(--trim-panel) 78%, var(--trim-fg) 22%)"
    : panel;
  return {
    grid: readVar("--trim-border", isDark ? "rgba(255,255,255,0.06)" : "#e8e8e8"),
    axis: readVar("--trim-subtle", isDark ? "#71717a" : "#8a8a8a"),
    axisLine: readVar("--trim-border-strong", isDark ? "rgba(255,255,255,0.14)" : "#cfcfcf"),
    muted: readVar("--trim-muted", isDark ? "#8a8f98" : "#5c5c5c"),
    fg: readVar("--trim-fg", isDark ? "#f7f8f8" : "#171717"),
    panel,
    tooltipBg,
    border: readVar("--trim-border", isDark ? "rgba(255,255,255,0.06)" : "#e8e8e8"),
    borderStrong: readVar("--trim-border-strong", isDark ? "rgba(255,255,255,0.14)" : "#cfcfcf"),
    ink: readVar("--trim-ink", isDark ? "#f7f8f8" : "#171717"),
    subtle: readVar("--trim-subtle", isDark ? "#71717a" : "#8a8a8a"),
    bg: readVar("--trim-bg", isDark ? "#08090a" : "#fafafa"),
    floatShadow: readVar(
      "--trim-float-shadow",
      isDark ? "0 10px 30px rgba(0,0,0,0.45)" : "0 8px 24px rgba(23,23,23,0.12)",
    ),
    isDark,
  };
}

function themeEqual(a: TrimChartTheme, b: TrimChartTheme): boolean {
  return (
    a.isDark === b.isDark &&
    a.grid === b.grid &&
    a.axis === b.axis &&
    a.axisLine === b.axisLine &&
    a.muted === b.muted &&
    a.fg === b.fg &&
    a.panel === b.panel &&
    a.tooltipBg === b.tooltipBg &&
    a.border === b.border &&
    a.borderStrong === b.borderStrong &&
    a.ink === b.ink &&
    a.subtle === b.subtle &&
    a.bg === b.bg &&
    a.floatShadow === b.floatShadow
  );
}

/** Cached snapshot - useSyncExternalStore requires referential stability when unchanged. */
let cachedTheme: TrimChartTheme | null = null;

function getTheme(): TrimChartTheme {
  const next = readTheme();
  if (cachedTheme && themeEqual(cachedTheme, next)) return cachedTheme;
  cachedTheme = next;
  return cachedTheme;
}

/** SSR fallback matches light tokens; client hydrates from live CSS vars. */
const SSR_THEME: TrimChartTheme = {
  grid: "#e8e8e8",
  axis: "#8a8a8a",
  axisLine: "#cfcfcf",
  muted: "#5c5c5c",
  fg: "#171717",
  panel: "#ffffff",
  tooltipBg: "#ffffff",
  border: "#e8e8e8",
  borderStrong: "#cfcfcf",
  ink: "#171717",
  subtle: "#8a8a8a",
  bg: "#fafafa",
  floatShadow: "0 8px 24px rgba(23,23,23,0.12)",
  isDark: false,
};

function getServerSnapshot(): TrimChartTheme {
  return SSR_THEME;
}

function subscribe(onStoreChange: () => void) {
  if (typeof document === "undefined") return () => {};
  const obs = new MutationObserver(onStoreChange);
  obs.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });
  return () => obs.disconnect();
}

/** Live chart chrome colors from CSS theme tokens (updates on light/dark toggle). */
export function useTrimChartTheme(): TrimChartTheme {
  return useSyncExternalStore(subscribe, getTheme, getServerSnapshot);
}

/** Floating tooltip chrome - elevated vs chart panel so dark popovers stay visible. */
export function chartTooltipStyle(t: TrimChartTheme): CSSProperties {
  return {
    background: t.tooltipBg,
    border: `1px solid ${t.borderStrong}`,
    borderRadius: 8,
    color: t.fg,
    boxShadow: t.floatShadow,
    padding: "8px 12px",
  };
}

/**
 * Recharts DefaultTooltipContent paints item text with the series/slice color.
 * Prefer TrimChartTooltipBody for pie/bar charts; these styles help if default content is kept.
 */
export function chartTooltipItemStyle(t: TrimChartTheme): CSSProperties {
  return {
    color: t.fg,
  };
}

export function chartTooltipLabelStyle(t: TrimChartTheme): CSSProperties {
  return {
    color: t.muted,
    marginBottom: 4,
  };
}

/** Greyscale pie slices that stay visible in both themes (longer for many models). */
export function chartPiePalette(t: TrimChartTheme): string[] {
  if (t.isDark) {
    return [
      "#fafafa",
      "#e4e4e7",
      "#d4d4d8",
      "#a1a1aa",
      "#71717a",
      "#52525b",
      "#3f3f46",
      "#27272a",
      "#c4b5fd",
      "#7dd3fc",
      "#86efac",
      "#fcd34d",
      "#f9a8d4",
      "#fdba74",
    ];
  }
  return [
    "#0a0a0a",
    "#262626",
    "#404040",
    "#525252",
    "#737373",
    "#a3a3a3",
    "#d4d4d4",
    "#e5e5e5",
    "#5b21b6",
    "#0369a1",
    "#166534",
    "#a16207",
    "#9d174d",
    "#c2410c",
  ];
}

/** Semantic fills for success / error outcome bars. */
export function chartOutcomeColors(t: TrimChartTheme): { success: string; error: string } {
  if (t.isDark) {
    return { success: "#4ade80", error: "#f87171" };
  }
  return { success: "#166534", error: "#b91c1c" };
}

/** Stacked-area series hues - 16 stops so 20 models don’t collide as hard. */
export function chartSeriesPalette(t: TrimChartTheme): string[] {
  if (t.isDark) {
    return [
      "#4ade80",
      "#38bdf8",
      "#60a5fa",
      "#a78bfa",
      "#f472b6",
      "#fb923c",
      "#facc15",
      "#2dd4bf",
      "#94a3b8",
      "#c084fc",
      "#34d399",
      "#f87171",
      "#22d3ee",
      "#e879f9",
      "#a3e635",
      "#fdba74",
    ];
  }
  return [
    "#166534",
    "#0369a1",
    "#1d4ed8",
    "#6d28d9",
    "#be185d",
    "#c2410c",
    "#a16207",
    "#0f766e",
    "#475569",
    "#7c3aed",
    "#15803d",
    "#b91c1c",
    "#0891b2",
    "#a21caf",
    "#4d7c0f",
    "#9a3412",
  ];
}
