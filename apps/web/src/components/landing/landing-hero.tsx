"use client";

import { CLI_SCRIPT_UNIX, type CliTone } from "@/components/landing/landing-cli-script";
import { STATUS_GREEN, STATUS_GREEN_BG, statusTextClass } from "@/lib/status-color";
import { cn } from "@/lib/utils";
import Link from "next/link";
import { useCallback, useEffect, useRef, useState, type MouseEvent } from "react";

type HeroChrome = {
  eyebrow?: string;
  headline?: string;
  tagline?: string;
  cta_start?: string;
  cta_demo?: string;
  hero_note?: string;
  nav_sign_in_href?: string;
  nav_install?: string;
  install_title?: string;
};

const BEFORE = `vendor/react/cjs/react.development.js
  createElement() { /* 380 lines */ }
  // framework internals …

node_modules/lodash/lodash.js
  /* 500+ lines utility noise */

.next/cache/webpack/…
  /* stale build artifacts */

[DEBUG] cache miss x 220
[INFO]  compiling packages…
[WARN]  deprecated left-pad@1.3.0

src/auth/session.ts
  export async function requireUser(req) {
    const session = await getSession(req);
    if (!session) throw new AuthError();
    return session.user;
  }

src/app/api/me/route.ts
  export async function GET(req) {
    const user = await requireUser(req);
    return Response.json({ user });
  }`;

const AFTER = `src/auth/session.ts
  export async function requireUser(req) {
    const session = await getSession(req);
    if (!session) throw new AuthError();
    return session.user;
  }

src/app/api/me/route.ts
  export async function GET(req) {
    const user = await requireUser(req);
    return Response.json({ user });
  }

-- Fast Mode summary --
vendor/react     → skeleton   (−3,120)
node_modules/*   → pruned     (−4,080)
.next + logs     → dropped    (−2,500)
DEBUG/INFO spam  → collapsed  (−1,100)
signal kept      → session + route
forwarded        → 2,140 tok`;

/** Stable keys for static demo lines (avoids array-index keys in JSX). */
const BEFORE_LINES = BEFORE.split("\n").map((text, lineNo) => ({
  id: `before-${lineNo}`,
  text,
}));

const IDE_STEPS = [
  { t: "capture Composer request", tone: "dim" as const },
  { t: "open workspace context pack", tone: "dim" as const },
  { t: "scan 12,840 tokens · 48 files", tone: "dim" as const },
  { t: "classify vendor vs product code", tone: "dim" as const },
  { t: "drop  vendor/react/cjs/*           −3,120 tok", tone: "dim" as const },
  { t: "drop  node_modules/lodash/*        −4,080 tok", tone: "dim" as const },
  { t: "drop  .next/cache + webpack noise  −1,400 tok", tone: "dim" as const },
  { t: "drop  DEBUG / INFO log spam        −1,100 tok", tone: "dim" as const },
  { t: "skeleton  heavy imports → stubs", tone: "ok" as const },
  { t: "keep  src/auth/session.ts", tone: "ok" as const },
  { t: "keep  src/app/api/me/route.ts", tone: "ok" as const },
  { t: "keep  stack frames + task files", tone: "ok" as const },
  { t: "forward  2,140 tokens → model", tone: "saved" as const },
  { t: "reply    streamed · quality unchanged", tone: "ok" as const },
];

type Scene = "ide" | "terminal";
type TermLine =
  | { k: "in"; t: string; caret?: boolean }
  | { k: "out"; t: string; tone?: CliTone }
  | { k: "blank" };

type Activity = { t: string; tone: "dim" | "ok" | "saved" };

