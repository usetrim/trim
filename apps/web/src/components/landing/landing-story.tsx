"use client";

import { statusTextClass } from "@/lib/status-color";
import { cn } from "@/lib/utils";
import type { AuthProvidersSiteChrome } from "@/types/auth";

type Props = { site: AuthProvidersSiteChrome };

function Stat({
  value,
  label,
  hint,
  emphasize,
}: {
  value?: string;
  label?: string;
  hint?: string;
  emphasize?: boolean;
}) {
  if (!(value || "").trim() || !(label || "").trim()) return null;
  return (
    <div className="rounded-2xl border border-[var(--trim-border)] bg-[var(--trim-panel)] px-6 py-8 shadow-[var(--trim-card-shadow)]">
      <p
        className={cn(
          "font-display text-4xl font-semibold tracking-tight sm:text-5xl",
          emphasize ? statusTextClass : "text-[var(--trim-fg)]",
        )}
      >
        {value}
      </p>
      <p className="mt-3 text-sm font-medium text-[var(--trim-muted)]">{label}</p>
      {hint?.trim() ? (
        <p className="mt-2 text-xs leading-relaxed text-[var(--trim-subtle)]">{hint}</p>
      ) : null}
    </div>
  );
}

export function LandingStats({ site }: Props) {
  const title = (site.stats_title || "").trim();
  if (!title) return null;
  return (
    <section id="savings" className="w-full px-5 py-16 sm:px-8 xl:px-12">
      <div className="max-w-2xl">
        <h2 className="font-display text-xl font-semibold tracking-tight text-[var(--trim-fg)] sm:text-2xl">
          {title}
        </h2>
        {site.stats_subtitle ? (
          <p className="mt-3 text-sm text-[var(--trim-muted)] sm:text-base">
            {site.stats_subtitle}
          </p>
        ) : null}
      </div>
      <div className="mt-10 grid gap-3 sm:grid-cols-3">
        <Stat value={site.stat_1_value} label={site.stat_1_label} emphasize />
        <Stat value={site.stat_2_value} label={site.stat_2_label} />
        <Stat value={site.stat_3_value} label={site.stat_3_label} hint={site.stat_3_hint} />
      </div>
    </section>
  );
}

export function LandingFlow({ site }: Props) {
  const title = (site.flow_title || "").trim();
  const steps = [
    { t: site.flow_1_title, b: site.flow_1_body },
    { t: site.flow_2_title, b: site.flow_2_body },
    { t: site.flow_3_title, b: site.flow_3_body },
    { t: site.flow_4_title, b: site.flow_4_body },
  ].filter((s) => (s.t || "").trim() && (s.b || "").trim());
  if (!title || steps.length === 0) return null;

  return (
    <section id="how" className="w-full px-5 py-16 sm:px-8 xl:px-12">
      <div className="max-w-2xl">
        <h2 className="font-display text-xl font-semibold tracking-tight text-[var(--trim-fg)] sm:text-2xl">
          {title}
        </h2>
        {site.flow_subtitle ? (
          <p className="mt-3 text-sm text-[var(--trim-muted)] sm:text-base">{site.flow_subtitle}</p>
        ) : null}
      </div>
      <ol className="mt-10 max-w-4xl">
        {steps.map((step, i) => {
          const last = i === steps.length - 1;
          const n = String(i + 1).padStart(2, "0");
          return (
            <li
              key={step.t}
              className="relative grid grid-cols-1 gap-3 border-l border-[var(--trim-border)] py-7 pl-8 md:grid-cols-[17.5rem_minmax(0,1fr)] md:items-start md:gap-x-12"
            >
              <span
                className={cn(
                  "absolute -left-1.5 top-[2.05rem] h-3 w-3 rounded-full ring-4 ring-[var(--trim-bg)]",
                  last ? "bg-[var(--trim-status)]" : "bg-[var(--trim-subtle)]",
                )}
              />
              <div className="min-w-0">
                <span className="font-mono text-[11px] tracking-wide text-[var(--trim-subtle)]">
                  Step {n}
                </span>
                <h3 className="mt-1 text-base font-semibold leading-snug text-[var(--trim-fg)] sm:text-lg">
                  {step.t}
                </h3>
              </div>
              <p className="min-w-0 text-sm leading-relaxed text-[var(--trim-muted)] md:pt-[1.375rem]">
                {step.b}
              </p>
            </li>
          );
        })}
      </ol>
    </section>
  );
}

