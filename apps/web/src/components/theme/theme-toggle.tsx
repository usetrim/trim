"use client";

import { applyDocumentDarkClass } from "@/lib/theme-dom";
import { cn } from "@/lib/utils";
import { Moon, Sun } from "lucide-react";
import { useTheme } from "next-themes";
import { useEffect, useState } from "react";

/** Binary light ↔ dark only (no system third state). */
export function ThemeToggle({ className }: { className?: string }) {
  const { resolvedTheme, setTheme } = useTheme();
  const [mounted, setMounted] = useState(false);
  /** Optimistic target so the icon flips in the same click turn as the DOM. */
  const [pending, setPending] = useState<"light" | "dark" | null>(null);

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    if (pending && resolvedTheme === pending) {
      setPending(null);
    }
  }, [resolvedTheme, pending]);

  // Until mount, do not read resolvedTheme for labels/title - SSR has no
  // localStorage/OS preference, so theme-dependent attrs cause hydration mismatch.
  const isDark = mounted ? (pending ?? resolvedTheme) !== "light" : false;

  return (
    <button
      type="button"
      aria-label={mounted ? (isDark ? "Switch to light mode" : "Switch to dark mode") : "Theme"}
      title={mounted ? (isDark ? "Light mode" : "Dark mode") : "Theme"}
      disabled={!mounted}
      onClick={() => {
        const next = isDark ? "light" : "dark";
        setPending(next);
        applyDocumentDarkClass(next === "dark", next);
        setTheme(next);
      }}
      className={cn(
        "inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-[var(--trim-border)] bg-[var(--trim-bg)] text-[var(--trim-muted)] transition hover:border-[var(--trim-border-strong)] hover:bg-[var(--trim-hover)] hover:text-[var(--trim-fg)]",
        className,
      )}
    >
      {mounted ? (
        isDark ? (
          <Sun className="h-3.5 w-3.5" strokeWidth={1.75} />
        ) : (
          <Moon className="h-3.5 w-3.5" strokeWidth={1.75} />
        )
      ) : (
        <span className="h-3.5 w-3.5" aria-hidden />
      )}
    </button>
  );
}
