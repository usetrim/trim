"use client";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";

/**
 * Controlled confirm dialog. Description + cancel + confirm labels required (fail closed).
 * pendingLabel is optional: when missing, confirm still works without a spinner label.
 */
export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  cancelLabel,
  confirmLabel,
  pendingLabel,
  isPending,
  onConfirm,
  destructive,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title?: string;
  description: string;
  cancelLabel: string;
  confirmLabel: string;
  pendingLabel?: string;
  isPending?: boolean;
  onConfirm: () => void;
  destructive?: boolean;
}) {
  const ready =
    Boolean(description.trim()) && Boolean(cancelLabel.trim()) && Boolean(confirmLabel.trim());

  return (
    <AlertDialog
      open={open && ready}
      onOpenChange={(next) => {
        if (isPending) return;
        onOpenChange(next);
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          {title?.trim() ? <AlertDialogTitle>{title}</AlertDialogTitle> : null}
          <AlertDialogDescription>{description}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isPending}>{cancelLabel}</AlertDialogCancel>
          <AlertDialogAction asChild>
            <Button
              type="button"
              variant={destructive ? "destructive" : "default"}
              isLoading={Boolean(pendingLabel?.trim()) && isPending}
              pendingLabel={pendingLabel}
              disabled={isPending}
              onClick={(e) => {
                e.preventDefault();
                onConfirm();
              }}
            >
              {confirmLabel}
            </Button>
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
