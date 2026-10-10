"use client";

import { createClient } from "@/lib/supabase/client";
import type { AuthChangeEvent, Session } from "@supabase/supabase-js";
import { useEffect, useState } from "react";

/**
 * Browser access token with SSR handoff.
 * Industry practice: never wipe a known-good token on a transient null
 * INITIAL_SESSION (cookie hydration race). Only clear on SIGNED_OUT.
 *
 * Pass `enabled: false` when a parent already owns the session (AdminGate shell)
 * so soft-nav page remounts do not re-subscribe Supabase.
 */
export function useAccessToken(initial?: string, enabled = true): string {
  const [token, setToken] = useState(initial?.trim() || "");

  useEffect(() => {
    if (!enabled) return;

    const seed = initial?.trim() || "";
    if (seed) setToken(seed);

    const supabase = createClient();

    void Promise.race([
      supabase.auth.getSession(),
      new Promise<{ data: { session: null } }>((resolve) =>
        setTimeout(() => resolve({ data: { session: null } }), 5000),
      ),
    ]).then(({ data }) => {
      const next = data.session?.access_token?.trim() || "";
      setToken((prev) => next || prev || seed);
    });

    const { data: sub } = supabase.auth.onAuthStateChange(
      (event: AuthChangeEvent, session: Session | null) => {
        if (event === "SIGNED_OUT") {
          setToken("");
          return;
        }
        const next = session?.access_token?.trim() || "";
        if (next) {
          setToken(next);
          return;
        }
        // Ignore transient null (INITIAL_SESSION / TOKEN_REFRESHED edge); keep prior.
      },
    );

    return () => {
      sub.subscription.unsubscribe();
    };
  }, [initial, enabled]);

  return token;
}
