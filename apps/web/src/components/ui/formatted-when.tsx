"use client";

import {
  formatDateShort,
  formatDateTimeFull,
  formatDateTimeRelative,
  formatDateTimeShort,
} from "@/lib/format-datetime";
import { useEffect, useMemo, useState } from "react";

type Variant = "short" | "full" | "date" | "relative";

/**
 * Locale + timezone–aware timestamp display.
 * Prefer over raw ISO / DB strings in product UI.
 */
export function FormattedWhen({
  value,
  locale,
  variant = "short",
  className,
  empty = "",
}: {
  value: string | null | undefined;
  /** SITE_HTML_LANG / money_locale / html_lang - empty skips formatting. */
  locale: string;
  variant?: Variant;
  className?: string;
  empty?: string;
}) {
  const absoluteTitle = useMemo(() => formatDateTimeFull(value, locale), [value, locale]);
  const absoluteShort = useMemo(() => formatDateTimeShort(value, locale), [value, locale]);
  const absoluteDate = useMemo(() => formatDateShort(value, locale), [value, locale]);
  const absoluteFull = absoluteTitle;

  // Relative only after mount to avoid SSR/client clock mismatch.
  const [relativeLabel, setRelativeLabel] = useState(absoluteShort);
  useEffect(() => {
    if (variant !== "relative") {
      setRelativeLabel(absoluteShort);
      return;
    }
    setRelativeLabel(formatDateTimeRelative(value, locale).label);
  }, [value, locale, variant, absoluteShort]);

  let label = "";
  if (variant === "full") label = absoluteFull;
  else if (variant === "date") label = absoluteDate;
  else if (variant === "relative") label = relativeLabel;
  else label = absoluteShort;

  if (!label) {
    return empty ? <span className={className}>{empty}</span> : null;
  }

  const title =
    variant === "relative" || variant === "short" || variant === "date"
      ? absoluteTitle || undefined
      : undefined;

  return (
    <span className={className} title={title}>
      {label}
    </span>
  );
}
