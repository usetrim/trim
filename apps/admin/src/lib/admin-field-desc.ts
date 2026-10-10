/** Per-field admin chrome description; omit prop when unset (no generic fallback). */
export function chromeFieldDesc(
  chrome: Record<string, string | undefined> | undefined,
  code: string,
): { description?: string } {
  const description = chrome?.[code]?.trim() ?? "";
  return description ? { description } : {};
}