export function LandingWhy({ site }: Props) {
  const title = (site.why_title || "").trim();
  const cards = [
    { t: site.why_1_title, b: site.why_1_body },
    { t: site.why_2_title, b: site.why_2_body },
    { t: site.why_3_title, b: site.why_3_body },
    { t: site.why_4_title, b: site.why_4_body },
    { t: site.why_5_title, b: site.why_5_body },
    { t: site.why_6_title, b: site.why_6_body },
    { t: site.why_7_title, b: site.why_7_body },
    { t: site.why_8_title, b: site.why_8_body },
    { t: site.why_9_title, b: site.why_9_body },
    { t: site.why_10_title, b: site.why_10_body },
    { t: site.why_11_title, b: site.why_11_body },
    { t: site.why_12_title, b: site.why_12_body },
  ].filter((c) => (c.t || "").trim() && (c.b || "").trim());
  if (!title || cards.length === 0) return null;

  return (
    <section id="why" className="w-full px-5 py-16 sm:px-8 xl:px-12">
      <div className="max-w-2xl">
        <h2 className="font-display text-xl font-semibold tracking-tight text-[var(--trim-fg)] sm:text-2xl">
          {title}
        </h2>
        {site.why_subtitle ? (
          <p className="mt-3 text-sm text-[var(--trim-muted)] sm:text-base">{site.why_subtitle}</p>
        ) : null}
      </div>
      <div className="mt-10 grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {cards.map((c) => (
          <article
            key={c.t}
            className="border-l border-[var(--trim-border)] py-1 pl-5 transition hover:border-[var(--trim-border-strong)]"
          >
            <h3 className="text-base font-semibold text-[var(--trim-fg)]">{c.t}</h3>
            <p className="mt-2 text-sm leading-relaxed text-[var(--trim-muted)]">{c.b}</p>
          </article>
        ))}
      </div>
    </section>
  );
}

export function LandingUse({ site }: Props) {
  const title = (site.use_title || "").trim();
  const steps = [
    { t: site.use_1_title, b: site.use_1_body },
    { t: site.use_2_title, b: site.use_2_body },
    { t: site.use_3_title, b: site.use_3_body },
  ].filter((s) => (s.t || "").trim() && (s.b || "").trim());
  if (!title || steps.length === 0) return null;

  return (
    <section id="use" className="w-full px-5 py-16 sm:px-8 xl:px-12">
      <div className="max-w-2xl">
        <h2 className="font-display text-xl font-semibold tracking-tight text-[var(--trim-fg)] sm:text-2xl">
          {title}
        </h2>
        {site.use_subtitle ? (
          <p className="mt-3 text-sm text-[var(--trim-muted)] sm:text-base">{site.use_subtitle}</p>
        ) : null}
      </div>
      <ol className="mt-10 grid gap-4 md:grid-cols-3">
        {steps.map((step, i) => (
          <li
            key={step.t}
            className="rounded-2xl border border-[var(--trim-border)] bg-[var(--trim-panel)] p-6 shadow-[var(--trim-card-shadow)]"
          >
            <span className="font-mono text-xs text-[var(--trim-subtle)]">
              Step {String(i + 1).padStart(2, "0")}
            </span>
            <h3 className="mt-3 text-lg font-semibold text-[var(--trim-fg)]">{step.t}</h3>
            <p className="mt-2 text-sm leading-relaxed text-[var(--trim-muted)]">{step.b}</p>
          </li>
        ))}
      </ol>
    </section>
  );
}
