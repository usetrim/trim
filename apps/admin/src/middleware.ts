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

export async function middleware(request: NextRequest) {
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

  // OAuth returns here with ?code= before a session exists. Never gate this path
  // (protected prefix "/auth" must not catch "/auth/callback"). Hard-allow the
  // canonical path even if NEXT_PUBLIC_APP_PATH_AUTH_CALLBACK is unset.
  if (
    pathname === "/auth/callback" ||
    pathname.startsWith("/auth/callback/") ||
    (callbackPath && (pathname === callbackPath || pathname.startsWith(`${callbackPath}/`)))
  ) {
    return response;
  }

  // Stray PKCE code on a non-callback URL (e.g. /login?code=...): forward to callback.
  if (callbackPath && pathname !== loginPath && request.nextUrl.searchParams.has("code")) {
    const dest = request.nextUrl.clone();
    dest.pathname = callbackPath;
    return NextResponse.redirect(dest);
  }
  if (callbackPath && pathname === loginPath && request.nextUrl.searchParams.has("code")) {
    const dest = request.nextUrl.clone();
    dest.pathname = callbackPath;
    // Keep code + next; drop unrelated noise later in the route handler.
    return NextResponse.redirect(dest);
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

  const protectedPrefixes = envProtectedPrefixes();

  if (protectedPrefixes.length > 0 && isProtected(pathname, protectedPrefixes) && !user) {
    // Soft-nav / RSC: a transient getUser timeout must NOT hard-redirect to login
    // when auth cookies are present - that remounts the whole app (looks like a
    // full page refresh on every sidebar click). AdminGate enforces auth client-side.
    if (hasSupabaseAuthCookie(request)) {
      return response;
    }
    if (!loginPath) {
      return response;
    }
    const login = request.nextUrl.clone();
    login.pathname = loginPath;
    login.search = "";
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

export const config = {
  matcher: [
    "/((?!_next/static|_next/image|favicon.ico|site\\.webmanifest|.*\\.(?:svg|png|jpg|jpeg|gif|webp|ico|webmanifest)$).*)",
  ],
};
