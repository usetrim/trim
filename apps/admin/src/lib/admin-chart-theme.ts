"use client";

import { useSyncExternalStore } from "react";
import type { CSSProperties } from "react";

export type AdminChartTheme = {
  grid: string;
  axis: string;
  axisLine: string;
  muted: string;
  subtle: string;
  border: string;
  popover: string;
  popoverFg: string;
  fg: string;
  isDark: boolean;
};

function readHsl(name: string, fallback: string): string {
  if (typeof window === "undefined") return fallback;
  const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
  return v ? `hsl(${v})` : fallback;
}

function readTheme(): AdminChartTheme {
  const isDark =
    typeof document !== "undefined" && document.documentElement.classList.contains("dark");
  const border = readHsl("--border", isDark ? "hsl(240 3.7% 15.9%)" : "hsl(240 5.9% 90%)");
  const muted = readHsl("--muted-foreground", isDark ? "hsl(240 5% 64.9%)" : "hsl(240 3.8% 46.1%)");
  return {
    grid: border,
    axis: muted,
    axisLine: border,
    muted,
    subtle: muted,
    border,
    popover: readHsl("--popover", isDark ? "hsl(240 10% 3.9%)" : "hsl(0 0% 100%)"),
    popoverFg: readHsl("--popover-foreground", isDark ? "hsl(0 0% 98%)" : "hsl(240 10% 3.9%)"),
    fg: readHsl("--foreground", isDark ? "hsl(0 0% 98%)" : "hsl(240 10% 3.9%)"),
    isDark,
  };
}

function themeEqual(a: AdminChartTheme, b: AdminChartTheme): boolean {
  return (
    a.isDark === b.isDark &&
    a.grid === b.grid &&
    a.axis === b.axis &&
    a.axisLine === b.axisLine &&
    a.muted === b.muted &&
    a.subtle === b.subtle &&
    a.border === b.border &&
    a.popover === b.popover &&
    a.popoverFg === b.popoverFg &&
    a.fg === b.fg
  );
}

let cached: AdminChartTheme | null = null;

function getTheme(): AdminChartTheme {
  const next = readTheme();
  if (cached && themeEqual(cached, next)) return cached;
  cached = next;
  return cached;
}

const SSR: AdminChartTheme = {
  grid: "hsl(240 5.9% 90%)",
  axis: "hsl(240 3.8% 46.1%)",
  axisLine: "hsl(240 5.9% 90%)",
  muted: "hsl(240 3.8% 46.1%)",
  subtle: "hsl(240 3.8% 46.1%)",
  border: "hsl(240 5.9% 90%)",
  popover: "hsl(0 0% 100%)",
  popoverFg: "hsl(240 10% 3.9%)",
  fg: "hsl(240 10% 3.9%)",
  isDark: false,
};

function subscribe(onStoreChange: () => void) {
  if (typeof document === "undefined") return () => {};
  const obs = new MutationObserver(onStoreChange);
  obs.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });
  return () => obs.disconnect();
}

export function useAdminChartTheme(): AdminChartTheme {
  return useSyncExternalStore(subscribe, getTheme, () => SSR);
}

export function adminChartTooltipStyle(t: AdminChartTheme): CSSProperties {
  return {
    background: t.popover,
    border: `1px solid ${t.border}`,
    borderRadius: 8,
    color: t.popoverFg,
  };
}

export function adminChartSeriesPalette(t: AdminChartTheme): string[] {
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
