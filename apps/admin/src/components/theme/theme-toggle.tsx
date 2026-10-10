"use client";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useThemeStore } from "@/stores/theme";
import { Moon, Sun } from "lucide-react";
import { useEffect, useState } from "react";

/** Binary light ↔ dark only (no system third state). */
export function ThemeToggle({
  className,
  lightLabel = "",
  darkLabel = "",
}: {
  className?: string;
  /** Backend chrome: ADMIN_THEME_LIGHT */
  lightLabel?: string;
  /** Backend chrome: ADMIN_THEME_DARK */
  darkLabel?: string;
}) {
  const mode = useThemeStore((s) => s.mode);
  const setMode = useThemeStore((s) => s.setMode);
  const hydrate = useThemeStore((s) => s.hydrate);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    hydrate();
    setMounted(true);
  }, [hydrate]);

  const isDark = mounted && mode !== "light";
  const actionLabel = mounted ? (isDark ? lightLabel : darkLabel) : "";

  return (
    <Button
      type="button"
      variant="outline"
      size="icon"
      aria-label={actionLabel || undefined}
      title={actionLabel || undefined}
      disabled={!mounted}
      onClick={() => {
        setMode(isDark ? "light" : "dark");
      }}
      className={cn(className)}
    >
      {mounted ? (
        isDark ? (
          <Sun className="h-4 w-4" strokeWidth={1.75} />
        ) : (
          <Moon className="h-4 w-4" strokeWidth={1.75} />
        )
      ) : (
        <span className="h-4 w-4" aria-hidden />
      )}
    </Button>
  );
}
