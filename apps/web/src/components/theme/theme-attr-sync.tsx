"use client";

import { applyDocumentDarkClass, resolvePrefersDark } from "@/lib/theme-dom";
import { useTheme } from "next-themes";
import { useEffect } from "react";

/**
 * Keep `.dark` + data-trim-theme aligned with next-themes.
 * Migrates legacy "system" → concrete light/dark once (binary toggle UX).
 */
export function ThemeAttrSync() {
  const { theme, resolvedTheme, setTheme } = useTheme();

  useEffect(() => {
    if (theme === undefined) return;

    if (theme === "system") {
      const next = (
        resolvedTheme === "light" || resolvedTheme === "dark"
          ? resolvedTheme
          : resolvePrefersDark()
            ? "dark"
            : "light"
      ) as "light" | "dark";
      applyDocumentDarkClass(next === "dark", next);
      setTheme(next);
      return;
    }

    if (theme !== "light" && theme !== "dark") return;
    applyDocumentDarkClass(theme === "dark", theme);
  }, [theme, resolvedTheme, setTheme]);

  return null;
}
