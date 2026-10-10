"use client";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import * as React from "react";

/** Backend-driven skip/limit controls. Labels and skips come only from API meta. */
export type SkipPaginationMeta = {
  skip?: number;
  limit?: number;
  page?: number;
  total_pages?: number;
  prev_skip?: number | null;
  next_skip?: number | null;
  first_skip?: number | null;
  last_skip?: number | null;
  has_more?: boolean;
  prev_action_label?: string;
  next_action_label?: string;
  first_action_label?: string;
  last_action_label?: string;
  first_pending_label?: string;
  prev_pending_label?: string;
  next_pending_label?: string;
  last_pending_label?: string;
  skip_to_label?: string;
  skip_to_action_label?: string;
  skip_to_pending_label?: string;
  skip_to_invalid?: string;
  /** Backend OFFSET per 1-based page (page_skips[page-1]). Required for skip-to. */
  page_skips?: number[];
  /** Canonical field from pagination.BuildMeta */
  page_summary?: string;
  /** Legacy alias; prefer page_summary */
  summary_label?: string;
};

type NavAction = "first" | "prev" | "next" | "last" | "skip";

export function SkipPagination({
  meta,
  onPrev,
  onNext,
  disabled,
  className,
  inputId,
}: {
  meta?: SkipPaginationMeta | null;
  onPrev: (skip: number) => void;
  onNext: (skip: number) => void;
  disabled?: boolean;
  className?: string;
  /** Unique id when more than one skip-to form is on the page. */
  inputId?: string;
}) {
  const generatedSkipId = React.useId();
  const skipInputId = inputId?.trim() || generatedSkipId;
  const [skipPage, setSkipPage] = React.useState("");
  const [skipError, setSkipError] = React.useState("");
  const [pendingAction, setPendingAction] = React.useState<NavAction | null>(null);

  React.useEffect(() => {
    if (typeof meta?.page === "number" && meta.page >= 1) {
      setSkipPage(String(meta.page));
      setSkipError("");
    }
  }, [meta?.page]);

  React.useEffect(() => {
    if (!disabled) {
      setPendingAction(null);
    }
  }, [disabled]);

  const applySkip = (skip: number, action: NavAction) => {
    setPendingAction(action);
    onNext(skip);
  };

  const firstOk = meta != null && typeof meta.first_skip === "number";
  const lastOk = meta != null && typeof meta.last_skip === "number";
  const prevOk = meta != null && typeof meta.prev_skip === "number";
  const nextOk = meta != null && Boolean(meta.has_more) && typeof meta.next_skip === "number";
  const summary = meta?.page_summary || meta?.summary_label;
  const canSkipTo =
    Boolean(meta?.skip_to_label?.trim()) &&
    Boolean(meta?.skip_to_action_label?.trim()) &&
    Array.isArray(meta?.page_skips) &&
    (meta?.page_skips?.length ?? 0) > 0 &&
    typeof meta.total_pages === "number" &&
    meta.total_pages > 0 &&
    meta.page_skips?.length === meta.total_pages;

  function submitSkipTo() {
    if (!canSkipTo || !meta?.page_skips) return;
    const page = Number.parseInt(skipPage, 10);
    const max = meta.total_pages ?? 0;
    if (!Number.isInteger(page) || page < 1 || page > max) {
      setSkipError(meta.skip_to_invalid || "");
      return;
    }
    const skip = meta.page_skips[page - 1];
    if (typeof skip !== "number" || skip < 0) {
      setSkipError(meta.skip_to_invalid || "");
      return;
    }
    setSkipError("");
    applySkip(skip, "skip");
  }

  const navLoading = (action: NavAction, pendingLabel?: string) =>
    Boolean(disabled && pendingAction === action && pendingLabel?.trim());

  return (
    <div className={cn("flex flex-wrap items-center gap-2", className)}>
      {summary ? <span className="mr-auto text-xs text-muted-foreground">{summary}</span> : null}
      {meta?.first_action_label ? (
        <Button
          size="sm"
          variant="outline"
          disabled={disabled || !firstOk}
          isLoading={navLoading("first", meta.first_pending_label)}
          pendingLabel={meta.first_pending_label || undefined}
          onClick={() => {
            if (typeof meta?.first_skip === "number") {
              applySkip(meta.first_skip, "first");
            }
          }}
        >
          {meta.first_action_label}
        </Button>
      ) : null}
      {meta?.prev_action_label ? (
        <Button
          size="sm"
          variant="outline"
          disabled={disabled || !prevOk}
          isLoading={navLoading("prev", meta.prev_pending_label)}
          pendingLabel={meta.prev_pending_label || undefined}
          onClick={() => {
            if (typeof meta?.prev_skip === "number") {
              setPendingAction("prev");
              onPrev(meta.prev_skip);
            }
          }}
        >
          {meta.prev_action_label}
        </Button>
      ) : null}
      {canSkipTo ? (
        <form
          className="flex items-center gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            submitSkipTo();
          }}
        >
          <label className="sr-only" htmlFor={skipInputId}>
            {meta?.skip_to_label}
          </label>
          <Input
            id={skipInputId}
            type="number"
            inputMode="numeric"
            min={1}
            max={meta?.total_pages}
            className="h-8 w-16"
            value={skipPage}
            disabled={disabled}
            placeholder={meta?.skip_to_label?.trim() || ""}
            aria-invalid={Boolean(skipError) || undefined}
            onChange={(e) => {
              setSkipPage(e.target.value);
              setSkipError("");
            }}
          />
          <Button
            type="submit"
            size="sm"
            variant="outline"
            disabled={disabled || !meta?.skip_to_action_label}
            isLoading={navLoading("skip", meta?.skip_to_pending_label)}
            pendingLabel={meta?.skip_to_pending_label || undefined}
          >
            {meta?.skip_to_action_label}
          </Button>
        </form>
      ) : null}
      {meta?.next_action_label ? (
        <Button
          size="sm"
          variant="outline"
          disabled={disabled || !nextOk}
          isLoading={navLoading("next", meta.next_pending_label)}
          pendingLabel={meta.next_pending_label || undefined}
          onClick={() => {
            if (typeof meta?.next_skip === "number") {
              applySkip(meta.next_skip, "next");
            }
          }}
        >
          {meta.next_action_label}
        </Button>
      ) : null}
      {meta?.last_action_label ? (
        <Button
          size="sm"
          variant="outline"
          disabled={disabled || !lastOk}
          isLoading={navLoading("last", meta.last_pending_label)}
          pendingLabel={meta.last_pending_label || undefined}
          onClick={() => {
            if (typeof meta?.last_skip === "number") {
              applySkip(meta.last_skip, "last");
            }
          }}
        >
          {meta.last_action_label}
        </Button>
      ) : null}
      {skipError ? <span className="w-full text-xs text-destructive">{skipError}</span> : null}
    </div>
  );
}
