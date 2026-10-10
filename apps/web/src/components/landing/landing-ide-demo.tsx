"use client";

import { statusSoftBgClass, statusTextClass } from "@/lib/status-color";
import { cn } from "@/lib/utils";
import { useEffect, useRef, useState, type ReactNode } from "react";

type Step = {
  id: string;
  label: string;
  eyebrow: string;
  body: ReactNode;
};

const STEPS: Step[] = [
  {
    id: "settings",
    label: "Settings",
    eyebrow: "Cursor / VS Code → Models",
    body: (
      <div className="space-y-3 font-mono text-[12px] leading-relaxed">
        <p className="text-[var(--trim-subtle)]">Open Cursor or VS Code settings</p>
        <ul className="space-y-1.5 text-[var(--trim-muted)]">
          <li className="text-[var(--trim-subtle)]">├─ General</li>
          <li className="text-[var(--trim-fg)]">
            ├─ Models / provider <span className="text-[var(--trim-subtle)]">← open this</span>
          </li>
          <li className="text-[var(--trim-subtle)]">├─ Features</li>
          <li className="text-[var(--trim-subtle)]">└─ …</li>
        </ul>
        <p className="pt-2 text-[11px] text-[var(--trim-subtle)]">
          Trim is an OpenAI-compatible endpoint. Cursor and VS Code talk to it like any other base
          URL.
        </p>
      </div>
    ),
  },
  {
    id: "override",
    label: "Base URL",
    eyebrow: "Override Base URL",
    body: (
      <div className="space-y-4 font-mono text-[12px]">
        <div className="rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel-2)] px-3 py-2.5">
          <p className="text-[10px] uppercase tracking-[0.14em] text-[var(--trim-subtle)]">
            Provider
          </p>
          <p className="mt-1 text-[var(--trim-muted)]">OpenAI Compatible</p>
        </div>
        <div className="rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel-2)] px-3 py-2.5">
          <p className="text-[10px] uppercase tracking-[0.14em] text-[var(--trim-subtle)]">
            Override Base URL
          </p>
          <p className="mt-1 text-[var(--trim-fg)]">http://127.0.0.1:8888/v1</p>
        </div>
        <p className="text-[11px] leading-relaxed text-[var(--trim-subtle)]">
          Compress on your machine. Slim prompt → your model provider.
        </p>
      </div>
    ),
  },
  {
    id: "chat",
    label: "Chat",
    eyebrow: "Composer through Trim",
    body: (
      <div className="space-y-3 font-mono text-[12px] leading-relaxed">
        <div className="rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel-2)] px-3 py-2.5">
          <p className="text-[10px] uppercase tracking-[0.14em] text-[var(--trim-subtle)]">You</p>
          <p className="mt-1 text-[var(--trim-muted)]">fix the requireUser guard in session.ts</p>
        </div>
        <ul className="space-y-1.5 text-[var(--trim-subtle)]">
          <li>
            <span className="text-[var(--trim-subtle)]">1.</span> Composer →{" "}
            <span className="text-[var(--trim-muted)]">127.0.0.1:8888</span>
          </li>
          <li>
            <span className="text-[var(--trim-subtle)]">2.</span> Trim scans context · drops vendor
            / logs
          </li>
          <li>
            <span className="text-[var(--trim-subtle)]">3.</span> Slim prompt → model · reply
            unchanged
          </li>
        </ul>
        <p className={cn("text-[11px]", statusTextClass)}>
          Same answer. Far fewer input tokens billed.
        </p>
      </div>
    ),
  },
  {
    id: "dashboard",
    label: "Savings",
    eyebrow: "What you just saved",
    body: (
      <div className="space-y-3 font-mono text-[12px]">
        <div className="grid grid-cols-2 gap-2">
          <div className="rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel-2)] px-3 py-2.5">
            <p className="text-[10px] uppercase tracking-[0.14em] text-[var(--trim-subtle)]">
              Tokens in
            </p>
            <p className="mt-1 text-[var(--trim-muted)]">12,840</p>
          </div>
          <div className="rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel-2)] px-3 py-2.5">
            <p className="text-[10px] uppercase tracking-[0.14em] text-[var(--trim-subtle)]">
              After Trim
            </p>
            <p className={cn("mt-1", statusTextClass)}>2,140</p>
          </div>
        </div>
        <div className="flex items-center justify-between rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel-2)] px-3 py-2.5">
          <span className="text-[var(--trim-subtle)]">Saved</span>
          <span
            className={cn(
              "rounded px-2 py-0.5 text-[12px] font-medium",
              statusSoftBgClass,
              statusTextClass,
            )}
          >
            83% · $0.032
          </span>
        </div>
        <ul className="space-y-1 text-[11px] text-[var(--trim-subtle)]">
          <li>dropped vendor / lodash / cache noise</li>
          <li>kept src/auth/session.ts + stack</li>
          <li>dashboard rolls this up per day / team</li>
        </ul>
      </div>
    ),
  },
];

