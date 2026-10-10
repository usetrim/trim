export type AuthProviderId = "google" | "github" | "gitlab";

const ALL: AuthProviderId[] = ["google", "github", "gitlab"];

function parseProviderList(raw: string[]): AuthProviderId[] {
  return raw
    .map((p) => p.trim().toLowerCase())
    .filter(Boolean)
    .filter((p): p is AuthProviderId => ALL.includes(p as AuthProviderId));
}

/**
 * Env fallback only for build-time / SSR when the API is unreachable.
 * Live UI and OAuth callback prefer fetchAllowedAuthProviders (DB via API).
 */
export function allowedAuthProvidersFromEnv(): AuthProviderId[] {
  const raw = process.env.NEXT_PUBLIC_ALLOWED_AUTH_PROVIDERS?.trim();
  if (!raw) return [];
  return parseProviderList(raw.split(","));
}

/**
 * Skeleton button count. Prefers live API count; otherwise the operator
 * allow-list in NEXT_PUBLIC_ALLOWED_AUTH_PROVIDERS. Never invents 3.
 */
export function authProviderSlotCount(apiCount?: number): number {
  if (typeof apiCount === "number" && apiCount > 0) {
    return apiCount;
  }
  return allowedAuthProvidersFromEnv().length;
}

/** @deprecated Prefer fetchAllowedAuthProviders; kept for SSR phrase fallback. */
export function allowedAuthProviders(): AuthProviderId[] {
  return allowedAuthProvidersFromEnv();
}

export type AllowedAuthProvidersResult = {
  /** True only when the public auth-providers API returned a successful body. */
  ok: boolean;
  providers: AuthProviderId[];
};

/**
 * Source of truth: GET /api/v1/public/auth-providers (auth_settings in Postgres).
 * Fail closed on missing API URL or non-OK response (ok: false).
 * Callers must not treat ok:false as an empty allow-list for destructive actions.
 */
export async function fetchAllowedAuthProviders(
  apiBase?: string,
): Promise<AllowedAuthProvidersResult> {
  const base = (apiBase || process.env.NEXT_PUBLIC_API_URL || "").replace(/\/$/, "");
  if (!base) return { ok: false, providers: [] };
  try {
    const res = await fetch(`${base}/api/v1/public/auth-providers`, {
      headers: { Accept: "application/json" },
      cache: "no-store",
      signal: AbortSignal.timeout(30_000),
    });
    if (!res.ok) return { ok: false, providers: [] };
    const body = (await res.json()) as { providers?: string[] };
    return {
      ok: true,
      providers: parseProviderList(body.providers ?? []),
    };
  } catch {
    return { ok: false, providers: [] };
  }
}

export function isAllowedAuthProvider(
  provider: string | undefined | null,
  allowList?: AuthProviderId[],
): boolean {
  if (!provider) return false;
  const list = allowList ?? allowedAuthProvidersFromEnv();
  return list.includes(provider.toLowerCase() as AuthProviderId);
}

/**
 * Join provider display names from the API.
 * phraseOr / phraseComma must come from auth-providers (no client invent).
 */
export function formatProviderPhrase(
  labels: string[],
  phraseOr?: string,
  phraseComma?: string,
): string {
  const clean = labels.map((l) => l.trim()).filter(Boolean);
  if (clean.length === 0) return "";
  if (clean.length === 1) return clean[0];
  const or = phraseOr ?? "";
  const comma = phraseComma ?? "";
  if (clean.length === 2) {
    if (!or) return "";
    return `${clean[0]}${or}${clean[1]}`;
  }
  if (!comma) return "";
  return `${clean.slice(0, -1).join(", ")}${comma}${clean[clean.length - 1]}`;
}
