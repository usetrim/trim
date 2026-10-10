"use client";

import { PublicAuthChrome } from "@/components/auth/user-menu";
import { TrimWordmark, TrimWordmarkLink } from "@/components/brand/trim-wordmark";
import { DocsShell } from "@/components/docs/docs-shell";
import { LandingHeroCopy } from "@/components/landing/landing-hero";
import { LandingPricing } from "@/components/landing/landing-pricing";
import { ContactPageContent } from "@/components/site/contact-page";
import { LegalPageShell, LegalRightToc, LegalSidebar } from "@/components/site/legal-page-shell";
import {
  DocsPageSkeleton,
  LandingPageSkeleton,
  type LandingSkeletonMode,
} from "@/components/skeletons/page-skeletons";
import { ThemeToggle } from "@/components/theme/theme-toggle";
import { useAuthProviders } from "@/hooks/queries/auth";
import { apiFetch } from "@/lib/api/client";
import { allowedAuthProvidersFromEnv } from "@/lib/auth-providers";
import { parseLegalPath } from "@/lib/legal/nav";
import { qk } from "@/lib/query-keys";
import { landingSkeletonChrome } from "@/lib/skeleton-chrome";
import { cn } from "@/lib/utils";
import type { AuthProvidersSiteChrome } from "@/types/auth";
import type { PlansResponse } from "@/types/billing";
import { useQueryClient } from "@tanstack/react-query";
import { Menu, X } from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { type MouseEvent, type ReactNode, useCallback, useEffect, useState } from "react";

export const PRICING_PATH = "/pricing";

type Mode = "home" | "pricing" | "login" | "legal" | "docs" | "contact" | "other";

function pathMode(pathname: string, loginPath: string, contactPath: string): Mode {
  if (pathname === "/" || pathname === "") return "home";
  if (pathname === PRICING_PATH || pathname.startsWith(`${PRICING_PATH}/`)) return "pricing";
  if (pathname === "/docs" || pathname.startsWith("/docs/")) return "docs";
  if (pathname === "/privacy" || pathname.startsWith("/privacy/")) return "legal";
  if (pathname === "/terms" || pathname.startsWith("/terms/")) return "legal";
  if (contactPath && (pathname === contactPath || pathname.startsWith(`${contactPath}/`))) {
    return "contact";
  }
  if (pathname === "/contact" || pathname.startsWith("/contact/")) return "contact";
  if (loginPath && (pathname === loginPath || pathname.startsWith(`${loginPath}/`))) return "login";
  // Before chrome loads, loginPath may be empty; still match the public login route.
  if (pathname === "/login" || pathname.startsWith("/login/")) return "login";
  return "other";
}

