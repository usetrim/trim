import type { AuthProvidersResponse } from "@/types/auth";
import { cache } from "react";

/** Local API origins are often down during `next build`; skip network (fail closed, no ECONNREFUSED spam). */
function isLocalApiBase(base: string): boolean {
  try {
    const host = new URL(base).hostname;
    return host === "localhost" || host === "127.0.0.1" || host === "::1";
  } catch {
    return false;
  }
}

/**
 * Server-side fetch of public auth-providers for route loading skeletons / site chrome.
 * Cached per-request so root layout + generateMetadata share one fetch (no double wait).
 * Short timeout: fail closed during SSG instead of hitting Next staticPageGenerationTimeout.
 */
export const loadPublicAuthProviders = cache(async (): Promise<AuthProvidersResponse | null> => {
  const base = process.env.NEXT_PUBLIC_API_URL?.replace(/\/$/, "");
  if (!base) return null;
  // Production Vercel builds use a real API host and still fetch; local builds do not.
  if (process.env.NEXT_PHASE === "phase-production-build" && isLocalApiBase(base)) {
    return null;
  }
  try {
    const res = await fetch(`${base}/api/v1/public/auth-providers`, {
      next: { revalidate: 300 },
      signal: AbortSignal.timeout(8_000),
    });
    if (!res.ok) return null;
    return (await res.json()) as AuthProvidersResponse;
  } catch {
    return null;
  }
});
