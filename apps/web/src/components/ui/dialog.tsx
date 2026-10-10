"use client";

import { releaseBodyPointerEvents } from "@/hooks/use-deferred-dialog-selection";
import { cn } from "@/lib/utils";
import * as DialogPrimitive from "@radix-ui/react-dialog";
import { X } from "lucide-react";
import * as React from "react";

const Dialog = DialogPrimitive.Root;
const DialogTrigger = DialogPrimitive.Trigger;
const DialogPortal = DialogPrimitive.Portal;
const DialogClose = DialogPrimitive.Close;

const DialogOverlay = React.forwardRef<
  React.ElementRef<typeof DialogPrimitive.Overlay>,
  React.ComponentPropsWithoutRef<typeof DialogPrimitive.Overlay>
>(({ className, ...props }, ref) => (
  <DialogPrimitive.Overlay
    ref={ref}
    className={cn(
      "fixed inset-0 z-[100] bg-[var(--trim-overlay)] data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0",
      className,
    )}
    {...props}
  />
));
DialogOverlay.displayName = DialogPrimitive.Overlay.displayName;

function DialogFooter({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      data-slot="dialog-footer"
      className={cn(
        "flex shrink-0 flex-col-reverse gap-2 border-t border-[var(--trim-border)] bg-[var(--trim-panel)] px-6 py-4 sm:flex-row sm:justify-end",
        className,
      )}
      {...props}
    />
  );
}

function isDialogFooterElement(
  node: React.ReactNode,
): node is React.ReactElement<React.HTMLAttributes<HTMLDivElement>> {
  return React.isValidElement(node) && node.type === DialogFooter;
}

const DialogContent = React.forwardRef<
  React.ElementRef<typeof DialogPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof DialogPrimitive.Content> & {
    /** Backend-owned sr-only close label (no invent "Close"). */
    closeLabel?: string;
  }
>(({ className, children, closeLabel, onCloseAutoFocus, onInteractOutside, ...props }, ref) => {
  const childList = React.Children.toArray(children);
  const footers = childList.filter(isDialogFooterElement);
  const body = childList.filter((child) => !isDialogFooterElement(child));

  React.useEffect(() => {
    return () => {
      releaseBodyPointerEvents();
    };
  }, []);

  return (
    <DialogPortal>
      <DialogOverlay />
      <DialogPrimitive.Content
        ref={ref}
        className={cn(
          // Cap height; scroll body only; footers stay pinned; close stays outside scroll.
          "trim-float fixed left-[50%] top-[50%] z-[100] flex w-full max-w-lg max-h-[min(90dvh,56rem)] translate-x-[-50%] translate-y-[-50%] flex-col overflow-hidden border border-[var(--trim-border-strong)] bg-[var(--trim-panel)] p-0 text-[var(--trim-fg)] duration-200 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[state=closed]:slide-out-to-left-1/2 data-[state=closed]:slide-out-to-top-[48%] data-[state=open]:slide-in-from-left-1/2 data-[state=open]:slide-in-from-top-[48%] sm:rounded-lg",
          className,
        )}
        onCloseAutoFocus={(e) => {
          e.preventDefault();
          releaseBodyPointerEvents();
          onCloseAutoFocus?.(e);
        }}
        onInteractOutside={(e) => {
          const target = e.target as HTMLElement | null;
          if (
            target?.closest?.(
              "[data-radix-select-content],[data-radix-popper-content-wrapper],[role='listbox']",
            )
          ) {
            e.preventDefault();
            return;
          }
          onInteractOutside?.(e);
        }}
        {...props}
      >
        <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
          <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain p-6 pr-12">
            <div className="grid gap-4">{body}</div>
          </div>
          {footers}
        </div>
        <DialogPrimitive.Close className="absolute right-4 top-4 z-10 rounded-sm text-[var(--trim-muted)] opacity-70 transition hover:bg-[var(--trim-hover)] hover:text-[var(--trim-fg)] hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-[var(--trim-border-strong)]">
          <X className="h-4 w-4" />
          <span className="sr-only">{closeLabel || ""}</span>
        </DialogPrimitive.Close>
      </DialogPrimitive.Content>
    </DialogPortal>
  );
});
DialogContent.displayName = DialogPrimitive.Content.displayName;

function DialogHeader({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("flex flex-col space-y-1.5 text-center sm:text-left", className)}
      {...props}
    />
  );
}

function DialogTitle({
  className,
  ...props
}: React.ComponentPropsWithoutRef<typeof DialogPrimitive.Title>) {
  return (
    <DialogPrimitive.Title
      className={cn(
        "text-lg font-semibold leading-none tracking-tight text-[var(--trim-fg)]",
        className,
      )}
      {...props}
    />
  );
}

function DialogDescription({
  className,
  ...props
}: React.ComponentPropsWithoutRef<typeof DialogPrimitive.Description>) {
  return (
    <DialogPrimitive.Description
      className={cn("text-sm text-[var(--trim-muted)]", className)}
      {...props}
    />
  );
}

export {
  Dialog,
  DialogPortal,
  DialogOverlay,
  DialogClose,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogFooter,
  DialogTitle,
  DialogDescription,
};
