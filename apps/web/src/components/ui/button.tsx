import { cn } from "@/lib/utils";
import { Slot } from "@radix-ui/react-slot";
import { type VariantProps, cva } from "class-variance-authority";
import { Loader2 } from "lucide-react";
import * as React from "react";

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[var(--trim-border-strong)] disabled:pointer-events-none disabled:opacity-50",
  {
    variants: {
      variant: {
        default: "bg-[var(--trim-ink)] text-[var(--trim-ink-inverse)] hover:opacity-90",
        secondary:
          "border border-[var(--trim-border)] bg-[var(--trim-panel)] text-[var(--trim-fg)] shadow-[var(--trim-card-shadow)] hover:border-[var(--trim-border-strong)] hover:bg-[var(--trim-hover)]",
        outline:
          "border border-[var(--trim-border)] bg-[var(--trim-panel)] text-[var(--trim-fg)] shadow-[var(--trim-card-shadow)] hover:border-[var(--trim-border-strong)] hover:bg-[var(--trim-hover)]",
        ghost: "text-[var(--trim-fg)] hover:bg-[var(--trim-hover)] hover:text-[var(--trim-fg)]",
        destructive:
          "bg-[hsl(var(--destructive))] text-[hsl(var(--destructive-foreground))] hover:opacity-90",
      },
      size: {
        default: "h-9 px-4 py-2",
        sm: "h-8 rounded-md px-3 text-xs",
        lg: "h-10 rounded-md px-8",
        icon: "h-9 w-9",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
  /** When true, disables the control and shows pendingLabel with spinner. */
  isLoading?: boolean;
  /**
   * Action-aware pending copy from the API (e.g. "Saving...", "Deleting...").
   * Required when isLoading is true: never invent pending text from idle children.
   */
  pendingLabel?: string;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      className,
      variant,
      size,
      asChild = false,
      isLoading = false,
      pendingLabel,
      disabled,
      children,
      ...props
    },
    ref,
  ) => {
    const Comp = asChild ? Slot : "button";
    const pendingText = pendingLabel?.trim() || "";
    const showPending = isLoading && !asChild;

    if (process.env.NODE_ENV !== "production" && isLoading && !asChild && !pendingText) {
      console.error(
        "Button: isLoading requires pendingLabel from the API (never invent idle children).",
      );
    }

    return (
      <Comp
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        disabled={disabled || isLoading}
        aria-busy={isLoading || undefined}
        {...props}
      >
        {showPending ? (
          <>
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
            {pendingText || null}
          </>
        ) : (
          children
        )}
      </Comp>
    );
  },
);
Button.displayName = "Button";

export { Button, buttonVariants };
