"use client";

import { UserMenu } from "@/components/auth/user-menu";
import { TrimWordmark, TrimWordmarkLink } from "@/components/brand/trim-wordmark";
import { NotificationBell } from "@/components/notifications/notification-bell";
import { ThemeToggle } from "@/components/theme/theme-toggle";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { useAuthProviders } from "@/hooks/queries/auth";
import { useSubscriptionStatus } from "@/hooks/queries/billing";
import { useDashboardSessionOptional } from "@/hooks/use-dashboard-session";
import { dashboardNavIcon } from "@/lib/dashboard-nav-icons";
import { refreshBillingUntilSettled } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import { useQueryClient } from "@tanstack/react-query";
import { Menu } from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { toast } from "sonner";

type NavItem = { id: string; href: string; label: string };

/** Stable shimmer keys for pending chrome slots. */
function chromeKeys(prefix: string, count: number): string[] {
  return Array.from({ length: count }, (_, i) => `${prefix}-${i}`);
}

/** Expected console nav count (Dashboard → Settings). */
const NAV_SLOT_COUNT = 6;

/**
 * Persistent customer console chrome - mirrors admin AdminShell.
 * Prefer auth-providers site chrome for all six nav items so they paint together.
 * Fall back to subscription labels when an older API has not yet shipped nav_team/nav_settings.
 */
function DashboardCheckoutSuccessEffect() {
  const search = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();
  const qc = useQueryClient();
  const authProviders = useAuthProviders();
  const handled = useRef(false);

  useEffect(() => {
    if (search.get("checkout") !== "success") return;
    if (handled.current) return;
    handled.current = true;
    void refreshBillingUntilSettled(qc);
    const toastMsg = (authProviders.data?.site?.checkout_success_toast || "").trim();
    if (toastMsg) toast.success(toastMsg);
    const params = new URLSearchParams(search.toString());
    params.delete("checkout");
    params.delete("plan");
    const qs = params.toString();
    router.replace(qs ? `${pathname}?${qs}` : pathname, { scroll: false });
  }, [authProviders.data?.site?.checkout_success_toast, pathname, qc, router, search]);

  return null;
}

