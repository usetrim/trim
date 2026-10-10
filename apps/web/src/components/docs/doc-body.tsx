"use client";

import { CopyableCode } from "@/components/ui/copy-button";
import { cn } from "@/lib/utils";
import type { DocBlock } from "@/lib/docs/content";

export function DocBody({ blocks }: { blocks: DocBlock[] }) {
  return (
    <div className="docs-prose space-y-5 text-[15px] leading-7 text-[var(--trim-muted)]">
      {blocks.map((block, i) => {
        const key = `${block.type}-${i}`;
        if (block.type === "p") {
          return (
            <p key={key} className="text-[var(--trim-muted)]">
              {block.text}
            </p>
          );
        }
        if (block.type === "h2") {
          return (
            <h2
              key={key}
              id={block.id}
              className="scroll-mt-24 pt-4 text-xl font-semibold tracking-tight text-[var(--trim-fg)]"
            >
              {block.text}
            </h2>
          );
        }
        if (block.type === "h3") {
          return (
            <h3
              key={key}
              id={block.id}
              className="scroll-mt-24 pt-2 text-base font-semibold text-[var(--trim-fg)]"
            >
              {block.text}
            </h3>
          );
        }
        if (block.type === "ul") {
          return (
            <ul key={key} className="list-disc space-y-2 pl-5">
              {block.items.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          );
        }
        if (block.type === "ol") {
          return (
            <ol key={key} className="list-decimal space-y-2 pl-5">
              {block.items.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ol>
          );
        }
        if (block.type === "code") {
          return <CopyableCode key={key} code={block.code} language={block.language} />;
        }
        if (block.type === "callout") {
          return (
            <div
              key={key}
              className="rounded-lg border border-[var(--trim-border)] bg-[var(--trim-bg)] px-4 py-3 shadow-[var(--trim-card-shadow)]"
            >
              {block.title ? (
                <p className="text-sm font-semibold text-[var(--trim-fg)]">{block.title}</p>
              ) : null}
              <p className={cn("text-sm leading-relaxed", block.title && "mt-1")}>{block.text}</p>
            </div>
          );
        }
        return (
          <p
            key={key}
            className="rounded-md border border-[var(--trim-border)] px-3 py-2 text-sm text-[var(--trim-subtle)]"
          >
            {block.text}
          </p>
        );
      })}
    </div>
  );
}
