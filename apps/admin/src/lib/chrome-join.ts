/**
 * Chrome-driven multi-part join. Never invents separators.
 * Single non-empty part: returns it.
 * Two+ parts: requires sep from site_messages; empty sep fails closed to "".
 */
export function joinChromeParts(parts: unknown[], sep: string): string {
  const clean = parts
    .map((p) => (typeof p === "string" ? p.trim() : p != null ? String(p).trim() : ""))
    .filter(Boolean);
  if (clean.length === 0) return "";
  if (clean.length === 1) return clean[0];
  const s = sep.trim();
  if (!s) return "";
  return clean.join(s);
}

/**
 * Labelled field from ADMIN_META_FIELD_FMT (`{label}: {value}`).
 * Fail closed when fmt missing placeholders or inputs empty.
 */
export function formatChromeField(fmt: string, label: string, value: string): string {
  const f = fmt.trim();
  const l = label.trim();
  const v = value.trim();
  if (!f || !l || !v) return "";
  if (!f.includes("{label}") || !f.includes("{value}")) return "";
  return f.replaceAll("{label}", l).replaceAll("{value}", v);
}
