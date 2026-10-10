/**
 * Semantic success green for Trim marketing + product UI.
 *
 * Industry reference (Linear): #27a644 “ready / saved”.
 * Soft wash uses --trim-status-bg so light and dark both read clearly.
 * Use only for status, savings %, and completed outcomes.
 */
export const STATUS_GREEN = "#27a644";
/** Prefer CSS var so light/dark soft wash stays readable. */
export const STATUS_GREEN_BG = "var(--trim-status-bg)";

export const statusTextClass = "text-[var(--trim-status,#27a644)]";
export const statusBorderClass = "border-[var(--trim-status,#27a644)]/80";
export const statusSoftBgClass = "bg-[var(--trim-status-bg)]";
