"use client";

import Link from "next/link";
import { useAuthProviders } from "@/hooks/queries/auth";

export default function ErrorPage({
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  const auth = useAuthProviders();
  const site = auth.data?.site;
  const heading = site?.page_error_heading?.trim() || "";
  const body = site?.page_error_body?.trim() || "";
  const retry = site?.page_error_retry_cta?.trim() || "";
  const homeCta = site?.page_error_home_cta?.trim() || "";
  const homeHref = site?.path_home?.trim() || "/";

  if (!heading || !body || !retry || !homeCta) {
    return null;
  }

  return (
    <main className="mx-auto flex min-h-[70vh] w-full max-w-lg flex-col items-center justify-center px-6 py-16 text-center">
      <h1 className="font-display text-3xl font-semibold tracking-tight text-[var(--trim-fg)] sm:text-4xl">
        {heading}
      </h1>
      <p className="mt-4 text-base leading-relaxed text-[var(--trim-muted)]">{body}</p>
      <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
        <button
          type="button"
          onClick={() => reset()}
          className="inline-flex h-10 items-center justify-center rounded-md bg-[var(--trim-ink)] px-4 text-sm font-medium text-[var(--trim-ink-inverse)] transition hover:opacity-90"
        >
          {retry}
        </button>
        <Link
          href={homeHref}
          className="inline-flex h-10 items-center justify-center rounded-md border border-[var(--trim-border-strong)] bg-[var(--trim-panel)] px-4 text-sm font-medium text-[var(--trim-fg)] transition hover:bg-[var(--trim-hover)]"
        >
          {homeCta}
        </Link>
      </div>
    </main>
  );
}
