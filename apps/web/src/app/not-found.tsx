import Link from "next/link";
import type { Metadata } from "next";
import { TrimWordmark } from "@/components/brand/trim-wordmark";
import { loadSiteChrome } from "@/lib/site-metadata";

export async function generateMetadata(): Promise<Metadata> {
  const site = await loadSiteChrome();
  const title = site?.page_404_title?.trim() || "";
  const robotsRaw = site?.seo_robots_noindex?.trim() || "";
  if (!title) return {};
  return {
    title,
    description: site?.page_404_body?.trim() || undefined,
    robots: robotsRaw.toLowerCase().includes("noindex")
      ? { index: false, follow: false }
      : undefined,
  };
}

export default async function NotFound() {
  const site = await loadSiteChrome();
  const code = site?.page_404_code?.trim() || "";
  const heading = site?.page_404_heading?.trim() || "";
  const body = site?.page_404_body?.trim() || "";
  const homeCta = site?.page_404_home_cta?.trim() || "";
  const homeHref = site?.path_home?.trim() || "/";
  const docsCta = site?.page_404_docs_cta?.trim() || "";
  const docsHref = site?.page_404_docs_href?.trim() || "";
  const brand = site?.brand?.trim() || "";

  if (!heading || !body || !homeCta) {
    return null;
  }

  return (
    <main className="mx-auto flex min-h-[70vh] w-full max-w-lg flex-col items-center justify-center px-6 py-16 text-center">
      {code ? (
        <p className="font-mono text-sm font-semibold tracking-[0.2em] text-[var(--trim-subtle)]">
          {code}
        </p>
      ) : null}
      {brand ? (
        <div className="mt-4 flex justify-center">
          <TrimWordmark size="lg" alt={brand} />
        </div>
      ) : null}
      <h1 className="font-display mt-4 text-3xl font-semibold tracking-tight text-[var(--trim-fg)] sm:text-4xl">
        {heading}
      </h1>
      <p className="mt-4 text-base leading-relaxed text-[var(--trim-muted)]">{body}</p>
      <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
        <Link
          href={homeHref}
          className="inline-flex h-10 items-center justify-center rounded-md bg-[var(--trim-ink)] px-4 text-sm font-medium text-[var(--trim-ink-inverse)] transition hover:opacity-90"
        >
          {homeCta}
        </Link>
        {docsCta && docsHref ? (
          <Link
            href={docsHref}
            className="inline-flex h-10 items-center justify-center rounded-md border border-[var(--trim-border-strong)] bg-[var(--trim-panel)] px-4 text-sm font-medium text-[var(--trim-fg)] transition hover:bg-[var(--trim-hover)]"
          >
            {docsCta}
          </Link>
        ) : null}
      </div>
    </main>
  );
}
