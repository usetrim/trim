"use client";

import { PublicAuthChrome } from "@/components/auth/user-menu";
import { TrimWordmark, TrimWordmarkLink } from "@/components/brand/trim-wordmark";
import { LegalPageHeaderSkeleton } from "@/components/skeletons/page-skeletons";
import { ThemeToggle } from "@/components/theme/theme-toggle";
import { useAuthProviders } from "@/hooks/queries/auth";
import Link from "next/link";

export function SiteNavHeader() {
  const chrome = useAuthProviders();
  const site = chrome.data?.site;

  if (chrome.isLoading && !site) {
    return <LegalPageHeaderSkeleton authSlot="pending" />;
  }

  if (chrome.isError || !site) {
    return (
      <header className="border-b border-[var(--trim-border)] bg-[var(--trim-panel)]">
        <div className="mx-auto flex w-full max-w-[1400px] items-center justify-between px-6 py-5">
          <span className="text-sm text-destructive">
            {chrome.data?.login_failed_message || ""}
          </span>
          <ThemeToggle />
        </div>
      </header>
    );
  }

  const homeHref =
    site.path_home?.startsWith("/") && !site.path_home.startsWith("//") ? site.path_home : "";

  return (
    <header className="border-b border-[var(--trim-border)] bg-[var(--trim-panel)]">
      <div className="mx-auto flex w-full max-w-[1400px] items-center justify-between px-6 py-5">
        {homeHref ? (
          <TrimWordmarkLink href={homeHref} size="lg" alt={site.brand || "Trim"} priority />
        ) : (
          <TrimWordmark size="lg" alt={site.brand || "Trim"} priority />
        )}
        <div className="flex items-center gap-3">
          <Link
            href="/docs"
            className="text-sm text-[var(--trim-muted)] transition hover:text-[var(--trim-fg)]"
          >
            Docs
          </Link>
          {site.nav_contact && site.path_contact ? (
            <Link
              href={site.path_contact}
              className="text-sm text-[var(--trim-muted)] transition hover:text-[var(--trim-fg)]"
            >
              {site.nav_contact}
            </Link>
          ) : null}
          <PublicAuthChrome
            site={site}
            signInClassName="inline-flex h-9 items-center whitespace-nowrap rounded-md bg-[var(--trim-ink)] px-4 text-sm font-medium text-[var(--trim-ink-inverse)]"
          />
          <ThemeToggle />
        </div>
      </div>
    </header>
  );
}
