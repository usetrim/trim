"use client";

import { cn } from "@/lib/utils";
import { ChevronDown } from "lucide-react";
import { useEffect, useState } from "react";

export function DocsToc({
  headings,
}: {
  headings: { id: string; title: string }[];
}) {
  const [active, setActive] = useState(headings[0]?.id || "");
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    if (headings.length === 0) return;
    const els = headings
      .map((h) => document.getElementById(h.id))
      .filter((el): el is HTMLElement => Boolean(el));
    if (els.length === 0) return;

    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((e) => e.isIntersecting)
          .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top);
        if (visible[0]?.target?.id) setActive(visible[0].target.id);
      },
      { rootMargin: "-20% 0px -65% 0px", threshold: [0, 1] },
    );
    for (const el of els) observer.observe(el);
    return () => observer.disconnect();
  }, [headings]);

  if (headings.length === 0) return null;

  const links = (
    <ul className="space-y-1.5 border-l border-[var(--trim-border)]">
      {headings.map((h) => (
        <li key={h.id}>
          <a
            href={`#${h.id}`}
            onClick={() => setMobileOpen(false)}
            className={cn(
              "-ml-px block border-l-2 py-0.5 pl-3 text-[12.5px] transition",
              active === h.id
                ? "border-[var(--trim-fg)] text-[var(--trim-fg)]"
                : "border-transparent text-[var(--trim-muted)] hover:text-[var(--trim-fg)]",
            )}
          >
            {h.title}
          </a>
        </li>
      ))}
    </ul>
  );

  return (
    <>
      {/* Next.js-style mobile "On this page" disclosure */}
      <div className="mb-8 xl:hidden">
        <button
          type="button"
          aria-expanded={mobileOpen}
          onClick={() => setMobileOpen((v) => !v)}
          className="flex w-full items-center justify-between rounded-lg border border-[var(--trim-border)] bg-[var(--trim-panel)] px-3 py-2.5 text-left text-sm text-[var(--trim-fg)]"
        >
          <span>On this page</span>
          <ChevronDown
            className={cn(
              "h-4 w-4 text-[var(--trim-muted)] transition-transform duration-200",
              mobileOpen && "rotate-180",
            )}
          />
        </button>
        <div
          className={cn(
            "grid transition-[grid-template-rows] duration-200 ease-out",
            mobileOpen ? "grid-rows-[1fr]" : "grid-rows-[0fr]",
          )}
        >
          <div className="min-h-0 overflow-hidden">
            <div className="mt-2 rounded-lg border border-[var(--trim-border)] bg-[var(--trim-panel)] p-3">
              {links}
            </div>
          </div>
        </div>
      </div>

      <nav aria-label="On this page" className="hidden space-y-2 xl:block">
        <p className="text-[11px] font-medium uppercase tracking-[0.14em] text-[var(--trim-muted)]">
          On this page
        </p>
        {links}
      </nav>
    </>
  );
}
