"use client";

import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { DateRangePicker, type DateRangeValue } from "@/components/ui/date-range-picker";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { filterAllSentinel, isFilterAll, type FilterOption } from "@/lib/admin-filter-options";
import { cn } from "@/lib/utils";
import { memo } from "react";

type SearchFilter = {
  kind: "search";
  id: string;
  label: string;
  description?: string;
  value: string;
  onChange: (value: string) => void;
  className?: string;
};

type SelectFilter = {
  kind: "select";
  id: string;
  label: string;
  description?: string;
  value: string;
  onChange: (value: string) => void;
  options: FilterOption[];
  className?: string;
  /** Chrome-owned clear/"all" option label. When empty and not required, clear option is omitted. */
  allLabel?: string;
  /** Required picks (e.g. distribution range): no clear option; empty shows placeholder only. */
  required?: boolean;
};

type TextFilter = {
  kind: "text";
  id: string;
  label: string;
  description?: string;
  value: string;
  onChange: (value: string) => void;
  className?: string;
};

type DateRangeFilter = {
  kind: "dateRange";
  id: string;
  label: string;
  description?: string;
  value: DateRangeValue;
  onChange: (next: DateRangeValue) => void;
  placeholder: string;
  clearLabel: string;
  applyLabel: string;
  locale: string;
  numberOfMonths: number;
  className?: string;
};

export type AdminFilterSpec = SearchFilter | SelectFilter | TextFilter | DateRangeFilter;

/**
 * Aligned filter toolbar: search wider, enum Selects fixed width, free-text only when needed.
 * Memoized so parent list refetch re-renders do not rebuild search inputs.
 */
export const AdminFilterBar = memo(function AdminFilterBar({
  filters,
  className,
}: {
  filters: AdminFilterSpec[];
  className?: string;
}) {
  const visible = filters.filter((f) => Boolean(f.label?.trim()));
  if (!visible.length) return null;
  const all = filterAllSentinel();

  return (
    <div className={cn("flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-end", className)}>
      {visible.map((f) => {
        if (f.kind === "search") {
          return (
            <Field
              key={f.id}
              id={f.id}
              label={f.label}
              description={f.description}
              className={cn("w-full sm:min-w-[16rem] sm:flex-1 sm:max-w-md", f.className)}
            >
              <Input
                id={f.id}
                value={f.value}
                onChange={(e) => f.onChange(e.target.value)}
                onKeyDown={(e) => {
                  // Never submit a nearby pagination <form> via Enter.
                  if (e.key === "Enter") e.preventDefault();
                }}
                placeholder={f.label}
                autoComplete="off"
                spellCheck={false}
              />
            </Field>
          );
        }
        if (f.kind === "select") {
          const clearLabel = f.allLabel?.trim() || "";
          const showClear = !f.required && Boolean(clearLabel);
          // Keep the field in layout even before options load (no jump when catalog arrives).
          return (
            <Field
              key={f.id}
              id={f.id}
              label={f.label}
              description={f.description}
              className={cn("w-full sm:w-44", f.className)}
            >
              <Select
                value={f.value ? f.value : showClear ? all : undefined}
                onValueChange={(v) => f.onChange(isFilterAll(v) ? "" : v)}
                disabled={!f.options.length}
              >
                <SelectTrigger id={f.id} aria-label={f.label}>
                  <SelectValue placeholder={f.label} />
                </SelectTrigger>
                <SelectContent>
                  {showClear ? <SelectItem value={all}>{clearLabel}</SelectItem> : null}
                  {f.options.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
          );
        }
        if (f.kind === "dateRange") {
          const pickerReady =
            Boolean(f.placeholder?.trim()) &&
            Boolean(f.clearLabel?.trim()) &&
            Boolean(f.applyLabel?.trim()) &&
            Boolean(f.locale?.trim()) &&
            Number.isFinite(f.numberOfMonths) &&
            f.numberOfMonths >= 1;
          // Always keep the Field mounted (stable key) so late calendar chrome
          // never remounts sibling search inputs.
          return (
            <Field
              key={f.id}
              id={f.id}
              label={f.label}
              description={f.description}
              className={cn("w-full sm:w-auto", f.className)}
            >
              {pickerReady ? (
                <DateRangePicker
                  id={f.id}
                  value={f.value}
                  onChange={f.onChange}
                  placeholder={f.placeholder}
                  clearLabel={f.clearLabel}
                  applyLabel={f.applyLabel}
                  locale={f.locale}
                  numberOfMonths={f.numberOfMonths}
                />
              ) : (
                <div className="h-10 w-full min-w-[12rem] rounded-md border border-dashed border-border sm:w-56" />
              )}
            </Field>
          );
        }
        return (
          <Field
            key={f.id}
            id={f.id}
            label={f.label}
            description={f.description}
            className={cn("w-full sm:w-44", f.className)}
          >
            <Input
              id={f.id}
              value={f.value}
              onChange={(e) => f.onChange(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") e.preventDefault();
              }}
              placeholder={f.label}
              autoComplete="off"
              spellCheck={false}
            />
          </Field>
        );
      })}
    </div>
  );
});
