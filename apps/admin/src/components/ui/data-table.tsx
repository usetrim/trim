"use client";

import { Checkbox } from "@/components/ui/checkbox";
import { FetchProgressBar } from "@/components/ui/fetch-progress";
import { SkipPagination, type SkipPaginationMeta } from "@/components/ui/pagination";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";
import {
  type ColumnDef,
  type RowSelectionState,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from "@tanstack/react-table";
import * as React from "react";

export type { ColumnDef, RowSelectionState };

export type DataTableSelectionChrome = {
  /** Aria / title for select-all. Empty = selection column hidden (fail closed). */
  selectAllLabel: string;
  /** Aria for per-row checkbox. Empty = selection column hidden. */
  selectRowLabel: string;
  /** Toolbar count template with {count}. Empty = no count text. */
  selectedCountFmt?: string;
};

export type DataTableProps<TData, TValue = unknown> = {
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
  meta?: SkipPaginationMeta | null;
  onPage?: (skip: number) => void;
  pageDisabled?: boolean;
  pageInputId?: string;
  getRowId?: (originalRow: TData, index: number) => string;
  className?: string;
  tableClassName?: string;
  headerRowClassName?: string;
  bodyRowClassName?: string | ((row: TData) => string | undefined);
  requireHeaders?: boolean;
  bordered?: boolean;
  /** Show top progress while fetching next page/filter without unmounting rows. */
  isFetching?: boolean;
  /**
   * Row selection (shadcn / TanStack). Only enable when chrome labels are set
   * and getRowId is provided. Parent owns bulk actions via toolbar / onSelectionChange.
   * Never invent client-only filters on skip-paginated lists.
   */
  selection?: {
    chrome: DataTableSelectionChrome;
    rowSelection: RowSelectionState;
    onRowSelectionChange: (
      updater: RowSelectionState | ((prev: RowSelectionState) => RowSelectionState),
    ) => void;
    /** Optional toolbar rendered when one or more rows are selected. */
    toolbar?: React.ReactNode;
  };
};

function columnHeaderText<TData, TValue>(col: ColumnDef<TData, TValue>): string {
  if (typeof col.header === "string") return col.header.trim();
  return "";
}

function selectedIds(state: RowSelectionState): string[] {
  return Object.entries(state)
    .filter(([, on]) => on)
    .map(([id]) => id);
}

/**
 * Shared shadcn DataTable (TanStack Table).
 * Pages pass column defs + row data; chrome labels gate columns (empty header = hidden).
 * Selection and row menus are opt-in and chrome-gated (fail closed).
 */
export function DataTable<TData, TValue = unknown>({
  columns,
  data,
  meta,
  onPage,
  pageDisabled,
  pageInputId,
  getRowId,
  className,
  tableClassName,
  headerRowClassName,
  bodyRowClassName,
  requireHeaders = true,
  bordered = true,
  isFetching = false,
  selection,
}: DataTableProps<TData, TValue>) {
  const selectAllLabel = selection?.chrome.selectAllLabel?.trim() || "";
  const selectRowLabel = selection?.chrome.selectRowLabel?.trim() || "";
  const selectedFmt = selection?.chrome.selectedCountFmt?.trim() || "";
  const selectionEnabled = Boolean(selection && selectAllLabel && selectRowLabel && getRowId);

  const visibleColumns = React.useMemo(() => {
    const base = requireHeaders
      ? columns.filter((col) => {
          const text = columnHeaderText(col);
          if (text) return true;
          const id = typeof col.id === "string" ? col.id : "";
          return id.startsWith("_");
        })
      : columns;
    if (!selectionEnabled || !selection) return base;
    const selectCol: ColumnDef<TData, TValue> = {
      id: "_select",
      header: ({ table }) => (
        <Checkbox
          checked={
            table.getIsAllPageRowsSelected()
              ? true
              : table.getIsSomePageRowsSelected()
                ? "indeterminate"
                : false
          }
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(Boolean(value))}
          aria-label={selectAllLabel}
          title={selectAllLabel}
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          disabled={!row.getCanSelect()}
          onCheckedChange={(value) => row.toggleSelected(Boolean(value))}
          aria-label={selectRowLabel}
          title={selectRowLabel}
        />
      ),
      enableSorting: false,
      enableHiding: false,
      size: 40,
    };
    return [selectCol, ...base];
  }, [columns, requireHeaders, selectionEnabled, selection, selectAllLabel, selectRowLabel]);

  const table = useReactTable({
    data,
    columns: visibleColumns,
    getCoreRowModel: getCoreRowModel(),
    getRowId: getRowId ? (row, index) => getRowId(row, index) : undefined,
    enableRowSelection: selectionEnabled,
    onRowSelectionChange:
      selectionEnabled && selection ? selection.onRowSelectionChange : undefined,
    state: selectionEnabled && selection ? { rowSelection: selection.rowSelection } : undefined,
  });

  if (visibleColumns.length === 0) return null;

  const ids = selectionEnabled && selection ? selectedIds(selection.rowSelection) : [];
  const countLabel =
    selectedFmt?.includes("{count}") && ids.length > 0
      ? selectedFmt.replaceAll("{count}", String(ids.length))
      : "";

  const tableEl = (
    <Table className={tableClassName}>
      <TableHeader>
        {table.getHeaderGroups().map((headerGroup) => (
          <TableRow key={headerGroup.id} className={cn("hover:bg-transparent", headerRowClassName)}>
            {headerGroup.headers.map((header) => (
              <TableHead
                key={header.id}
                style={header.getSize() !== 150 ? { width: header.getSize() } : undefined}
              >
                {header.isPlaceholder
                  ? null
                  : flexRender(header.column.columnDef.header, header.getContext())}
              </TableHead>
            ))}
          </TableRow>
        ))}
      </TableHeader>
      <TableBody>
        {table.getRowModel().rows.map((row) => {
          const extra =
            typeof bodyRowClassName === "function"
              ? bodyRowClassName(row.original)
              : bodyRowClassName;
          return (
            <TableRow
              key={row.id}
              data-state={row.getIsSelected() ? "selected" : undefined}
              className={extra}
            >
              {row.getVisibleCells().map((cell) => (
                <TableCell key={cell.id}>
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </TableCell>
              ))}
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );

  return (
    <div className={cn("space-y-4", className)}>
      {selectionEnabled && ids.length > 0 ? (
        <div className="flex flex-wrap items-center gap-3">
          {countLabel ? <span className="text-sm text-muted-foreground">{countLabel}</span> : null}
          {selection?.toolbar}
        </div>
      ) : null}
      {bordered ? (
        <div className="rounded-md border border-border bg-background">
          <FetchProgressBar active={isFetching} />
          <div className="overflow-x-auto">{tableEl}</div>
        </div>
      ) : (
        <div>
          <FetchProgressBar active={isFetching} />
          <div className="overflow-x-auto">{tableEl}</div>
        </div>
      )}
      {meta && onPage ? (
        <SkipPagination
          meta={meta}
          disabled={pageDisabled}
          inputId={pageInputId}
          onPrev={onPage}
          onNext={onPage}
        />
      ) : null}
    </div>
  );
}

/** Build a text column gated by a chrome label (empty label => omitted by DataTable). */
export function dataTableTextColumn<TData>(
  id: keyof TData & string,
  header: string,
  opts?: {
    cell?: ColumnDef<TData, unknown>["cell"];
    className?: string;
    mono?: boolean;
  },
): ColumnDef<TData, unknown> {
  return {
    id,
    accessorKey: id,
    header,
    cell:
      opts?.cell ??
      (({ getValue }) => {
        const v = getValue();
        if (v == null) return "";
        if (typeof v === "string") return v.trim();
        if (typeof v === "number" || typeof v === "boolean") return String(v);
        return "";
      }),
    meta: { className: opts?.className, mono: opts?.mono },
  };
}

export function getSelectedRowIds(state: RowSelectionState): string[] {
  return selectedIds(state);
}
