/**
 * Appends URL query parameters to an API path.
 * Used only by hooks under hooks/queries and hooks/mutations (not by UI components).
 * Empty or undefined values are omitted (fail closed; no invent defaults).
 */
export function withQuery(
  path: string,
  params: Record<string, string | number | undefined>,
): string {
  const search = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined || v === "") continue;
    search.set(k, String(v));
  }
  const s = search.toString();
  return s ? `${path}?${s}` : path;
}
