"use client";

import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import type * as React from "react";

/**
 * shadcn-style field: label + control + optional description.
 * Description is body copy under the control (not a placeholder).
 */
export function Field({
  id,
  label,
  description,
  className,
  children,
}: {
  id?: string;
  label: string;
  description?: string;
  className?: string;
  children: React.ReactNode;
}) {
  const title = label.trim();
  if (!title) return null;
  return (
    <div className={cn("space-y-1.5", className)}>
      <Label htmlFor={id}>{title}</Label>
      {children}
      {description?.trim() ? (
        <p className="text-xs text-[var(--trim-muted)]">{description.trim()}</p>
      ) : null}
    </div>
  );
}
