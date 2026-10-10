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
};

/**
 * Chrome-gated row actions menu. Fail closed: no triggerLabel or no actions => null.
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
  return (
    <DropdownMenu modal={false}>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="h-8 w-8 text-[var(--trim-muted)] hover:text-[var(--trim-fg)]"
          aria-label={label}
          title={label}
          onClick={(e) => {
            e.stopPropagation();
          }}
          onPointerDown={(e) => {
            e.stopPropagation();
          }}
        >
          <MoreHorizontal className="h-4 w-4" />
          <span className="sr-only">{label}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        sideOffset={4}
        className="z-[100] min-w-[10rem]"
        onCloseAutoFocus={(e) => {
          // Opening dialogs/panels from a row action must not fight focus restore.
          e.preventDefault();
        }}
      >
        {items.map((action) => (
          <DropdownMenuItem
            key={action.id}
            disabled={action.disabled}
            className={
              action.destructive
                ? "text-destructive focus:bg-[var(--trim-hover)] focus:text-destructive"
                : undefined
            }
            onSelect={() => {
              action.onSelect();
            }}
          >
            {action.label.trim()}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
