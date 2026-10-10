import { useStepUpStore } from "@/stores/step-up";
import type { ApiError } from "@/types/admin";

function apiBase(): string {
  const base = process.env.NEXT_PUBLIC_API_URL;
  if (!base) {
    const missing = process.env.NEXT_PUBLIC_API_URL_MISSING?.trim();
    throw new Error(missing || "");
  }
  return base.replace(/\/$/, "");
}

async function readJSONOrFail(res: Response): Promise<unknown> {
  try {
    return await res.json();
  } catch {
    throw new Error("");
  }
}

export type AdminFetchInit = RequestInit & {
  token?: string;
  stepUp?: string;
  stepUpToken?: string;
};

/** Remote DB + multi-query admin routes (dashboard) need headroom beyond chart cache hits. */
const ADMIN_FETCH_MS = 60_000;

/**
 * Low-level admin fetch. Only hooks under hooks/queries and hooks/mutations
 * should call this. UI components must use those hooks.
 */
export async function adminFetch<T>(path: string, init?: AdminFetchInit): Promise<T> {
  const headers = new Headers(init?.headers);
  headers.set("Accept", "application/json");
  if (init?.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  if (init?.token) {
    headers.set("Authorization", `Bearer ${init.token}`);
  }
  const step =
    (init?.stepUpToken || init?.stepUp || "").trim() || useStepUpStore.getState().activeToken();
  if (step) {
    headers.set("X-Trim-Step-Up", step);
  }
  const {
    token: _token,
    stepUp: _stepUp,
    stepUpToken: _stepUpToken,
    signal: callerSignal,
    ...rest
  } = init ?? {};
  void _token;
  void _stepUp;
  void _stepUpToken;
  const signal = callerSignal ?? AbortSignal.timeout(ADMIN_FETCH_MS);
  let res: Response;
  try {
    res = await fetch(`${apiBase()}${path}`, { ...rest, headers, signal });
  } catch {
    throw new Error("");
  }
  if (!res.ok) {
    const text = await res.text();
    try {
      const body = JSON.parse(text) as Record<string, unknown>;
      const code = typeof body.code === "string" ? body.code.trim() : "";
      const rawMsg =
        (typeof body.error === "string" && body.error.trim()) ||
        (typeof body.message === "string" && body.message.trim()) ||
        "";
      const detail = typeof body.detail === "string" ? body.detail.trim() : "";
      // Enroll-only: never surface per-action step-up / obsolete confirm chrome.
      const obsoleteStepUp =
        code === "ADMIN_STEP_UP_REQUIRED" ||
        code === "ADMIN_STEP_UP_FACTOR_REQUIRED" ||
        /confirm this action with your authenticator/i.test(rawMsg);
      if (obsoleteStepUp) {
        const err = new Error("") as ApiError;
        err.body = { ...body, error: "", code: code || "ADMIN_STEP_UP_REQUIRED" };
        err.status = res.status;
        throw err;
      }
      // Prefer operator detail (e.g. real Paddle API reason) when present.
      const display =
        detail && rawMsg && !rawMsg.includes(detail) ? `${rawMsg} (${detail})` : detail || rawMsg;
      const err = new Error(display) as ApiError;
      err.body = body;
      err.status = res.status;
      throw err;
    } catch (e) {
      if (e instanceof Error && (e as ApiError).body) {
        throw e;
      }
      const err = new Error("") as ApiError;
      err.status = res.status;
      throw err;
    }
  }
  if (res.status === 204) {
    return undefined as T;
  }
  return (await readJSONOrFail(res)) as T;
}
