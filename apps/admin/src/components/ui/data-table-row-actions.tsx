"use client";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { MoreHorizontal } from "lucide-react";

export type DataTableRowAction = {
  id: string;
  label: string;
  onSelect: () => void;
  disabled?: boolean;
  destructive?: boolean;
  isLoading?: boolean;
  pendingLabel?: string;
};

/**
 * Chrome-gated row actions menu. Fail closed: no triggerLabel or no actions => null.
 * When an action is loading with pendingLabel, that item shows the pending text and is disabled.
 */
export function DataTableRowActions({
  triggerLabel,
  actions,
}: {
  triggerLabel: string;
  actions: DataTableRowAction[];
}) {
  const label = triggerLabel.trim();
  const items = actions.filter((a) => a.label.trim());
  if (!label || items.length === 0) return null;
  const anyLoading = items.some((a) => Boolean(a.isLoading && a.pendingLabel?.trim()));
  return (
    <DropdownMenu modal={false}>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="h-8 w-8 text-muted-foreground hover:text-foreground"
          aria-label={label}
          title={label}
          disabled={anyLoading}
          onClick={(e) => {
            e.stopPropagation();
          }}
          onPointerDown={(e) => {
            e.stopPropagation();
          }}
        >
          <MoreHorizontal className="h-4 w-4" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        sideOffset={4}
        className="z-[100] min-w-[10rem]"
        onCloseAutoFocus={(e) => {
          e.preventDefault();
        }}
      >
        {items.map((action) => {
          const pending = action.pendingLabel?.trim() || "";
          const loading = Boolean(action.isLoading && pending);
          const text = loading ? pending : action.label.trim();
          return (
            <DropdownMenuItem
              key={action.id}
              disabled={action.disabled || loading}
              className={
                action.destructive
                  ? "text-destructive focus:bg-destructive/10 focus:text-destructive"
                  : undefined
              }
              onSelect={() => {
                if (loading) return;
                queueMicrotask(() => action.onSelect());
              }}
            >
              {text}
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
