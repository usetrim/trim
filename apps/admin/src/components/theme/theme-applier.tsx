"use client";

import { useThemeStore } from "@/stores/theme";
import { useEffect } from "react";

/** Apply stored light/dark on mount. No OS listener - app theme is explicit. */
export function ThemeApplier() {
  const hydrate = useThemeStore((s) => s.hydrate);

  useEffect(() => {
    hydrate();
  }, [hydrate]);

  return null;
}
