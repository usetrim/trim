"use client";

import { cn } from "@/lib/utils";
import { DOC_NAV, docHref, findDocSectionId } from "@/lib/docs/nav";
import { ChevronRight } from "lucide-react";
import Link from "next/link";
import { useEffect, useState } from "react";

/**
 * Accordion sidebar: only one section open at a time (Cursor / Next.js style).
 * Opening a section closes the previous. Active route keeps its section open.
 */
export function DocsSidebar({ slug }: { slug: string }) {
  const activeSection = findDocSectionId(slug) || DOC_NAV[0]?.id || "";
  const [openId, setOpenId] = useState(activeSection);

  useEffect(() => {
    setOpenId(activeSection);
  }, [activeSection]);

  return (
    <nav className="space-y-0.5 pb-10" aria-label="Documentation">
      {DOC_NAV.map((section) => {
        const open = openId === section.id;
        return (
          <div key={section.id} className="border-b border-[var(--trim-border)]">
            <button
              type="button"
              aria-expanded={open}
              onClick={() => {
                // Exclusive accordion: clicking another section replaces openId.
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
                    const active = item.slug === slug;
                    return (
                      <li key={item.slug || "intro"}>
                        <Link
                          href={docHref(item.slug)}
                          scroll={false}
                          onClick={() => setOpenId(section.id)}
                          className={cn(
                            "block rounded-md px-2 py-1.5 text-[13px] transition",
                            active
                              ? "bg-[var(--trim-hover)] font-medium text-[var(--trim-fg)]"
                              : "text-[var(--trim-muted)] hover:bg-[var(--trim-hover)] hover:text-[var(--trim-fg)]",
                          )}
                        >
                          {item.title}
                        </Link>
                      </li>
                    );
                  })}
                </ul>
              </div>
            </div>
          </div>
        );
      })}
    </nav>
  );
}
