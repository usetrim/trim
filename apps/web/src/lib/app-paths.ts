/**
 * Edge-safe app path helpers. Next.js middleware cannot call the API on every
 * request, so protected route prefixes come from NEXT_PUBLIC_APP_PATH_* env
 * that must stay aligned with site_messages APP_PATH_* seeds.
 */

function trimPath(value: string | undefined): string {
  return (value || "").trim();
}

/** Login path from env (aligned with APP_PATH_LOGIN). Empty = unset. */
export function envPathLogin(): string {
  return trimPath(process.env.NEXT_PUBLIC_APP_PATH_LOGIN);
}

/** Home path from env (aligned with APP_PATH_HOME). Empty = unset. */
export function envPathHome(): string {
  return trimPath(process.env.NEXT_PUBLIC_APP_PATH_HOME);
}

/** OAuth PKCE return path. Must never be behind the auth gate. */
export function envPathAuthCallback(): string {
  return trimPath(process.env.NEXT_PUBLIC_APP_PATH_AUTH_CALLBACK);
}

/**
 * Protected path prefixes for middleware auth gate.
 * Built only from configured NEXT_PUBLIC_APP_PATH_* values (no invent defaults).
 */
export function envProtectedPrefixes(): string[] {
  const keys = [
    process.env.NEXT_PUBLIC_APP_PATH_DASHBOARD,
    process.env.NEXT_PUBLIC_APP_PATH_TEAM,
    process.env.NEXT_PUBLIC_APP_PATH_SETTINGS,
    process.env.NEXT_PUBLIC_APP_PATH_RECEIPTS_PREFIX,
  ];
  const out: string[] = [];
  const seen = new Set<string>();
  for (const raw of keys) {
    const p = trimPath(raw).replace(/\/$/, "");
    if (!p || !p.startsWith("/") || p.startsWith("//")) continue;
    if (seen.has(p)) continue;
    seen.add(p);
    out.push(p);
  }
  return out;
}