export function LandingHeroStage() {
  const stageRef = useRef<HTMLDivElement>(null);
  const beforeRef = useRef<HTMLPreElement>(null);
  const termScrollRef = useRef<HTMLDivElement>(null);
  const activityScrollRef = useRef<HTMLUListElement>(null);
  const [scene, setScene] = useState<Scene>("ide");
  const [tilt, setTilt] = useState({ x: 0, y: 0 });
  const [progress, setProgress] = useState(0);
  const [tokensOut, setTokensOut] = useState(12840);
  const [afterText, setAfterText] = useState("");
  const [phase, setPhase] = useState<"idle" | "scan" | "trim" | "done">("idle");
  const [cliPhase, setCliPhase] = useState<"install" | "proxy" | "trim" | "status">("install");
  const [term, setTerm] = useState<TermLine[]>([]);
  const [activity, setActivity] = useState<Activity[]>([]);
  const [beamY, setBeamY] = useState(0);
  const [paused, setPaused] = useState(false);
  const timers = useRef<number[]>([]);
  const raf = useRef<number | null>(null);
  const gen = useRef(0);
  const pausedRef = useRef(false);
  const sceneLock = useRef<Scene | null>(null);
  const ignorePauseUntil = useRef(0);

  const tokensIn = 12840;
  const tokensTarget = 2140;
  const saved = Math.round((1 - tokensOut / tokensIn) * 100);
  const isSuccess = phase === "done" || (scene === "terminal" && cliPhase === "status");

  useEffect(() => {
    pausedRef.current = paused;
  }, [paused]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    if (termScrollRef.current) {
      termScrollRef.current.scrollTop = termScrollRef.current.scrollHeight;
    }
  }, [term]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    if (activityScrollRef.current) {
      activityScrollRef.current.scrollTop = activityScrollRef.current.scrollHeight;
    }
  }, [activity]);

  const clear = useCallback(() => {
    for (const id of timers.current) window.clearTimeout(id);
    timers.current = [];
    if (raf.current != null) cancelAnimationFrame(raf.current);
  }, []);

  const later = useCallback((ms: number, fn: () => void, g: number) => {
    const tick = () => {
      if (g !== gen.current) return;
      if (pausedRef.current) {
        timers.current.push(window.setTimeout(tick, 120));
        return;
      }
      fn();
    };
    timers.current.push(window.setTimeout(tick, ms));
  }, []);

  const playTerminal = useCallback(
    (g: number, onDone: () => void) => {
      setScene("terminal");
      setCliPhase("install");
      setTerm([]);
      let shown: TermLine[] = [];
      let t = 200;
      const script = CLI_SCRIPT_UNIX;

      script.forEach((step, idx) => {
        if (step.kind === "blank") {
          t += 160;
          later(
            t,
            () => {
              shown = [...shown, { k: "blank" }];
              setTerm([...shown]);
            },
            g,
          );
          return;
        }
        if (step.kind === "in") {
          t += step.delay || 200;
          const start = t;
          const cmd = step.text;
          later(
            start,
            () => {
              if (cmd.startsWith("trim start")) setCliPhase("proxy");
              if (cmd.startsWith("trim status")) setCliPhase("status");
              shown = [...shown, { k: "in", t: "", caret: true }];
              setTerm([...shown]);
            },
            g,
          );
          for (let i = 1; i <= cmd.length; i++) {
            const slice = cmd.slice(0, i);
            later(
              start + i * 11,
              () => {
                shown = [...shown.slice(0, -1), { k: "in", t: slice, caret: i < cmd.length }];
                setTerm([...shown]);
              },
              g,
            );
          }
          t = start + cmd.length * 11 + 100;
          return;
        }
        t += 70;
        const text = step.text;
        const tone = step.tone;
        later(
          t,
          () => {
            if (text.includes("Fast Mode pass") || text.includes("tokens in")) setCliPhase("trim");
            shown = [...shown, { k: "out", t: text, tone }];
            setTerm([...shown]);
          },
          g,
        );
        t += tone === "saved" ? 280 : 140;
        if (idx === script.length - 1) {
          later(t + 1000, onDone, g);
        }
      });
    },
    [later],
  );

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  const playIde = useCallback(
    (g: number, onDone: () => void) => {
      setScene("ide");
      setPhase("scan");
      setProgress(0);
      setTokensOut(tokensIn);
      setAfterText("");
      setActivity([{ t: IDE_STEPS[0].t, tone: IDE_STEPS[0].tone }]);
      setBeamY(0);
      if (activityScrollRef.current) activityScrollRef.current.scrollTop = 0;

      // Panels stay mounted; wait one frame so layout/scrollHeight is ready
      // after CLI→IDE (avoids null/zero-height scan that made the switch feel broken).
      const kick = () => {
        if (g !== gen.current) return;
        const el = beforeRef.current;
        if (el) el.scrollTop = 0;

        const max = el ? Math.max(0, el.scrollHeight - el.clientHeight) : 0;
        const t0 = performance.now();
        const scanDur = 2400;
        const step = (now: number) => {
          if (g !== gen.current) return;
          if (pausedRef.current) {
            raf.current = requestAnimationFrame(step);
            return;
          }
          const p = Math.min(1, (now - t0) / scanDur);
          const eased = 0.5 - 0.5 * Math.cos(Math.PI * p);
          if (el) el.scrollTop = max * eased;
          setBeamY(eased * 100);
          if (p < 1) raf.current = requestAnimationFrame(step);
        };
        raf.current = requestAnimationFrame(step);
      };
      requestAnimationFrame(() => {
        if (g !== gen.current) return;
        requestAnimationFrame(kick);
      });

      // ~520ms per line - readable pacing, ~8s through the full activity log
      const stepGap = 520;
      const stepStart = 450;
      IDE_STEPS.forEach((s, i) => {
        later(
          stepStart + i * stepGap,
          () => {
            setActivity((a) => [...a.slice(0, i), { t: s.t, tone: s.tone }]);
            if (i === 3) setPhase("trim");
          },
          g,
        );
      });

      const trimAt = stepStart + 3 * stepGap; // when classify finishes
      later(
        trimAt,
        () => {
          if (raf.current != null) cancelAnimationFrame(raf.current);
          setPhase("trim");
          const t0 = performance.now();
          const dur = 4200;
          const step = (now: number) => {
            if (g !== gen.current) return;
            if (pausedRef.current) {
              raf.current = requestAnimationFrame(step);
              return;
            }
            const p = Math.min(1, (now - t0) / dur);
            const eased = 1 - (1 - p) * (1 - p);
            setProgress(eased);
            setTokensOut(Math.round(tokensIn + (tokensTarget - tokensIn) * eased));
            setAfterText(AFTER.slice(0, Math.floor(AFTER.length * eased)));
            setBeamY(eased * 100);
            if (p < 1) raf.current = requestAnimationFrame(step);
            else {
              setAfterText(AFTER);
              setTokensOut(tokensTarget);
              setProgress(1);
              setPhase("done");
            }
          };
          raf.current = requestAnimationFrame(step);
        },
        g,
      );

      // Last activity line, then ~1s hold on the finished frame before switching
      later(stepStart + (IDE_STEPS.length - 1) * stepGap + 1000, onDone, g);
    },
    [later, tokensIn, tokensTarget],
  );

  const run = useCallback(() => {
    const g = ++gen.current;
    clear();
    setTerm([]);

    const prefer = sceneLock.current;
    if (prefer === "terminal") {
      playTerminal(g, () =>
        later(
          1000,
          () => {
            if (g === gen.current) run();
          },
          g,
        ),
      );
      return;
    }
    if (prefer === "ide") {
      playIde(g, () =>
        later(
          1000,
          () => {
            if (g === gen.current) run();
          },
          g,
        ),
      );
      return;
    }
    // Auto showcase: IDE then CLI - ~2s total dwell after each finishes
    playIde(g, () => {
      later(
        1000,
        () => {
          playTerminal(g, () =>
            later(
              1000,
              () => {
                if (g === gen.current) run();
              },
              g,
            ),
          );
        },
        g,
      );
    });
  }, [clear, later, playIde, playTerminal]);

  useEffect(() => {
    run();
    return () => clear();
  }, [run, clear]);

  const onMove = (e: MouseEvent) => {
    const el = stageRef.current;
    if (!el) return;
    const r = el.getBoundingClientRect();
    const nx = (e.clientX - r.left) / r.width - 0.5;
    const ny = (e.clientY - r.top) / r.height - 0.5;
    setTilt({ x: ny * -3, y: nx * 4 });
  };

  const switchScene = (next: Scene) => {
    // Tab clicks sit over the stage - don't let hover-pause freeze the first beats
    ignorePauseUntil.current = performance.now() + 500;
    pausedRef.current = false;
    setPaused(false);
    sceneLock.current = next;
    setScene(next);
    run();
  };

  const pauseIfAllowed = () => {
    if (performance.now() < ignorePauseUntil.current) return;
    setPaused(true);
  };

  const statusLabel =
    scene === "terminal"
      ? cliPhase
      : phase === "scan"
        ? "scanning"
        : phase === "trim"
          ? "compressing"
          : phase === "done"
            ? "ready"
            : "idle";

  return (
    <div className="relative w-full" style={{ perspective: "1600px" }}>
      <div
        ref={stageRef}
        role="presentation"
        onMouseMove={onMove}
        onMouseLeave={() => {
          setPaused(false);
          setTilt({ x: 0, y: 0 });
        }}
        onClick={(e) => {
          if ((e.target as HTMLElement).closest("button")) return;
          sceneLock.current = null;
          pausedRef.current = false;
          setPaused(false);
          run();
        }}
        onKeyDown={(e) => {
          if (e.key !== "Enter" && e.key !== " ") return;
          e.preventDefault();
          sceneLock.current = null;
          pausedRef.current = false;
          setPaused(false);
          run();
        }}
        className="trim-product-frame overflow-hidden rounded-[10px] border border-[var(--trim-border)] bg-[var(--trim-panel)] transition-transform duration-200 ease-out"
        style={{
          transform: `rotateX(${tilt.x}deg) rotateY(${tilt.y}deg)`,
          transformStyle: "preserve-3d",
        }}
      >
        <div className="flex h-10 items-center gap-3 border-b border-[var(--trim-border)] bg-[var(--trim-panel)] px-3">
          <div className="flex gap-[6px]">
            <span className="h-[10px] w-[10px] rounded-full bg-[var(--trim-panel-2)]" />
            <span className="h-[10px] w-[10px] rounded-full bg-[var(--trim-panel-2)]" />
            <span className="h-[10px] w-[10px] rounded-full bg-[var(--trim-panel-2)]" />
          </div>
          <div className="flex min-w-0 flex-1 justify-center">
            <div className="flex h-6 w-full max-w-sm items-center rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel-2)] px-2.5 font-mono text-[11px] text-[var(--trim-subtle)]">
              <span className="truncate">
                {scene === "terminal"
                  ? "trim · local proxy · zsh"
                  : "localhost:8888/v1 · trim proxy"}
              </span>
            </div>
          </div>
          <div className="flex gap-0.5">
            {(
              [
                ["ide", "IDE"],
                ["terminal", "CLI"],
              ] as const
            ).map(([id, label]) => (
              <button
                key={id}
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  switchScene(id);
                }}
                className={cn(
                  "rounded px-2 py-1 font-mono text-[10px] transition",
                  scene === id
                    ? "bg-[var(--trim-panel-2)] text-[var(--trim-fg)]"
                    : "text-[var(--trim-muted)] hover:text-[var(--trim-fg)]",
                )}
              >
                {label}
              </button>
            ))}
          </div>
        </div>

        <div className="flex h-9 items-center gap-3 border-b border-[var(--trim-border)] bg-[var(--trim-panel-2)] px-3 font-mono text-[10px]">
          <span
            className="inline-block h-1.5 w-1.5 rounded-full"
            style={{
              backgroundColor: isSuccess ? STATUS_GREEN : "var(--trim-subtle)",
            }}
          />
          <span className="text-[var(--trim-subtle)]">tokens</span>
          <span className="text-[var(--trim-muted)]">{tokensIn.toLocaleString()}</span>
          <span className="text-[var(--trim-subtle)]">→</span>
          <span className="text-[var(--trim-fg)]">{tokensOut.toLocaleString()}</span>
          {isSuccess ? (
            <span
              className="rounded px-1.5 py-0.5 text-[10px] font-medium"
              style={{ color: STATUS_GREEN, backgroundColor: STATUS_GREEN_BG }}
            >
              {saved}% saved
            </span>
          ) : (
            <span className="text-[var(--trim-subtle)]">{statusLabel}</span>
          )}
          <div className="ml-auto flex items-center gap-3 text-[var(--trim-subtle)]">
            <span className="hidden sm:inline">{paused ? "paused" : "live"}</span>
          </div>
        </div>

        <div className="relative">
          {/* Both stay mounted; inactive is visibility:hidden so refs/scrollHeight stay valid */}
          <div
            className={cn(
              scene === "ide"
                ? "relative"
                : "pointer-events-none invisible absolute inset-0 overflow-hidden",
            )}
            aria-hidden={scene !== "ide"}
            onMouseEnter={pauseIfAllowed}
            onMouseLeave={() => setPaused(false)}
          >
            <div className="relative grid min-h-[420px] lg:grid-cols-[1fr_1fr_0.85fr]">
              <div className="relative border-b border-[var(--trim-border)] lg:border-b-0 lg:border-r">
                <div className="flex h-7 items-center justify-between border-b border-[var(--trim-border)] px-3 font-mono text-[10px] text-[var(--trim-subtle)]">
                  <span>before · noisy context</span>
                  <span>{tokensIn.toLocaleString()}</span>
                </div>
                <pre
                  ref={beforeRef}
                  className="h-[360px] overflow-auto px-3 py-3 font-mono text-[11px] leading-[1.55] text-[var(--trim-subtle)]"
                >
                  <code>
                    {BEFORE_LINES.map((line) => (
                      <div
                        key={line.id}
                        className={cn(
                          line.text.startsWith("src/") && "text-[var(--trim-muted)]",
                          (line.text.trim().startsWith("export") ||
                            line.text.trim().startsWith("const") ||
                            line.text.trim().startsWith("if") ||
                            line.text.trim().startsWith("return")) &&
                            "text-[var(--trim-muted)]",
                        )}
                      >
                        {line.text || " "}
                      </div>
                    ))}
                  </code>
                </pre>
                {(phase === "scan" || phase === "trim") && (
                  <div
                    className="pointer-events-none absolute inset-x-0 z-10 h-px bg-[var(--trim-fg)]/50"
                    style={{
                      top: `${14 + beamY * 0.7}%`,
                      boxShadow: "0 0 16px 2px color-mix(in srgb, var(--trim-fg) 18%, transparent)",
                    }}
                  />
                )}
              </div>

              <div className="relative border-b border-[var(--trim-border)] lg:border-b-0 lg:border-r">
                <div className="flex h-7 items-center justify-between border-b border-[var(--trim-border)] px-3 font-mono text-[10px] text-[var(--trim-subtle)]">
                  <span>after · what the model sees</span>
                  <span className={phase === "done" ? statusTextClass : "text-[var(--trim-muted)]"}>
                    {tokensOut.toLocaleString()}
                  </span>
                </div>
                <pre className="h-[360px] overflow-auto px-3 py-3 font-mono text-[11px] leading-[1.55] text-[var(--trim-muted)]">
                  <code>
                    {afterText ? (
                      afterText.split("\n").map((text, lineNo) => {
                        const id = `after-${lineNo}`;
                        return (
                          <div
                            key={id}
                            className={cn(
                              (text.includes("→") ||
                                text.includes("pruned") ||
                                text.includes("dropped") ||
                                text.includes("skeleton") ||
                                text.includes("--")) &&
                                "text-[var(--trim-subtle)]",
                              text.includes("−") && statusTextClass,
                              text.startsWith("src/") && "text-[var(--trim-fg)]",
                            )}
                          >
                            {text || " "}
                          </div>
                        );
                      })
                    ) : (
                      <span className="text-[var(--trim-subtle)]">
                        {phase === "idle" ? "waiting…" : " "}
                      </span>
                    )}
                    {phase === "trim" ? (
                      <span className="animate-pulse text-[var(--trim-fg)]">▋</span>
                    ) : null}
                  </code>
                </pre>
                <div className="absolute inset-x-0 bottom-0 h-px bg-[var(--trim-hover)]">
                  <div
                    className="h-full bg-[var(--trim-fg)]/40 transition-[width] duration-75"
                    style={{ width: `${Math.round(progress * 100)}%` }}
                  />
                </div>
              </div>

              <div className="bg-[var(--trim-panel-2)]">
                <div className="flex h-7 items-center border-b border-[var(--trim-border)] px-3 font-mono text-[10px] text-[var(--trim-subtle)]">
                  Fast Mode · live pass
                </div>
                <ul
                  ref={activityScrollRef}
                  className="h-[360px] space-y-2 overflow-auto px-3 py-3 font-mono text-[12px] leading-6"
                >
                  {activity.length === 0 ? (
                    <li className="text-[var(--trim-subtle)]">idle…</li>
                  ) : (
                    activity.map((a) => (
                      <li
                        key={a.t}
                        className={cn(
                          "break-words",
                          a.tone === "saved" && statusTextClass,
                          a.tone === "ok" && "text-[var(--trim-muted)]",
                          a.tone === "dim" && "text-[var(--trim-subtle)]",
                        )}
                      >
                        {a.tone === "saved" ? "→ " : a.tone === "ok" ? "✓ " : "· "}
                        {a.t}
                      </li>
                    ))
                  )}
                </ul>
              </div>
            </div>
          </div>

          <div
            ref={termScrollRef}
            className={cn(
              "h-[460px] overflow-auto bg-[var(--trim-panel-2)] px-4 py-4 font-mono text-[12px] leading-6",
              scene === "terminal"
                ? "relative"
                : "pointer-events-none invisible absolute inset-0 overflow-hidden",
            )}
            aria-hidden={scene !== "terminal"}
            onMouseEnter={pauseIfAllowed}
            onMouseLeave={() => setPaused(false)}
          >
            {term.length === 0 ? (
              <p className="text-[var(--trim-subtle)]">$ _</p>
            ) : (
              term.map((line, i) => {
                if (line.k === "blank") {
                  // biome-ignore lint/suspicious/noArrayIndexKey: append-only demo terminal rows
                  return <div key={i} className="h-2.5" />;
                }
                if (line.k === "in") {
                  return (
                    // biome-ignore lint/suspicious/noArrayIndexKey: append-only demo terminal rows
                    <p key={i} className="text-[var(--trim-fg)]">
                      <span className="text-[var(--trim-subtle)]">$ </span>
                      {line.t}
                      {line.caret ? (
                        <span className="animate-pulse text-[var(--trim-fg)]">▋</span>
                      ) : null}
                    </p>
                  );
                }
                return (
                  <p
                    // biome-ignore lint/suspicious/noArrayIndexKey: append-only demo terminal rows
                    key={i}
                    className={cn(
                      line.tone === "saved" && statusTextClass,
                      line.tone === "ok" && "text-[var(--trim-muted)]",
                      line.tone === "dim" && "text-[var(--trim-subtle)]",
                      line.tone === "label" && "text-[var(--trim-muted)]",
                      !line.tone && "text-[var(--trim-subtle)]",
                    )}
                  >
                    {line.t}
                  </p>
                );
              })
            )}
          </div>
        </div>

        <div className="flex h-7 items-center gap-3 border-t border-[var(--trim-border)] bg-[var(--trim-panel)] px-3 font-mono text-[10px] text-[var(--trim-subtle)]">
          <span
            className="inline-block h-1.5 w-1.5 rounded-full"
            style={{
              backgroundColor: isSuccess ? STATUS_GREEN : "var(--trim-subtle)",
            }}
          />
          <span>
            {scene === "terminal"
              ? "install → login → proxy → trim → status"
              : "scan → drop noise → keep signal → forward"}
          </span>
          <span className="ml-auto hidden text-[var(--trim-subtle)] sm:inline">
            click replay · hover pause · IDE / CLI
          </span>
        </div>
      </div>
    </div>
  );
}

