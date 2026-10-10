/**
 * Low-level web API fetch. Only hooks under hooks/queries and hooks/mutations
 * should call this. UI components must use those hooks.
 */

/** Populated from GET /api/v1/public/auth-providers; fail-closed until synced. */
let authNotSignedInMessage = "";

export function rememberAuthNotSignedIn(message: string | undefined) {
  if (typeof message === "string" && message.trim()) {
    authNotSignedInMessage = message.trim();
  }
}

export function requireAccessToken(token: string | undefined): string {
  if (!token) {
    throw new Error(authNotSignedInMessage);
  }
  return token;
}

function apiBase(): string {
  const base = process.env.NEXT_PUBLIC_API_URL;
  if (!base) {
    const missing = process.env.NEXT_PUBLIC_API_URL_MISSING?.trim();
    throw new Error(missing || "");
  }
  return base.replace(/\/$/, "");
}

export function getApiBase(): string {
  return apiBase();
}

/**
 * Parses a JSON body, failing closed on anything that is not JSON so a proxy
 * HTML page or an empty body can never reach the UI as user-visible copy.
 */
export async function readJSONOrFail(res: Response): Promise<unknown> {
  try {
    return await res.json();
  } catch {
    throw new Error("");
  }
}

export type ApiFetchInit = RequestInit & { token?: string };

/** Remote DB + chart/agg routes need headroom beyond warm Redis cache hits. */
const API_FETCH_MS = 60_000;

export async function apiFetch<T>(path: string, init?: ApiFetchInit): Promise<T> {
  const headers = new Headers(init?.headers);
  headers.set("Accept", "application/json");
  if (init?.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  if (init?.token) {
    headers.set("Authorization", `Bearer ${init.token}`);
  }
  const { token: _token, signal: callerSignal, ...rest } = init ?? {};
  void _token;
  const signal = callerSignal ?? AbortSignal.timeout(API_FETCH_MS);
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
      const msg =
        (typeof body.error === "string" && body.error.trim()) ||
        (typeof body.message === "string" && body.message.trim()) ||
        "";
      const err = new Error(msg) as Error & {
        body?: Record<string, unknown>;
        status?: number;
      };
      err.body = body;
      err.status = res.status;
      throw err;
    } catch (e) {
      if (e instanceof Error && (e as Error & { body?: unknown }).body) {
        throw e;
      }
      const err = new Error("") as Error & { status?: number };
      err.status = res.status;
      throw err;
    }
  }
  return (await readJSONOrFail(res)) as T;
}
