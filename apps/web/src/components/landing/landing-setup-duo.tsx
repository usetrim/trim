"use client";

import { LandingCliDemo } from "@/components/landing/landing-cli-demo";
import { LandingIdeDemo } from "@/components/landing/landing-ide-demo";

/** Side-by-side install showcase - reuses the enhanced CLI + IDE demos. */
export function LandingSetupDuo() {
  return (
    <section id="setup" className="w-full px-5 py-16 sm:px-8 xl:px-12">
      <div className="max-w-2xl">
        <h2 className="font-display text-xl font-semibold tracking-tight text-[var(--trim-fg)] sm:text-2xl">
          Install once. Wire Cursor or VS Code. Keep coding.
        </h2>
        <p className="mt-3 text-sm text-[var(--trim-muted)] sm:text-base">
          Live CLI and IDE: install → local proxy → Composer / VS Code chat → savings meter.
        </p>
      </div>
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
    </section>
  );
}
