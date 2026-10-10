"use client";

import { cn } from "@/lib/utils";
import { Check, Copy } from "lucide-react";
import { useCallback, useState } from "react";

type Props = {
  value: string;
  className?: string;
  /** Accessible name for the control. */
  label?: string;
  /** Optional compact icon-only button for inline code chrome. */
  size?: "sm" | "md";
};

export function CopyButton({ value, className, label = "Copy", size = "sm" }: Props) {
  const [copied, setCopied] = useState(false);

  const onCopy = useCallback(async () => {
    const text = value.trim();
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      setCopied(false);
    }
  }, [value]);

  return (
    <button
      type="button"
      onClick={() => void onCopy()}
      aria-label={copied ? "Copied" : label}
      title={copied ? "Copied" : label}
      className={cn(
        "inline-flex items-center gap-1.5 rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel)] font-medium text-[var(--trim-muted)] transition hover:border-[var(--trim-border-strong)] hover:bg-[var(--trim-hover)] hover:text-[var(--trim-fg)]",
        size === "sm" ? "h-7 px-2 text-[11px]" : "h-8 px-2.5 text-[12px]",
        className,
      )}
    >
      {copied ? (
        <Check className="h-3.5 w-3.5 text-[var(--trim-status)]" />
      ) : (
        <Copy className="h-3.5 w-3.5" />
      )}
      <span className="hidden sm:inline">{copied ? "Copied" : "Copy"}</span>
    </button>
  );
}

export function CopyableCode({
  code,
  language,
  className,
}: {
  code: string;
  language?: string;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "overflow-hidden rounded-lg border border-[var(--trim-border)] bg-[var(--trim-code)]",
        className,
      )}
    >
      <div className="flex items-center justify-between gap-2 border-b border-[var(--trim-border)] px-3 py-1.5">
        <span className="font-mono text-[10px] uppercase tracking-[0.14em] text-[var(--trim-subtle)]">
          {language || "code"}
        </span>
        <CopyButton value={code} label="Copy code" />
      </div>
      <pre className="overflow-x-auto p-4 font-mono text-[12.5px] leading-6 text-[var(--trim-fg)]">
        <code>{code}</code>
      </pre>
    </div>
  );
}
