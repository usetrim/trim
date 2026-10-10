import { envPathLogin } from "@/lib/app-paths";
import { fetchAllowedAuthProviders, isAllowedAuthProvider } from "@/lib/auth-providers";
import { type CookieOptions, createServerClient } from "@supabase/ssr";
import { cookies } from "next/headers";
import { NextResponse } from "next/server";

function isSafeAppPath(path: string): boolean {
  return path.startsWith("/") && !path.startsWith("//");
}

type AuthProvidersChrome = {
  login_default_next?: string;
  oauth_link_intent?: string;
  site?: {
    path_login?: string;
    path_settings?: string;
    path_auth_callback?: string;
    nav_sign_in_href?: string;
  };
};

/** Remote API + multi-query auth chrome; keep below hung-login UX pain. */
const API_FETCH_MS = 30_000;
// Keep tight: hung Auth exchange must not look like a frozen Continue redirect.
const SUPABASE_AUTH_MS = 8_000;

async function fetchWithTimeout(input: string, init?: RequestInit): Promise<Response> {
  return fetch(input, { ...init, signal: AbortSignal.timeout(API_FETCH_MS) });
}

async function withTimeout<T>(promise: Promise<T>, ms: number): Promise<T> {
  let timer: ReturnType<typeof setTimeout> | undefined;
  try {
    return await Promise.race([
      promise,
      new Promise<never>((_, reject) => {
        timer = setTimeout(() => reject(new Error("timeout")), ms);
      }),
    ]);
  } finally {
    if (timer) clearTimeout(timer);
  }
}

async function signOutBestEffort(supabase: {
  auth: { signOut: () => Promise<unknown> };
}): Promise<void> {
  try {
    await withTimeout(supabase.auth.signOut(), 2500);
  } catch {
    // Fail closed: redirect proceeds even if signOut hangs.
  }
}

/** Match DB handle_new_user: denylist only blocks brand-new signups, not returning users. */
function isBrandNewOAuthUser(createdAtRaw: string | undefined): boolean {
  const createdAt = Date.parse(createdAtRaw || "");
  if (!Number.isFinite(createdAt)) return true;
  return Date.now() - createdAt < 120_000;
}

async function fetchAuthChrome(): Promise<AuthProvidersChrome | null> {
  const apiBase = process.env.NEXT_PUBLIC_API_URL?.replace(/\/$/, "");
  if (!apiBase) return null;
  try {
    const res = await fetchWithTimeout(`${apiBase}/api/v1/public/auth-providers`, {
      headers: { Accept: "application/json" },
      cache: "no-store",
    });
    if (!res.ok) return null;
    return (await res.json()) as AuthProvidersChrome;
  } catch {
    return null;
  }
}

function resolveLoginPath(authChrome: AuthProvidersChrome | null): string {
  const fromApi = (authChrome?.site?.path_login || authChrome?.site?.nav_sign_in_href || "").trim();
  if (fromApi && isSafeAppPath(fromApi)) return fromApi;
  const fromEnv = envPathLogin();
  if (fromEnv && isSafeAppPath(fromEnv)) return fromEnv;
  return "";
}

function resolveSettingsPath(authChrome: AuthProvidersChrome | null): string {
  const fromApi = (authChrome?.site?.path_settings || "").trim();
  if (fromApi && isSafeAppPath(fromApi)) return fromApi;
  return "";
}

type PendingCookie = { name: string; value: string; options: CookieOptions };

/**
 * OAuth PKCE callback. Session cookies must be attached to the redirect Response
 * (cookieStore.set alone is not reliable on NextResponse.redirect in App Router).
 *
 * Best practice: start the code exchange immediately; load chrome in parallel
 * (do not block PKCE on auth-providers latency).
 */
