"use client";

import { Button } from "@/components/ui/button";
import { createClient } from "@/lib/supabase/client";
import { envPathLogin } from "@/lib/app-paths";
import { useState } from "react";

/** Full E2E operator sign-out: clear Supabase session, then hard-navigate to login. */
export function AdminSignOutButton({
  actionLabel,
  pendingLabel,
  className,
}: {
  actionLabel: string;
  pendingLabel: string;
  className?: string;
}) {
  const [signingOut, setSigningOut] = useState(false);
  const label = actionLabel.trim();
  const pending = pendingLabel.trim();
  if (!label || !pending) return null;

  return (
    <Button
      type="button"
      variant="outline"
      className={className}
      isLoading={signingOut}
      pendingLabel={pending}
      onClick={() => {
        setSigningOut(true);
        void (async () => {
          const login = envPathLogin();
          try {
            await Promise.race([
              createClient().auth.signOut(),
              new Promise<void>((resolve) => setTimeout(resolve, 2500)),
            ]);
          } catch {
            // Still leave the protected surface.
          }
          if (login.startsWith("/") && !login.startsWith("//")) {
            window.location.replace(login);
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
