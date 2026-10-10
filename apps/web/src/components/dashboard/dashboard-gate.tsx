"use client";

import { DashboardBootSkeleton } from "@/components/dashboard/dashboard-boot-skeleton";
import {
  DashboardSessionProvider,
  useDashboardSessionOptional,
  type DashboardSession,
} from "@/hooks/use-dashboard-session";
import { createClient } from "@/lib/supabase/client";
import type { AuthChangeEvent, Session } from "@supabase/supabase-js";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";

function firstNonEmpty(...values: (string | undefined)[]): string | undefined {
  for (const v of values) {
    const t = (v || "").trim();
    if (t) return t;
  }
  return undefined;
}

function sessionFromSupabase(s: Session | null): DashboardSession | null {
  const accessToken = s?.access_token?.trim() || "";
  if (!accessToken) return null;
  const meta = (s?.user.user_metadata || {}) as Record<string, unknown>;
  return {
    accessToken,
    userId: s?.user.id,
    email: s?.user.email,
    fullName: firstNonEmpty(
      meta.full_name as string | undefined,
      meta.name as string | undefined,
      meta.user_name as string | undefined,
      meta.preferred_username as string | undefined,
      meta.nickname as string | undefined,
      meta.login as string | undefined,
    ),
    avatarUrl: firstNonEmpty(
      meta.avatar_url as string | undefined,
      meta.picture as string | undefined,
      meta.avatar as string | undefined,
    ),
  };
}

/**
 * Provides session context for the console.
 * Always renders `children` (DashboardShell) so refresh never remounts
 * ShellSkeleton → Shell (that double-shimmer flash). Body uses BootSkeleton
 * until a session exists via DashboardMain.
 */
export function DashboardGate({ children }: { children: ReactNode }) {
  const router = useRouter();
  const login = (process.env.NEXT_PUBLIC_APP_PATH_LOGIN || "").trim();
  const [live, setLive] = useState<DashboardSession | null>(null);
  const latchedRef = useRef<DashboardSession | null>(null);
  const bouncedRef = useRef(false);

  useEffect(() => {
    const supabase = createClient();

    void Promise.race([
      supabase.auth.getSession(),
      new Promise<{ data: { session: null } }>((resolve) =>
        setTimeout(() => resolve({ data: { session: null } }), 5000),
      ),
    ]).then(({ data }) => {
      const next = sessionFromSupabase(data.session);
      setLive(next);
      if (!next && login && !bouncedRef.current) {
        bouncedRef.current = true;
        router.replace(login);
      }
    });

    const { data: sub } = supabase.auth.onAuthStateChange(
      (event: AuthChangeEvent, next: Session | null) => {
        if (event === "SIGNED_OUT") {
          setLive(null);
          latchedRef.current = null;
          if (login && !bouncedRef.current) {
            bouncedRef.current = true;
            router.replace(login);
          }
          return;
        }
        const parsed = sessionFromSupabase(next);
        if (parsed) {
          setLive(parsed);
        }
      },
    );

    return () => {
      sub.subscription.unsubscribe();
    };
  }, [router, login]);

  if (live) {
    latchedRef.current = live;
  }

  const display = live ?? latchedRef.current;
  const session = useMemo(() => display, [display]);

  return <DashboardSessionProvider session={session}>{children}</DashboardSessionProvider>;
}

/** Body slot: route boot shimmer until session exists, then the page segment. */
export function DashboardMain({ children }: { children: ReactNode }) {
  const session = useDashboardSessionOptional();
  if (!session?.accessToken) {
    return <DashboardBootSkeleton />;
  }
  return <>{children}</>;
}
