"use client";

import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { CalendarIcon } from "lucide-react";
import * as React from "react";
import type { DateRange } from "react-day-picker";

export type DateRangeValue = {
  from: string | null;
  to: string | null;
};

function toISODate(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

function parseISODate(s: string | null | undefined): Date | undefined {
  if (!s || !/^\d{4}-\d{2}-\d{2}$/.test(s)) return undefined;
  const [y, m, d] = s.split("-").map(Number);
  return new Date(y, m - 1, d);
}

function rangeFromValue(value: DateRangeValue): DateRange | undefined {
  const from = parseISODate(value.from);
  const to = parseISODate(value.to);
  if (!from && !to) return undefined;
  return { from, to };
}

function valueFromRange(range: DateRange | undefined): DateRangeValue {
  return {
    from: range?.from ? toISODate(range.from) : null,
    to: range?.to ? toISODate(range.to) : null,
  };
}

/** Locale-aware short date; locale must come from SITE_HTML_LANG (no undefined invent). */
function formatLocalDate(d: Date, locale: string): string {
  if (!locale.trim()) return "";
  return new Intl.DateTimeFormat(locale, {
    year: "numeric",
    month: "short",
    day: "numeric",
  }).format(d);
}

/** Tailwind `sm` (640px): two months side-by-side only when there is room. */
function useWideCalendar(minWidthPx = 640) {
  // Assume narrow until measured so mobile never flashes a clipped two-month stack.
  const [wide, setWide] = React.useState(false);
  React.useEffect(() => {
    const mq = window.matchMedia(`(min-width: ${minWidthPx}px)`);
    const apply = () => setWide(mq.matches);
    apply();
    mq.addEventListener("change", apply);
    return () => mq.removeEventListener("change", apply);
  }, [minWidthPx]);
  return wide;
}

export function DateRangePicker({
  id,
  value,
  onChange,
  placeholder,
  clearLabel,
  applyLabel,
  locale,
  numberOfMonths,
  className,
}: {
  id?: string;
  value: DateRangeValue;
  onChange: (next: DateRangeValue) => void;
  /** Backend-owned placeholder when no range selected. */
  placeholder: string;
  /** Backend-owned clear action label. */
  clearLabel: string;
  /** Backend-owned apply / done label (commits draft + closes popover). */
  applyLabel: string;
  /** SITE_HTML_LANG from API (required; empty skips locale formatting). */
  locale: string;
  /** billing_settings.date_range_months from API; fail closed when < 1. */
  numberOfMonths: number;
  className?: string;
}) {
  const [open, setOpen] = React.useState(false);
  const [draft, setDraft] = React.useState<DateRange | undefined>(() => rangeFromValue(value));
  const wide = useWideCalendar();
  const months =
    Number.isFinite(numberOfMonths) && numberOfMonths >= 1 ? Math.floor(numberOfMonths) : 0;
  // Mobile / narrow: one month + month/year dropdowns (scroll months via caption).
  const visibleMonths = months < 1 ? 0 : wide ? months : 1;

  const label = React.useMemo(() => {
    const from = parseISODate(value.from);
    const to = parseISODate(value.to);
    if (from && to) {
      const a = formatLocalDate(from, locale);
      const b = formatLocalDate(to, locale);
      if (!a || !b) return placeholder;
      return `${a} - ${b}`;
    }
    if (from) {
      return formatLocalDate(from, locale) || placeholder;
    }
    return placeholder;
  }, [value.from, value.to, placeholder, locale]);

  const handleOpenChange = (next: boolean) => {
    if (next) {
      // Sync draft from committed value when opening so Cancel/outside-dismiss is safe.
      setDraft(rangeFromValue(value));
    }
    setOpen(next);
  };

  if (visibleMonths < 1) {
    return null;
  }

  return (
    <div className={cn("flex flex-wrap items-center gap-2", className)}>
      <Popover open={open} onOpenChange={handleOpenChange}>
        <PopoverTrigger asChild>
          <Button
            id={id}
            type="button"
            variant="outline"
            className={cn(
              "w-full max-w-full justify-start text-left font-normal sm:w-auto",
              !value.from && "text-muted-foreground",
            )}
          >
            <CalendarIcon className="mr-2 h-4 w-4 shrink-0" />
            <span className="truncate">{label}</span>
          </Button>
        </PopoverTrigger>
        <PopoverContent
          className="flex w-auto max-w-[calc(100vw-1.5rem)] flex-col overflow-hidden p-0"
          align="start"
          side="bottom"
          sideOffset={8}
          collisionPadding={16}
          avoidCollisions
          onOpenAutoFocus={(e) => e.preventDefault()}
          onPointerDownOutside={(e) => {
            const node = e.target as HTMLElement | null;
            if (node?.closest?.("[data-radix-select-content]")) {
              e.preventDefault();
            }
          }}
        >
          <div className="max-h-[min(22.5rem,calc(var(--radix-popover-content-available-height,100dvh)-3.5rem))] overflow-y-auto overscroll-contain">
            <Calendar
              mode="range"
              captionLayout="dropdown"
              defaultMonth={draft?.from ?? rangeFromValue(value)?.from}
              selected={draft}
              onSelect={(range) => {
                // Draft only - parent query must not refetch until Apply/Clear.
                setDraft(range);
              }}
              numberOfMonths={visibleMonths}
              formatters={{
                formatMonthDropdown: (date) => {
                  const loc = locale.trim();
                  if (!loc) return date.toLocaleString("default", { month: "short" });
                  try {
                    return new Intl.DateTimeFormat(loc, { month: "short" }).format(date);
                  } catch {
                    return date.toLocaleString("default", { month: "short" });
                  }
                },
              }}
            />
          </div>
          <div className="flex shrink-0 items-center justify-end gap-2 border-t border-border bg-popover p-2">
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => {
                setDraft(undefined);
                onChange({ from: null, to: null });
                setOpen(false);
              }}
            >
              {clearLabel}
            </Button>
            <Button
              type="button"
              size="sm"
              onClick={() => {
                onChange(valueFromRange(draft));
                setOpen(false);
              }}
            >
              {applyLabel}
            </Button>
          </div>
        </PopoverContent>
      </Popover>
    </div>
  );
}
