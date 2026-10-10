"use client";

import { cn } from "@/lib/utils";

/** Thin indeterminate bar for table/header fetch without unmounting content. */
export function FetchProgressBar({
  active,
  className,
}: {
  active: boolean;
  className?: string;
}) {
  if (!active) return null;
  return (
    <div
      className={cn("relative h-0.5 w-full overflow-hidden bg-[var(--trim-panel-2)]", className)}
      role="status"
      aria-live="polite"
      aria-busy="true"
    >
      <div className="absolute inset-y-0 w-1/3 animate-fetch-bar rounded-full bg-[var(--trim-fg)]" />
    </div>
  );
}
