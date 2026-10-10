"use client";

import { LandingCliDemo } from "@/components/landing/landing-cli-demo";
import { LandingDemo } from "@/components/landing/landing-demo";
import { LandingHeroStage } from "@/components/landing/landing-hero";
import { useLandingSite } from "@/components/landing/landing-shell";
import { LandingIdeDemo } from "@/components/landing/landing-ide-demo";
import {
  LandingFlow,
  LandingStats,
  LandingUse,
  LandingWhy,
} from "@/components/landing/landing-story";
import { LandingHomeContentSkeleton } from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { CopyableCode } from "@/components/ui/copy-button";
import Link from "next/link";
import { useEffect } from "react";

export default function HomePage() {
  const site = useLandingSite();

  useEffect(() => {
    const scrollTo = (id: string) => {
      const el = document.getElementById(id);
      if (el) el.scrollIntoView({ behavior: "smooth" });
    };

    let fromNav = "";
    try {
      fromNav = sessionStorage.getItem("landing-scroll") || "";
      if (fromNav) sessionStorage.removeItem("landing-scroll");
    } catch {
      /* ignore */
    }
    const hash = window.location.hash.replace(/^#/, "");
    const id = fromNav || hash;
    if (id) {
      // Wait a tick so the right-rail content is painted.
      requestAnimationFrame(() => scrollTo(id));
    }

    const onHash = () => {
      const h = window.location.hash.replace(/^#/, "");
      if (h) scrollTo(h);
    };
    window.addEventListener("hashchange", onHash);
    return () => window.removeEventListener("hashchange", onHash);
  }, []);

  if (!site) return <LandingHomeContentSkeleton />;

  return (
    <>
      <div className="px-5 py-8 sm:px-8 xl:px-12 xl:py-10">
        <section id="hero-demo" aria-label="Live product demo">
          <LandingHeroStage />
        </section>
      </div>

      <div className="space-y-2 border-t border-[var(--trim-border)]">
        <LandingStats site={site} />
        <LandingDemo chrome={site} />

        <section id="install" className="px-5 py-16 sm:px-8 xl:px-12">
          <h2 className="font-display text-xl font-semibold tracking-tight text-[var(--trim-fg)] sm:text-2xl">
            Install once. Wire Cursor or VS Code. Keep coding.
          </h2>
          <p className="mt-2 max-w-xl text-sm text-[var(--trim-muted)]">
            Live CLI and IDE setup. Install → local proxy → Base URL in Cursor or VS Code → savings
            meter. Optional Trim IDE extension for telemetry in both editors.
          </p>
          <div className="mt-8 grid gap-4 xl:grid-cols-2">
            <div>
              <p className="mb-2 font-mono text-[10px] uppercase tracking-[0.16em] text-[var(--trim-subtle)]">
                CLI · install to meter
              </p>
              <LandingCliDemo compact />
            </div>
            <div>
              <p className="mb-2 font-mono text-[10px] uppercase tracking-[0.16em] text-[var(--trim-subtle)]">
                IDE · Cursor / VS Code
              </p>
              <LandingIdeDemo compact />
            </div>
          </div>
          {site.install_snippet ? (
            <CopyableCode className="mt-6" code={site.install_snippet} language="bash" />
          ) : null}
          <div className="mt-5 flex flex-wrap gap-2">
            <Button asChild size="sm" className="rounded-md px-5">
              <Link href={site.nav_sign_in_href || ""}>{site.cta_start}</Link>
            </Button>
            <Button asChild size="sm" variant="outline" className="rounded-md px-5">
              <a href={site.source_url} rel="noreferrer">
                {site.cta_source}
              </a>
            </Button>
            <Button asChild size="sm" variant="outline" className="rounded-md px-5">
              <Link href="/docs/ide/extension">IDE extension</Link>
            </Button>
          </div>
        </section>

        <LandingFlow site={site} />
        <LandingWhy site={site} />
        <LandingUse site={site} />
      </div>
    </>
  );
}
