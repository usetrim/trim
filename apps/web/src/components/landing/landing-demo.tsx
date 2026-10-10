"use client";

import { cn } from "@/lib/utils";
import { statusSoftBgClass, statusTextClass } from "@/lib/status-color";
import { useCallback, useEffect, useRef, useState } from "react";

export type LandingDemoChrome = {
  demo_title?: string;
  demo_subtitle?: string;
  demo_before_label?: string;
  demo_after_label?: string;
  demo_tokens_in_fmt?: string;
  demo_tokens_out_fmt?: string;
  demo_saved_fmt?: string;
  demo_cost_hint?: string;
};

type Phase = "idle" | "sorting" | "writing" | "done";

type Scenario = {
  id: string;
  label: string;
  tokensIn: number;
  tokensOut: number;
  usdIn: number;
  usdOut: number;
  before: string;
  after: string;
  cuts: string[];
};

const SCENARIOS: Scenario[] = [
  {
    id: "vendor",
    label: "Vendor dump",
    tokensIn: 14200,
    tokensOut: 2380,
    usdIn: 0.0426,
    usdOut: 0.0071,
    before: `// vendor/react/cjs/react.development.js
function createElement(type, config, children) {
  // 380 lines of framework internals...
}

// node_modules/lodash/lodash.js
/* ... 500+ lines of utility noise ... */

// src/auth/session.ts  <-- signal
export async function requireUser(req: Request) {
  const session = await getSession(req);
  if (!session) throw new AuthError("unauthorized");
  return session.user;
}

// .next/cache/... stale build noise`,
    after: `// src/auth/session.ts (kept)
export async function requireUser(req: Request) {
  const session = await getSession(req);
  if (!session) throw new AuthError("unauthorized");
  return session.user;
}

// vendor/react: skeleton
// lodash: pruned
// .next cache: dropped`,
    cuts: [
      "capture noisy Composer context",
      "scan 14,200 tokens",
      "drop vendor/react bodies −3.4k",
      "drop lodash utility dump −4.1k",
      "drop .next/cache noise −2.2k",
      "keep src/auth/session.ts",
      "skeleton heavy imports",
      "forward 2,380 tokens → model",
    ],
  },
  {
    id: "ci",
    label: "CI logs",
    tokensIn: 18640,
    tokensOut: 1920,
    usdIn: 0.0559,
    usdOut: 0.0058,
    before: `[INFO] compiling packages...
[DEBUG] cache miss x 220
[WARN] deprecated dep: left-pad@1.3.0
... 400 more noise lines ...
[ERROR] TypeError: Cannot read properties of undefined (reading 'id')
    at requireUser (src/auth/session.ts:12:18)
    at handler (src/app/api/me/route.ts:8:20)
[INFO] uploaded artifact build.zip`,
    after: `[ERROR] TypeError: Cannot read properties of undefined (reading 'id')
    at requireUser (src/auth/session.ts:12:18)
    at handler (src/app/api/me/route.ts:8:20)

// INFO/DEBUG/WARN spam collapsed (400 lines)
// artifact upload noise dropped`,
    cuts: [
      "capture CI log dump",
      "scan 18,640 tokens",
      "collapse INFO/DEBUG spam −14k",
      "drop artifact upload noise",
      "keep ERROR + stack frames",
      "keep session.ts + route.ts refs",
      "forward 1,920 tokens → model",
    ],
  },
  {
    id: "agent",
    label: "Agent context",
    tokensIn: 22100,
    tokensOut: 4100,
    usdIn: 0.0663,
    usdOut: 0.0123,
    before: `<tool_result path="package-lock.json">
{ ... 8k lines ... }
</tool_result>

<open_file path="README.md">
Long onboarding docs...
</open_file>

<diff path="src/billing/checkout.ts">
- const priceId = oldPrice
+ const priceId = plan.paddlePriceId
</diff>`,
    after: `<diff path="src/billing/checkout.ts">
- const priceId = oldPrice
+ const priceId = plan.paddlePriceId
</diff>

// lockfile omitted · README dropped · tool spam dropped`,
    cuts: [
      "capture agent tool results",
      "scan 22,100 tokens",
      "stub package-lock.json −8k",
      "drop README + tool spam",
      "keep billing checkout diff",
      "forward 4,100 tokens → model",
    ],
  },
  {
    id: "chat",
    label: "Long chat",
    tokensIn: 9800,
    tokensOut: 2100,
    usdIn: 0.0294,
    usdOut: 0.0063,
    before: `User: how do I set up auth?
Assistant: use sessions...
... 26 more stale turns ...
User: show me the requireUser guard and why checkout 500s`,
    after: `User: show me the requireUser guard and why checkout 500s

// Related: src/auth/session.ts · src/billing/checkout.ts
// Older turns summarized`,
    cuts: [
      "capture long chat history",
      "scan 9,800 tokens",
      "summarize 26 stale turns",
      "keep latest user ask",
      "pin session.ts + checkout.ts",
      "forward 2,100 tokens → model",
    ],
  },
];

