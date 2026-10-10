import { cn } from "@/lib/utils";
import * as React from "react";

const Input = React.forwardRef<HTMLInputElement, React.ComponentProps<"input">>(
  ({ className, type, ...props }, ref) => {
    return (
      <input
        type={type}
        className={cn(
          "flex h-10 w-full rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel)] px-3 py-2 text-sm text-[var(--trim-fg)] transition-colors file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-[var(--trim-muted)] focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[var(--trim-border-strong)] focus-visible:ring-offset-1 focus-visible:ring-offset-[var(--trim-bg)] disabled:cursor-not-allowed disabled:opacity-50",
          className,
        )}
        ref={ref}
        {...props}
      />
    );
  },
);
Input.displayName = "Input";

export { Input };
