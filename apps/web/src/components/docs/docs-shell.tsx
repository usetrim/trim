"use client";

import { PublicAuthChrome } from "@/components/auth/user-menu";
import { TrimWordmarkLink } from "@/components/brand/trim-wordmark";
import { DocsSidebar } from "@/components/docs/docs-sidebar";
import { DocsPageSkeleton } from "@/components/skeletons/page-skeletons";
import { ThemeToggle } from "@/components/theme/theme-toggle";
import { useAuthProviders } from "@/hooks/queries/auth";
import { apiFetch } from "@/lib/api/client";
import { qk } from "@/lib/query-keys";
import { cn } from "@/lib/utils";
import type { PlansResponse } from "@/types/billing";
import { useQueryClient } from "@tanstack/react-query";
import { Menu, X } from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState, type ReactNode } from "react";

function slugFromPath(pathname: string) {
  if (!pathname.startsWith("/docs")) return "";
  const rest = pathname.slice("/docs".length).replace(/^\//, "");
  return rest;
}

/** Persistent docs chrome. Lives in layout so slug changes do not remount the shell. */
export function DocsShell({ children }: { children: ReactNode }) {
  const chrome = useAuthProviders();
  const site = chrome.data?.site;
  const brand = site?.brand || "Trim";
  const pathname = usePathname() || "/docs";
  const slug = slugFromPath(pathname);
  const router = useRouter();
  const queryClient = useQueryClient();
  const [mobileOpen, setMobileOpen] = useState(false);

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setMobileOpen(false);
  }, [pathname]);

  // Soft-nav: pricing stays mounted in LandingShell; keep plans warm anyway.
  useEffect(() => {
    router.prefetch("/pricing");
    router.prefetch("/");
    for (const interval of ["monthly", "annual"] as const) {
      void queryClient.prefetchQuery({
        queryKey: [...qk.plans, interval, "anon"],
        queryFn: () =>
          apiFetch<PlansResponse>(`/api/v1/public/plans?interval=${encodeURIComponent(interval)}`),
        staleTime: 60_000,
      });
    }
  }, [queryClient, router]);

  // Parent LandingShell already gates on chrome; keep a local skeleton only for standalone mounts.
  if (chrome.isPending && !site) {
    return <DocsPageSkeleton authSlot="pending" />;
  }

  return (
    <div className="min-h-screen bg-[var(--trim-bg)] text-[var(--trim-fg)]">
      <header className="sticky top-0 z-40 border-b border-[var(--trim-border)] bg-[var(--trim-panel)]/95 backdrop-blur">
        <div className="mx-auto flex h-14 w-full max-w-none items-center gap-3 px-4 sm:px-6 lg:px-10">
          <button
            type="button"
            className="inline-flex h-8 w-8 items-center justify-center rounded-md border border-[var(--trim-border)] bg-[var(--trim-bg)] text-[var(--trim-fg)] transition hover:border-[var(--trim-border-strong)] hover:bg-[var(--trim-hover)] lg:hidden"
            aria-label={mobileOpen ? "Close menu" : "Open menu"}
            onClick={() => setMobileOpen((v) => !v)}
          >
            {mobileOpen ? <X className="h-4 w-4" /> : <Menu className="h-4 w-4" />}
          </button>
          <TrimWordmarkLink href="/" size="md" alt={brand} priority />
          <span className="hidden text-[var(--trim-muted)] sm:inline">/</span>
          <Link
            href="/docs"
            scroll={false}
            className="hidden text-sm text-[var(--trim-muted)] hover:text-[var(--trim-fg)] sm:inline"
          >
            Docs
          </Link>
          <div className="ml-auto flex items-center gap-2">
            <Link
              href="/pricing"
              scroll={false}
              className="hidden rounded-md px-2 py-1 text-[12px] text-[var(--trim-muted)] transition hover:text-[var(--trim-fg)] sm:inline"
            >
              Pricing
            </Link>
            {site?.path_privacy ? (
              <Link
                href={site.path_privacy}
                scroll={false}
                className="hidden rounded-md px-2 py-1 text-[12px] text-[var(--trim-muted)] transition hover:text-[var(--trim-fg)] md:inline"
              >
                Privacy
              </Link>
            ) : null}
            {site ? (
              <div className="hidden sm:block">
                <PublicAuthChrome
                  site={site}
                  compact
                  signInClassName="inline-flex whitespace-nowrap rounded-md bg-[var(--trim-ink)] px-3 py-1.5 text-[12px] font-medium text-[var(--trim-ink-inverse)]"
                />
              </div>
            ) : null}
            <ThemeToggle />
          </div>
        </div>
      </header>

      <div className="mx-auto flex w-full max-w-none items-start">
        <aside
          className={cn(
            "fixed inset-y-0 left-0 z-30 w-[min(86vw,300px)] overflow-y-auto border-r border-[var(--trim-border)] bg-[var(--trim-panel)] px-3 pb-8 pt-[4.25rem] transition-transform lg:sticky lg:top-14 lg:z-0 lg:block lg:h-[calc(100vh-3.5rem)] lg:w-[260px] lg:shrink-0 lg:translate-x-0 lg:self-start lg:overflow-y-auto lg:pt-6",
            mobileOpen ? "translate-x-0" : "-translate-x-full lg:translate-x-0",
          )}
        >
          <DocsSidebar slug={slug} />
        </aside>
        {mobileOpen ? (
          <button
            type="button"
            aria-label="Close overlay"
            className="fixed inset-0 z-20 bg-[var(--trim-overlay)] lg:hidden"
            onClick={() => setMobileOpen(false)}
          />
        ) : null}

        <div className="min-w-0 flex-1 px-4 py-8 sm:px-8 lg:px-12 xl:px-16">{children}</div>
      </div>
    </div>
  );
}
