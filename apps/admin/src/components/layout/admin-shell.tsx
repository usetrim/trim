"use client";

import { StepUpVerifyDialog } from "@/components/admin/step-up-verify-dialog";
import { AdminUserMenu, useAdminBrowserProfile } from "@/components/auth/user-menu";
import { TrimWordmark } from "@/components/brand/trim-wordmark";
import { AdminNotificationBell } from "@/components/notifications/notification-bell";
import { AdminShellSkeleton } from "@/components/skeletons/page-skeletons";
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
import { useAdminChromeNav } from "@/hooks/queries/chrome";
import { useAdminMe } from "@/hooks/queries/me";
import { adminNavIcon } from "@/lib/admin-nav-icons";
import { cn } from "@/lib/utils";
import type { AdminNavItem, AdminNavSection } from "@/types/admin/me";
import { useThemeStore } from "@/stores/theme";
import { ChevronRight, Menu } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useMemo, useState } from "react";

type Me = NonNullable<ReturnType<typeof useAdminMe>["data"]>;

function findActiveSectionId(pathname: string, sections: AdminNavSection[]): string {
  for (const section of sections) {
    const id = typeof section.id === "string" ? section.id : "";
    const items = Array.isArray(section.items) ? section.items : [];
    for (const item of items) {
      const href = typeof item.href === "string" ? item.href.trim() : "";
      if (!href) continue;
      const active =
        href === "/" ? pathname === "/" : pathname === href || pathname.startsWith(`${href}/`);
      if (active && id) return id;
    }
  }
  return sections[0]?.id?.trim() || "";
}

function NavAccordion({
  sections,
  pathname,
}: {
  sections: AdminNavSection[];
  pathname: string;
}) {
  const activeSection = useMemo(
    () => findActiveSectionId(pathname, sections),
    [pathname, sections],
  );
  const [openId, setOpenId] = useState(activeSection);

  useEffect(() => {
    setOpenId(activeSection);
  }, [activeSection]);

  return (
    <nav className="flex flex-1 flex-col gap-0 overflow-y-auto p-2" aria-label="Admin">
      {sections.map((section) => {
        const id = typeof section.id === "string" ? section.id : "";
        const label = typeof section.label === "string" ? section.label.trim() : "";
        const items = (Array.isArray(section.items) ? section.items : []).filter((item) => {
          const href = typeof item.href === "string" ? item.href.trim() : "";
          const itemLabel = (typeof item.label === "string" && item.label.trim()) || "";
          return Boolean(href && itemLabel);
        });
        if (!id || !label || items.length === 0) return null;

        // Single-item sections (e.g. Overview → Command center): show one top-level
        // link using the section label - no pointless accordion dropdown.
        if (items.length === 1) {
          const only = items[0];
          const href = typeof only.href === "string" ? only.href.trim() : "";
          const itemId = typeof only.id === "string" ? only.id.trim() : "";
          if (!href) return null;
          const Icon = adminNavIcon(itemId);
          const active =
            href === "/" ? pathname === "/" : pathname === href || pathname.startsWith(`${href}/`);
          return (
            <div key={id} className="border-b border-border/80">
              <Link
                href={href}
                scroll={false}
                prefetch
                className={cn(
                  "flex items-center gap-2 rounded-md px-3 py-2.5 text-sm font-medium transition-colors",
                  active
                    ? "bg-accent text-accent-foreground"
                    : "text-foreground hover:bg-accent/60",
                )}
              >
                {Icon ? <Icon className="h-3.5 w-3.5 shrink-0 opacity-80" aria-hidden /> : null}
                <span className="min-w-0 truncate">{label}</span>
              </Link>
            </div>
          );
        }

        const open = openId === id;
        return (
          <div key={id} className="border-b border-border/80">
            <button
              type="button"
              aria-expanded={open}
              onClick={() => {
                setOpenId((current) => (current === id ? "" : id));
              }}
              className="flex w-full items-center justify-between gap-2 rounded-md px-3 py-2.5 text-left text-sm font-medium text-foreground transition-colors hover:bg-accent/60"
            >
              <span>{label}</span>
              <ChevronRight
                className={cn(
                  "h-3.5 w-3.5 shrink-0 text-muted-foreground transition-transform duration-200 ease-out",
                  open && "rotate-90",
                )}
              />
            </button>
            <div
              className={cn(
                "grid transition-[grid-template-rows] duration-200 ease-out",
                open ? "grid-rows-[1fr]" : "grid-rows-[0fr]",
              )}
            >
              <div className="min-h-0 overflow-hidden">
                <ul className="space-y-0.5 pb-2 pl-1">
                  {items.map((item) => {
                    const href = typeof item.href === "string" ? item.href : "";
                    const itemLabel = typeof item.label === "string" ? item.label.trim() : "";
                    const itemId = typeof item.id === "string" ? item.id.trim() : "";
                    const Icon = adminNavIcon(itemId);
                    const active =
                      href === "/"
                        ? pathname === "/"
                        : pathname === href || pathname.startsWith(`${href}/`);
                    return (
                      <li key={itemId || href}>
                        <Link
                          href={href}
                          scroll={false}
                          prefetch
                          onClick={() => setOpenId(id)}
                          className={cn(
                            "flex items-center gap-2 rounded-md px-3 py-1.5 text-sm transition-colors",
                            active
                              ? "bg-accent font-medium text-accent-foreground"
                              : "text-muted-foreground hover:bg-accent/60 hover:text-accent-foreground",
                          )}
                        >
                          {Icon ? (
                            <Icon className="h-3.5 w-3.5 shrink-0 opacity-80" aria-hidden />
                          ) : null}
                          <span className="min-w-0 truncate">{itemLabel}</span>
                        </Link>
                      </li>
                    );
                  })}
                </ul>
              </div>
            </div>
          </div>
        );
      })}
    </nav>
  );
}

