"use client";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";
import { createClient } from "@/lib/supabase/client";
import { cn } from "@/lib/utils";
import { ChevronDown, LayoutDashboard, LogOut, Settings } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { useEffect, useMemo, useState } from "react";

export type UserMenuProfile = {
  name?: string;
  email?: string;
  avatarUrl?: string;
};

type UserMenuProps = {
  profile: UserMenuProfile;
  /** Public marketing chrome: Dashboard + Settings + Sign out */
  variant: "public" | "dashboard" | "admin";
  dashboardHref?: string;
  settingsHref?: string;
  dashboardLabel?: string;
  settingsLabel?: string;
  signOutLabel: string;
  signOutPendingLabel: string;
  className?: string;
  /** Compact trigger (avatar only) for tight headers. */
  compact?: boolean;
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

/**
 * Standard avatar + name menu. Public: Dashboard / Settings / Sign out.
 * Dashboard & admin: Settings / Sign out.
 */
export function UserMenu({
  profile,
  variant,
  dashboardHref = "",
  settingsHref = "",
  dashboardLabel = "",
  settingsLabel = "",
  signOutLabel,
  signOutPendingLabel,
  className,
  compact = false,
}: UserMenuProps) {
  const [signingOut, setSigningOut] = useState(false);
  const [imgFailed, setImgFailed] = useState(false);
  const name = (profile.name || "").trim();
  const email = (profile.email || "").trim();
  const avatarUrl = (profile.avatarUrl || "").trim();
  const display = name || email || "";
  const mark = useMemo(() => initials(name, email), [name, email]);
  const outLabel = signOutLabel.trim();
  const outPending = signOutPendingLabel.trim();
  const dashHref = dashboardHref.trim();
  const dashLabel = dashboardLabel.trim();
  const setHref = settingsHref.trim();
  const setLabel = settingsLabel.trim();

  if (!outLabel || !outPending) return null;

  const showDashboard = variant === "public" && Boolean(dashHref && dashLabel);
  const showSettings = Boolean(setHref && setLabel);

  const signOut = () => {
    setSigningOut(true);
    void (async () => {
      try {
        await Promise.race([
          createClient().auth.signOut(),
          new Promise<void>((resolve) => setTimeout(resolve, 2500)),
        ]);
      } catch {
        /* still leave */
      }
      const login = (process.env.NEXT_PUBLIC_APP_PATH_LOGIN || "/login").trim();
      window.location.replace(login.startsWith("/") ? login : "/login");
    })();
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          className={cn(
            "inline-flex max-w-[14rem] items-center gap-2 rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel)] px-1.5 py-1 text-left text-sm text-[var(--trim-fg)] transition hover:border-[var(--trim-border-strong)] hover:bg-[var(--trim-hover)]",
            compact && "max-w-none px-1",
            className,
          )}
          aria-label={display || outLabel}
        >
          <span className="relative flex h-7 w-7 shrink-0 items-center justify-center overflow-hidden rounded-full bg-[var(--trim-panel-2)] text-[11px] font-semibold text-[var(--trim-muted)]">
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
          {!compact ? (
            <>
              <span className="min-w-0 flex-1 truncate font-medium">{display}</span>
              <ChevronDown className="h-3.5 w-3.5 shrink-0 text-[var(--trim-muted)]" />
            </>
          ) : null}
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56">
        {(name || email) && (
          <>
            <DropdownMenuLabel className="font-normal">
              {name ? <p className="truncate text-sm font-medium">{name}</p> : null}
              {email ? <p className="truncate text-xs text-[var(--trim-muted)]">{email}</p> : null}
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
          </>
        )}
        {showDashboard ? (
          <DropdownMenuItem asChild>
            <Link href={dashHref} scroll={false} className="cursor-pointer">
              <LayoutDashboard className="mr-2 h-4 w-4" />
              {dashLabel}
            </Link>
          </DropdownMenuItem>
        ) : null}
        {showSettings ? (
          <DropdownMenuItem asChild>
            <Link href={setHref} scroll={false} className="cursor-pointer">
              <Settings className="mr-2 h-4 w-4" />
              {setLabel}
            </Link>
          </DropdownMenuItem>
        ) : null}
        {(showDashboard || showSettings) && <DropdownMenuSeparator />}
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

/**
 * Public header auth control: session-aware Sign-in vs avatar menu.
 * While session settles, shows a profile-shaped shimmer (avoids Sign-in flash when logged in).
 */
export function PublicAuthChrome({
  site,
  compact = false,
  signInClassName,
}: {
  site: {
    nav_sign_in?: string;
    nav_sign_in_href?: string;
    path_login?: string;
    nav_dashboard?: string;
    nav_dashboard_href?: string;
    path_dashboard?: string;
    nav_settings?: string;
    path_settings?: string;
    sign_out_action_label?: string;
    sign_out_pending_label?: string;
  };
  compact?: boolean;
  /** Extra classes for the signed-out Sign-in control. */
  signInClassName?: string;
}) {
  const { ready, signedIn, profile } = useBrowserAuthProfile();
  const loginHref = (site.nav_sign_in_href || site.path_login || "").trim();
  const signInLabel = (site.nav_sign_in || "").trim();
  const dashHref = (site.nav_dashboard_href || site.path_dashboard || "").trim();
  const dashLabel = (site.nav_dashboard || "").trim();
  const setHref = (site.path_settings || "").trim();
  const setLabel = (site.nav_settings || "").trim();
  const outLabel = (site.sign_out_action_label || "").trim();
  const outPending = (site.sign_out_pending_label || "").trim();

  if (!ready) {
    return <Skeleton className={cn("h-9 shrink-0 rounded-md", compact ? "w-9" : "w-36")} />;
  }

  if (signedIn && outLabel && outPending) {
    return (
      <UserMenu
        variant="public"
        profile={profile}
        dashboardHref={dashHref}
        dashboardLabel={dashLabel}
        settingsHref={setHref}
        settingsLabel={setLabel}
        signOutLabel={outLabel}
        signOutPendingLabel={outPending}
        compact={compact}
      />
    );
  }

  if (!loginHref || !signInLabel) return null;

  return (
    <Link
      href={loginHref}
      className={
        signInClassName ||
        "inline-flex h-8 shrink-0 items-center whitespace-nowrap rounded-md bg-[var(--trim-ink)] px-3 text-[12px] font-medium text-[var(--trim-ink-inverse)]"
      }
    >
      {signInLabel}
    </Link>
  );
}

/** Live Supabase session profile for marketing chrome (outside DashboardGate). */
export function useBrowserAuthProfile(): {
  ready: boolean;
  signedIn: boolean;
  profile: UserMenuProfile;
  accessToken: string;
  userId: string;
} {
  const [ready, setReady] = useState(false);
  const [profile, setProfile] = useState<UserMenuProfile>({});
  const [signedIn, setSignedIn] = useState(false);
  const [accessToken, setAccessToken] = useState("");
  const [userId, setUserId] = useState("");

  useEffect(() => {
    const supabase = createClient();
    const apply = (
      session: {
        access_token?: string;
        user?: {
          id?: string;
          email?: string;
          user_metadata?: Record<string, unknown>;
        };
      } | null,
    ) => {
      if (!session?.user) {
        setSignedIn(false);
        setProfile({});
        setAccessToken("");
        setUserId("");
        return;
      }
      const meta = session.user.user_metadata || {};
      const name = [meta.full_name, meta.name, meta.user_name, meta.preferred_username]
        .map((v) => (typeof v === "string" ? v.trim() : ""))
        .find(Boolean);
      const avatarUrl = [meta.avatar_url, meta.picture, meta.avatar]
        .map((v) => (typeof v === "string" ? v.trim() : ""))
        .find(Boolean);
      setSignedIn(true);
      setAccessToken((session.access_token || "").trim());
      setUserId((session.user.id || "").trim());
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

  return { ready, signedIn, profile, accessToken, userId };
}