/** Soft-nav left rail: keep all copy variants mounted; only visibility swaps. */
function LeftCopy({
  mode,
  site,
  loginTitle,
}: {
  mode: Mode;
  site: AuthProvidersSiteChrome;
  loginTitle: string;
}) {
  const show = (m: Mode) => (mode === m ? undefined : "hidden");

  return (
    <>
      <div className={show("home")} aria-hidden={mode !== "home"}>
        <LandingHeroCopy site={site} align="left" />
      </div>

      <div className={show("pricing")} aria-hidden={mode !== "pricing"}>
        <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-[var(--trim-muted)]">
          {site.nav_pricing || "Pricing"}
        </p>
        <h1 className="font-display mt-4 max-w-[22rem] text-[1.5rem] font-medium leading-[1.2] tracking-[-0.025em] text-[var(--trim-fg)] xl:text-[1.75rem]">
          {site.pricing_title}
        </h1>
        {site.pricing_subtitle ? (
          <p className="mt-4 max-w-[22rem] text-[14px] leading-relaxed text-[var(--trim-muted)] xl:text-[15px]">
            {site.pricing_subtitle}
          </p>
        ) : null}
        {site.nav_sign_in_href ? (
          <Link
            href={site.nav_sign_in_href}
            className="mt-8 inline-flex h-9 items-center whitespace-nowrap rounded-md bg-[var(--trim-ink)] px-5 text-[13px] font-medium text-[var(--trim-ink-inverse)] transition hover:opacity-90"
          >
            {site.cta_start}
            <span className="ml-1.5 text-[var(--trim-ink-inverse)] opacity-70">→</span>
          </Link>
        ) : null}
      </div>

      <div className={show("login")} aria-hidden={mode !== "login"}>
        <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-[var(--trim-muted)]">
          {site.nav_sign_in}
        </p>
        <h1 className="font-display mt-4 max-w-[22rem] text-[1.5rem] font-medium leading-[1.2] tracking-[-0.025em] text-[var(--trim-fg)] xl:text-[1.75rem]">
          {loginTitle || site.nav_sign_in}
        </h1>
        <p className="mt-4 max-w-[22rem] text-[14px] leading-relaxed text-[var(--trim-muted)] xl:text-[15px]">
          {site.tagline}
        </p>
        <Link
          href={site.path_home || "/"}
          className="mt-8 inline-flex h-9 items-center gap-2 whitespace-nowrap rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel)] px-4 text-[13px] font-medium text-[var(--trim-fg)] shadow-[var(--trim-card-shadow)] transition hover:border-[var(--trim-border-strong)] hover:bg-[var(--trim-hover)]"
        >
          <span aria-hidden>←</span>
          <TrimWordmark size="sm" alt={site.brand || "Trim"} />
        </Link>
      </div>

      <div className={show("contact")} aria-hidden={mode !== "contact"}>
        <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-[var(--trim-muted)]">
          {site.nav_contact}
        </p>
        <h1 className="font-display mt-4 max-w-[22rem] text-[1.5rem] font-medium leading-[1.2] tracking-[-0.025em] text-[var(--trim-fg)] xl:text-[1.75rem]">
          {site.contact_page_heading}
        </h1>
        {site.contact_page_body ? (
          <p className="mt-4 max-w-[22rem] text-[14px] leading-relaxed text-[var(--trim-muted)] xl:text-[15px]">
            {site.contact_page_body}
          </p>
        ) : null}
      </div>

      <div className={show("legal")} aria-hidden={mode !== "legal"}>
        <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-[var(--trim-muted)]">
          Legal
        </p>
        <h1 className="font-display mt-4 max-w-[22rem] text-[1.5rem] font-medium leading-[1.2] tracking-[-0.025em] text-[var(--trim-fg)] xl:text-[1.75rem]">
          Policies that keep Trim clear for developers and teams
        </h1>
        <p className="mt-4 max-w-[22rem] text-[14px] leading-relaxed text-[var(--trim-muted)] xl:text-[15px]">
          Privacy and terms for the hosted Service. Read them in full in the main column.
        </p>
        <div className="mt-8 flex flex-wrap gap-3 text-[13px]">
          {site.path_privacy ? (
            <Link
              href={site.path_privacy}
              scroll={false}
              className="underline underline-offset-4 hover:text-[var(--trim-fg)]"
            >
              {site.privacy_title || "Privacy"}
            </Link>
          ) : null}
          {site.path_terms ? (
            <Link
              href={site.path_terms}
              scroll={false}
              className="underline underline-offset-4 hover:text-[var(--trim-fg)]"
            >
              {site.terms_title || "Terms"}
            </Link>
          ) : null}
        </div>
      </div>

      {/* Fallback for unexpected routes while still under the marketing shell. */}
      <div
        className={
          mode === "home" ||
          mode === "pricing" ||
          mode === "login" ||
          mode === "legal" ||
          mode === "docs" ||
          mode === "contact"
            ? "hidden"
            : undefined
        }
        aria-hidden={
          mode === "home" ||
          mode === "pricing" ||
          mode === "login" ||
          mode === "legal" ||
          mode === "docs" ||
          mode === "contact"
        }
      >
        <LandingHeroCopy site={site} align="left" />
      </div>
    </>
  );
}