function FlatNav({ items, pathname }: { items: AdminNavItem[]; pathname: string }) {
  return (
    <nav className="flex flex-1 flex-col gap-0.5 overflow-y-auto p-2">
      {items.map((item) => {
        const href = typeof item.href === "string" ? item.href : "";
        const label = typeof item.label === "string" ? item.label.trim() : "";
        const itemId = typeof item.id === "string" ? item.id.trim() : "";
        if (!href || !label) return null;
        const Icon = adminNavIcon(itemId);
        const active =
          href === "/" ? pathname === "/" : pathname === href || pathname.startsWith(`${href}/`);
        return (
          <Link
            key={itemId || href}
            href={href}
            scroll={false}
            prefetch
            className={cn(
              "flex items-center gap-2 rounded-md px-3 py-2 text-sm transition-colors",
              active
                ? "bg-accent text-accent-foreground"
                : "text-muted-foreground hover:bg-accent/60 hover:text-accent-foreground",
            )}
          >
            {Icon ? <Icon className="h-3.5 w-3.5 shrink-0 opacity-80" aria-hidden /> : null}
            <span className="min-w-0 truncate">{label}</span>
          </Link>
        );
      })}
    </nav>
  );
}

/** Prefer Auth settings for the profile menu; fall back to Product / Billing settings. */
function resolveAdminSettings(nav: AdminNavItem[]): { href: string; label: string } {
  const prefer = ["auth", "product", "billing_settings"];
  for (const id of prefer) {
    const hit = nav.find((item) => (item.id || "").trim() === id);
    const href = (hit?.href || "").trim();
    const label = (hit?.label || "").trim();
    if (href && label) return { href, label };
  }
  return { href: "", label: "" };
}

