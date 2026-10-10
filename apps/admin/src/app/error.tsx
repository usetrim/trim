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
  const heading = site?.admin_page_error_heading?.trim() || "";
  const body = site?.admin_page_error_body?.trim() || "";
  const retry = site?.admin_page_error_retry_cta?.trim() || "";
  const homeCta = site?.admin_page_error_home_cta?.trim() || "";

  if (!heading || !body || !retry || !homeCta) {
    return null;
  }

  return (
    <main className="mx-auto flex min-h-[70vh] w-full max-w-lg flex-col items-center justify-center px-6 py-16 text-center">
      <h1 className="text-3xl font-semibold tracking-tight text-foreground sm:text-4xl">
        {heading}
      </h1>
      <p className="mt-4 text-base leading-relaxed text-muted-foreground">{body}</p>
      <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
        <button
          type="button"
          onClick={() => reset()}
          className="inline-flex h-10 items-center justify-center rounded-md bg-foreground px-4 text-sm font-medium text-background transition hover:opacity-90"
        >
          {retry}
        </button>
        <Link
          href="/"
          className="inline-flex h-10 items-center justify-center rounded-md border border-border bg-background px-4 text-sm font-medium text-foreground transition hover:bg-accent"
        >
          {homeCta}
        </Link>
      </div>
    </main>
  );
}
