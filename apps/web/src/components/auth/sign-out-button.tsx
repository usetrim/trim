"use client";

import { Button } from "@/components/ui/button";
import { useAuthProviders } from "@/hooks/queries/auth";
import { createClient } from "@/lib/supabase/client";
import { cn } from "@/lib/utils";
import { useState } from "react";

/**
 * Full E2E sign-out: clear Supabase session (bounded), then hard-navigate to
 * login (or home) so client caches and protected routes cannot linger.
 */
export function SignOutButton({
  className,
  variant = "outline",
  actionLabel,
  pendingLabel,
}: {
  className?: string;
  variant?: "default" | "secondary" | "outline" | "ghost" | "destructive";
  /** Prefer subscription chrome when present; falls back to auth-providers site. */
  actionLabel?: string;
  pendingLabel?: string;
}) {
  const auth = useAuthProviders();
  const [signingOut, setSigningOut] = useState(false);

  const label = (
    actionLabel?.trim() ||
    auth.data?.site?.sign_out_action_label?.trim() ||
    ""
  ).trim();
  const pending = (
    pendingLabel?.trim() ||
    auth.data?.site?.sign_out_pending_label?.trim() ||
    ""
  ).trim();
  const login = (auth.data?.site?.path_login || "").trim();
  const home = (auth.data?.site?.path_home || "").trim();

  if (!label || !pending) return null;

  return (
    <Button
      type="button"
      variant={variant}
      className={cn(className)}
      isLoading={signingOut}
      pendingLabel={pending}
      onClick={() => {
        setSigningOut(true);
        void (async () => {
          try {
            await Promise.race([
              createClient().auth.signOut(),
              new Promise<void>((resolve) => setTimeout(resolve, 2500)),
            ]);
          } catch {
            // Still leave the signed-in surface.
          }
          const dest =
            login.startsWith("/") && !login.startsWith("//")
              ? login
              : home.startsWith("/") && !home.startsWith("//")
                ? home
                : "";
          if (dest) {
            window.location.replace(dest);
            return;
          }
          setSigningOut(false);
        })();
      }}
    >
      {label}
    </Button>
  );
}