export function AdminShell({
  token,
  me: meProp,
  children,
}: {
  token: string;
  me?: Me;
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const meQuery = useAdminMe(token);
  const me = meProp ?? meQuery.data;
  const chrome = useAdminChromeNav(token);
  const { ready: profileReady, profile: browserProfile } = useAdminBrowserProfile();
  const hydrate = useThemeStore((s) => s.hydrate);
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    hydrate();
  }, [hydrate]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setMobileOpen(false);
  }, [pathname]);

  if (!meProp && meQuery.isError) {
    return (
      <div className="mx-auto flex min-h-screen max-w-lg flex-col justify-center gap-3 p-8">
        <p className="text-sm text-muted-foreground">{(meQuery.error as Error)?.message || ""}</p>
      </div>
    );
  }

  if (!me) {
    return <AdminShellSkeleton>{children}</AdminShellSkeleton>;
  }

  const meChrome = me.chrome && typeof me.chrome === "object" ? me.chrome : {};
  const mapChrome = chrome.data && typeof chrome.data === "object" ? chrome.data : {};
  const labels: Record<string, string> = { ...mapChrome };
  for (const [code, body] of Object.entries(meChrome)) {
    const trimmed = typeof body === "string" ? body.trim() : "";
    if (trimmed) labels[code] = trimmed;
  }
  const nav = Array.isArray(me.nav) ? me.nav : [];
  const sections = Array.isArray(me.nav_sections) ? me.nav_sections : [];
  const routeAllowed = navAllowsPath(pathname, nav);
  const menuLabel = labels.ADMIN_NAV_MENU?.trim() || "";
  const closeLabel = labels.ADMIN_DIALOG_CLOSE?.trim() || labels.ADMIN_CLOSE?.trim() || "";
  const signOutLabel = labels.ADMIN_SIGN_OUT?.trim() || "";
  const signOutPending = labels.ADMIN_SIGN_OUT_PENDING?.trim() || "";
  const settings = resolveAdminSettings(nav);
  const profile = {
    name: browserProfile.name,
    email: browserProfile.email,
    avatarUrl: browserProfile.avatarUrl,
  };

  const navLinks =
    sections.length > 0 ? (
      <NavAccordion sections={sections} pathname={pathname} />
    ) : (
      <FlatNav items={nav} pathname={pathname} />
    );

  const brandBlock = (
    <div className="border-b border-border px-4 py-4">
      <TrimWordmark size="sm" alt={me.brand || "Trim"} />
      <p className="mt-1 text-xs text-muted-foreground">{me.tagline || ""}</p>
      <p className="mt-2 text-[11px] text-muted-foreground">{me.role_slug}</p>
    </div>
  );

  const profileMenu =
    signOutLabel && signOutPending && profileReady ? (
      <AdminUserMenu
        profile={profile}
        settingsHref={settings.href}
        settingsLabel={settings.label}
        signOutLabel={signOutLabel}
        signOutPendingLabel={signOutPending}
      />
    ) : signOutLabel && signOutPending ? (
      <Skeleton className="h-9 w-36 shrink-0 rounded-md" />
    ) : null;

  return (
    <div className="flex min-h-screen bg-background text-foreground">
      <aside className="sticky top-0 hidden h-screen w-60 shrink-0 flex-col self-start overflow-hidden border-r border-border bg-card print:hidden md:flex">
        {brandBlock}
        {navLinks}
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="sticky top-0 z-40 flex h-14 shrink-0 items-center gap-2 border-b border-border bg-card/95 px-3 backdrop-blur print:hidden supports-[backdrop-filter]:bg-card/80 sm:px-6">
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
                <SheetTitle>{me.brand || menuLabel}</SheetTitle>
                {me.tagline ? <SheetDescription>{me.tagline}</SheetDescription> : null}
              </SheetHeader>
              <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
                <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain">{navLinks}</div>
              </div>
            </SheetContent>
          </Sheet>
          <div className="min-w-0 flex-1 md:hidden">
            <TrimWordmark size="sm" alt={me.brand || "Trim"} />
          </div>
          <div className="ml-auto flex items-center gap-2">
            <ThemeToggle
              lightLabel={labels.ADMIN_THEME_LIGHT?.trim() || ""}
              darkLabel={labels.ADMIN_THEME_DARK?.trim() || ""}
            />
            {/* Bell owns its pending slot; do not unmount when ui-map chrome settles. */}
            <AdminNotificationBell token={token} />
            {profileMenu}
          </div>
        </header>
        <main className="flex-1 p-3 sm:p-4 md:p-6 print:p-0">
          <div className="mx-auto w-full max-w-7xl print:max-w-none">
            {routeAllowed ? (
              children
            ) : (
              <p className="text-sm text-muted-foreground">
                {labels.ADMIN_ROUTE_FORBIDDEN?.trim() || ""}
              </p>
            )}
          </div>
        </main>
        <StepUpVerifyDialog token={token} />
      </div>
    </div>
  );
}

function navAllowsPath(pathname: string, nav: Array<{ href?: string }>): boolean {
  if (pathname === "/login" || pathname.startsWith("/auth")) return true;
  return nav.some((item) => {
    const href = typeof item.href === "string" ? item.href.trim() : "";
    if (!href) return false;
    if (href === "/") return pathname === "/";
    return pathname === href || pathname.startsWith(`${href}/`);
  });
}
