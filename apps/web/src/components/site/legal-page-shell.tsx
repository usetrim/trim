"use client";

import { TrimWordmarkLink } from "@/components/brand/trim-wordmark";
import { DocsToc } from "@/components/docs/docs-toc";
import { SiteNavHeader } from "@/components/site/site-nav-header";
import { ThemeToggle } from "@/components/theme/theme-toggle";
import { LegalArticleSkeleton } from "@/components/skeletons/page-skeletons";
import { useAuthProviders } from "@/hooks/queries/auth";
import {
  buildLegalNav,
  findLegalSectionGroupId,
  legalSectionId,
  type LegalKind,
} from "@/lib/legal/nav";
import { cn } from "@/lib/utils";
import type { LegalSectionChrome } from "@/types/auth";
import { ArrowLeft, ArrowRight, ChevronRight, Menu, X } from "lucide-react";
import Link from "next/link";
import { useEffect, useMemo, useState } from "react";

function LegalSectionBlock({
  section,
  id,
}: {
  section: LegalSectionChrome;
  id: string;
}) {
  const hasContact = Boolean(section.contact_email);
  return (
    <section id={id} className="scroll-mt-24 space-y-3">
      {section.heading ? (
        <h2 className="pt-2 text-lg font-medium text-[var(--trim-fg)] sm:text-xl">
          {section.heading}
        </h2>
      ) : null}
      {section.body ? (
        <p className="text-[15px] leading-relaxed text-[var(--trim-muted)] sm:text-base">
          {section.body}
        </p>
      ) : null}
      {hasContact ? (
        <p className="text-[15px] leading-relaxed text-[var(--trim-muted)] sm:text-base">
          {section.contact_lead ? `${section.contact_lead} ` : null}
          <a
            className="text-[var(--trim-fg)] underline underline-offset-4"
            href={`mailto:${section.contact_email}`}
          >
            {section.contact_email}
          </a>
          {section.contact_trail ? ` ${section.contact_trail}` : null}
        </p>
      ) : null}
    </section>
  );
}

/** Docs-style Prev / Next between Privacy and Terms. */
function LegalPager({
  kind,
  privacyHref,
  termsHref,
  privacyLabel,
  termsLabel,
}: {
  kind: LegalKind;
  privacyHref: string;
  termsHref: string;
  privacyLabel: string;
  termsLabel: string;
}) {
  const prev =
    kind === "terms" ? { href: privacyHref, title: privacyLabel || "Privacy Policy" } : null;
  const next =
    kind === "privacy" ? { href: termsHref, title: termsLabel || "Terms of Service" } : null;

  if (!prev && !next) return null;

  return (
    <div className="mt-14 grid gap-3 border-t border-[var(--trim-border)] pt-8 sm:grid-cols-2">
      {prev ? (
        <Link
          href={prev.href}
          scroll={false}
          className="group flex flex-col rounded-lg border border-[var(--trim-border)] bg-[var(--trim-panel)] px-4 py-3 shadow-[var(--trim-card-shadow)] transition hover:border-[var(--trim-border-strong)] hover:bg-[var(--trim-hover)]"
        >
          <span className="inline-flex items-center gap-1 text-[11px] font-medium uppercase tracking-wide text-[var(--trim-muted)]">
            <ArrowLeft className="h-3 w-3" /> Previous
          </span>
          <span className="mt-1 text-sm font-semibold text-[var(--trim-fg)] group-hover:underline">
            {prev.title}
          </span>
        </Link>
      ) : (
        <div />
      )}
      {next ? (
        <Link
          href={next.href}
          scroll={false}
          className="group flex flex-col items-end rounded-lg border border-[var(--trim-border)] bg-[var(--trim-panel)] px-4 py-3 text-right shadow-[var(--trim-card-shadow)] transition hover:border-[var(--trim-border-strong)] hover:bg-[var(--trim-hover)]"
        >
          <span className="inline-flex items-center gap-1 text-[11px] font-medium uppercase tracking-wide text-[var(--trim-muted)]">
            Next <ArrowRight className="h-3 w-3" />
          </span>
          <span className="mt-1 text-sm font-semibold text-[var(--trim-fg)] group-hover:underline">
            {next.title}
          </span>
        </Link>
      ) : null}
    </div>
  );
}

/**
 * Docs-style exclusive accordion: Policies switcher + categorized in-page sections.
 */
