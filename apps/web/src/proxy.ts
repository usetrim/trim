import { envPathAuthCallback, envPathLogin, envProtectedPrefixes } from "@/lib/app-paths";
import { createServerClient } from "@supabase/ssr";
import { type NextRequest, NextResponse } from "next/server";

function isProtected(pathname: string, prefixes: string[]): boolean {
  return prefixes.some((prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`));
}

/** Supabase SSR auth cookies - presence means a browser session likely exists. */
function hasSupabaseAuthCookie(request: NextRequest): boolean {
  return request.cookies.getAll().some((c) => {
    const n = c.name;
    return n.includes("sb-") && (n.includes("auth-token") || n.endsWith("-auth-token"));
  });
}

function isLoginPath(pathname: string, loginPath: string): boolean {
  if (loginPath && (pathname === loginPath || pathname.startsWith(`${loginPath}/`))) return true;
  // Fail-closed match before chrome/env login path is known.
  return pathname === "/login" || pathname.startsWith("/login/");
}

/**
 * Auth lookup (getUser) is only required for gated routes and login redirect-if-authed.
 * Calling it on every marketing soft-nav freezes navigation when Auth is slow.
 */
function needsAuthLookup(
  pathname: string,
  loginPath: string,
  protectedPrefixes: string[],
): boolean {
  if (isLoginPath(pathname, loginPath)) return true;
  if (protectedPrefixes.length > 0 && isProtected(pathname, protectedPrefixes)) return true;
  return false;
}

export async function proxy(request: NextRequest) {
  let response = NextResponse.next({
    request: { headers: request.headers },
  });

  const url = process.env.NEXT_PUBLIC_SUPABASE_URL;
  const key = process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY;
  if (!url || !key) {
    return response;
  }

  const { pathname } = request.nextUrl;
  const loginPath = envPathLogin();
  const callbackPath = envPathAuthCallback();

  // OAuth returns here with ?code= before a session exists. Never gate this path.
  // Hard-allow the canonical path even if NEXT_PUBLIC_APP_PATH_AUTH_CALLBACK is unset.
  if (
    pathname === "/auth/callback" ||
    pathname.startsWith("/auth/callback/") ||
    (callbackPath && (pathname === callbackPath || pathname.startsWith(`${callbackPath}/`)))
  ) {
    return response;
  }

  // Stray PKCE code on a non-callback URL: forward to callback (preserves query).
  if (callbackPath && request.nextUrl.searchParams.has("code")) {
    if (pathname !== callbackPath) {
      const dest = request.nextUrl.clone();
      dest.pathname = callbackPath;
      return NextResponse.redirect(dest);
    }
  }

  const protectedPrefixes = envProtectedPrefixes();

  // Public marketing soft-nav: skip getUser entirely (home/pricing/docs/contact/legal/…).
  if (!needsAuthLookup(pathname, loginPath, protectedPrefixes)) {
    return response;
  }

  const supabase = createServerClient(url, key, {
    cookies: {
      getAll() {
        return request.cookies.getAll();
      },
      setAll(
        cookiesToSet: {
          name: string;
          value: string;
          options?: Record<string, unknown>;
        }[],
      ) {
        for (const { name, value } of cookiesToSet) {
          request.cookies.set(name, value);
        }
        response = NextResponse.next({
          request: { headers: request.headers },
        });
        for (const { name, value, options } of cookiesToSet) {
          response.cookies.set(name, value, options);
        }
      },
    },
  });

  // Never block Edge navigation on a hung Supabase Auth call (post-OAuth freeze).
  let user: { id: string } | null = null;
  try {
    const result = await Promise.race([
      supabase.auth.getUser(),
      new Promise<{ data: { user: null }; error: Error }>((resolve) =>
        setTimeout(() => resolve({ data: { user: null }, error: new Error("timeout") }), 8000),
      ),
    ]);
    user = result.data.user;
  } catch {
    user = null;
  }

  if (protectedPrefixes.length > 0 && isProtected(pathname, protectedPrefixes) && !user) {
    // Soft-nav / RSC: do not hard-redirect to login on transient getUser failure
    // when auth cookies exist - that remounts the whole app on every navigation.
    // Client gates still enforce auth.
    if (hasSupabaseAuthCookie(request)) {
      return response;
    }
    if (!loginPath) {
      return response;
    }
    const login = request.nextUrl.clone();
    login.pathname = loginPath;
    login.searchParams.set("next", pathname);
    return NextResponse.redirect(login);
  }

  if (loginPath && pathname === loginPath && user) {
    // Keep users on login when an auth error is being shown (toast + social buttons).
    if (request.nextUrl.searchParams.has("error")) {
      return response;
    }
    const next = request.nextUrl.searchParams.get("next");
    if (
      next?.startsWith("/") &&
      !next.startsWith("//") &&
      next !== "/auth/callback" &&
      (!callbackPath || next !== callbackPath)
    ) {
      const dest = request.nextUrl.clone();
      dest.pathname = next;
      dest.search = "";
      return NextResponse.redirect(dest);
    }
  }

  return response;
}

// Broad matcher: path prefixes are resolved at runtime from NEXT_PUBLIC_APP_PATH_*.
export const config = {
  matcher: [
    "/((?!_next/static|_next/image|favicon.ico|site\\.webmanifest|.*\\.(?:svg|png|jpg|jpeg|gif|webp|ico|webmanifest)$).*)",
  ],
};
