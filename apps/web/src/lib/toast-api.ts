import { toast } from "sonner";

/** Fail-closed: only toast non-empty API / chrome text. */
export function toastApiError(err: unknown): void {
  const msg = err instanceof Error ? err.message.trim() : "";
  if (msg) toast.error(msg);
}

/** Prefer `message`, then prefs-style `saved_message`. Empty = no toast. */
export function toastApiSuccess(data: unknown): void {
  if (!data || typeof data !== "object") return;
  const o = data as Record<string, unknown>;
  const raw =
    (typeof o.message === "string" && o.message) ||
    (typeof o.saved_message === "string" && o.saved_message) ||
    "";
  const msg = raw.trim();
  if (msg) toast.success(msg);
}