export async function GET(request: Request) {
  const url = new URL(request.url);
  const code = url.searchParams.get("code");
  const nextParam = (url.searchParams.get("next") || "").trim();
  const oauthError = (url.searchParams.get("error") || "").trim();

  const pendingCookies: PendingCookie[] = [];
  const withCookies = (res: NextResponse) => {
    for (const { name, value, options } of pendingCookies) {
      res.cookies.set(name, value, options);
    }
    return res;
  };

  let loginPath = envPathLogin();
  let settingsPath = "";
  // Named authChrome - avoid clashing with the DOM `chrome` global typings.
  let authChrome: AuthProvidersChrome | null = null;
  const authChromePromise = fetchAuthChrome();

  const redirectLogin = (errorCode: string) => {
    if (!loginPath) {
      return NextResponse.json({ error: "login_path_missing", detail: errorCode }, { status: 500 });
    }
    return withCookies(
      NextResponse.redirect(new URL(`${loginPath}?error=${errorCode}`, url.origin)),
    );
  };

  const redirectSettings = (errorCode: string) => {
    if (!settingsPath) {
      return redirectLogin(errorCode);
    }
    return withCookies(
      NextResponse.redirect(new URL(`${settingsPath}?error=${errorCode}`, url.origin)),
    );
  };

  const applyAuthChrome = (c: AuthProvidersChrome | null): AuthProvidersChrome | null => {
    authChrome = c;
    loginPath = resolveLoginPath(c) || loginPath;
    settingsPath = resolveSettingsPath(c);
    return c;
  };

  const intentParam = (url.searchParams.get("intent") || "").trim();

  const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL;
  const supabaseAnon = process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY;
  if (!supabaseUrl || !supabaseAnon) {
    const c = applyAuthChrome(await authChromePromise);
    const linkIntent = (c?.oauth_link_intent || "").trim();
    const isLink = Boolean(linkIntent) && intentParam === linkIntent;
    return isLink
      ? redirectSettings("missing_supabase_env")
      : redirectLogin("missing_supabase_env");
  }

  if (oauthError && !code) {
    const c = applyAuthChrome(await authChromePromise);
    const linkIntent = (c?.oauth_link_intent || "").trim();
    const isLink = Boolean(linkIntent) && intentParam === linkIntent;
    return isLink ? redirectSettings("oauth_link_failed") : redirectLogin("oauth_exchange_failed");
  }

  if (code) {
    const cookieStore = await cookies();
    const supabase = createServerClient(supabaseUrl, supabaseAnon, {
      cookies: {
        getAll() {
          return cookieStore.getAll();
        },
        setAll(
          cookiesToSet: Array<{
            name: string;
            value: string;
            options: CookieOptions;
          }>,
        ) {
          for (const { name, value, options } of cookiesToSet) {
            pendingCookies.push({ name, value, options });
            try {
              cookieStore.set(name, value, options);
            } catch {
              // Route Handler may still attach via withCookies on the redirect.
            }
          }
        },
      },
    });
    let exchanged: Awaited<ReturnType<typeof supabase.auth.exchangeCodeForSession>>;
    try {
      exchanged = await withTimeout(supabase.auth.exchangeCodeForSession(code), SUPABASE_AUTH_MS);
    } catch {
      const c = applyAuthChrome(await authChromePromise);
      const linkIntent = (c?.oauth_link_intent || "").trim();
      const isLink = Boolean(linkIntent) && intentParam === linkIntent;
      return isLink
        ? redirectSettings("oauth_link_failed")
        : redirectLogin("oauth_exchange_failed");
    }
    const { data, error } = exchanged;
    const c = applyAuthChrome(await authChromePromise);
    const linkIntent = (c?.oauth_link_intent || "").trim();
    const isLink = Boolean(linkIntent) && intentParam === linkIntent;
    const failAuth = (errorCode: string) =>
      isLink ? redirectSettings(errorCode) : redirectLogin(errorCode);

    if (error) {
      return failAuth(isLink ? "oauth_link_failed" : "oauth_exchange_failed");
    }

    const email = (data.session?.user?.email || "").trim();
    if (!email) {
      if (!isLink) {
        await signOutBestEffort(supabase);
      }
      return failAuth("oauth_email_missing");
    }

    // Overlap allow-list with email-policy so callback stays fast after IdP Continue.
    const allowPromise = fetchAllowedAuthProviders(process.env.NEXT_PUBLIC_API_URL);

    // Email denylist applies to new signups only (same rule as handle_new_user).
    // Returning users must not be blocked or deleted by a policy/network glitch.
    if (isBrandNewOAuthUser(data.session?.user?.created_at)) {
      const apiBase = process.env.NEXT_PUBLIC_API_URL?.replace(/\/$/, "");
      if (!apiBase) {
        if (!isLink) {
          await signOutBestEffort(supabase);
        }
        return failAuth("auth_providers_unavailable");
      }
      try {
        const policyRes = await fetchWithTimeout(
          `${apiBase}/api/v1/public/email-policy?email=${encodeURIComponent(email)}`,
          { headers: { Accept: "application/json" }, cache: "no-store" },
        );
        if (!policyRes.ok) {
          // API/DB unavailable - do not mislabel as disposable-email denial.
          if (!isLink) {
            await signOutBestEffort(supabase);
          }
          return failAuth("auth_providers_unavailable");
        }
        const policy = (await policyRes.json()) as { allowed?: boolean };
        if (policy.allowed !== true) {
          if (!isLink) {
            const accessToken = data.session?.access_token;
            if (accessToken) {
              try {
                await fetchWithTimeout(`${apiBase}/api/v1/me/account`, {
                  method: "DELETE",
                  headers: {
                    Authorization: `Bearer ${accessToken}`,
                    Accept: "application/json",
                  },
                  cache: "no-store",
                });
              } catch {
                // Best-effort purge when profile trigger may have failed.
              }
            }
            await signOutBestEffort(supabase);
          }
          return failAuth("oauth_email_denied");
        }
      } catch {
        if (!isLink) {
          await signOutBestEffort(supabase);
        }
        return failAuth("auth_providers_unavailable");
      }
    }

    const identities = data.session?.user?.identities ?? [];
    const provider =
      data.session?.user?.app_metadata?.provider ??
      data.session?.user?.app_metadata?.providers?.[0] ??
      identities[0]?.provider;

    const allow = await allowPromise;
    if (!allow.ok) {
      if (!isLink) {
        await signOutBestEffort(supabase);
      }
      return failAuth("auth_providers_unavailable");
    }
    const identityProviders = identities
      .map((identity) => String(identity.provider || "").toLowerCase())
      .filter(Boolean);
    const disallowedIdentity = identityProviders.find(
      (id) => !isAllowedAuthProvider(id, allow.providers),
    );
    if (
      !provider ||
      !isAllowedAuthProvider(String(provider), allow.providers) ||
      Boolean(disallowedIdentity)
    ) {
      if (isLink) {
        return redirectSettings("provider_not_allowed");
      }
      const accessToken = data.session?.access_token;
      const apiBase = process.env.NEXT_PUBLIC_API_URL?.replace(/\/$/, "");
      if (accessToken && apiBase) {
        try {
          await fetchWithTimeout(`${apiBase}/api/v1/me/account`, {
            method: "DELETE",
            headers: {
              Authorization: `Bearer ${accessToken}`,
              Accept: "application/json",
            },
            cache: "no-store",
          });
        } catch {
          // Best-effort purge; still sign out below.
        }
      }
      await signOutBestEffort(supabase);
      return redirectLogin(provider ? "provider_not_allowed" : "provider_missing");
    }
  } else {
    authChrome = applyAuthChrome(await authChromePromise);
  }

  let next = nextParam;
  if (!next || !isSafeAppPath(next)) {
    next = (authChrome?.login_default_next || "").trim();
  }
  const callbackPath = (authChrome?.site?.path_auth_callback || "/auth/callback").trim();
  if (callbackPath && next === callbackPath) {
    next = (authChrome?.login_default_next || "").trim();
  }
  if (!next || !isSafeAppPath(next) || next === callbackPath) {
    return redirectLogin("login_default_next_missing");
  }

  return withCookies(NextResponse.redirect(new URL(next, url.origin)));
}
