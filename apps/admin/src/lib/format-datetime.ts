/**
 * UTC-in-DB → locale + timezone display (professional SaaS).
 * Locale must come from site chrome (html_lang / money_locale / SITE_HTML_LANG).
 * Empty locale → empty string (fail-closed; never invent browser language).
 * Timezone uses the runtime environment (browser IANA zone).
 */

export function parseUtcTimestamp(raw: string | null | undefined): Date | null {
  const s = (raw ?? "").trim();
  if (!s) return null;

  let candidate = s;
  // Postgres / Go: "2026-10-02 13:11:32.645223+00"
  if (/^\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}/.test(candidate)) {
    candidate = candidate.replace(" ", "T");
  }
  // +00 / -05 → +00:00 / -05:00
  candidate = candidate.replace(/([+-]\d{2})$/, "$1:00");
  // +0000 → +00:00
  candidate = candidate.replace(/([+-]\d{2})(\d{2})$/, "$1:$2");

  const parsed = new Date(candidate);
  if (!Number.isNaN(parsed.getTime())) return parsed;

  // Calendar day only (billing period / receipt day labels as ISO dates).
  if (/^\d{4}-\d{2}-\d{2}$/.test(s)) {
    const day = new Date(`${s}T00:00:00.000Z`);
    if (!Number.isNaN(day.getTime())) return day;
  }
  return null;
}

function resolveLocale(locale: string): string {
  return (locale || "").trim();
}

function resolveTimeZone(): string | undefined {
  if (typeof Intl === "undefined") return undefined;
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || undefined;
  } catch {
    return undefined;
  }
}

function formatWith(date: Date, locale: string, options: Intl.DateTimeFormatOptions): string {
  const loc = resolveLocale(locale);
  if (!loc) return "";
  const timeZone = resolveTimeZone();
  try {
    return new Intl.DateTimeFormat(loc, timeZone ? { ...options, timeZone } : options).format(date);
  } catch {
    return "";
  }
}

/** Lists / tables: Oct 2, 2026, 4:11 PM */
export function formatDateTimeShort(raw: string | null | undefined, locale: string): string {
  const date = parseUtcTimestamp(raw);
  if (!date) return "";
  return formatWith(date, locale, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

/** Date-only: Oct 2, 2026 */
export function formatDateShort(raw: string | null | undefined, locale: string): string {
  const date = parseUtcTimestamp(raw);
  if (!date) return "";
  return formatWith(date, locale, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

/** Heatmap / long calendar day: Monday, October 2, 2026 */
export function formatDateLong(raw: string | null | undefined, locale: string): string {
  const date = parseUtcTimestamp(raw);
  if (!date) return "";
  return formatWith(date, locale, {
    weekday: "long",
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}

/** Narrow month for dense heatmap headers (locale-aware). */
export function formatMonthNarrow(raw: string | null | undefined, locale: string): string {
  const s = (raw ?? "").trim();
  const normalized = /^\d{4}-\d{2}$/.test(s) ? `${s}-01` : s;
  const date = parseUtcTimestamp(normalized);
  if (!date) return "";
  return formatWith(date, locale, { month: "narrow" });
}

/** Month name from YYYY-MM or YYYY-MM-DD: October 2026 */
export function formatMonthLong(raw: string | null | undefined, locale: string): string {
  const s = (raw ?? "").trim();
  const normalized = /^\d{4}-\d{2}$/.test(s) ? `${s}-01` : s;
  const date = parseUtcTimestamp(normalized);
  if (!date) return "";
  return formatWith(date, locale, {
    year: "numeric",
    month: "long",
  });
}

/** Receipts / legal / detail: includes timezone or offset when available */
export function formatDateTimeFull(raw: string | null | undefined, locale: string): string {
  const date = parseUtcTimestamp(raw);
  if (!date) return "";
  return formatWith(date, locale, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
    second: "2-digit",
    timeZoneName: "short",
  });
}

const RELATIVE_MAX_MS = 7 * 24 * 60 * 60 * 1000;

/**
 * Notifications: relative when recent; otherwise short absolute.
 * `title` is always full absolute (hover / accessibility).
 */
export function formatDateTimeRelative(
  raw: string | null | undefined,
  locale: string,
  now: Date = new Date(),
): { label: string; title: string } {
  const date = parseUtcTimestamp(raw);
  const title = formatDateTimeFull(raw, locale);
  if (!date) return { label: "", title: "" };

  const loc = resolveLocale(locale);
  if (!loc) return { label: "", title: "" };

  const diffMs = date.getTime() - now.getTime();
  const absMs = Math.abs(diffMs);
  if (absMs > RELATIVE_MAX_MS) {
    return { label: formatDateTimeShort(raw, locale), title };
  }

  const absSec = Math.round(absMs / 1000);
  let value: number;
  let unit: Intl.RelativeTimeFormatUnit;
  if (absSec < 60) {
    value = Math.round(diffMs / 1000);
    unit = "second";
  } else if (absSec < 3600) {
    value = Math.round(diffMs / 60_000);
    unit = "minute";
  } else if (absSec < 86_400) {
    value = Math.round(diffMs / 3_600_000);
    unit = "hour";
  } else {
    value = Math.round(diffMs / 86_400_000);
    unit = "day";
  }

  try {
    const label = new Intl.RelativeTimeFormat(loc, { numeric: "auto" }).format(value, unit);
    return { label, title };
  } catch {
    return { label: formatDateTimeShort(raw, locale), title };
  }
}
