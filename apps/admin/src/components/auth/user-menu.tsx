"use client";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { createClient } from "@/lib/supabase/client";
import { envPathLogin } from "@/lib/app-paths";
import { cn } from "@/lib/utils";
import { ChevronDown, LogOut, Settings } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { useEffect, useMemo, useState } from "react";

export type AdminUserMenuProfile = {
  name?: string;
  email?: string;
  avatarUrl?: string;
};

type AdminUserMenuProps = {
  profile: AdminUserMenuProfile;
  settingsHref?: string;
  settingsLabel?: string;
  signOutLabel: string;
  signOutPendingLabel: string;
  className?: string;
};

function initials(name: string, email: string): string {
  const n = name.trim();
  if (n) {
    const parts = n.split(/\s+/).filter(Boolean);
    if (parts.length >= 2) {
      return `${parts[0]?.[0] ?? ""}${parts[1]?.[0] ?? ""}`.toUpperCase();
    }
    return n.slice(0, 2).toUpperCase();
  }
  const e = email.trim();
  return e ? e.slice(0, 2).toUpperCase() : "?";
}

export function AdminUserMenu({
  profile,
  settingsHref = "",
  settingsLabel = "",
  signOutLabel,
  signOutPendingLabel,
  className,
}: AdminUserMenuProps) {
  const [signingOut, setSigningOut] = useState(false);
  const [imgFailed, setImgFailed] = useState(false);
  const name = (profile.name || "").trim();
  const email = (profile.email || "").trim();
  const avatarUrl = (profile.avatarUrl || "").trim();
  const display = name || email || "";
  const mark = useMemo(() => initials(name, email), [name, email]);
  const outLabel = signOutLabel.trim();
  const outPending = signOutPendingLabel.trim();
  const setHref = settingsHref.trim();
  const setLabel = settingsLabel.trim();

  if (!outLabel || !outPending) return null;

  const signOut = () => {
    setSigningOut(true);
    void (async () => {
      const login = envPathLogin();
      try {
        await Promise.race([
          createClient().auth.signOut(),
          new Promise<void>((resolve) => setTimeout(resolve, 2500)),
        ]);
      } catch {
        /* still leave */
      }
      if (login) {
        window.location.replace(login);
        return;
      }
      setSigningOut(false);
    })();
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          className={cn(
            "inline-flex max-w-[14rem] items-center gap-2 rounded-md border border-border bg-card px-1.5 py-1 text-left text-sm transition hover:bg-accent/60",
            className,
          )}
          aria-label={display || outLabel}
        >
          <span className="relative flex h-7 w-7 shrink-0 items-center justify-center overflow-hidden rounded-full bg-muted text-[11px] font-semibold text-muted-foreground">
            {avatarUrl && !imgFailed ? (
              <Image
                src={avatarUrl}
                alt=""
                width={28}
                height={28}
                className="h-7 w-7 object-cover"
                unoptimized
                onError={() => setImgFailed(true)}
              />
            ) : (
              mark
            )}
          </span>
          <span className="min-w-0 flex-1 truncate font-medium">{display}</span>
          <ChevronDown className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56">
        {(name || email) && (
          <>
            <DropdownMenuLabel className="font-normal">
              {name ? <p className="truncate text-sm font-medium">{name}</p> : null}
              {email ? <p className="truncate text-xs text-muted-foreground">{email}</p> : null}
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
          </>
        )}
        {setHref && setLabel ? (
          <>
            <DropdownMenuItem asChild>
              <Link href={setHref} scroll={false} className="cursor-pointer">
                <Settings className="mr-2 h-4 w-4" />
                {setLabel}
              </Link>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
          </>
        ) : null}
        <DropdownMenuItem
          disabled={signingOut}
          onSelect={(e) => {
            e.preventDefault();
            signOut();
          }}
          className="cursor-pointer text-destructive focus:text-destructive"
        >
          <LogOut className="mr-2 h-4 w-4" />
          {signingOut ? outPending : outLabel}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function useAdminBrowserProfile(): {
  ready: boolean;
  profile: AdminUserMenuProfile;
} {
  const [ready, setReady] = useState(false);
  const [profile, setProfile] = useState<AdminUserMenuProfile>({});

  useEffect(() => {
    const supabase = createClient();
    const apply = (
      session: {
        user?: { email?: string; user_metadata?: Record<string, unknown> };
      } | null,
    ) => {
      if (!session?.user) {
        setProfile({});
        return;
      }
      const meta = session.user.user_metadata || {};
      const name = [meta.full_name, meta.name, meta.user_name]
        .map((v) => (typeof v === "string" ? v.trim() : ""))
        .find(Boolean);
      const avatarUrl = [meta.avatar_url, meta.picture, meta.avatar]
        .map((v) => (typeof v === "string" ? v.trim() : ""))
        .find(Boolean);
      setProfile({
        name,
        email: session.user.email?.trim() || "",
        avatarUrl,
      });
    };
    void supabase.auth.getSession().then(({ data }) => {
      apply(data.session);
      setReady(true);
    });
    const { data: sub } = supabase.auth.onAuthStateChange((_e, session) => {
      apply(session);
      setReady(true);
    });
    return () => sub.subscription.unsubscribe();
  }, []);

  return { ready, profile };
}