function fmt(template: string | undefined, value: string) {
  if (!template?.includes("%s")) return value;
  return template.replace("%s", value);
}

function money(n: number) {
  return `$${n.toFixed(4)}`;
}

export function LandingDemo({ chrome }: { chrome: LandingDemoChrome }) {
  const [scenarioIndex, setScenarioIndex] = useState(0);
  const scenario = SCENARIOS[scenarioIndex] || SCENARIOS[0];
  const [phase, setPhase] = useState<Phase>("idle");
  const [log, setLog] = useState<string[]>([]);
  const [tokensOut, setTokensOut] = useState(scenario.tokensIn);
  const [usdOut, setUsdOut] = useState(scenario.usdIn);
  const [afterText, setAfterText] = useState("");
  const [paused, setPaused] = useState(false);
  const timers = useRef<number[]>([]);
  const raf = useRef<number | null>(null);
  const afterRef = useRef<HTMLPreElement>(null);
  const stageRef = useRef<HTMLDivElement>(null);
  const logScrollRef = useRef<HTMLUListElement>(null);
  const inView = useRef(false);
  const pausedRef = useRef(false);
  const runGen = useRef(0);
  const scenarioIndexRef = useRef(0);
  scenarioIndexRef.current = scenarioIndex;

  useEffect(() => {
    pausedRef.current = paused;
  }, [paused]);

  const clearTimers = useCallback(() => {
    for (const id of timers.current) window.clearTimeout(id);
    timers.current = [];
    if (raf.current != null) {
      cancelAnimationFrame(raf.current);
      raf.current = null;
    }
  }, []);

  const run = useCallback(
    (s: Scenario, onComplete?: () => void) => {
      const gen = ++runGen.current;
      clearTimers();
      setPhase("sorting");
      setTokensOut(s.tokensIn);
      setUsdOut(s.usdIn);
      setAfterText("");
      setLog([]);

      const push = (ms: number, fn: () => void) => {
        timers.current.push(
          window.setTimeout(() => {
            if (gen !== runGen.current) return;
            fn();
          }, ms),
        );
      };

      push(200, () => setLog([s.cuts[0] || "Captured noisy prompt"]));
      s.cuts.forEach((cut, i) => {
        if (i === 0) return;
        push(280 + i * 380, () => {
          setLog((l) => [...l, cut]);
          if (i === Math.min(2, s.cuts.length - 1)) setPhase("writing");
        });
      });

      const writeAt = 280 + Math.max(1, s.cuts.length - 1) * 380 + 200;
      push(writeAt, () => {
        setPhase("writing");
        const full = s.after;
        const start = performance.now();
        const typeMs = Math.min(2200, 14 * full.length);
        const meterMs = 1600;

        const tick = (now: number) => {
          if (gen !== runGen.current) return;
          const tp = Math.min(1, (now - start) / typeMs);
          const mp = Math.min(1, (now - start) / meterMs);
          const eased = 1 - (1 - mp) * (1 - mp);
          setAfterText(full.slice(0, Math.floor(full.length * tp)));
          setTokensOut(Math.round(s.tokensIn + (s.tokensOut - s.tokensIn) * eased));
          setUsdOut(s.usdIn + (s.usdOut - s.usdIn) * eased);
          if (afterRef.current) afterRef.current.scrollTop = afterRef.current.scrollHeight;
          if (tp < 1 || mp < 1) {
            raf.current = requestAnimationFrame(tick);
          } else {
            setPhase("done");
            setTokensOut(s.tokensOut);
            setUsdOut(s.usdOut);
            setAfterText(full);
            setLog((l) => [
              ...l,
              `saved ${Math.round((1 - s.tokensOut / s.tokensIn) * 100)}% · ${money(
                s.usdIn,
              )} → ${money(s.usdOut)}`,
            ]);
            push(2400, () => {
              if (!pausedRef.current) onComplete?.();
            });
          }
        };
        raf.current = requestAnimationFrame(tick);
      });
    },
    [clearTimers],
  );

  const advance = useCallback(() => {
    setScenarioIndex((i) => (i + 1) % SCENARIOS.length);
  }, []);

  useEffect(() => {
    if (!inView.current) return;
    run(scenario, advance);
  }, [scenario, run, advance]);

  useEffect(() => {
    const el = stageRef.current;
    if (!el) return;
    const io = new IntersectionObserver(
      (entries) => {
        const visible = entries.some((e) => e.isIntersecting);
        inView.current = visible;
        if (visible) {
          const s = SCENARIOS[scenarioIndexRef.current] || SCENARIOS[0];
          run(s, advance);
        } else {
          clearTimers();
        }
      },
      { threshold: 0.25 },
    );
    io.observe(el);
    return () => {
      io.disconnect();
      clearTimers();
    };
  }, [run, advance, clearTimers]);

  useEffect(() => () => clearTimers(), [clearTimers]);

  useEffect(() => {
    if (!paused && phase === "done" && inView.current) {
      const id = window.setTimeout(() => advance(), 600);
      return () => window.clearTimeout(id);
    }
  }, [paused, phase, advance]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    if (logScrollRef.current) {
      logScrollRef.current.scrollTop = logScrollRef.current.scrollHeight;
    }
  }, [log]);

  const savedPct = Math.round((1 - scenario.tokensOut / scenario.tokensIn) * 100);
  const title = (chrome.demo_title || "").trim();
  if (!title) return null;

  return (
    <section id="demo" className="w-full px-5 py-16 sm:px-8 xl:px-12">
      <div className="max-w-2xl">
        <h2 className="font-display text-xl font-semibold tracking-tight text-[var(--trim-fg)] sm:text-2xl">
          {title}
        </h2>
        {chrome.demo_subtitle ? (
          <p className="mt-3 text-sm text-[var(--trim-muted)] sm:text-base">
            {chrome.demo_subtitle}
          </p>
        ) : null}
      </div>

      <div
        ref={stageRef}
        className="trim-product-frame relative mt-8 overflow-hidden rounded-[10px] border border-[var(--trim-border)] bg-[var(--trim-panel)]"
        onMouseEnter={() => setPaused(true)}
        onMouseLeave={() => setPaused(false)}
      >
        <div className="flex items-center gap-0 overflow-x-auto border-b border-[var(--trim-border)] px-2">
          {SCENARIOS.map((s, i) => {
            const active = i === scenarioIndex;
            return (
              <button
                key={s.id}
                type="button"
                onClick={() => setScenarioIndex(i)}
                className={cn(
                  "relative shrink-0 px-2.5 py-2 font-mono text-[11px] transition",
                  active
                    ? "text-[var(--trim-fg)]"
                    : "text-[var(--trim-muted)] hover:text-[var(--trim-fg)]",
                )}
              >
                {s.label}
                {active ? (
                  <span className="absolute inset-x-1.5 bottom-0 h-px bg-[var(--trim-fg)]/80" />
                ) : null}
              </button>
            );
          })}
          <span className="ml-auto hidden pr-2 font-mono text-[10px] text-[var(--trim-subtle)] sm:inline">
            {paused ? "paused" : "live"}
          </span>
        </div>

        <div className="grid min-w-0 gap-0 lg:grid-cols-2">
          {/* Process log - same padding / mono density as other CLIs */}
          <div className="flex min-w-0 flex-col border-b border-[var(--trim-border)] lg:border-b-0 lg:border-r">
            <div className="border-b border-[var(--trim-border)] px-3.5 py-3">
              <p className="font-mono text-[10px] uppercase tracking-[0.14em] text-[var(--trim-subtle)]">
                Fast Mode pass
              </p>
              <p className="mt-1.5 font-mono text-[12px] leading-snug text-[var(--trim-muted)]">
                Shrink this context before the model bills it
              </p>
              <div className="mt-3 space-y-1 font-mono text-[11px] leading-5 text-[var(--trim-subtle)]">
                <p className="break-words">
                  <span
                    className={
                      phase !== "idle" && phase !== "sorting"
                        ? statusTextClass
                        : "text-[var(--trim-subtle)]"
                    }
                  >
                    [x]
                  </span>{" "}
                  Drop vendor / log / lockfile noise
                </p>
                <p className="break-words">
                  <span
                    className={
                      phase === "done" || phase === "writing"
                        ? statusTextClass
                        : "text-[var(--trim-subtle)]"
                    }
                  >
                    [x]
                  </span>{" "}
                  Keep task files + stack traces
                </p>
                <p className="break-words">
                  <span
                    className={phase === "done" ? statusTextClass : "text-[var(--trim-subtle)]"}
                  >
                    [x]
                  </span>{" "}
                  Forward slim prompt upstream
                </p>
              </div>
            </div>

            <ul
              ref={logScrollRef}
              className="landing-demo-scroll h-[240px] min-w-0 space-y-1.5 overflow-auto px-3.5 py-3 font-mono text-[11.5px] leading-[1.55] text-[var(--trim-muted)]"
            >
              {log.length === 0 ? (
                <li className="text-[var(--trim-subtle)]">waiting…</li>
              ) : (
                log.map((line, i) => {
                  const savedLine = line.startsWith("saved") || line.startsWith("forward");
                  const active = i === log.length - 1 && phase !== "done";
                  return (
                    <li
                      // biome-ignore lint/suspicious/noArrayIndexKey: append-only demo log rows
                      key={`${line}-${i}`}
                      className="flex min-w-0 items-start gap-2"
                    >
                      <span
                        className={cn(
                          "mt-1.5 h-1.5 w-1.5 shrink-0 rotate-45 border bg-transparent",
                          savedLine || (!active && phase !== "idle")
                            ? "border-[var(--trim-status)]"
                            : active
                              ? "border-[var(--trim-fg)]/55"
                              : "border-[var(--trim-border)]",
                        )}
                      />
                      <span
                        className={cn(
                          "min-w-0 flex-1 break-words",
                          savedLine && statusTextClass,
                          !savedLine &&
                            (active ? "text-[var(--trim-fg)]" : "text-[var(--trim-subtle)]"),
                        )}
                      >
                        {line}
                      </span>
                    </li>
                  );
                })
              )}
            </ul>

            <div className="mt-auto border-t border-[var(--trim-border)] px-3.5 py-3 font-mono text-[11px] leading-relaxed text-[var(--trim-subtle)]">
              <p className="break-words">
                tokens {scenario.tokensIn.toLocaleString()} →{" "}
                <span className={phase === "done" ? statusTextClass : "text-[var(--trim-fg)]"}>
                  {tokensOut.toLocaleString()}
                </span>
                {" · "}
                {money(scenario.usdIn)} →{" "}
                <span className={phase === "done" ? statusTextClass : "text-[var(--trim-fg)]"}>
                  {money(usdOut)}
                </span>
              </p>
              {phase === "done" ? (
                <span
                  className={cn(
                    "mt-2 inline-block rounded px-1.5 py-0.5 text-[10px] font-medium",
                    statusSoftBgClass,
                    statusTextClass,
                  )}
                >
                  {savedPct}% saved
                </span>
              ) : null}
              {chrome.demo_cost_hint && phase === "done" ? (
                <p className="mt-2 break-words text-[10px] leading-relaxed text-[var(--trim-subtle)]">
                  {chrome.demo_cost_hint}
                </p>
              ) : null}
            </div>
          </div>

          {/* Before / after code */}
          <div className="grid min-h-[320px] min-w-0 grid-rows-2">
            <div className="min-w-0 border-b border-[var(--trim-border)]">
              <div className="flex items-center justify-between gap-2 px-3.5 py-1.5 font-mono text-[10px] text-[var(--trim-subtle)]">
                <span className="shrink-0">{chrome.demo_before_label || "before"}</span>
                <span className="truncate">
                  {fmt(chrome.demo_tokens_in_fmt, scenario.tokensIn.toLocaleString())}
                </span>
              </div>
              <pre className="landing-demo-scroll h-[160px] overflow-auto px-3.5 pb-3 font-mono text-[11px] leading-[1.55] text-[var(--trim-subtle)]">
                <code className="block whitespace-pre-wrap break-words">{scenario.before}</code>
              </pre>
            </div>
            <div className="min-w-0">
              <div className="flex items-center justify-between gap-2 px-3.5 py-1.5 font-mono text-[10px] text-[var(--trim-subtle)]">
                <span className="shrink-0">{chrome.demo_after_label || "after"}</span>
                <span
                  className={cn(
                    "truncate",
                    phase === "done" ? statusTextClass : "text-[var(--trim-muted)]",
                  )}
                >
                  {fmt(chrome.demo_tokens_out_fmt, tokensOut.toLocaleString())}
                </span>
              </div>
              <pre
                ref={afterRef}
                className="landing-demo-scroll h-[160px] overflow-auto px-3.5 pb-3 font-mono text-[11px] leading-[1.55] text-[var(--trim-muted)]"
              >
                <code className="block whitespace-pre-wrap break-words">
                  {afterText || " "}
                  {phase === "writing" ? (
                    <span className="animate-pulse text-[var(--trim-fg)]">▋</span>
                  ) : null}
                </code>
              </pre>
            </div>
          </div>
        </div>

        <div className="flex h-7 items-center gap-2 border-t border-[var(--trim-border)] px-3 font-mono text-[10px] text-[var(--trim-subtle)]">
          <span
            className={cn(
              "inline-block h-1.5 w-1.5 rounded-full",
              phase === "done" ? "bg-[var(--trim-status)]" : "bg-[var(--trim-subtle)]",
            )}
          />
          <span>capture → drop → keep → meter</span>
          <span className="ml-auto hidden sm:inline">hover pause · click scenario</span>
        </div>
      </div>
    </section>
  );
}