export function LandingHeroCopy({
  site,
  align = "left",
}: {
  site: HeroChrome;
  align?: "left" | "center";
}) {
  const center = align === "center";
  return (
    <div className={cn(center && "mx-auto max-w-[36rem] text-center")}>
      <p className="font-mono text-[11px] tracking-[0.2em] text-[var(--trim-muted)] uppercase">
        {site.eyebrow}
      </p>
      <h1
        className={cn(
          "font-display mt-5 text-[1.5rem] font-medium leading-[1.2] tracking-[-0.025em] text-[var(--trim-fg)] xl:text-[1.75rem]",
          !center && "max-w-[22rem]",
        )}
      >
        {site.headline}
      </h1>
      <p
        className={cn(
          "mt-4 text-[14px] leading-relaxed text-[var(--trim-muted)] xl:text-[15px]",
          center ? "mx-auto max-w-md" : "max-w-[22rem]",
        )}
      >
        {site.tagline}
      </p>
      <div className={cn("mt-8 flex flex-wrap items-center gap-2.5", center && "justify-center")}>
        <Link
          href={site.nav_sign_in_href || ""}
          className="inline-flex h-9 items-center whitespace-nowrap rounded-md bg-[var(--trim-ink)] px-5 text-[13px] font-medium text-[var(--trim-ink-inverse)] transition hover:opacity-90"
        >
          {site.cta_start}
          <span className="ml-1.5 text-[var(--trim-ink-inverse)] opacity-70">→</span>
        </Link>
        <a
          href="#demo"
          className="inline-flex h-9 items-center whitespace-nowrap rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel)] px-4 text-[13px] font-medium text-[var(--trim-fg)] shadow-[var(--trim-card-shadow)] transition hover:border-[var(--trim-border-strong)] hover:bg-[var(--trim-hover)]"
        >
          {site.cta_demo || "See demo"}
        </a>
      </div>
      <a
        href="#install"
        className={cn(
          "mt-4 inline-flex items-center gap-2 font-mono text-[12px] text-[var(--trim-muted)] transition hover:text-[var(--trim-fg)]",
          center && "justify-center",
        )}
      >
        {site.nav_install || site.install_title || "Install"}
        <kbd className="rounded border border-[var(--trim-border)] bg-[var(--trim-bg)] px-1.5 py-0.5 text-[10px] text-[var(--trim-muted)]">
          ⌘I
        </kbd>
      </a>
      {site.hero_note ? (
        <p
          className={cn(
            "mt-6 font-mono text-[11px] leading-relaxed text-[var(--trim-muted)]",
            !center && "max-w-[20rem]",
          )}
        >
          {site.hero_note}
        </p>
      ) : null}
    </div>
  );
}

/** @deprecated use LandingHeroStage + LandingHeroCopy in split layout */
export function LandingHero({ site }: { site: HeroChrome }) {
  return (
    <section className="relative z-10 px-6 pb-16 pt-10">
      <LandingHeroCopy site={site} align="center" />
      <div className="mt-12">
        <LandingHeroStage />
      </div>
    </section>
  );
}
