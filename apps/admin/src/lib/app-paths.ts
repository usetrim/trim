function trimPath(value: string | undefined): string {
  return (value || "").trim();
}

export function envPathLogin(): string {
  return trimPath(process.env.NEXT_PUBLIC_APP_PATH_LOGIN);
}

export function envPathHome(): string {
  return trimPath(process.env.NEXT_PUBLIC_APP_PATH_HOME);
}

/** OAuth PKCE return path. Must never be behind the auth gate. */
export function envPathAuthCallback(): string {
  const fromEnv = trimPath(process.env.NEXT_PUBLIC_APP_PATH_AUTH_CALLBACK);
  return fromEnv;
}

/** Comma-separated protected prefixes from env (no invent list in code). */
export function envProtectedPrefixes(): string[] {
  const raw = trimPath(process.env.NEXT_PUBLIC_ADMIN_PROTECTED_PREFIXES);
  if (!raw) return [];
  const out: string[] = [];
  const seen = new Set<string>();
  for (const part of raw.split(",")) {
    const p = trimPath(part).replace(/\/$/, "");
    if (!p || !p.startsWith("/") || p.startsWith("//")) continue;
    if (seen.has(p)) continue;
    seen.add(p);
    out.push(p);
  }
  return out;
}
