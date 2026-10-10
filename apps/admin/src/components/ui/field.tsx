"use client";

import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import type * as React from "react";

/**
 * shadcn-style field: label + control + optional description.
 * Description is a single reserved line by default so sibling fields stay
 * bottom-aligned; pass descriptionWrap for longer operator guidance.
 */
export function Field({
  id,
  label,
  description,
  descriptionWrap = false,
  className,
  children,
}: {
  id?: string;
  label: string;
  description?: string;
  /** When true, description can wrap (e.g. Sync to Paddle guidance). */
  descriptionWrap?: boolean;
  className?: string;
  children: React.ReactNode;
}) {
  const title = label.trim();
  if (!title) return null;
  const desc = description?.trim() || "";
  return (
    <div className={cn("flex flex-col gap-1.5", className)}>
      <Label htmlFor={id} className="line-clamp-1 min-h-5 leading-5" title={title}>
        {title}
      </Label>
      <div className="min-h-10 w-full [&>textarea]:min-h-[min(32dvh,16rem)]">{children}</div>
      <p
        className={cn(
          "min-h-5 text-xs leading-5 text-muted-foreground",
          descriptionWrap ? "text-pretty" : "line-clamp-1",
        )}
        title={desc || undefined}
      >
        {desc ? desc : "\u00a0"}
      </p>
    </div>
  );
}
