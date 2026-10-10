import Link from "next/link";
import type { Metadata } from "next";
import { loadAdminSiteChrome } from "@/lib/site-metadata";

export async function generateMetadata(): Promise<Metadata> {
  const site = await loadAdminSiteChrome();
  const title = site?.admin_page_404_title?.trim() || "";
  if (!title) return { robots: { index: false, follow: false } };
  return {
    title,
    description: site?.admin_page_404_body?.trim() || undefined,
    robots: { index: false, follow: false },
  };
}

export default async function NotFound() {
  const site = await loadAdminSiteChrome();
  const heading = site?.admin_page_404_heading?.trim() || "";
  const body = site?.admin_page_404_body?.trim() || "";
  const homeCta = site?.admin_page_404_home_cta?.trim() || "";

  if (!heading || !body || !homeCta) {
    return null;
  }

  return (
    <main className="mx-auto flex min-h-[70vh] w-full max-w-lg flex-col items-center justify-center px-6 py-16 text-center">
      <p className="font-mono text-sm font-semibold tracking-[0.2em] text-muted-foreground">404</p>
      <h1 className="mt-4 text-3xl font-semibold tracking-tight text-foreground sm:text-4xl">
        {heading}
      </h1>
      <p className="mt-4 text-base leading-relaxed text-muted-foreground">{body}</p>
      <Link
        href="/"
        className="mt-8 inline-flex h-10 items-center justify-center rounded-md bg-foreground px-4 text-sm font-medium text-background transition hover:opacity-90"
      >
        {homeCta}
      </Link>
    </main>
  );
}
