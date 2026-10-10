"use client";

import { AdminBootSkeleton } from "@/components/admin/admin-boot-skeleton";
import { AdminShellSkeleton } from "@/components/skeletons/page-skeletons";
import { useAdminMe } from "@/hooks/queries/me";
import { useAccessToken } from "@/hooks/use-access-token";
import { envPathLogin } from "@/lib/app-paths";
import { createClient } from "@/lib/supabase/client";
import type { ApiError } from "@/types/admin";
import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";

type AdminSession = {
  token: string;
  me: NonNullable<ReturnType<typeof useAdminMe>["data"]>;
};

const AdminSessionContext = createContext<AdminSession | null>(null);

export function useAdminSession(): AdminSession {
  const ctx = useContext(AdminSessionContext);
  if (!ctx) {
    throw new Error("useAdminSession requires AdminGate");
  }
  return ctx;
}

function loginErrorCode(status: number | undefined): string {
  if (status === 401) return "admin_credentials";
  if (status === 503) return "admin_bootstrap";
  if (status === 403) return "admin_forbidden";
  return "admin_forbidden";
}

/**
 * Renders children only when GET /api/v1/admin/me succeeds.
 * Uses context (not render props) so the shell tree identity stays stable across
 * soft-nav and query updates - render props were recreating AdminShell every render.
 *
 * Once a session is established, latch it so a transient empty token /me pause
 * cannot tear the whole console down to a skeleton (looked like a full refresh
 * while typing or soft-navigating).
 */
export function AdminGate({
  initialToken,
  children,
}: {
  initialToken?: string;
  children: ReactNode;
}) {
  const token = useAccessToken(initialToken);
  const meQuery = useAdminMe(token);
  const [sessionSettled, setSessionSettled] = useState(Boolean(initialToken?.trim()));
  const [rejecting, setRejecting] = useState(false);
  const bouncedRef = useRef(false);
  const latchedSessionRef = useRef<AdminSession | null>(null);

  useEffect(() => {
    if (initialToken?.trim()) {
      setSessionSettled(true);
      return;
    }
    const supabase = createClient();
    void Promise.race([
      supabase.auth.getSession(),
      new Promise<void>((resolve) => setTimeout(resolve, 5000)),
    ]).finally(() => setSessionSettled(true));
  }, [initialToken]);

  useEffect(() => {
    if (!sessionSettled || rejecting || bouncedRef.current) return;
    const login = envPathLogin();
    if (!login) return;

    if (!meQuery.isError) return;

    const status = (meQuery.error as ApiError)?.status;
    const code = loginErrorCode(status);
    bouncedRef.current = true;
    setRejecting(true);
    latchedSessionRef.current = null;

    void (async () => {
      const dest = new URL(login, window.location.origin);
      dest.searchParams.set("error", code);
      try {
        await Promise.race([
          createClient().auth.signOut(),
          new Promise<void>((resolve) => setTimeout(resolve, 2500)),
        ]);
      } catch {
        // Still leave the protected surface.
      } finally {
        window.location.replace(dest.toString());
      }
    })();
  }, [sessionSettled, meQuery.isError, meQuery.error, rejecting]);

  useEffect(() => {
    if (!sessionSettled || rejecting || bouncedRef.current) return;
    if (token || initialToken?.trim() || latchedSessionRef.current) return;
    const login = envPathLogin();
    if (!login) return;
    bouncedRef.current = true;
    setRejecting(true);
    const dest = new URL(login, window.location.origin);
    dest.searchParams.set("error", "admin_credentials");
    window.location.replace(dest.toString());
  }, [sessionSettled, token, initialToken, rejecting]);

  const effectiveToken = (token || initialToken || "").trim();
  const session = useMemo<AdminSession | null>(() => {
    if (!meQuery.data || !effectiveToken) return null;
    return { token: effectiveToken, me: meQuery.data };
  }, [effectiveToken, meQuery.data]);

  if (session) {
    latchedSessionRef.current = session;
  }

  const displaySession = rejecting ? null : session ?? latchedSessionRef.current;

  if (!sessionSettled || rejecting) {
    return (
      <AdminShellSkeleton>
        <AdminBootSkeleton />
      </AdminShellSkeleton>
    );
  }

  // Soft-nav: never tear down the shell when /me or a latched session exists.
  if (meQuery.isError && !meQuery.data && !latchedSessionRef.current) {
    return (
      <AdminShellSkeleton>
        <AdminBootSkeleton />
      </AdminShellSkeleton>
    );
  }
  if (!displaySession) {
    return (
      <AdminShellSkeleton>
        <AdminBootSkeleton />
      </AdminShellSkeleton>
    );
  }

  return (
    <AdminSessionContext.Provider value={displaySession}>{children}</AdminSessionContext.Provider>
  );
}