export function LandingShell({ children }: { children: ReactNode }) {
  const chrome = useAuthProviders();
  const site = chrome.data?.site;
  const pathname = usePathname();
  const router = useRouter();
  const queryClient = useQueryClient();
  const [legalNavOpen, setLegalNavOpen] = useState(false);
  /** Soft-nav keep-alive: mount each heavy panel on first visit, then keep mounted (hidden). */
  const [mountedPanels, setMountedPanels] = useState({
    pricing: false,
    contact: false,
    legal: false,
    docs: false,
  });

  const loginPath = (site?.nav_sign_in_href || site?.path_login || "").trim();
  const contactPath = (site?.path_contact || "").trim() || "/contact";
  const mode = pathMode(pathname, loginPath, contactPath);
  const homePath = site?.path_home || "/";
  const legalRoute = parseLegalPath(pathname);
  const legalKind = legalRoute?.kind ?? "privacy";

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setLegalNavOpen(false);
  }, [pathname]);

  useEffect(() => {
    if (mode !== "pricing" && mode !== "contact" && mode !== "legal" && mode !== "docs") return;
    setMountedPanels((prev) => (prev[mode] ? prev : { ...prev, [mode]: true }));
  }, [mode]);

  const goHow = useCallback(
    (e: MouseEvent<HTMLButtonElement>) => {
      e.preventDefault();
      if (mode === "home") {
        document.getElementById("how")?.scrollIntoView({ behavior: "smooth" });
        window.history.replaceState(null, "", "/#how");
        return;
      }
      try {
        sessionStorage.setItem("landing-scroll", "how");
      } catch {
        /* ignore */
      }
      router.push("/");
    },
    [mode, router],
  );

  // Soft-nav: warm pricing data before the user opens /pricing (no card skeleton flash).
  useEffect(() => {
    const prefetchPlans = (interval: "monthly" | "annual") => {
      void queryClient.prefetchQuery({
        queryKey: [...qk.plans, interval, "anon"],
        queryFn: () =>
          apiFetch<PlansResponse>(`/api/v1/public/plans?interval=${encodeURIComponent(interval)}`),
        staleTime: 60_000,
      });
    };
    prefetchPlans("monthly");
    prefetchPlans("annual");
  }, [queryClient]);

  useEffect(() => {
    if (site?.path_privacy) router.prefetch(site.path_privacy);
    if (site?.path_terms) router.prefetch(site.path_terms);
    if (site?.path_contact) router.prefetch(site.path_contact);
    router.prefetch(PRICING_PATH);
    router.prefetch("/docs");
  }, [router, site?.path_privacy, site?.path_terms, site?.path_contact]);

  // Keep sticky shell visible while using cached chrome; only block on first load.
  if (chrome.isPending && !site) {
    const pendingMode = pathMode(pathname, loginPath, "/contact");
    if (pendingMode === "docs") {
      return <DocsPageSkeleton authSlot="pending" />;
    }
    const skeletonMode: LandingSkeletonMode =
      pendingMode === "pricing"
        ? "pricing"
        : pendingMode === "login"
          ? "login"
          : pendingMode === "legal"
            ? "legal"
            : pendingMode === "contact"
              ? "contact"
              : "home";
    return (
      <LandingPageSkeleton
        chrome={landingSkeletonChrome(site)}
        mode={skeletonMode}
        providerSlots={allowedAuthProvidersFromEnv().length}
        pricingCardCount={4}
        authSlot="pending"
      />
    );
  }

  if (chrome.isError || !site) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-[var(--trim-bg)] px-6">
        <div className="w-full max-w-md border border-[var(--trim-border)] bg-[var(--trim-panel)] p-8">
          <p className="text-sm text-destructive">
            {chrome.data?.login_failed_message || chrome.data?.providers_empty_message || ""}
          </p>
        </div>
      </main>
    );
  }

  const year = new Date().getFullYear();
  const copyright = (site.copyright_fmt || "").includes("%d")
    ? (site.copyright_fmt || "").replace("%d", String(year))
    : site.copyright_fmt || "";

  const navLinkClass = (active: boolean) =>
    cn(
      "rounded-md px-2 py-1 font-mono text-[11px] transition",
      active
        ? "bg-[var(--trim-hover)] font-medium text-[var(--trim-fg)]"
        : "text-[var(--trim-muted)] hover:bg-[var(--trim-hover)] hover:text-[var(--trim-fg)]",
    );

  const footerLinks = (
    <>
      <Link href="/docs" scroll={false} className="hover:text-[var(--trim-fg)]">
        Docs
      </Link>
      {site.nav_contact && contactPath ? (
        <Link href={contactPath} scroll={false} className="hover:text-[var(--trim-fg)]">
          {site.nav_contact}
        </Link>
      ) : null}
      {site.path_privacy ? (
        <Link href={site.path_privacy} scroll={false} className="hover:text-[var(--trim-fg)]">
          {site.footer_privacy}
        </Link>
      ) : null}
      {site.path_terms ? (
        <Link href={site.path_terms} scroll={false} className="hover:text-[var(--trim-fg)]">
          {site.footer_terms}
        </Link>
      ) : null}
      <a href={site.source_url} className="hover:text-[var(--trim-fg)]" rel="noreferrer">
        {site.footer_github}
      </a>
      {site.nav_pricing ? (
        <Link href={PRICING_PATH} scroll={false} className="hover:text-[var(--trim-fg)]">
          {site.nav_pricing}
        </Link>
      ) : null}
    </>
  );

  const loginTitle = chrome.data?.login_title || "";
  // Legal docs need a wide reading column; keep shell chrome but drop the left rail.
  const fullBleed = mode === "legal";
  const showMarketing = mode !== "docs";

  return (
    <>
      {/* Soft-nav: docs under the same LandingShell parent as pricing - no layout remount. */}
      {mountedPanels.docs || mode === "docs" ? (
        <div className={mode === "docs" ? undefined : "hidden"} aria-hidden={mode !== "docs"}>
          <DocsShell>{mode === "docs" ? children : null}</DocsShell>
        </div>
      ) : null}

      <main
        className={cn(
          "min-h-screen bg-[var(--trim-bg)] text-[var(--trim-fg)]",
          showMarketing ? undefined : "hidden",
        )}
        aria-hidden={!showMarketing}
      >
        <header
          className={cn(
            "flex items-center justify-between gap-3 border-b border-[var(--trim-border)] bg-[var(--trim-panel)] px-5 py-3",
            fullBleed
              ? "sticky top-0 z-40 h-14 bg-[var(--trim-panel)]/95 px-4 backdrop-blur sm:px-6 lg:px-10"
              : "sticky top-0 z-40 bg-[var(--trim-panel)]/95 backdrop-blur supports-[backdrop-filter]:bg-[var(--trim-panel)]/80 lg:hidden",
          )}
        >
          <TrimWordmarkLink href={homePath} size="md" alt={site.brand || "Trim"} priority />
          <div className="flex min-w-0 flex-1 items-center justify-end gap-1.5 overflow-x-auto sm:gap-2">
            <Link
              href="/docs"
              scroll={false}
              className={cn(navLinkClass(pathname.startsWith("/docs")), "shrink-0")}
            >
              Docs
            </Link>
            {site.nav_pricing ? (
              <Link
                href={PRICING_PATH}
                scroll={false}
                className={cn(navLinkClass(mode === "pricing"), "shrink-0")}
              >
                {site.nav_pricing}
              </Link>
            ) : null}
            {site.nav_contact && contactPath ? (
              <Link
                href={contactPath}
                scroll={false}
                className={cn(navLinkClass(mode === "contact"), "shrink-0")}
              >
                {site.nav_contact}
              </Link>
            ) : null}
            {site.path_privacy ? (
              <Link
                href={site.path_privacy}
                scroll={false}
                className={cn(
                  navLinkClass(pathname.startsWith("/privacy")),
                  "hidden shrink-0 sm:inline-flex",
                )}
              >
                {site.footer_privacy || "Privacy"}
              </Link>
            ) : null}
            {site.path_terms ? (
              <Link
                href={site.path_terms}
                scroll={false}
                className={cn(
                  navLinkClass(pathname.startsWith("/terms")),
                  "hidden shrink-0 sm:inline-flex",
                )}
              >
                {site.footer_terms || "Terms"}
              </Link>
            ) : null}
            <PublicAuthChrome
              site={site}
              compact
              signInClassName="inline-flex h-8 shrink-0 items-center whitespace-nowrap rounded-md bg-[var(--trim-ink)] px-3 text-[12px] font-medium text-[var(--trim-ink-inverse)]"
            />
            <ThemeToggle />
          </div>
        </header>

        <div className={cn("mx-auto flex w-full", fullBleed ? "max-w-none" : "max-w-[1850px]")}>
          {/* Soft-nav: keep marketing left rail mounted; only hide on legal full-bleed. */}
          <aside
            className={cn(
              "relative w-[min(28vw,430px)] shrink-0 border-r border-[var(--trim-border)] bg-[var(--trim-panel)] xl:w-[450px]",
              fullBleed ? "hidden" : "hidden lg:block",
            )}
            aria-hidden={fullBleed}
          >
            <div className="sticky top-0 flex h-screen flex-col px-8 py-7 xl:px-10">
              <div className="flex items-center justify-between gap-3">
                <TrimWordmarkLink href={homePath} size="md" alt={site.brand || "Trim"} priority />
                <nav className="flex items-center gap-0.5">
                  <Link
                    href="/docs"
                    scroll={false}
                    className={navLinkClass(pathname.startsWith("/docs"))}
                  >
                    Docs
                  </Link>
                  {site.nav_how ? (
                    <button type="button" onClick={goHow} className={navLinkClass(false)}>
                      {site.nav_how}
                    </button>
                  ) : null}
                  {site.nav_pricing ? (
                    <Link
                      href={PRICING_PATH}
                      scroll={false}
                      className={navLinkClass(mode === "pricing")}
                    >
                      {site.nav_pricing}
                    </Link>
                  ) : null}
                  {site.nav_contact && contactPath ? (
                    <Link
                      href={contactPath}
                      scroll={false}
                      className={navLinkClass(mode === "contact")}
                    >
                      {site.nav_contact}
                    </Link>
                  ) : null}
                  <PublicAuthChrome
                    site={site}
                    compact
                    signInClassName={cn(
                      "ml-1 inline-flex h-7 shrink-0 items-center whitespace-nowrap rounded-md px-3 text-[12px] font-medium transition",
                      mode === "login"
                        ? "border border-[var(--trim-border-strong)] bg-[var(--trim-panel)] text-[var(--trim-fg)] shadow-[var(--trim-card-shadow)]"
                        : "bg-[var(--trim-ink)] text-[var(--trim-ink-inverse)] hover:opacity-90",
                    )}
                  />
                </nav>
              </div>

              <div className="flex flex-1 flex-col justify-center py-10">
                <LeftCopy mode={mode} site={site} loginTitle={loginTitle} />
              </div>

              <footer className="border-t border-[var(--trim-border)] pt-5 text-[12px] text-[var(--trim-muted)]">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="text-[var(--trim-muted)]">{copyright}</p>
                    <div className="mt-3 flex flex-wrap gap-x-4 gap-y-2">{footerLinks}</div>
                  </div>
                  <ThemeToggle className="mt-0.5" />
                </div>
              </footer>
            </div>
          </aside>

          <div className="min-w-0 flex-1">
            <div
              className={cn(
                "border-b border-[var(--trim-border)] px-5 py-10 lg:hidden",
                mode === "home" ? undefined : "hidden",
              )}
              aria-hidden={mode !== "home"}
            >
              <LandingHeroCopy site={site} align="center" />
            </div>

            {/* Soft-nav: keep legal mounted after first visit; only visibility swaps. */}
            {mountedPanels.legal || mode === "legal" ? (
              <div
                className={mode === "legal" ? undefined : "hidden"}
                aria-hidden={mode !== "legal"}
              >
                <div className="mx-auto flex w-full max-w-none items-start">
                  <aside
                    className={cn(
                      "fixed inset-y-0 left-0 z-30 w-[min(86vw,300px)] overflow-y-auto border-r border-[var(--trim-border)] bg-[var(--trim-panel)] px-3 pb-8 pt-[4.25rem] transition-transform lg:sticky lg:top-14 lg:z-0 lg:block lg:h-[calc(100vh-3.5rem)] lg:w-[260px] lg:shrink-0 lg:translate-x-0 lg:self-start lg:overflow-y-auto lg:pt-6",
                      legalNavOpen ? "translate-x-0" : "-translate-x-full lg:translate-x-0",
                    )}
                  >
                    <LegalSidebar
                      kind={legalKind}
                      privacyHref={site.path_privacy || "/privacy"}
                      termsHref={site.path_terms || "/terms"}
                      privacyLabel={site.privacy_title || site.footer_privacy || "Privacy"}
                      termsLabel={site.terms_title || site.footer_terms || "Terms"}
                    />
                  </aside>
                  {legalNavOpen && mode === "legal" ? (
                    <button
                      type="button"
                      aria-label="Close overlay"
                      className="fixed inset-0 z-20 bg-[var(--trim-overlay)] lg:hidden"
                      onClick={() => setLegalNavOpen(false)}
                    />
                  ) : null}
                  <div className="min-w-0 flex-1 px-4 py-8 sm:px-8 lg:px-10 xl:px-12">
                    <button
                      type="button"
                      className="mb-6 inline-flex h-8 w-8 items-center justify-center rounded-md border border-[var(--trim-border)] bg-[var(--trim-bg)] text-[var(--trim-fg)] transition hover:border-[var(--trim-border-strong)] hover:bg-[var(--trim-hover)] lg:hidden"
                      aria-label={legalNavOpen ? "Close menu" : "Open menu"}
                      onClick={() => setLegalNavOpen((v) => !v)}
                    >
                      {legalNavOpen ? <X className="h-4 w-4" /> : <Menu className="h-4 w-4" />}
                    </button>
                    <div className={pathname.startsWith("/terms") ? "hidden" : undefined}>
                      <LegalPageShell kind="privacy" embedded hideLeftNav hideRightToc />
                    </div>
                    <div className={pathname.startsWith("/terms") ? undefined : "hidden"}>
                      <LegalPageShell kind="terms" embedded hideLeftNav hideRightToc />
                    </div>
                  </div>
                  <aside className="hidden w-[220px] shrink-0 self-start xl:sticky xl:top-24 xl:block xl:max-h-[calc(100vh-7rem)] xl:overflow-y-auto xl:py-8 xl:pr-6 2xl:pr-10">
                    <div className={pathname.startsWith("/terms") ? "hidden" : undefined}>
                      <LegalRightToc kind="privacy" />
                    </div>
                    <div className={pathname.startsWith("/terms") ? undefined : "hidden"}>
                      <LegalRightToc kind="terms" />
                    </div>
                  </aside>
                </div>
              </div>
            ) : null}

            {/* Soft-nav: keep pricing mounted after first visit so Monthly/Annual never remount. */}
            {mountedPanels.pricing || mode === "pricing" ? (
              <div
                className={mode === "pricing" ? undefined : "hidden"}
                aria-hidden={mode !== "pricing"}
              >
                <div className="min-h-[70vh]">
                  <LandingPricing
                    site={site}
                    signInHref={site.nav_sign_in_href || ""}
                    sidebarChrome
                  />
                </div>
              </div>
            ) : null}

            {/* Soft-nav: keep contact mounted after first visit (same pattern as pricing). */}
            {mountedPanels.contact || mode === "contact" ? (
              <div
                className={mode === "contact" ? undefined : "hidden"}
                aria-hidden={mode !== "contact"}
              >
                <div className="min-h-[70vh]">
                  <ContactPageContent />
                </div>
              </div>
            ) : null}

            <div
              className={
                mode !== "legal" && mode !== "pricing" && mode !== "docs" && mode !== "contact"
                  ? undefined
                  : "hidden"
              }
              aria-hidden={
                mode === "legal" || mode === "pricing" || mode === "docs" || mode === "contact"
              }
            >
              {mode === "docs" ? null : children}
            </div>

            <footer
              className={cn(
                "flex flex-wrap items-center gap-4 border-t border-[var(--trim-border)] px-5 py-8 text-sm text-[var(--trim-muted)] lg:hidden",
                (mode === "legal" || mode === "docs") && "hidden",
              )}
              aria-hidden={mode === "legal" || mode === "docs"}
            >
              <span className="mr-auto">{copyright}</span>
              {footerLinks}
              <ThemeToggle />
            </footer>
          </div>
        </div>
      </main>
    </>
  );
}

export function useLandingSite(): AuthProvidersSiteChrome | null {
  const chrome = useAuthProviders();
  return chrome.data?.site ?? null;
}