export function LandingIdeDemo({ compact = false }: { compact?: boolean }) {
  const [index, setIndex] = useState(0);
  const rootRef = useRef<HTMLDivElement>(null);
  const paused = useRef(false);
  const step = STEPS[index] || STEPS[0];

  useEffect(() => {
    const el = rootRef.current;
    if (!el) return;
    const io = new IntersectionObserver(() => {}, { threshold: 0.25 });
    io.observe(el);
    const timer = window.setInterval(() => {
      if (!paused.current) setIndex((i) => (i + 1) % STEPS.length);
    }, 3800);
    return () => {
      io.disconnect();
      window.clearInterval(timer);
    };
  }, []);

  const panel = (
    <div
      ref={rootRef}
      className="trim-product-frame overflow-hidden rounded-[10px] border border-[var(--trim-border)] bg-[var(--trim-panel)]"
      onMouseEnter={() => {
        paused.current = true;
      }}
      onMouseLeave={() => {
        paused.current = false;
      }}
    >
      <div className="flex h-9 items-center gap-3 border-b border-[var(--trim-border)] px-3">
        <div className="flex gap-[5px]">
          <span className="h-[9px] w-[9px] rounded-full bg-[var(--trim-panel-2)]" />
          <span className="h-[9px] w-[9px] rounded-full bg-[var(--trim-panel-2)]" />
          <span className="h-[9px] w-[9px] rounded-full bg-[var(--trim-panel-2)]" />
        </div>
        <div className="flex min-w-0 flex-1 justify-center">
          <span className="truncate font-mono text-[11px] text-[var(--trim-subtle)]">
            Cursor / VS Code · wire Trim
          </span>
        </div>
        <span className="font-mono text-[10px] text-[var(--trim-subtle)]">
          {index + 1}/{STEPS.length}
        </span>
      </div>

      <div className="flex items-center gap-0 overflow-x-auto border-b border-[var(--trim-border)] px-2">
        {STEPS.map((s, i) => {
          const active = i === index;
          const done = i < index;
          return (
            <button
              key={s.id}
              type="button"
              onClick={() => setIndex(i)}
              className={cn(
                "relative shrink-0 px-2.5 py-2 font-mono text-[11px] transition",
                active
                  ? "text-[var(--trim-fg)]"
                  : done
                    ? statusTextClass
                    : "text-[var(--trim-muted)] hover:text-[var(--trim-fg)]",
              )}
            >
              {i + 1}. {s.label}
              {active ? (
                <span className="absolute inset-x-1.5 bottom-0 h-px bg-[var(--trim-fg)]/80" />
              ) : null}
            </button>
          );
        })}
      </div>

      <div className="min-h-[280px] px-4 py-4 sm:min-h-[300px]">
        <p className="font-mono text-[10px] uppercase tracking-[0.16em] text-[var(--trim-subtle)]">
          {step.eyebrow}
        </p>
        <div className="mt-3">{step.body}</div>
      </div>

      <div className="flex h-7 items-center gap-2 border-t border-[var(--trim-border)] px-3 font-mono text-[10px] text-[var(--trim-subtle)]">
        <span
          className={cn(
            "inline-block h-1.5 w-1.5 rounded-full",
            step.id === "dashboard" ? "bg-[var(--trim-status)]" : "bg-[var(--trim-subtle)]",
          )}
        />
        <span>settings → URL → chat → savings</span>
        <span className="ml-auto hidden sm:inline">hover pause · click step</span>
      </div>
    </div>
  );

  if (compact) return panel;

  return (
    <section id="ide-demo" className="w-full px-5 py-16 sm:px-8 xl:px-12">
      <div className="max-w-2xl">
        <h2 className="font-display text-xl font-semibold tracking-tight text-[var(--trim-fg)] sm:text-2xl">
          Point Cursor or VS Code at Trim in four clicks
        </h2>
        <p className="mt-3 text-sm text-[var(--trim-muted)] sm:text-base">
          Models → base URL → chat through the proxy → see tokens and USD drop.
        </p>
      </div>
      <div className="mt-8 max-w-3xl">{panel}</div>
    </section>
  );
}