export function LegalSidebar({
  kind,
  privacyHref,
  termsHref,
  privacyLabel,
  termsLabel,
}: {
  kind: LegalKind;
  privacyHref: string;
  termsHref: string;
  privacyLabel: string;
  termsLabel: string;
}) {
  const providers = useAuthProviders();
  const sections =
    kind === "privacy" ? providers.data?.privacy_sections : providers.data?.terms_sections;
  const nav = useMemo(() => buildLegalNav(kind, sections ?? []), [kind, sections]);

  const [hashId, setHashId] = useState("");
  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    const read = () =>
      setHashId(typeof window !== "undefined" ? window.location.hash.replace(/^#/, "") : "");
    read();
    window.addEventListener("hashchange", read);
    return () => window.removeEventListener("hashchange", read);
  }, [kind]);

  const activeSectionId = useMemo(() => {
    if (hashId && nav.some((s) => s.items.some((i) => i.id === hashId))) return hashId;
    return nav[0]?.items[0]?.id || "";
  }, [hashId, nav]);

  const activeGroupId = findLegalSectionGroupId(nav, activeSectionId);
  const [openId, setOpenId] = useState(activeGroupId);

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setOpenId(activeGroupId);
  }, [kind, activeGroupId]);

  const policyItems = [
    { href: privacyHref, label: privacyLabel || "Privacy Policy", active: kind === "privacy" },
    { href: termsHref, label: termsLabel || "Terms of Service", active: kind === "terms" },
  ].filter((item) => item.href);

  const scrollToSection = (id: string) => {
    const el = document.getElementById(id);
    if (el) {
      el.scrollIntoView({ behavior: "smooth", block: "start" });
      window.history.replaceState(null, "", `#${id}`);
      setHashId(id);
    }
  };

  return (
    <nav className="space-y-0.5 pb-10" aria-label="Legal">
      <div className="border-b border-[var(--trim-border)]">
        <p className="px-2 py-2.5 text-[13px] font-medium text-[var(--trim-fg)]">Policies</p>
        <ul className="space-y-0.5 pb-2.5 pl-1">
          {policyItems.map((item) => (
            <li key={item.href}>
              <Link
                href={item.href}
                scroll={false}
                className={cn(
                  "block rounded-md px-2 py-1.5 text-[13px] transition",
                  item.active
                    ? "bg-[var(--trim-hover)] font-medium text-[var(--trim-fg)]"
                    : "text-[var(--trim-muted)] hover:bg-[var(--trim-hover)] hover:text-[var(--trim-fg)]",
                )}
              >
                {item.label}
              </Link>
            </li>
          ))}
        </ul>
      </div>

      {nav.map((section) => {
        const open = openId === section.id;
        return (
          <div key={section.id} className="border-b border-[var(--trim-border)]">
            <button
              type="button"
              aria-expanded={open}
              onClick={() => {
                setOpenId((current) => (current === section.id ? "" : section.id));
              }}
              className="flex w-full items-center justify-between gap-2 rounded-md px-2 py-2.5 text-left text-[13px] font-medium text-[var(--trim-fg)] transition hover:bg-[var(--trim-hover)]"
            >
              <span>{section.title}</span>
              <ChevronRight
                className={cn(
                  "h-3.5 w-3.5 shrink-0 text-[var(--trim-muted)] transition-transform duration-200 ease-out",
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
                <ul className="space-y-0.5 pb-2.5 pl-1">
                  {section.items.map((item) => {
                    const active = item.id === activeSectionId;
                    return (
                      <li key={item.id}>
                        <a
                          href={`#${item.id}`}
                          onClick={(event) => {
                            event.preventDefault();
                            setOpenId(section.id);
                            scrollToSection(item.id);
                          }}
                          className={cn(
                            "block rounded-md px-2 py-1.5 text-[13px] transition",
                            active
                              ? "bg-[var(--trim-hover)] font-medium text-[var(--trim-fg)]"
                              : "text-[var(--trim-muted)] hover:bg-[var(--trim-hover)] hover:text-[var(--trim-fg)]",
                          )}
                        >
                          {item.title}
                        </a>
                      </li>
                    );
                  })}
                </ul>
              </div>
            </div>
          </div>
        );
      })}

      <div className="px-2 pt-3">
        <Link
          href="/docs"
          className="block rounded-md px-2 py-1.5 text-[13px] text-[var(--trim-muted)] transition hover:bg-[var(--trim-hover)] hover:text-[var(--trim-fg)]"
        >
          Documentation
        </Link>
        <Link
          href="/"
          className="block rounded-md px-2 py-1.5 text-[13px] text-[var(--trim-muted)] transition hover:bg-[var(--trim-hover)] hover:text-[var(--trim-fg)]"
        >
          Home
        </Link>
      </div>
    </nav>
  );
}

export function LegalPageShell({
  kind,
  embedded = false,
  hideLeftNav = false,
  /** When true, omit right TOC (parent renders a sticky sibling column). */
  hideRightToc = false,
}: {
  kind: LegalKind;
  embedded?: boolean;
  hideLeftNav?: boolean;
  hideRightToc?: boolean;
}) {
  const providers = useAuthProviders();
  const site = providers.data?.site;
  const sections =
    kind === "privacy" ? providers.data?.privacy_sections : providers.data?.terms_sections;
  const sectionCount =
    kind === "privacy"
      ? providers.data?.privacy_section_count ?? providers.data?.privacy_sections?.length ?? 0
      : providers.data?.terms_section_count ?? providers.data?.terms_sections?.length ?? 0;
  const [mobileOpen, setMobileOpen] = useState(false);

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    setMobileOpen(false);
  }, [kind]);

  const tocItems = useMemo(() => {
    if (!sections) return [];
    return sections.map((section, index) => ({
      id: legalSectionId(kind, section.heading, index),
      title: section.heading?.trim() || "Introduction",
    }));
  }, [kind, sections]);

  if (!providers.data && providers.isLoading) {
    if (embedded) {
      return (
        <div className="w-full px-5 py-6 sm:px-8 xl:px-12">
          <LegalArticleSkeleton sectionCount={sectionCount} />
        </div>
      );
    }
    return (
      <main className="min-h-screen bg-[var(--trim-bg)] text-[var(--trim-fg)]">
        <SiteNavHeader />
        <LegalArticleSkeleton sectionCount={sectionCount} />
      </main>
    );
  }

  if (providers.isError || !site || !sections) {
    const err = (
      <div className="mx-auto w-full max-w-none px-5 pb-24 pt-6 sm:px-8 lg:px-12 xl:px-16 2xl:px-20">
        <p className="text-sm text-destructive">
          {providers.data?.login_failed_message || providers.data?.providers_empty_message || ""}
        </p>
      </div>
    );
    if (embedded) return err;
    return (
      <main className="min-h-screen bg-[var(--trim-bg)] text-[var(--trim-fg)]">
        <SiteNavHeader />
        {err}
      </main>
    );
  }

  const title = kind === "privacy" ? site.privacy_title : site.terms_title;
  const privacyHref = site.path_privacy || "/privacy";
  const termsHref = site.path_terms || "/terms";
  const privacyLabel = site.privacy_title || site.footer_privacy || "Privacy Policy";
  const termsLabel = site.terms_title || site.footer_terms || "Terms of Service";

  const articleInner = (
    <article className="min-w-0">
      <div className="xl:hidden">
        <DocsToc headings={tocItems} />
      </div>
      <p className="font-mono text-[11px] uppercase tracking-[0.18em] text-[var(--trim-subtle)]">
        Legal
      </p>
      <h1 className="font-display mt-3 text-3xl font-semibold tracking-tight text-[var(--trim-fg)] sm:text-4xl">
        {title}
      </h1>
      <p className="mt-3 max-w-3xl text-base leading-relaxed text-[var(--trim-muted)]">
        {site.updated_prefix} {site.updated_date}
      </p>
      <div className="mt-10 space-y-8">
        {sections.map((section, index) => (
          <LegalSectionBlock
            key={`${section.heading ?? "intro"}-${index}`}
            id={legalSectionId(kind, section.heading, index)}
            section={section}
          />
        ))}
      </div>
      <LegalPager
        kind={kind}
        privacyHref={privacyHref}
        termsHref={termsHref}
        privacyLabel={privacyLabel}
        termsLabel={termsLabel}
      />
    </article>
  );

  // Sticky must live on the aside itself (self-start + sticky). An inner sticky
  // inside a short self-start aside has no travel range and scrolls away.
  const article = hideRightToc ? (
    articleInner
  ) : (
    <div className="mx-auto grid w-full max-w-none gap-10 xl:grid-cols-[minmax(0,1fr)_220px]">
      {articleInner}
      <aside className="hidden max-h-[calc(100vh-7rem)] w-[220px] shrink-0 overflow-y-auto xl:sticky xl:top-24 xl:block xl:self-start">
        <DocsToc headings={tocItems} />
      </aside>
    </div>
  );

  if (embedded && hideLeftNav) return article;

  const withLeftNav = (
    <div className="mx-auto flex w-full max-w-none items-start">
      {!hideLeftNav ? (
        <>
          <aside
            className={cn(
              "fixed inset-y-0 left-0 z-30 w-[min(86vw,300px)] overflow-y-auto border-r border-[var(--trim-border)] bg-[var(--trim-panel)] px-3 pb-8 pt-[4.25rem] transition-transform lg:sticky lg:top-14 lg:z-0 lg:block lg:h-[calc(100vh-3.5rem)] lg:w-[260px] lg:shrink-0 lg:translate-x-0 lg:self-start lg:overflow-y-auto lg:pt-6",
              mobileOpen ? "translate-x-0" : "-translate-x-full lg:translate-x-0",
            )}
          >
            <LegalSidebar
              kind={kind}
              privacyHref={privacyHref}
              termsHref={termsHref}
              privacyLabel={privacyLabel}
              termsLabel={termsLabel}
            />
          </aside>
          {mobileOpen ? (
            <button
              type="button"
              aria-label="Close overlay"
              className="fixed inset-0 z-20 bg-[var(--trim-overlay)] lg:hidden"
              onClick={() => setMobileOpen(false)}
            />
          ) : null}
        </>
      ) : null}

      <div className="min-w-0 flex-1 px-4 py-8 sm:px-8 lg:px-12 xl:px-16">
        {!hideLeftNav ? (
          <button
            type="button"
            className="mb-6 inline-flex h-8 w-8 items-center justify-center rounded-md border border-[var(--trim-border)] bg-[var(--trim-bg)] text-[var(--trim-fg)] transition hover:border-[var(--trim-border-strong)] hover:bg-[var(--trim-hover)] lg:hidden"
            aria-label={mobileOpen ? "Close menu" : "Open menu"}
            onClick={() => setMobileOpen((v) => !v)}
          >
            {mobileOpen ? <X className="h-4 w-4" /> : <Menu className="h-4 w-4" />}
          </button>
        ) : null}
        {article}
      </div>
    </div>
  );

  if (embedded) return withLeftNav;

  return (
    <main className="min-h-screen bg-[var(--trim-bg)] text-[var(--trim-fg)]">
      <header className="sticky top-0 z-40 border-b border-[var(--trim-border)] bg-[var(--trim-panel)]/95 backdrop-blur">
        <div className="mx-auto flex h-14 w-full max-w-none items-center gap-3 px-4 sm:px-6 lg:px-10">
          <TrimWordmarkLink
            href={site.path_home || "/"}
            size="md"
            alt={site.brand || "Trim"}
            priority
          />
          <span className="hidden text-[var(--trim-muted)] sm:inline">/</span>
          <span className="hidden text-sm text-[var(--trim-muted)] sm:inline">Legal</span>
          <div className="ml-auto flex items-center gap-2">
            <Link
              href="/docs"
              className="hidden rounded-md px-2 py-1 text-[12px] text-[var(--trim-muted)] transition hover:text-[var(--trim-fg)] sm:inline"
            >
              Docs
            </Link>
            <ThemeToggle />
          </div>
        </div>
      </header>
      {withLeftNav}
    </main>
  );
}

/** Shared sticky right TOC for LandingShell three-column legal layout. */
export function LegalRightToc({ kind }: { kind: LegalKind }) {
  const providers = useAuthProviders();
  const sections =
    kind === "privacy" ? providers.data?.privacy_sections : providers.data?.terms_sections;
  const tocItems = useMemo(() => {
    if (!sections) return [];
    return sections.map((section, index) => ({
      id: legalSectionId(kind, section.heading, index),
      title: section.heading?.trim() || "Introduction",
    }));
  }, [kind, sections]);

  if (tocItems.length === 0) return null;

  return <DocsToc headings={tocItems} />;
}
