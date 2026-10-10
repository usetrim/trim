"use client";

import { adjacentDocs, docHref } from "@/lib/docs/nav";
import { ArrowLeft, ArrowRight } from "lucide-react";
import Link from "next/link";

export function DocsPager({ slug }: { slug: string }) {
  const { prev, next } = adjacentDocs(slug);
  if (!prev && !next) return null;

  return (
    <div className="mt-14 grid gap-3 border-t border-[var(--trim-border)] pt-8 sm:grid-cols-2">
      {prev ? (
        <Link
          href={docHref(prev.slug)}
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
          href={docHref(next.slug)}
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