export function DashboardShell({ children }: { children: ReactNode }) {
  const session = useDashboardSessionOptional();
  const accessToken = session?.accessToken;
  const email = session?.email;
  const fullName = session?.fullName;
  const avatarUrl = session?.avatarUrl;
  const pathname = usePathname();
  const authProviders = useAuthProviders();
  const subscription = useSubscriptionStatus(accessToken);
  const [mobileOpen, setMobileOpen] = useState(false);

  // biome-ignore lint/correctness/useExhaustiveDependencies: close mobile nav on route change
  useEffect(() => {
    setMobileOpen(false);
  }, [pathname]);

  const site = authProviders.data?.site;
  const sub = subscription.data;

  const siteHasFullNav = Boolean(site?.nav_team?.trim() && site?.nav_settings?.trim());
  // Wait for subscription only when site chrome is missing Team/Settings labels.
  // Also pending while session has not settled yet (shell stays mounted on refresh).
  const chromePending =
    !accessToken ||
    (authProviders.isPending && !site) ||
    (Boolean(accessToken) &&
      !siteHasFullNav &&
      subscription.isPending &&
      !subscription.data &&
      !subscription.isError);

  const brand = (site?.brand || "").trim();
  const tagline = (site?.tagline || site?.eyebrow || "").trim();
  const homeHref =
    site?.path_home?.startsWith("/") && !site.path_home.startsWith("//") ? site.path_home : "";
  const menuLabel = (site?.nav_dashboard || "").trim();
  const closeLabel = (site?.dialog_cancel || authProviders.data?.dialog_close_label || "").trim();

  const signOutLabel = (
    site?.sign_out_action_label?.trim() ||
    sub?.sign_out_action_label?.trim() ||
    ""
  ).trim();
  const signOutPending = (
    site?.sign_out_pending_label?.trim() ||
    sub?.sign_out_pending_label?.trim() ||
    ""
  ).trim();

  const settingsHref = (site?.path_settings || sub?.path_settings || "").trim();
  const settingsLabel = (site?.nav_settings || sub?.nav_settings_label || "").trim();

  const nav = useMemo(() => {
    const items: NavItem[] = [];

    const dashHref = (
      site?.path_dashboard ||
      site?.nav_dashboard_href ||
      sub?.path_dashboard ||
      ""
    ).trim();
    const dashLabel = (site?.nav_dashboard || "").trim();
    if (dashHref && dashLabel) items.push({ id: "dashboard", href: dashHref, label: dashLabel });

    const tracesHref = (site?.path_traces || sub?.path_traces || "").trim();
    const tracesLabel = (site?.nav_traces || sub?.nav_traces_label || "").trim();
    if (tracesHref && tracesLabel)
      items.push({ id: "traces", href: tracesHref, label: tracesLabel });

    const receiptsHref = (site?.path_receipts || sub?.path_receipts || "").trim();
    const receiptsLabel = (site?.nav_receipts || sub?.nav_receipts_label || "").trim();
    if (receiptsHref && receiptsLabel) {
      items.push({ id: "receipts", href: receiptsHref, label: receiptsLabel });
    }

    const enterpriseHref = (site?.path_enterprise || sub?.path_enterprise || "").trim();
    const enterpriseLabel = (site?.nav_enterprise || sub?.nav_enterprise_label || "").trim();
    if (enterpriseHref && enterpriseLabel) {
      items.push({ id: "enterprise", href: enterpriseHref, label: enterpriseLabel });
    }

    const teamHref = (site?.path_team || sub?.path_team || "").trim();
    const teamLabel = (site?.nav_team || sub?.nav_team_label || "").trim();
    if (teamHref && teamLabel) items.push({ id: "team", href: teamHref, label: teamLabel });

    if (settingsHref && settingsLabel) {
      items.push({ id: "settings", href: settingsHref, label: settingsLabel });
    }

    return items;
  }, [site, sub, settingsHref, settingsLabel]);

  const navLinks = chromePending ? (
    <div className="flex flex-1 flex-col gap-1 overflow-y-auto p-2">
      {chromeKeys("nav", NAV_SLOT_COUNT).map((id) => (
        <Skeleton key={id} className="h-9 w-full rounded-md" />
      ))}
    </div>
  ) : (
    <nav className="flex flex-1 flex-col gap-0.5 overflow-y-auto p-2">
      {nav.map((item) => {
        const active = navItemActive(pathname, item.href, nav);
        const Icon = dashboardNavIcon(item.id);
        return (
          <Link
            key={item.id || item.href}
            href={item.href}
            scroll={false}
            prefetch
            className={cn(
              "flex items-center gap-2 rounded-md px-3 py-2 text-sm transition-colors",
              active
                ? "bg-[var(--trim-panel-2)] text-[var(--trim-fg)]"
                : "text-[var(--trim-muted)] hover:bg-[var(--trim-panel-2)]/70 hover:text-[var(--trim-fg)]",
            )}
          >
            {Icon ? <Icon className="h-3.5 w-3.5 shrink-0 opacity-80" aria-hidden /> : null}
            <span className="min-w-0 truncate">{item.label}</span>
          </Link>
        );
      })}
    </nav>
  );

  const brandTitle = brand ? (
    homeHref ? (
      <TrimWordmarkLink href={homeHref} size="sm" alt={brand} />
    ) : (
      <TrimWordmark size="sm" alt={brand} />
    )
  ) : (
    <TrimWordmark size="sm" alt="Trim" />
  );

  const brandBlock = (
    <div className="border-b border-[var(--trim-border)] px-4 py-4">
      {chromePending && !brand ? (
        <div className="space-y-2">
          <Skeleton className="h-4 w-24" />
          <Skeleton className="h-3 w-36" />
        </div>
      ) : (
        <>
          {brandTitle}
          {tagline ? <p className="mt-1 text-xs text-[var(--trim-muted)]">{tagline}</p> : null}
          {chromePending && !tagline ? <Skeleton className="mt-1 h-3 w-36" /> : null}
        </>
      )}
    </div>
  );

  const profileMenu =
    signOutLabel && signOutPending ? (
      <UserMenu
        variant="dashboard"
        profile={{ name: fullName, email, avatarUrl }}
        settingsHref={settingsHref}
        settingsLabel={settingsLabel}
        signOutLabel={signOutLabel}
        signOutPendingLabel={signOutPending}
      />
    ) : chromePending ? (
      <Skeleton className="h-9 w-36 shrink-0 rounded-md" />
    ) : null;

  return (
    <div className="flex min-h-screen bg-[var(--trim-bg)] text-[var(--trim-fg)]">
      <Suspense fallback={null}>
        <DashboardCheckoutSuccessEffect />
      </Suspense>
      <aside className="sticky top-0 hidden h-screen w-60 shrink-0 flex-col self-start overflow-hidden border-r border-[var(--trim-border)] bg-[var(--trim-panel)] print:hidden md:flex">
        {brandBlock}
        {navLinks}
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="sticky top-0 z-40 flex h-14 shrink-0 items-center gap-2 border-b border-[var(--trim-border)] bg-[var(--trim-panel)]/95 px-3 backdrop-blur print:hidden supports-[backdrop-filter]:bg-[var(--trim-panel)]/80 sm:px-6">
          <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
            <SheetTrigger asChild>
              <Button
                type="button"
                variant="outline"
                size="icon"
                className="md:hidden"
                aria-label={menuLabel || undefined}
                title={menuLabel || undefined}
              >
                <Menu className="h-4 w-4" />
              </Button>
            </SheetTrigger>
            <SheetContent side="left" closeLabel={closeLabel} className="p-0">
              <SheetHeader>
                <SheetTitle>{brand || menuLabel}</SheetTitle>
                {tagline ? <SheetDescription>{tagline}</SheetDescription> : null}
              </SheetHeader>
              <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
                <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain">{navLinks}</div>
              </div>
            </SheetContent>
          </Sheet>
          <div className="min-w-0 flex-1 md:hidden">
            {brand && homeHref ? (
              <TrimWordmarkLink href={homeHref} size="sm" alt={brand} />
            ) : (
              <TrimWordmark size="sm" alt={brand || "Trim"} />
            )}
          </div>
          <div className="ml-auto flex items-center gap-2">
            <ThemeToggle />
            {/* Bell owns its pending slot (skeleton→button); do not unmount on shell chrome. */}
            <NotificationBell accessToken={accessToken} />
            {profileMenu}
          </div>
        </header>
        <main className="flex-1 p-3 sm:p-4 md:p-6 print:p-0">
          <div className="mx-auto w-full max-w-[1400px] print:max-w-none">{children}</div>
        </main>
      </div>
    </div>
  );
}

/** Prefer the longest matching nav href so /dashboard does not stay active on /dashboard/team. */
function navItemActive(pathname: string, href: string, nav: NavItem[]): boolean {
  if (pathname === href) return true;
  if (!pathname.startsWith(`${href}/`)) return false;
  return !nav.some(
    (other) =>
      other.href !== href &&
      other.href.length > href.length &&
      (pathname === other.href || pathname.startsWith(`${other.href}/`)),
  );
}
