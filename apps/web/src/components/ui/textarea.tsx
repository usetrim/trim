import { cn } from "@/lib/utils";
import * as React from "react";

const Textarea = React.forwardRef<HTMLTextAreaElement, React.ComponentProps<"textarea">>(
  ({ className, ...props }, ref) => {
    return (
      <textarea
        className={cn(
          "flex min-h-[min(40dvh,20rem)] w-full resize-y rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel)] px-3 py-2 text-sm text-[var(--trim-fg)] placeholder:text-[var(--trim-muted)] focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[var(--trim-border-strong)] focus-visible:ring-offset-1 focus-visible:ring-offset-[var(--trim-bg)] disabled:cursor-not-allowed disabled:opacity-50",
          className,
        )}
        ref={ref}
        {...props}
      />
    );
  },
);
Textarea.displayName = "Textarea";

export { Textarea };
