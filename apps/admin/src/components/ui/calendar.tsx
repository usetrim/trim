"use client";

import { buttonVariants } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";
import { ChevronDown, ChevronLeft, ChevronRight } from "lucide-react";
import type * as React from "react";
import { DayPicker, getDefaultClassNames, type DropdownProps } from "react-day-picker";

export type CalendarProps = React.ComponentProps<typeof DayPicker>;

/**
 * shadcn Calendar on react-day-picker v9.
 * Default captionLayout="dropdown" with Radix Select for month + year
 * (worldwide calendar UX - not native opacity-0 selects).
 */
function CalendarDropdown({
  options,
  value,
  onChange,
  "aria-label": ariaLabel,
  className,
  disabled,
}: DropdownProps) {
  const selected = String(value ?? "");
  return (
    <Select
      value={selected}
      disabled={disabled}
      onValueChange={(next) => {
        if (!onChange) return;
        onChange({
          target: { value: next },
        } as React.ChangeEvent<HTMLSelectElement>);
      }}
    >
      <SelectTrigger
        aria-label={ariaLabel}
        className={cn(
          "h-8 w-auto min-w-[4.75rem] gap-1 px-2 text-sm font-medium shadow-none",
          className,
        )}
      >
        <SelectValue />
      </SelectTrigger>
      <SelectContent
        position="popper"
        collisionPadding={12}
        className="z-[110] max-h-60 min-w-[var(--radix-select-trigger-width)]"
      >
        {(options ?? []).map((option) => (
          <SelectItem
            key={String(option.value)}
            value={String(option.value)}
            disabled={option.disabled}
          >
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

function Calendar({
  className,
  classNames,
  showOutsideDays = true,
  captionLayout = "dropdown",
  components,
  startMonth: startMonthProp,
  endMonth: endMonthProp,
  formatters,
  ...props
}: CalendarProps) {
  const defaultClassNames = getDefaultClassNames();
  const now = new Date();
  const startMonth = startMonthProp ?? new Date(now.getFullYear() - 20, 0);
  const endMonth = endMonthProp ?? new Date(now.getFullYear() + 5, 11);

  return (
    <DayPicker
      {...props}
      showOutsideDays={showOutsideDays}
      captionLayout={captionLayout}
      navLayout="around"
      startMonth={startMonth}
      endMonth={endMonth}
      className={cn("rdp-root bg-background p-3 text-popover-foreground", className)}
      formatters={{
        formatMonthDropdown: (date) => date.toLocaleString("default", { month: "short" }),
        ...formatters,
      }}
      classNames={{
        root: cn("w-fit", defaultClassNames.root),
        months: cn("relative flex flex-col gap-4 sm:flex-row sm:gap-6", defaultClassNames.months),
        month: cn("flex w-full flex-col gap-3", defaultClassNames.month),
        month_caption: cn(
          "relative flex h-9 w-full items-center justify-center px-8",
          defaultClassNames.month_caption,
        ),
        caption_label: cn(
          "select-none text-sm font-medium",
          captionLayout === "label"
            ? ""
            : "flex h-8 items-center gap-1 rounded-md pl-2 pr-1 [&>svg]:size-3.5 [&>svg]:text-muted-foreground",
          defaultClassNames.caption_label,
        ),
        nav: cn(
          "absolute inset-x-0 top-0 flex w-full items-center justify-between gap-1",
          defaultClassNames.nav,
        ),
        button_previous: cn(
          buttonVariants({ variant: "outline" }),
          "z-10 h-7 w-7 bg-transparent p-0 opacity-70 hover:opacity-100",
          defaultClassNames.button_previous,
        ),
        button_next: cn(
          buttonVariants({ variant: "outline" }),
          "z-10 h-7 w-7 bg-transparent p-0 opacity-70 hover:opacity-100",
          defaultClassNames.button_next,
        ),
        dropdowns: cn(
          "relative z-10 flex h-9 w-full items-center justify-center gap-2 text-sm font-medium",
          defaultClassNames.dropdowns,
        ),
        dropdown_root: cn("relative", defaultClassNames.dropdown_root),
        dropdown: cn("absolute inset-0 opacity-0", defaultClassNames.dropdown),
        month_grid: cn("w-full border-collapse", defaultClassNames.month_grid),
        weekdays: cn("flex", defaultClassNames.weekdays),
        weekday: cn(
          "w-8 select-none rounded-md text-[0.8rem] font-normal text-muted-foreground",
          defaultClassNames.weekday,
        ),
        week: cn("mt-2 flex w-full", defaultClassNames.week),
        day: cn(
          "relative p-0 text-center text-sm focus-within:relative focus-within:z-20 [&:has([aria-selected])]:bg-accent [&:has([aria-selected].day-outside)]:bg-accent/50 [&:has([aria-selected].day-range-end)]:rounded-r-md",
          props.mode === "range"
            ? "[&:has(>.day-range-end)]:rounded-r-md [&:has(>.day-range-start)]:rounded-l-md first:[&:has([aria-selected])]:rounded-l-md last:[&:has([aria-selected])]:rounded-r-md"
            : "[&:has([aria-selected])]:rounded-md",
          defaultClassNames.day,
        ),
        day_button: cn(
          buttonVariants({ variant: "ghost" }),
          "h-8 w-8 p-0 font-normal aria-selected:opacity-100",
          defaultClassNames.day_button,
        ),
        range_start: cn("day-range-start", defaultClassNames.range_start),
        range_end: cn("day-range-end", defaultClassNames.range_end),
        selected: cn(
          "bg-primary text-primary-foreground hover:bg-primary hover:text-primary-foreground focus:bg-primary focus:text-primary-foreground",
          defaultClassNames.selected,
        ),
        today: cn("bg-accent text-popover-foreground", defaultClassNames.today),
        outside: cn(
          "day-outside text-muted-foreground aria-selected:bg-accent/50 aria-selected:text-muted-foreground",
          defaultClassNames.outside,
        ),
        disabled: cn("text-muted-foreground opacity-50", defaultClassNames.disabled),
        range_middle: cn(
          "aria-selected:bg-accent aria-selected:text-popover-foreground",
          defaultClassNames.range_middle,
        ),
        hidden: cn("invisible", defaultClassNames.hidden),
        ...classNames,
      }}
      components={{
        Dropdown: CalendarDropdown,
        Chevron: ({ orientation, className: chevronClass, ...chevronProps }) => {
          if (orientation === "left") {
            return <ChevronLeft className={cn("h-4 w-4", chevronClass)} {...chevronProps} />;
          }
          if (orientation === "right") {
            return <ChevronRight className={cn("h-4 w-4", chevronClass)} {...chevronProps} />;
          }
          return <ChevronDown className={cn("h-4 w-4", chevronClass)} {...chevronProps} />;
        },
        ...components,
      }}
    />
  );
}
Calendar.displayName = "Calendar";

export { Calendar };
