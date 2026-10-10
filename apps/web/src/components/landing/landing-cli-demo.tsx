"use client";

import {
  CLI_SCRIPT_POWERSHELL,
  CLI_SCRIPT_UNIX,
  type CliTone,
} from "@/components/landing/landing-cli-script";
import { statusSoftBgClass, statusTextClass } from "@/lib/status-color";
import { cn } from "@/lib/utils";
import { useCallback, useEffect, useRef, useState } from "react";

type Shell = "unix" | "powershell";

type Shown =
  | { kind: "in"; text: string; typing: boolean }
  | { kind: "out"; text: string; tone?: CliTone }
  | { kind: "blank" };

export function LandingCliDemo({ compact = false }: { compact?: boolean }) {
  const [shell, setShell] = useState<Shell>("unix");
  const [lines, setLines] = useState<Shown[]>([]);
  const [paused, setPaused] = useState(false);
  const [phase, setPhase] = useState<"install" | "proxy" | "trim" | "status">("install");
  const timers = useRef<number[]>([]);
  const scrollRef = useRef<HTMLDivElement>(null);
  const rootRef = useRef<HTMLDivElement>(null);
  const gen = useRef(0);
  const shellRef = useRef(shell);
  const pausedRef = useRef(false);
  shellRef.current = shell;
  pausedRef.current = paused;

  const clearTimers = useCallback(() => {
    for (const id of timers.current) window.clearTimeout(id);
    timers.current = [];
  }, []);

  const play = useCallback(
    (loop: boolean) => {
      const g = ++gen.current;
      clearTimers();
      setLines([]);
      setPhase("install");
      const script = shellRef.current === "unix" ? CLI_SCRIPT_UNIX : CLI_SCRIPT_POWERSHELL;
      let t = 280;
      let shown: Shown[] = [];

      const schedule = (ms: number, fn: () => void) => {
        timers.current.push(
          window.setTimeout(() => {
            if (g !== gen.current) return;
            if (pausedRef.current) {
              // Reschedule shortly while hovered/paused
              schedule(120, fn);
              return;
            }
            fn();
          }, ms),
        );
      };

      script.forEach((step, idx) => {
        if (step.kind === "blank") {
          t += 180;
          schedule(t, () => {
            shown = [...shown, { kind: "blank" }];
            setLines([...shown]);
          });
          return;
        }

        if (step.kind === "in") {
          t += step.delay || 220;
          const startAt = t;
          const cmd = step.text;
          schedule(startAt, () => {
            if (cmd.startsWith("trim start")) setPhase("proxy");
            if (cmd.startsWith("trim status")) setPhase("status");
            shown = [...shown, { kind: "in", text: "", typing: true }];
            setLines([...shown]);
          });
          for (let i = 1; i <= step.text.length; i++) {
            const slice = step.text.slice(0, i);
            schedule(startAt + i * 14, () => {
              shown = [
                ...shown.slice(0, -1),
                { kind: "in", text: slice, typing: i < step.text.length },
              ];
              setLines([...shown]);
            });
          }
          t = startAt + step.text.length * 14 + 140;
          return;
        }

        t += 90;
        const outText = step.text;
        const outTone = step.tone;
        schedule(t, () => {
          if (outText.includes("scan context")) setPhase("trim");
          shown = [...shown, { kind: "out", text: outText, tone: outTone }];
          setLines([...shown]);
        });
        t += step.tone === "saved" ? 320 : 160;

        if (idx === script.length - 1) {
          schedule(t + 2200, () => {
            if (loop) play(true);
          });
        }
      });
    },
    [clearTimers],
  );

  useEffect(() => {
    const el = rootRef.current;
    if (!el) return;
    const io = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) play(true);
        else clearTimers();
      },
      { threshold: 0.2 },
    );
    io.observe(el);
    return () => {
      io.disconnect();
      clearTimers();
    };
  }, [play, clearTimers]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    play(true);
  }, [shell, play]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: intentional when dependency values change
  useEffect(() => {
    if (scrollRef.current) scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
  }, [lines]);

  const prompt = shell === "unix" ? "$" : "PS>";
  const phaseLabel =
    phase === "install"
      ? "install"
      : phase === "proxy"
        ? "proxy"
        : phase === "trim"
          ? "trim"
          : "status";

  const panel = (
    <div
      ref={rootRef}
      className="trim-product-frame overflow-hidden rounded-[10px] border border-[var(--trim-border)] bg-[var(--trim-panel)]"
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
    >
      {/* Title bar */}
      <div className="flex h-9 items-center gap-3 border-b border-[var(--trim-border)] px-3">
        <div className="flex gap-[5px]">
          <span className="h-[9px] w-[9px] rounded-full bg-[var(--trim-panel-2)]" />
          <span className="h-[9px] w-[9px] rounded-full bg-[var(--trim-panel-2)]" />
          <span className="h-[9px] w-[9px] rounded-full bg-[var(--trim-panel-2)]" />
        </div>
        <div className="flex min-w-0 flex-1 justify-center">
          <span className="truncate font-mono text-[11px] text-[var(--trim-subtle)]">
            trim · local proxy · {shell === "unix" ? "zsh" : "powershell"}
          </span>
        </div>
        <span
          className={cn(
            "rounded px-1.5 py-0.5 font-mono text-[10px]",
            phase === "status" || phase === "trim"
              ? cn(statusTextClass, statusSoftBgClass)
              : "bg-[var(--trim-hover)] text-[var(--trim-subtle)]",
          )}
        >
          {phaseLabel}
        </span>
      </div>

      {/* Shell tabs */}
      <div className="flex items-center gap-0 border-b border-[var(--trim-border)] px-2">
        {(
          [
            ["unix", "macOS / Linux"],
            ["powershell", "Windows"],
          ] as const
        ).map(([id, label]) => {
          const active = shell === id;
          return (
            <button
              key={id}
              type="button"
              onClick={() => setShell(id)}
              className={cn(
                "relative shrink-0 px-2.5 py-2 font-mono text-[11px] transition",
                active
                  ? "text-[var(--trim-fg)]"
                  : "text-[var(--trim-muted)] hover:text-[var(--trim-fg)]",
              )}
            >
              {label}
              {active ? (
                <span className="absolute inset-x-1.5 bottom-0 h-px bg-[var(--trim-fg)]/80" />
              ) : null}
            </button>
          );
        })}
        <span className="ml-auto pr-2 font-mono text-[10px] text-[var(--trim-subtle)]">
          {paused ? "paused" : "live"}
        </span>
      </div>

      <div
        ref={scrollRef}
        className="h-[320px] overflow-auto px-3.5 py-3 font-mono text-[11.5px] leading-[1.55] text-[var(--trim-muted)] sm:h-[360px]"
      >
        {lines.length === 0 ? (
          <p className="text-[var(--trim-subtle)]">waiting…</p>
        ) : (
          lines.map((line, i) => {
            if (line.kind === "blank") {
              // biome-ignore lint/suspicious/noArrayIndexKey: append-only demo log rows
              return <div key={`b-${i}`} className="h-2.5" />;
            }
            if (line.kind === "in") {
              return (
                // biome-ignore lint/suspicious/noArrayIndexKey: append-only demo log rows
                <p key={`in-${i}`} className="text-[var(--trim-fg)]">
                  <span className="text-[var(--trim-subtle)]">{prompt} </span>
                  {line.text}
                  {line.typing ? (
                    <span className="animate-pulse text-[var(--trim-fg)]">▋</span>
                  ) : null}
                </p>
              );
            }
            return (
              <p
                // biome-ignore lint/suspicious/noArrayIndexKey: append-only demo log rows
                key={`out-${i}`}
                className={cn(
                  line.tone === "saved" && statusTextClass,
                  line.tone === "ok" && "text-[var(--trim-muted)]",
                  line.tone === "dim" && "text-[var(--trim-subtle)]",
                  line.tone === "label" && "text-[var(--trim-muted)]",
                  line.tone === "warn" && "text-amber-400/90",
                  !line.tone && "text-[var(--trim-muted)]",
                )}
              >
                {line.text}
              </p>
            );
          })
        )}
      </div>

      <div className="flex h-7 items-center gap-2 border-t border-[var(--trim-border)] px-3 font-mono text-[10px] text-[var(--trim-subtle)]">
        <span
          className={cn(
            "inline-block h-1.5 w-1.5 rounded-full",
            phase === "status" || phase === "trim"
              ? "bg-[var(--trim-status)]"
              : "bg-[var(--trim-subtle)]",
          )}
        />
        <span>install → proxy → trim → meter</span>
        <span className="ml-auto hidden sm:inline">hover pause · switch shell</span>
      </div>
    </div>
  );

  if (compact) return panel;

  return (
    <section id="cli-demo" className="w-full px-5 py-16 sm:px-8 xl:px-12">
      <div className="max-w-2xl">
        <h2 className="font-display text-xl font-semibold tracking-tight text-[var(--trim-fg)] sm:text-2xl">
          Install once. Watch Trim come alive.
        </h2>
        <p className="mt-3 text-sm text-[var(--trim-muted)] sm:text-base">
          Live CLI: install, start the local proxy, trim a Composer request, read the meter.
        </p>
      </div>
      <div className="mt-8 max-w-3xl">{panel}</div>
    </section>
  );
}
