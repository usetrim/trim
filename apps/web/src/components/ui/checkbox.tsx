"use client";

import { cn } from "@/lib/utils";
import * as CheckboxPrimitive from "@radix-ui/react-checkbox";
import { Check } from "lucide-react";
import * as React from "react";

const Checkbox = React.forwardRef<
  React.ElementRef<typeof CheckboxPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof CheckboxPrimitive.Root>
>(({ className, ...props }, ref) => (
  <CheckboxPrimitive.Root
    ref={ref}
    className={cn(
      "peer h-4 w-4 shrink-0 rounded-sm border border-[var(--trim-border-strong)] bg-[var(--trim-panel)] focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[var(--trim-border-strong)] disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:border-[var(--trim-ink)] data-[state=checked]:bg-[var(--trim-ink)] data-[state=checked]:text-[var(--trim-ink-inverse)]",
      className,
    )}
    {...props}
  >
    <CheckboxPrimitive.Indicator className={cn("flex items-center justify-center text-current")}>
      <Check className="h-3 w-3" />
    </CheckboxPrimitive.Indicator>
  </CheckboxPrimitive.Root>
));
Checkbox.displayName = CheckboxPrimitive.Root.displayName;

export { Checkbox };
