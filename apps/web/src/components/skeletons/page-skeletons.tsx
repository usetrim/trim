import { TrimWordmark } from "@/components/brand/trim-wordmark";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";
import type { ReactNode } from "react";

/** Stable ids for fixed-length shimmer slots. Placeholders never reorder. */
function shimmerKeys(prefix: string, count: number): string[] {
  return Array.from({ length: Math.max(0, count) }, (_, i) => `${prefix}-${i}`);
}

/** Public header auth slot while session settles (profile chip, not Sign-in). */
export type PublicAuthSkeletonSlot = "signin" | "profile" | "pending";

function PublicAuthSlotSkeleton({
  slot = "pending",
  signInLabel,
  className,
}: {
  slot?: PublicAuthSkeletonSlot;
  signInLabel?: string;
  className?: string;
}) {
  if (slot === "signin" && signInLabel?.trim()) {
    return (
      <span
        className={
          className ||
          "inline-flex h-8 shrink-0 items-center whitespace-nowrap rounded-md bg-[var(--trim-ink)] px-3 text-[12px] text-[var(--trim-ink-inverse)]"
        }
      >
        {signInLabel}
      </span>
    );
  }
  return <Skeleton className={cn("h-9 w-36 shrink-0 rounded-md", className)} />;
}

/**
 * Chrome-only shell while the gate settles (sidebar + top bar).
 * Main must stay empty - inventing a fake page body causes a wrong→correct
 * shimmer flash when the route's real page skeleton mounts.
 */
export function DashboardShellSkeleton({ children }: { children?: ReactNode }) {
  return (
    <div className="flex min-h-screen bg-[var(--trim-bg)] text-[var(--trim-fg)]">
      <aside className="sticky top-0 hidden h-screen w-60 shrink-0 flex-col self-start overflow-hidden border-r border-[var(--trim-border)] bg-[var(--trim-panel)] md:flex">
        <div className="space-y-2 border-b border-[var(--trim-border)] px-4 py-4">
          <TrimWordmark size="sm" alt="Trim" />
          <Skeleton className="h-3 w-36" />
        </div>
        <div className="flex flex-col gap-1 p-2">
          {/* Match live console: Dashboard, Traces, Receipts, Enterprise, Team, Settings */}
          {shimmerKeys("nav", 6).map((id) => (
            <Skeleton key={id} className="h-9 w-full rounded-md" />
          ))}
        </div>
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="sticky top-0 z-40 flex h-14 items-center gap-2 border-b border-[var(--trim-border)] bg-[var(--trim-panel)] px-3 sm:px-6">
          <Skeleton className="h-9 w-9 rounded-md md:hidden" />
          <div className="md:hidden">
            <TrimWordmark size="sm" alt="Trim" />
          </div>
          <div className="ml-auto flex items-center gap-2">
            {/* Match live header: theme (h-8) + bell (h-8 sm button) + profile menu (h-9). */}
            <Skeleton className="h-8 w-8 shrink-0 rounded-md" />
            <Skeleton className="h-8 w-8 shrink-0 rounded-md" />
            <Skeleton className="h-9 w-36 shrink-0 rounded-md" />
          </div>
        </header>
        <main className="flex-1 p-3 sm:p-4 md:p-6">
          <div className="mx-auto w-full max-w-[1400px]">{children}</div>
        </main>
      </div>
    </div>
  );
}

/** Toolbar action button placeholder (matches Button h-9). */
export function ToolbarButtonSkeleton({ className }: { className?: string }) {
  return <Skeleton aria-hidden className={className ?? "h-9 w-28 rounded-md"} />;
}

/** Single metric value shimmer matching text-2xl figure height. */
export function MetricValueSkeleton({ className }: { className?: string }) {
  return <Skeleton aria-hidden className={className ?? "h-8 w-24"} />;
}

/** Metric cards matching dashboard rows (4 primary or 3 secondary). */
export function MetricCardsSkeleton({
  labels,
  showCreditsBar = true,
  showHintLine = false,
}: {
  /** Backend chrome labels in display order; required length (no invent of 4 empty cards). */
  labels: Array<string | undefined>;
  showCreditsBar?: boolean;
  /** Extra line under the value (tab acceptance hint, etc.). */
  showHintLine?: boolean;
}) {
  if (!labels.length) {
    return null;
  }
  const cols = labels;
  const cards = cols.map((label, i) => ({
    id: `metric-${i}`,
    label,
    withCreditsBar: showCreditsBar && cols.length >= 4 && i === 1,
    withHintLine: showHintLine && i === 0,
  }));
  const gridClass =
    cols.length <= 3
      ? "grid gap-4 sm:grid-cols-2 lg:grid-cols-3"
      : "grid gap-4 sm:grid-cols-2 xl:grid-cols-4";
  return (
    <div className={gridClass}>
      {cards.map((card) => (
        <Card key={card.id}>
          <CardHeader>
            {card.label ? (
              <CardTitle className="text-sm font-medium text-[var(--trim-muted)]">
                {card.label}
              </CardTitle>
            ) : (
              <Skeleton className="h-4 w-24" />
            )}
          </CardHeader>
          <CardContent className="space-y-3">
            <MetricValueSkeleton />
            {card.withCreditsBar ? <Skeleton className="h-1.5 w-full rounded" /> : null}
            {card.withHintLine ? <Skeleton className="h-3 w-40" /> : null}
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

/** Chart card shell. `variant="usage"` mirrors UsageStackedChart; `outcomes` adds meta line. */
export function ChartSkeleton({
  title,
  variant = "default",
}: {
  title?: string;
  variant?: "default" | "usage" | "outcomes";
}) {
  if (variant === "usage") {
    return (
      <Card>
        <CardContent className="space-y-4 pt-6">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
            <div className="space-y-2">
              {title ? (
                <CardTitle className="text-base">{title}</CardTitle>
              ) : (
                <Skeleton className="h-5 w-40" />
              )}
              <Skeleton className="h-3 w-56" />
            </div>
            <Skeleton className="h-9 w-36 rounded-md" />
          </div>
          <Skeleton className="h-56 w-full rounded-lg sm:h-64" />
        </CardContent>
      </Card>
    );
  }
  return (
    <Card>
      <CardHeader>
        {title ? <CardTitle>{title}</CardTitle> : <Skeleton className="h-5 w-40" />}
      </CardHeader>
      <CardContent>
        {variant === "outcomes" ? <Skeleton className="mb-3 h-4 w-48" /> : null}
        <Skeleton className="h-64 w-full rounded-lg" />
      </CardContent>
    </Card>
  );
}

/** LOC heatmap shimmer matching LocHeatmap fixed-cell layout. */
export function LocHeatmapSkeleton() {
  const cell = 11;
  const gap = 3;
  const weeks = 12;
  const labelCol = 28;
  const gridWidth = weeks * cell + Math.max(0, weeks - 1) * gap;
  return (
    <Card>
      <CardContent className="space-y-4 pt-6">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="space-y-2">
            <Skeleton className="h-4 w-40" />
            <Skeleton className="h-9 w-24" />
          </div>
          <div className="flex gap-1">
            {shimmerKeys("scope", 3).map((id) => (
              <Skeleton key={id} className="h-8 w-16 rounded-md" />
            ))}
          </div>
        </div>
        <div className="overflow-x-auto pb-1">
          <div className="inline-block min-w-0" style={{ width: labelCol + gridWidth }}>
            <div className="flex" style={{ gap: `${gap}px` }}>
              <div className="flex flex-col" style={{ width: labelCol, gap: `${gap}px` }}>
                {shimmerKeys("wd", 7).map((id) => (
                  <Skeleton
                    key={id}
                    className="rounded-sm"
                    style={{ width: labelCol, height: cell }}
                  />
                ))}
              </div>
              <div
                className="grid"
                style={{
                  width: gridWidth,
                  gridTemplateColumns: `repeat(${weeks}, ${cell}px)`,
                  gridTemplateRows: `repeat(7, ${cell}px)`,
                  gap: `${gap}px`,
                  gridAutoFlow: "column",
                }}
              >
                {shimmerKeys("hm", weeks * 7).map((id) => (
                  <Skeleton key={id} className="rounded-sm" style={{ width: cell, height: cell }} />
                ))}
              </div>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

type ColumnSpec = { label: string; width?: string };

/** Label + control + description slot matching live `Field` (Settings / Team practice). */
function FieldSkeleton({
  label,
  description,
  controlClassName,
  className,
}: {
  label?: string;
  description?: string;
  /** Default matches Input h-10; date-range trigger uses Button h-9. */
  controlClassName?: string;
  className?: string;
}) {
  const title = label?.trim() || "";
  const desc = description?.trim() || "";
  return (
    <div className={cn("space-y-1.5", className)}>
      {title ? (
        <p className="text-sm font-medium leading-none text-[var(--trim-fg)]">{title}</p>
      ) : (
        <Skeleton className="h-4 w-28" />
      )}
      <Skeleton className={controlClassName ?? "h-10 w-full rounded-md"} />
      {desc ? (
        <p className="text-xs text-[var(--trim-muted)]">{desc}</p>
      ) : (
        <Skeleton className="h-3 w-48" />
      )}
    </div>
  );
}

/** First / prev / skip-to / next / last slots matching SkipPagination. */
export function PaginationBarSkeleton({ className }: { className?: string }) {
  return (
    <div className={className ?? "flex flex-wrap items-center gap-2"}>
      <Skeleton className="mr-auto h-3 w-24" />
      <Skeleton className="h-8 w-12" />
      <Skeleton className="h-8 w-14" />
      <Skeleton className="h-8 w-16" />
      <Skeleton className="h-8 w-10" />
      <Skeleton className="h-8 w-14" />
      <Skeleton className="h-8 w-12" />
    </div>
  );
}

/** Nested DataTable body shimmer (no Card). */
export function DataTableRowsSkeleton({
  columns,
  rows,
  showPagination = false,
}: {
  columns: ColumnSpec[];
  rows: number;
  showPagination?: boolean;
}) {
  // Never collapse to header-only on refresh while page size is still unknown.
  const n = rows > 0 ? rows : 8;
  const cells = columns.map((col, i) => ({ ...col, id: `col-${i}` }));
  if (!cells.length) return null;
  return (
    <div className="space-y-4">
      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow className="border-[var(--trim-border)] hover:bg-transparent">
              {cells.map((col) => {
                const header = col.label.trim();
                return (
                  <TableHead key={col.id} className="text-[var(--trim-muted)]">
                    {header ? header : <Skeleton className={col.width ?? "h-4 w-16"} />}
                  </TableHead>
                );
              })}
            </TableRow>
          </TableHeader>
          <TableBody>
            {shimmerKeys("row", n).map((rowId) => (
              <TableRow key={rowId} className="border-[var(--trim-border)]">
                {cells.map((col) => (
                  <TableCell key={`${rowId}-${col.id}`} className="text-[var(--trim-muted)]">
                    <Skeleton className={col.width ?? "h-4 w-20"} />
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      {showPagination ? <PaginationBarSkeleton /> : null}
    </div>
  );
}

export type DedicatedListToolbarChrome = {
  /** Date-range / status-filter field label, or sync button label. */
  primaryLabel?: string;
  primaryDescription?: string;
  searchLabel?: string;
  searchDescription?: string;
};

/** Card-wrapped DataTable shimmer for page-level loading states. */
export function DataTableSkeleton({
  title,
  columns,
  rows,
  showPagination = true,
  toolbar,
  toolbarChrome,
}: {
  title: string;
  columns: ColumnSpec[];
  /** From billing_settings.default_page_size; 0 = header chrome only. */
  rows: number;
  showPagination?: boolean;
  /**
   * Live toolbars mirrored while pending:
   * - dateRange: traces date picker + search
   * - sync: receipts sync + search
   * - statusSearch: enterprise status filter + search
   */
  toolbar?: "dateRange" | "sync" | "statusSearch";
  /** Chrome labels for Field / sync button (Settings-style real text when known). */
  toolbarChrome?: DedicatedListToolbarChrome;
}) {
  const primaryLabel = toolbarChrome?.primaryLabel?.trim() || "";
  const primaryDescription = toolbarChrome?.primaryDescription?.trim() || "";
  const searchLabel = toolbarChrome?.searchLabel?.trim() || "";
  const searchDescription = toolbarChrome?.searchDescription?.trim() || "";

  return (
    <Card className="w-full">
      {toolbar === "dateRange" ? (
        <CardHeader className="flex flex-col gap-3">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            {title ? (
              <CardTitle className="text-base">{title}</CardTitle>
            ) : (
              <Skeleton className="h-5 w-32" />
            )}
            <FieldSkeleton
              label={primaryLabel}
              description={primaryDescription}
              controlClassName="h-9 w-full rounded-md sm:w-56"
              className="w-full sm:w-auto"
            />
          </div>
          <FieldSkeleton
            label={searchLabel}
            description={searchDescription}
            className="w-full sm:max-w-md"
          />
        </CardHeader>
      ) : toolbar === "statusSearch" ? (
        <CardHeader className="flex flex-col gap-3">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            {title ? (
              <CardTitle className="text-base">{title}</CardTitle>
            ) : (
              <Skeleton className="h-5 w-32" />
            )}
            <FieldSkeleton
              label={primaryLabel}
              description={primaryDescription}
              className="w-full sm:w-auto sm:min-w-[12rem]"
            />
          </div>
          <FieldSkeleton
            label={searchLabel}
            description={searchDescription}
            className="w-full sm:max-w-md"
          />
        </CardHeader>
      ) : (
        <CardHeader className="flex flex-col gap-3">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            {title ? (
              <CardTitle className="text-base">{title}</CardTitle>
            ) : (
              <Skeleton className="h-5 w-32" />
            )}
            {toolbar === "sync" ? (
              primaryLabel ? (
                <span className="inline-flex h-8 items-center rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel)] px-3 text-xs text-[var(--trim-muted)] shadow-[var(--trim-card-shadow)]">
                  {primaryLabel}
                </span>
              ) : (
                <Skeleton className="h-8 w-32 rounded-md" />
              )
            ) : null}
          </div>
          {toolbar === "sync" ? (
            <FieldSkeleton
              label={searchLabel}
              description={searchDescription}
              className="w-full sm:max-w-md"
            />
          ) : null}
        </CardHeader>
      )}
      <CardContent>
        <DataTableRowsSkeleton columns={columns} rows={rows} showPagination={showPagination} />
      </CardContent>
    </Card>
  );
}

/**
 * Dedicated list pages (Traces / Receipts / Enterprise): page eyebrow + h1 + table card.
 * Matches live clients' `space-y-6` header + Card layout (not home Dashboard charts).
 */
export function DedicatedListPageSkeleton({
  title,
  eyebrow,
  columns,
  rows,
  toolbar,
  toolbarChrome,
  embedded = true,
}: {
  title: string;
  /** Live `page_eyebrow` / auth skeleton chrome. */
  eyebrow?: string;
  columns: ColumnSpec[];
  rows: number;
  toolbar?: "dateRange" | "sync" | "statusSearch";
  toolbarChrome?: DedicatedListToolbarChrome;
  embedded?: boolean;
}) {
  const eyebrowText = eyebrow?.trim() || "";
  return (
    <div className="w-full space-y-6">
      <div className={embedded ? "min-w-0" : "mb-6 min-w-0"}>
        {eyebrowText ? (
          <p className="text-sm text-[var(--trim-muted)]">{eyebrowText}</p>
        ) : (
          <Skeleton className="h-4 w-20" />
        )}
        {title ? (
          <h1 className="text-2xl font-semibold tracking-tight text-[var(--trim-fg)] sm:text-3xl">
            {title}
          </h1>
        ) : (
          <Skeleton className="mt-0 h-8 w-48 sm:h-9" />
        )}
      </div>
      <DataTableSkeleton
        title={title}
        columns={columns}
        rows={rows}
        toolbar={toolbar}
        toolbarChrome={toolbarChrome}
      />
    </div>
  );
}

export type DashboardSkeletonChrome = {
  traces_title?: string;
  receipts_title?: string;
  chart_token_series_title?: string;
  chart_models_title?: string;
  chart_outcomes_title?: string;
  chart_modes_title?: string;
  metric_plan_label?: string;
  metric_credits_label?: string;
  metric_remaining_label?: string;
  metric_tokens_saved_label?: string;
  metric_tab_label?: string;
  metric_lines_added_label?: string;
  metric_lines_deleted_label?: string;
  traces_col_when?: string;
  traces_col_model?: string;
  traces_col_mode?: string;
  traces_col_tokens?: string;
  traces_col_latency?: string;
  receipts_col_date?: string;
  receipts_col_invoice?: string;
  receipts_col_status?: string;
  receipts_col_total?: string;
  receipts_col_view?: string;
  /** Header action slots: only shimmer when live chrome would show them. */
  header_primary?: boolean;
  header_topup?: boolean;
  header_sign_out?: boolean;
  header_portal?: boolean;
  header_team?: boolean;
  header_settings?: boolean;
  header_notifications?: boolean;
};

/** Full dashboard body while auth/session boots. Pass chrome when available. */
export function DashboardPageSkeleton({
  chrome,
  tableRows: _tableRows = 0,
  embedded = false,
}: {
  chrome?: DashboardSkeletonChrome;
  /** @deprecated Home dashboard no longer embeds traces/receipts tables; kept for call-site compat. */
  tableRows?: number;
  /** When true, skip legacy sticky header (shell already provides chrome). */
  embedded?: boolean;
}) {
  void _tableRows;
  return (
    <div className="w-full">
      {embedded ? null : (
        <header className="sticky top-0 z-40 border-b border-[var(--trim-border)] bg-[var(--trim-bg)]/95 backdrop-blur supports-[backdrop-filter]:bg-[var(--trim-bg)]/80">
          <div className="mx-auto flex w-full max-w-[1400px] flex-col gap-4 px-3 py-4 sm:flex-row sm:flex-wrap sm:items-end sm:justify-between sm:px-6">
            <div className="min-w-0 space-y-2">
              <Skeleton className="h-4 w-20" />
              <Skeleton className="h-8 w-64 sm:h-9" />
              <Skeleton className="h-4 w-80 max-w-full" />
            </div>
            <div className="flex flex-wrap items-center gap-2">
              {chrome?.header_primary ? <Skeleton className="h-9 w-24" /> : null}
              {chrome?.header_topup ? <Skeleton className="h-9 w-24" /> : null}
              {chrome?.header_notifications ? <Skeleton className="h-8 w-8 rounded-md" /> : null}
              <Skeleton className="h-8 w-8 rounded-md" />
              {chrome?.header_sign_out ? <Skeleton className="h-9 w-24 rounded-md" /> : null}
              <div className="hidden flex-wrap items-center gap-2 md:flex">
                {chrome?.header_portal ? <Skeleton className="h-9 w-24" /> : null}
                {chrome?.header_team ? <Skeleton className="h-9 w-20" /> : null}
                {chrome?.header_settings ? <Skeleton className="h-9 w-24" /> : null}
              </div>
              {chrome?.header_portal || chrome?.header_team || chrome?.header_settings ? (
                <Skeleton className="h-9 w-9 rounded-md md:hidden" />
              ) : null}
            </div>
          </div>
        </header>
      )}

      <div
        className={
          embedded ? "w-full space-y-6" : "mx-auto w-full max-w-[1400px] px-3 py-6 sm:px-6 sm:py-8"
        }
      >
        {embedded ? (
          <div className="flex flex-col gap-4 sm:flex-row sm:flex-wrap sm:items-end sm:justify-between">
            <div className="min-w-0 space-y-2">
              <Skeleton className="h-4 w-20" />
              <Skeleton className="h-8 w-64 sm:h-9" />
              <Skeleton className="h-4 w-80 max-w-full" />
            </div>
            <div className="flex flex-wrap items-center gap-2">
              {/* Match live: Upgrade plan + Buy top-up; portal only when paid. */}
              {chrome?.header_primary !== false ? (
                <Skeleton className="h-9 w-28 rounded-md" />
              ) : null}
              {chrome?.header_topup !== false ? <Skeleton className="h-9 w-24 rounded-md" /> : null}
              {chrome?.header_portal ? <Skeleton className="h-9 w-28 rounded-md" /> : null}
            </div>
          </div>
        ) : null}
        <div className={embedded ? "w-full" : undefined}>
          <SubscriptionStatusSkeleton />

          <div className="mt-4">
            <MetricCardsSkeleton
              labels={[
                chrome?.metric_plan_label,
                chrome?.metric_credits_label,
                chrome?.metric_remaining_label,
                chrome?.metric_tokens_saved_label,
              ]}
            />
          </div>

          <div className="mt-4">
            <MetricCardsSkeleton
              labels={[
                chrome?.metric_tab_label,
                chrome?.metric_lines_added_label,
                chrome?.metric_lines_deleted_label,
              ]}
              showCreditsBar={false}
              showHintLine
            />
          </div>

          <div className="mt-8 space-y-8">
            <ChartSkeleton variant="usage" />
            <LocHeatmapSkeleton />
          </div>

          <div className="mt-8">
            <ChartSkeleton title={chrome?.chart_token_series_title} />
          </div>

          <div className="mt-8 grid gap-6 md:grid-cols-2 xl:grid-cols-3">
            <ChartSkeleton title={chrome?.chart_models_title} />
            <ChartSkeleton title={chrome?.chart_outcomes_title} variant="outcomes" />
            <ChartSkeleton title={chrome?.chart_modes_title} />
          </div>

          <Skeleton className="mt-6 h-3 w-64" />

          <Card className="mt-8">
            <CardHeader>
              <Skeleton className="h-4 w-28" />
            </CardHeader>
            <CardContent className="space-y-3">
              <Skeleton className="h-4 w-72 max-w-full" />
              <Skeleton className="h-9 w-36" />
              <Skeleton className="h-4 w-64 max-w-full" />
              <Skeleton className="h-9 w-40" />
              <Skeleton className="h-3 w-48" />
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}

export type TeamSkeletonChrome = {
  workspaces_title?: string;
  members_title?: string;
  invites_title?: string;
};

/** Team page: page header + workspace list + members panel mirrors. */
export function TeamPageSkeleton({
  chrome,
  workspaceRows,
  memberRows,
  embedded = false,
}: {
  chrome?: TeamSkeletonChrome;
  /** From billing/auth default_page_size (no invent). */
  workspaceRows: number;
  memberRows: number;
  /** When true, skip legacy sticky chrome (shell already provides it). */
  embedded?: boolean;
}) {
  const wRows = workspaceRows > 0 ? workspaceRows : 8;
  const mRows = memberRows > 0 ? memberRows : 8;
  return (
    <div className={embedded ? "w-full space-y-6" : "w-full"}>
      {embedded ? (
        <div className="min-w-0 space-y-2">
          <Skeleton className="h-4 w-16" />
          <Skeleton className="h-8 w-40 sm:h-9" />
          <Skeleton className="h-4 w-72 max-w-full" />
        </div>
      ) : (
        <header className="sticky top-0 z-40 border-b border-[var(--trim-border)] bg-[var(--trim-bg)]/95 backdrop-blur supports-[backdrop-filter]:bg-[var(--trim-bg)]/80">
          <div className="mx-auto flex w-full max-w-[1400px] flex-col gap-4 px-3 py-4 sm:flex-row sm:flex-wrap sm:items-end sm:justify-between sm:px-6">
            <div className="min-w-0 space-y-2">
              <Skeleton className="h-4 w-16" />
              <Skeleton className="h-8 w-40 sm:h-9" />
              <Skeleton className="h-4 w-72 max-w-full" />
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <Skeleton className="h-9 w-28" />
              <Skeleton className="h-8 w-8 rounded-md" />
              <Skeleton className="h-8 w-8 rounded-md" />
              <Skeleton className="h-9 w-24 rounded-md" />
            </div>
          </div>
        </header>
      )}
      <div
        className={embedded ? "w-full" : "mx-auto w-full max-w-[1400px] px-3 py-6 sm:px-6 sm:py-8"}
      >
        <div className="grid gap-6 lg:grid-cols-[1fr_1.2fr]">
          <Card>
            <CardHeader>
              {chrome?.workspaces_title ? (
                <CardTitle className="text-base text-[var(--trim-fg)]">
                  {chrome.workspaces_title}
                </CardTitle>
              ) : (
                <Skeleton className="h-5 w-36" />
              )}
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
                <div className="min-w-0 flex-1 space-y-1.5">
                  <Skeleton className="h-4 w-28" />
                  <Skeleton className="h-10 w-full" />
                  <Skeleton className="h-3 w-48" />
                </div>
                <Skeleton className="h-10 w-20" />
              </div>
              <div className="w-full space-y-1.5">
                <Skeleton className="h-4 w-28" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-3 w-48" />
              </div>
              <DataTableRowsSkeleton
                columns={[
                  { label: " ", width: "h-4 w-4" },
                  { label: chrome?.workspaces_title || "", width: "h-8 w-40" },
                  { label: " ", width: "h-8 w-8" },
                ]}
                rows={wRows}
                showPagination
              />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              {chrome?.members_title ? (
                <CardTitle className="text-base text-[var(--trim-fg)]">
                  {chrome.members_title}
                </CardTitle>
              ) : (
                <Skeleton className="h-5 w-24" />
              )}
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
                <div className="min-w-0 flex-1 space-y-1.5">
                  <Skeleton className="h-4 w-28" />
                  <Skeleton className="h-10 w-full" />
                  <Skeleton className="h-3 w-48" />
                </div>
                <Skeleton className="h-10 w-20" />
              </div>
              <div className="w-full space-y-1.5">
                <Skeleton className="h-4 w-28" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-3 w-48" />
              </div>
              <DataTableRowsSkeleton
                columns={[
                  { label: " ", width: "h-4 w-4" },
                  { label: chrome?.members_title || " ", width: "h-4 w-32" },
                  { label: " ", width: "h-4 w-14" },
                  { label: " ", width: "h-8 w-16" },
                  { label: " ", width: "h-8 w-8" },
                ]}
                rows={mRows}
                showPagination
              />
              <div className="border-t border-[var(--trim-border)] pt-4">
                {chrome?.invites_title ? (
                  <p className="mb-2 text-sm font-medium text-[var(--trim-fg)]">
                    {chrome.invites_title}
                  </p>
                ) : (
                  <Skeleton className="mb-2 h-4 w-24" />
                )}
                <div className="mb-3 w-full space-y-1.5">
                  <Skeleton className="h-4 w-28" />
                  <Skeleton className="h-10 w-full" />
                  <Skeleton className="h-3 w-48" />
                </div>
                <DataTableRowsSkeleton
                  columns={[
                    { label: " ", width: "h-4 w-4" },
                    { label: chrome?.invites_title || " ", width: "h-4 w-36" },
                    { label: " ", width: "h-8 w-16" },
                    { label: " ", width: "h-8 w-8" },
                  ]}
                  rows={mRows}
                  showPagination
                />
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}

export function WorkspaceListSkeleton({ rows }: { rows: number }) {
  return (
    <DataTableRowsSkeleton
      columns={[{ label: " ", width: "h-4 w-40" }]}
      rows={rows}
      showPagination
    />
  );
}

export function MemberListSkeleton({ rows }: { rows: number }) {
  return (
    <DataTableRowsSkeleton
      columns={[
        { label: " ", width: "h-4 w-32" },
        { label: " ", width: "h-4 w-14" },
        { label: " ", width: "h-8 w-16" },
      ]}
      rows={rows}
      showPagination
    />
  );
}

/** Pending invites: email + status line + revoke action (no role chip). */
export function InviteListSkeleton({ rows }: { rows: number }) {
  return (
    <DataTableRowsSkeleton
      columns={[
        { label: " ", width: "h-4 w-36" },
        { label: " ", width: "h-8 w-16" },
      ]}
      rows={rows}
      showPagination
    />
  );
}

export type ReceiptSkeletonChrome = {
  col_description?: string;
  col_product?: string;
  col_sku?: string;
  col_qty?: string;
  col_unit?: string;
  col_tax_rate?: string;
  col_amount?: string;
  section_bill_to?: string;
  section_invoice_from?: string;
  section_invoice_details?: string;
  section_transaction?: string;
  section_tax_breakdown?: string;
  section_period?: string;
  section_amount?: string;
  section_status?: string;
  section_payment?: string;
  label_subtotal?: string;
  label_tax?: string;
  label_total?: string;
  label_amount_paid?: string;
  footer?: string;
};

/**
 * Receipt detail shimmer - mirrors ReceiptView tax-invoice article 1:1:
 * toolbar, header band (title+badge+meta | brand), parties grid, invoice details,
 * transaction table with stacked product cell, totals, tax breakdown, footer band.
 */
export function ReceiptDetailSkeleton({
  chrome,
  lineRows,
}: {
  chrome?: ReceiptSkeletonChrome;
  /** Line count from receipt payload when known; 1 while unknown (layout fidelity). */
  lineRows: number;
}) {
  const n = lineRows > 0 ? lineRows : 1;
  const productCol = (chrome?.col_product || chrome?.col_description || "").trim();
  return (
    <div className="mx-auto w-full max-w-[920px] px-3 py-6 sm:px-6 sm:py-10">
      <div className="mb-5 flex flex-wrap items-center justify-between gap-3">
        <Skeleton className="h-8 w-28" />
        <div className="flex gap-2">
          <Skeleton className="h-8 w-28" />
          <Skeleton className="h-8 w-16" />
        </div>
      </div>
      <article data-receipt-print className="bg-card text-foreground">
        <header
          data-receipt-band
          className="bg-muted px-5 py-6 sm:px-8 sm:py-7"
          style={{ backgroundColor: "var(--receipt-band, #f4f4f5)" }}
        >
          <div className="flex flex-col gap-5 sm:flex-row sm:items-start sm:justify-between">
            <div className="min-w-0 space-y-1.5">
              <div className="flex flex-wrap items-center gap-2.5">
                <Skeleton className="h-7 w-36 sm:h-8 sm:w-40" />
                <Skeleton className="h-5 w-12 rounded" />
              </div>
              <Skeleton className="h-4 w-52" />
            </div>
            <div className="flex shrink-0 flex-col items-start gap-1 sm:items-end">
              <Skeleton className="h-8 w-24 sm:h-9" />
              <Skeleton className="h-4 w-16" />
              <Skeleton className="h-3 w-24" />
            </div>
          </div>
        </header>

        <section className="grid gap-10 px-5 py-8 sm:px-8 md:grid-cols-2">
          <div className="min-w-0 space-y-2">
            {chrome?.section_bill_to ? (
              <h2 className="text-[13px] font-bold text-[var(--trim-fg)]">
                {chrome.section_bill_to}
              </h2>
            ) : (
              <Skeleton className="h-4 w-20" />
            )}
            <div className="space-y-1.5">
              <Skeleton className="h-4 w-40" />
              <Skeleton className="h-4 w-56" />
              <Skeleton className="h-4 w-36" />
              <Skeleton className="h-4 w-28" />
              <Skeleton className="mt-2 h-4 w-44" />
            </div>
          </div>
          <div className="min-w-0 space-y-2">
            {chrome?.section_invoice_from ? (
              <h2 className="text-[13px] font-bold text-[var(--trim-fg)]">
                {chrome.section_invoice_from}
              </h2>
            ) : (
              <Skeleton className="h-4 w-24" />
            )}
            <div className="space-y-1.5">
              <Skeleton className="h-4 w-28" />
              <Skeleton className="h-4 w-48" />
              <Skeleton className="h-4 w-40" />
              <Skeleton className="h-4 w-20" />
            </div>
          </div>
        </section>

        <section className="px-5 pb-6 sm:px-8">
          {chrome?.section_invoice_details ? (
            <h2 className="text-[13px] font-bold text-[var(--trim-fg)]">
              {chrome.section_invoice_details}
            </h2>
          ) : (
            <Skeleton className="h-4 w-28" />
          )}
          <div className="mt-3 space-y-2">
            <Skeleton className="h-4 w-64 max-w-full" />
            <Skeleton className="h-4 w-72 max-w-full" />
            <Skeleton className="h-4 w-80 max-w-full" />
            <Skeleton className="h-4 w-40 max-w-full" />
          </div>
        </section>

        <div className="mx-5 border-t border-[var(--trim-border)] sm:mx-8" />

        <section className="px-5 py-6 sm:px-8">
          {chrome?.section_transaction ? (
            <h2 className="mb-4 text-[13px] font-bold text-[var(--trim-fg)]">
              {chrome.section_transaction}
            </h2>
          ) : (
            <Skeleton className="mb-4 h-4 w-24" />
          )}

          <div className="overflow-x-auto">
            <table className="w-full min-w-[520px] border-collapse text-left text-[13px]">
              <thead>
                <tr className="border-b border-[var(--trim-border)]">
                  <th className="pb-2.5 pr-3 text-[12px] font-bold text-[var(--trim-fg)]">
                    {productCol || <Skeleton className="inline-block h-3 w-16" />}
                  </th>
                  <th className="px-2 pb-2.5 text-right text-[12px] font-bold text-[var(--trim-fg)]">
                    {chrome?.col_qty || <Skeleton className="ml-auto inline-block h-3 w-8" />}
                  </th>
                  <th className="px-2 pb-2.5 text-right text-[12px] font-bold text-[var(--trim-fg)]">
                    {chrome?.col_unit || <Skeleton className="ml-auto inline-block h-3 w-14" />}
                  </th>
                  <th className="px-2 pb-2.5 text-right text-[12px] font-bold text-[var(--trim-fg)]">
                    {chrome?.col_tax_rate || <Skeleton className="ml-auto inline-block h-3 w-14" />}
                  </th>
                  <th className="pb-2.5 pl-2 text-right text-[12px] font-bold text-[var(--trim-fg)]">
                    {chrome?.col_amount || <Skeleton className="ml-auto inline-block h-3 w-14" />}
                  </th>
                </tr>
              </thead>
              <tbody>
                {Array.from({ length: n }, (_, i) => `receipt-line-${i}`).map((key) => (
                  <tr key={key} className="border-b border-[var(--trim-border)]">
                    <td className="py-3.5 pr-3 align-top">
                      <Skeleton className="h-4 w-28" />
                      <Skeleton className="mt-1.5 h-3 w-48" />
                      <Skeleton className="mt-1.5 h-3 w-24" />
                    </td>
                    <td className="px-2 py-3.5 text-right align-top">
                      <Skeleton className="ml-auto h-4 w-6" />
                    </td>
                    <td className="px-2 py-3.5 text-right align-top">
                      <Skeleton className="ml-auto h-4 w-12" />
                    </td>
                    <td className="px-2 py-3.5 text-right align-top">
                      <Skeleton className="ml-auto h-4 w-8" />
                    </td>
                    <td className="py-3.5 pl-2 text-right align-top">
                      <Skeleton className="ml-auto h-4 w-12" />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="mt-2 flex justify-end">
            <dl className="w-full max-w-[240px] space-y-0 text-[13px]">
              {[0, 1, 2, 3].map((i) => (
                <div
                  key={i}
                  className="flex items-center justify-between gap-8 border-b border-[var(--trim-border)] py-2"
                >
                  <Skeleton className={`h-4 ${i === 3 ? "w-24" : "w-16"}`} />
                  <Skeleton className={`h-4 ${i === 3 ? "w-14" : "w-12"}`} />
                </div>
              ))}
            </dl>
          </div>

          <div className="mt-8 max-w-[240px]">
            {chrome?.section_tax_breakdown ? (
              <h2 className="text-[13px] font-bold text-[var(--trim-fg)]">
                {chrome.section_tax_breakdown}
              </h2>
            ) : (
              <Skeleton className="h-4 w-28" />
            )}
            <div className="mt-2 space-y-2 border-b border-[var(--trim-border)] pb-2">
              <div className="flex justify-between gap-8">
                <Skeleton className="h-3 w-12" />
                <Skeleton className="h-3 w-10" />
              </div>
              <div className="flex justify-between gap-8">
                <Skeleton className="h-4 w-8" />
                <Skeleton className="h-4 w-12" />
              </div>
            </div>
            <div className="flex justify-between gap-8 pt-2.5">
              <Skeleton className="h-4 w-16" />
              <Skeleton className="h-4 w-12" />
            </div>
          </div>
        </section>

        <footer
          data-receipt-band
          className="mt-4 bg-muted px-5 py-7 text-center sm:px-8"
          style={{ backgroundColor: "var(--receipt-band, #f4f4f5)" }}
        >
          <div className="mx-auto flex max-w-lg flex-col items-center gap-3">
            {chrome?.footer ? (
              <p className="text-[12px] leading-relaxed text-[var(--trim-muted)]">
                {chrome.footer}
              </p>
            ) : (
              <Skeleton className="h-3 w-72 max-w-full" />
            )}
            <Skeleton className="h-5 w-16" />
            <Skeleton className="h-3 w-56 max-w-full" />
          </div>
        </footer>
      </article>
    </div>
  );
}

/** Feature-row widths mirror plan_catalog feature lines in the plan modal. */
const PLAN_MODAL_FEATURE_WIDTHS = [
  "w-full",
  "w-[92%]",
  "w-[88%]",
  "w-[95%]",
  "w-[80%]",
  "w-[90%]",
  "w-[72%]",
] as const;

/**
 * One plan-modal card shimmer - mirrors PlanModal article (same chrome as landing pricing):
 * optional popular/unlimited badges, title, description, price + period,
 * check+feature rows, full-width CTA.
 */
export function PlanModalCardSkeleton({
  featureCount = 7,
  showPerSeat = false,
  showUnlimitedBadge = false,
  showPopularBadge = false,
}: {
  featureCount?: number;
  showPerSeat?: boolean;
  showUnlimitedBadge?: boolean;
  showPopularBadge?: boolean;
} = {}) {
  const widths = PLAN_MODAL_FEATURE_WIDTHS.slice(0, Math.max(0, featureCount));
  return (
    <div
      className={cn(
        "relative flex flex-col rounded-2xl border bg-[var(--trim-panel)] p-6 pt-7 shadow-[var(--trim-card-shadow)]",
        showPopularBadge
          ? "border-[var(--trim-ink)] ring-1 ring-[var(--trim-ink)]/20 -translate-y-0.5 shadow-[var(--trim-float-shadow)]"
          : "border-[var(--trim-border)]",
      )}
    >
      {showPopularBadge ? (
        <span className="absolute -top-3 left-1/2 z-10 -translate-x-1/2">
          <Skeleton className="h-6 w-20 rounded-md" />
        </span>
      ) : null}

      <div className="flex min-h-[1.5rem] items-start justify-between gap-2">
        <Skeleton className="h-7 w-24" />
      </div>

      {showUnlimitedBadge ? <Skeleton className="mt-3 h-6 w-44 max-w-full rounded-md" /> : null}

      <div className="mt-2 space-y-1.5">
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-[90%]" />
      </div>

      <div className="mt-6 flex items-baseline gap-2">
        <Skeleton className="h-9 w-28" />
        <Skeleton className="h-4 w-14" />
      </div>
      {showPerSeat ? <Skeleton className="mt-2 h-3 w-28" /> : null}

      <ul className="mt-6 mb-0 flex-1 space-y-2">
        {widths.map((w) => (
          <li key={w} className="flex items-start gap-2">
            <Skeleton className="mt-0.5 h-4 w-4 shrink-0 rounded-sm" />
            <Skeleton className={cn("h-4", w)} />
          </li>
        ))}
      </ul>
      <Skeleton className="mt-8 h-9 w-full rounded-md" />
    </div>
  );
}

/**
 * Plan modal grid shimmer.
 * Upgrade: Monthly/Annual switch + seat field + subscription cards.
 * Top-up: pack cards only (no interval/seats chrome).
 * Count must come from API (no invent).
 */
export function PlanGridSkeleton({
  count,
  mode = "upgrade",
}: {
  count: number;
  mode?: "upgrade" | "topup";
}) {
  const n = count > 0 ? count : 0;
  const topup = mode === "topup";
  const featureCount = topup ? 5 : 7;

  return (
    <div className="space-y-4">
      {!topup ? (
        <div className="flex flex-col items-center py-2">
          <div className="inline-flex items-center gap-3 rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel)] px-3 py-2 shadow-[var(--trim-card-shadow)]">
            <div className="min-w-0 space-y-1 text-left">
              <Skeleton className="h-4 w-28" />
              <Skeleton className="h-3 w-20" />
            </div>
            <Skeleton className="h-5 w-9 rounded-full" />
          </div>
        </div>
      ) : null}

      {!topup ? (
        <div className="mx-auto max-w-xs space-y-2">
          <Skeleton className="h-4 w-24" />
          <Skeleton className="h-9 w-full rounded-md" />
          <Skeleton className="h-3 w-40" />
        </div>
      ) : null}

      <div className="grid gap-4 pt-3 md:grid-cols-2 xl:grid-cols-4">
        {shimmerKeys("plan-card", n).map((id, i) => (
          <PlanModalCardSkeleton
            key={id}
            featureCount={featureCount}
            /* Catalog sort: Free, Pro, Team, Enterprise - Free+Enterprise unlimited, Pro popular, Team+Enterprise per-seat. */
            showPerSeat={!topup && (i === 2 || i === 3)}
            showUnlimitedBadge={!topup && (i === 0 || i === 3)}
            showPopularBadge={!topup && i === 1}
          />
        ))}
      </div>
    </div>
  );
}

/** Invite accept page: workspace summary + CTA mirrors. */
export type InviteAcceptSkeletonChrome = {
  eyebrow?: string;
  title?: string;
  body?: string;
  status_title?: string;
  accept_label?: string;
};

export function InviteAcceptSkeleton({
  chrome,
}: {
  chrome?: InviteAcceptSkeletonChrome;
} = {}) {
  return (
    <div className="mx-auto flex min-h-screen w-full max-w-lg flex-col justify-center bg-[var(--trim-bg)] px-6 py-16">
      {chrome?.eyebrow ? (
        <p className="text-sm text-[var(--trim-muted)]">{chrome.eyebrow}</p>
      ) : (
        <Skeleton className="h-3 w-16" />
      )}
      {chrome?.title ? (
        <h1 className="mt-2 text-3xl font-semibold tracking-tight text-[var(--trim-fg)]">
          {chrome.title}
        </h1>
      ) : (
        <Skeleton className="mt-3 h-9 w-56" />
      )}
      {chrome?.body ? (
        <p className="mt-2 text-sm text-[var(--trim-muted)]">{chrome.body}</p>
      ) : (
        <>
          <Skeleton className="mt-3 h-4 w-full" />
          <Skeleton className="mt-2 h-4 w-3/4" />
        </>
      )}
      <Card className="mt-8 border-[var(--trim-border)] bg-[var(--trim-panel)]">
        <CardHeader>
          {chrome?.status_title ? (
            <CardTitle className="text-base text-[var(--trim-fg)]">{chrome.status_title}</CardTitle>
          ) : (
            <Skeleton className="h-5 w-32" />
          )}
        </CardHeader>
        <CardContent className="space-y-4">
          <Skeleton className="h-4 w-48" />
          <Skeleton className="h-4 w-40" />
          {chrome?.accept_label ? (
            <span className="inline-flex h-10 w-full items-center justify-center rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel)] text-sm text-[var(--trim-muted)] shadow-[var(--trim-card-shadow)]">
              {chrome.accept_label}
            </span>
          ) : (
            <Skeleton className="h-10 w-full" />
          )}
        </CardContent>
      </Card>
    </div>
  );
}

/** Settings: Fast/Deep toggles, engine select, target tokens, CLI help. */
export type SettingsSkeletonChrome = {
  page_title?: string;
  page_description?: string;
  tier_title?: string;
  deep_label?: string;
  cli_title?: string;
  keys_title?: string;
  save_label?: string;
  back_label?: string;
  account_title?: string;
  auth_provider_prefix?: string;
  link_hint?: string;
  unlink_label?: string;
  row_actions_label?: string;
};

export function SettingsPageSkeleton({
  chrome,
  keyRows = 0,
  providerSlots = 0,
  identityRows = 1,
  embedded = false,
}: {
  chrome?: SettingsSkeletonChrome;
  /** Row count from billing_settings.default_page_size; 0 = keys chrome only. */
  keyRows?: number;
  /** Allowed OAuth provider count from auth-providers (connect button slots). */
  providerSlots?: number;
  /** Linked identity rows to shimmer (at least one signed-in account). */
  identityRows?: number;
  /** When true, skip legacy sticky chrome (shell already provides it). */
  embedded?: boolean;
} = {}) {
  const accountRows = identityRows > 0 ? identityRows : 1;
  return (
    <div className={embedded ? "w-full space-y-6" : "w-full"}>
      {embedded ? (
        <div className="min-w-0">
          {chrome?.page_title ? (
            <h1 className="text-2xl font-semibold tracking-tight text-[var(--trim-fg)] sm:text-3xl">
              {chrome.page_title}
            </h1>
          ) : (
            <Skeleton className="h-8 w-64 sm:h-9" />
          )}
          {chrome?.page_description ? (
            <p className="mt-1 max-w-2xl text-sm text-[var(--trim-muted)]">
              {chrome.page_description}
            </p>
          ) : (
            <Skeleton className="mt-1 h-4 w-96 max-w-full" />
          )}
        </div>
      ) : (
        <header className="sticky top-0 z-40 border-b border-[var(--trim-border)] bg-[var(--trim-bg)]/95 backdrop-blur supports-[backdrop-filter]:bg-[var(--trim-bg)]/80">
          <div className="mx-auto flex w-full max-w-[1400px] flex-col gap-4 px-3 py-4 sm:flex-row sm:flex-wrap sm:items-end sm:justify-between sm:px-6">
            <div className="min-w-0">
              {chrome?.page_title ? (
                <h1 className="text-2xl font-semibold tracking-tight text-[var(--trim-fg)] sm:text-3xl">
                  {chrome.page_title}
                </h1>
              ) : (
                <Skeleton className="h-8 w-64 sm:h-9" />
              )}
              {chrome?.page_description ? (
                <p className="mt-1 max-w-2xl text-sm text-[var(--trim-muted)]">
                  {chrome.page_description}
                </p>
              ) : (
                <Skeleton className="mt-1 h-4 w-96 max-w-full" />
              )}
            </div>
            <div className="flex flex-wrap items-center gap-2">
              {chrome?.back_label ? (
                <span className="inline-flex h-9 items-center rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel)] px-4 text-sm text-[var(--trim-muted)] shadow-[var(--trim-card-shadow)]">
                  {chrome.back_label}
                </span>
              ) : (
                <Skeleton className="h-9 w-40" />
              )}
              <Skeleton className="h-8 w-8 rounded-md" />
              <Skeleton className="h-8 w-8 rounded-md" />
              <Skeleton className="h-9 w-24 rounded-md" />
            </div>
          </div>
        </header>
      )}
      <div
        className={embedded ? "w-full" : "mx-auto w-full max-w-[1400px] px-3 py-6 sm:px-6 sm:py-8"}
      >
        <Card className="mb-6 border-[var(--trim-border)] bg-[var(--trim-panel)]">
          <CardHeader>
            {chrome?.account_title ? (
              <CardTitle className="text-base text-[var(--trim-fg)]">
                {chrome.account_title}
              </CardTitle>
            ) : (
              <Skeleton className="h-5 w-36" />
            )}
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Matches live identity DataTable: provider cell (icon + prefix + name + badges) + row-actions ⋯ */}
            <div className="space-y-4">
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow className="border-[var(--trim-border)] hover:bg-transparent">
                      <TableHead className="text-[var(--trim-muted)]">
                        {chrome?.account_title?.trim() ? (
                          chrome.account_title
                        ) : (
                          <Skeleton className="h-4 w-36" />
                        )}
                      </TableHead>
                      <TableHead className="w-12 text-right text-[var(--trim-muted)]">
                        {chrome?.row_actions_label?.trim() ? (
                          <span className="sr-only">{chrome.row_actions_label}</span>
                        ) : null}
                      </TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {shimmerKeys("identity", accountRows).map((rowId) => (
                      <TableRow key={rowId} className="border-[var(--trim-border)]">
                        <TableCell>
                          <div className="flex min-w-0 flex-wrap items-center gap-3">
                            <Skeleton className="h-5 w-5 shrink-0 rounded-sm" />
                            {chrome?.auth_provider_prefix ? (
                              <span className="text-[var(--trim-muted)]">
                                {chrome.auth_provider_prefix}
                              </span>
                            ) : (
                              <Skeleton className="h-4 w-24" />
                            )}
                            <Skeleton className="h-4 w-16" />
                            <Skeleton className="h-4 w-40 max-w-full" />
                            <Skeleton className="h-5 w-14 rounded-md" />
                            <Skeleton className="h-5 w-16 rounded-md" />
                          </div>
                        </TableCell>
                        <TableCell className="w-12 text-right">
                          <div className="inline-flex justify-end">
                            <Skeleton className="h-8 w-8 rounded-md" />
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </div>
            {chrome?.link_hint ? (
              <p className="text-xs text-[var(--trim-muted)]">{chrome.link_hint}</p>
            ) : (
              <Skeleton className="h-3 w-72 max-w-full" />
            )}
            <div className="flex flex-wrap gap-2">
              {shimmerKeys("connect", providerSlots > 0 ? Math.max(1, providerSlots - 1) : 1).map(
                (id) => (
                  <Skeleton key={id} className="h-9 w-36 rounded-md" />
                ),
              )}
            </div>
          </CardContent>
        </Card>
        <div className="grid gap-6 lg:grid-cols-2">
          <Card className="border-[var(--trim-border)] bg-[var(--trim-panel)]">
            <CardHeader>
              {chrome?.tier_title ? (
                <CardTitle className="text-base text-[var(--trim-fg)]">
                  {chrome.tier_title}
                </CardTitle>
              ) : (
                <Skeleton className="h-5 w-32" />
              )}
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="flex items-center justify-between rounded-lg border border-[var(--trim-border)] px-4 py-3">
                <div className="space-y-2">
                  {chrome?.deep_label ? (
                    <p className="text-sm font-medium text-[var(--trim-fg)]">{chrome.deep_label}</p>
                  ) : (
                    <Skeleton className="h-4 w-24" />
                  )}
                  <Skeleton className="h-3 w-48" />
                </div>
                <Skeleton className="h-6 w-11 rounded-full" />
              </div>
              <div className="flex items-center justify-between rounded-lg border border-[var(--trim-border)] px-4 py-3">
                <div className="space-y-2">
                  <Skeleton className="h-4 w-32" />
                  <Skeleton className="h-3 w-52" />
                </div>
                <Skeleton className="h-6 w-11 rounded-full" />
              </div>
              <div className="space-y-1.5">
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-3 w-48" />
              </div>
              <div className="space-y-1.5">
                <Skeleton className="h-4 w-28" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-3 w-52" />
              </div>
              {chrome?.save_label ? (
                <span className="inline-flex h-9 items-center rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel)] px-4 text-sm text-[var(--trim-muted)] shadow-[var(--trim-card-shadow)]">
                  {chrome.save_label}
                </span>
              ) : (
                <Skeleton className="h-9 w-36" />
              )}
            </CardContent>
          </Card>
          <Card className="border-[var(--trim-border)] bg-[var(--trim-panel)]">
            <CardHeader>
              {chrome?.cli_title ? (
                <CardTitle className="text-base text-[var(--trim-fg)]">
                  {chrome.cli_title}
                </CardTitle>
              ) : (
                <Skeleton className="h-5 w-28" />
              )}
            </CardHeader>
            <CardContent className="space-y-3">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-[83%]" />
              <Skeleton className="h-4 w-[80%]" />
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="mt-4 h-3 w-full" />
            </CardContent>
          </Card>
          <Card className="border-[var(--trim-border)] bg-[var(--trim-panel)] lg:col-span-2">
            <CardHeader className="flex flex-row items-center justify-between">
              {chrome?.keys_title ? (
                <CardTitle className="text-base text-[var(--trim-fg)]">
                  {chrome.keys_title}
                </CardTitle>
              ) : (
                <Skeleton className="h-5 w-24" />
              )}
              <div className="flex gap-2">
                <Skeleton className="h-8 w-20" />
                <Skeleton className="h-8 w-24" />
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              <Skeleton className="h-4 w-full max-w-xl" />
              <div className="w-full max-w-md space-y-1.5">
                <Skeleton className="h-4 w-28" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-3 w-48" />
              </div>
              <div className="flex flex-wrap gap-2">
                <Skeleton className="h-9 w-40" />
                <Skeleton className="h-9 w-28" />
              </div>
              <ApiKeysListSkeleton rows={keyRows > 0 ? keyRows : 8} title={chrome?.keys_title} />
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}

/** API key rows: select + prefix + actions. Matches web DataTable. */
export function ApiKeysListSkeleton({
  rows,
  title,
}: {
  rows: number;
  title?: string;
}) {
  return (
    <DataTableRowsSkeleton
      columns={[
        { label: " ", width: "h-4 w-4" },
        { label: title || " ", width: "h-4 w-40" },
        { label: " ", width: "h-8 w-8" },
      ]}
      rows={rows}
      showPagination
    />
  );
}

/** Login panel shimmer: title + provider buttons matching LoginPanel embedded card. */
export function AuthPageSkeleton({
  title,
  providerSlots,
}: {
  title?: string;
  /** Operator/API OAuth button count. 0 = title chrome only. */
  providerSlots: number;
}) {
  const slots = shimmerKeys("provider", typeof providerSlots === "number" ? providerSlots : 0);
  return (
    <div className="mx-auto w-full max-w-sm space-y-6">
      <div className="space-y-2 text-center">
        {title ? (
          <h1 className="font-display text-xl font-semibold tracking-tight text-[var(--trim-fg)] lg:text-2xl">
            {title}
          </h1>
        ) : (
          <Skeleton className="mx-auto h-7 w-48" />
        )}
      </div>
      {slots.length > 0 ? (
        <div className="flex flex-col gap-3">
          {slots.map((id) => (
            <div
              key={id}
              className="flex h-9 w-full items-center justify-center gap-2 rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel)] px-4"
            >
              <Skeleton className="h-4 w-4 shrink-0 rounded-sm" />
              <Skeleton className="h-4 w-40" />
            </div>
          ))}
        </div>
      ) : null}
    </div>
  );
}

/** Marketing sticky shell: left brand/copy/footer + right scroll rail (matches LandingShell). */
export type LandingPageSkeletonChrome = {
  brand?: string;
  nav_dashboard?: string;
  nav_sign_in?: string;
  /** Live login panel heading (`login_title`), not the nav Sign-in label. */
  login_title?: string;
  nav_how?: string;
  nav_pricing?: string;
  nav_contact?: string;
  /** Contact left-rail / page heading from site chrome. */
  contact_page_heading?: string;
  contact_page_body?: string;
  eyebrow?: string;
  headline?: string;
  tagline?: string;
  cta_start?: string;
  cta_source?: string;
  cta_demo?: string;
  footer_privacy?: string;
  footer_terms?: string;
  footer_github?: string;
};

export type LandingSkeletonMode = "home" | "pricing" | "login" | "legal" | "contact";

/** Full-bleed legal chrome shimmer (matches LandingShell mode=legal). */
export function LandingLegalPageSkeleton({
  chrome,
  authSlot = "pending",
}: {
  chrome?: LandingPageSkeletonChrome;
  authSlot?: PublicAuthSkeletonSlot;
} = {}) {
  const brand = chrome?.brand;
  return (
    <main className="min-h-screen bg-[var(--trim-bg)] text-[var(--trim-fg)]">
      <header className="sticky top-0 z-40 flex h-14 items-center justify-between gap-3 border-b border-[var(--trim-border)] bg-[var(--trim-panel)]/95 px-4 backdrop-blur sm:px-6 lg:px-10">
        {brand ? <TrimWordmark size="md" alt={brand} /> : <Skeleton className="h-5 w-16" />}
        <div className="flex items-center gap-2">
          <Skeleton className="hidden h-7 w-10 sm:block" />
          {chrome?.nav_pricing ? (
            <span className="rounded px-2 py-1 font-mono text-[11px] text-[var(--trim-subtle)]">
              {chrome.nav_pricing}
            </span>
          ) : (
            <Skeleton className="h-7 w-14" />
          )}
          {chrome?.footer_privacy ? (
            <span className="rounded px-2 py-1 font-mono text-[11px] text-[var(--trim-subtle)]">
              {chrome.footer_privacy}
            </span>
          ) : (
            <Skeleton className="h-7 w-14" />
          )}
          {chrome?.footer_terms ? (
            <span className="hidden rounded px-2 py-1 font-mono text-[11px] text-[var(--trim-subtle)] md:inline">
              {chrome.footer_terms}
            </span>
          ) : (
            <Skeleton className="hidden h-7 w-12 md:block" />
          )}
          <PublicAuthSlotSkeleton slot={authSlot} signInLabel={chrome?.nav_sign_in} />
          <Skeleton className="h-8 w-8 shrink-0 rounded-md" />
        </div>
      </header>

      <div className="mx-auto flex w-full max-w-none items-start">
        <aside className="hidden w-[260px] shrink-0 self-start border-r border-[var(--trim-border)] bg-[var(--trim-panel)] px-3 pb-8 pt-6 lg:sticky lg:top-14 lg:block lg:h-[calc(100vh-3.5rem)] lg:overflow-y-auto">
          <div className="border-b border-[var(--trim-border)] pb-2.5">
            <Skeleton className="mx-2 mb-2 h-4 w-16" />
            <div className="space-y-1 pl-1">
              <Skeleton className="h-8 w-full rounded-md" />
              <Skeleton className="h-8 w-full rounded-md" />
            </div>
          </div>
          {shimmerKeys("legal-nav", 6).map((id, i) => (
            <div key={id} className="border-b border-[var(--trim-border)] py-1">
              <div className="flex items-center justify-between px-2 py-2.5">
                <Skeleton className="h-4 w-28" />
                <Skeleton className="h-3.5 w-3.5" />
              </div>
              {i === 0 ? (
                <div className="space-y-1 pb-2 pl-3">
                  {shimmerKeys("legal-nav-link", 4).map((lid) => (
                    <Skeleton key={lid} className="h-7 w-full rounded-md" />
                  ))}
                </div>
              ) : null}
            </div>
          ))}
        </aside>
        <div className="min-w-0 flex-1 px-4 py-8 sm:px-8 lg:px-10 xl:px-12">
          <Skeleton className="mb-6 h-8 w-8 rounded-md lg:hidden" />
          <LegalArticleSkeleton sectionCount={0} embedded />
        </div>
        <aside className="hidden w-[220px] shrink-0 self-start xl:sticky xl:top-24 xl:block xl:max-h-[calc(100vh-7rem)] xl:overflow-y-auto xl:py-8 xl:pr-6">
          <Skeleton className="h-3 w-20" />
          <div className="mt-2 space-y-2 border-l border-[var(--trim-border)] pl-3">
            {shimmerKeys("legal-right-toc", 10).map((id) => (
              <Skeleton key={id} className="h-3 w-28" />
            ))}
          </div>
        </aside>
      </div>
    </main>
  );
}

export function LandingPageSkeleton({
  chrome,
  mode = "home",
  providerSlots = 0,
  pricingCardCount = 4,
  authSlot = "pending",
}: {
  chrome?: LandingPageSkeletonChrome;
  /** Matches LandingShell pathMode so route loading mirrors the real page. */
  mode?: LandingSkeletonMode;
  /** OAuth button count for login right-rail. */
  providerSlots?: number;
  pricingCardCount?: number;
  authSlot?: PublicAuthSkeletonSlot;
} = {}) {
  if (mode === "legal") {
    return <LandingLegalPageSkeleton chrome={chrome} authSlot={authSlot} />;
  }

  const brand = chrome?.brand;
  return (
    <main className="min-h-screen bg-[var(--trim-bg)] text-[var(--trim-fg)]">
      <header className="sticky top-0 z-40 flex items-center justify-between gap-3 border-b border-[var(--trim-border)] bg-[var(--trim-panel)]/95 px-5 py-3 backdrop-blur supports-[backdrop-filter]:bg-[var(--trim-panel)]/80 lg:hidden">
        {brand ? <TrimWordmark size="md" alt={brand} /> : <Skeleton className="h-5 w-16" />}
        <div className="flex items-center gap-2">
          {chrome?.nav_pricing ? (
            <span className="rounded px-2 py-1 font-mono text-[11px] text-[var(--trim-subtle)]">
              {chrome.nav_pricing}
            </span>
          ) : (
            <Skeleton className="h-7 w-14 rounded px-2" />
          )}
          {chrome?.nav_contact ? (
            <span className="rounded px-2 py-1 font-mono text-[11px] text-[var(--trim-subtle)]">
              {chrome.nav_contact}
            </span>
          ) : mode === "contact" ? (
            <Skeleton className="h-7 w-16 rounded px-2" />
          ) : null}
          <PublicAuthSlotSkeleton slot={authSlot} signInLabel={chrome?.nav_sign_in} />
          <Skeleton className="h-8 w-8 shrink-0 rounded-md" />
        </div>
      </header>

      <div className="mx-auto flex w-full max-w-[1850px]">
        <aside className="relative hidden w-[min(28vw,430px)] shrink-0 border-r border-[var(--trim-border)] bg-[var(--trim-panel)] lg:block xl:w-[450px]">
          <div className="sticky top-0 flex h-screen flex-col px-8 py-7 xl:px-10">
            <div className="flex items-center justify-between gap-3">
              {brand ? <TrimWordmark size="md" alt={brand} /> : <Skeleton className="h-5 w-16" />}
              <div className="flex items-center gap-1">
                {chrome?.nav_how ? (
                  <span className="rounded px-2 py-1 font-mono text-[11px] text-[var(--trim-subtle)]">
                    {chrome.nav_how}
                  </span>
                ) : (
                  <Skeleton className="h-6 w-10" />
                )}
                {chrome?.nav_pricing ? (
                  <span className="rounded px-2 py-1 font-mono text-[11px] text-[var(--trim-subtle)]">
                    {chrome.nav_pricing}
                  </span>
                ) : (
                  <Skeleton className="h-6 w-14" />
                )}
                {chrome?.nav_contact ? (
                  <span className="rounded px-2 py-1 font-mono text-[11px] text-[var(--trim-subtle)]">
                    {chrome.nav_contact}
                  </span>
                ) : mode === "contact" ? (
                  <Skeleton className="h-6 w-16" />
                ) : null}
                <PublicAuthSlotSkeleton
                  slot={authSlot}
                  signInLabel={chrome?.nav_sign_in}
                  className="ml-1 h-7 w-28"
                />
              </div>
            </div>

            <div className="flex flex-1 flex-col justify-center py-10">
              {mode === "pricing" ? (
                <>
                  {chrome?.nav_pricing ? (
                    <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-[var(--trim-subtle)]">
                      {chrome.nav_pricing}
                    </p>
                  ) : (
                    <Skeleton className="h-3 w-20" />
                  )}
                  <Skeleton className="mt-4 h-7 w-56" />
                  <Skeleton className="mt-2 h-7 w-40" />
                  <Skeleton className="mt-4 h-4 w-full max-w-[20rem]" />
                  <Skeleton className="mt-2 h-4 w-3/4 max-w-[16rem]" />
                  <Skeleton className="mt-8 h-9 w-36 rounded-md" />
                </>
              ) : mode === "login" ? (
                <>
                  {chrome?.nav_sign_in ? (
                    <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-[var(--trim-subtle)]">
                      {chrome.nav_sign_in}
                    </p>
                  ) : (
                    <Skeleton className="h-3 w-20" />
                  )}
                  {chrome?.login_title ? (
                    <h1 className="font-display mt-4 max-w-[22rem] text-[1.5rem] font-medium leading-[1.2] tracking-[-0.025em] text-[var(--trim-fg)] xl:text-[1.75rem]">
                      {chrome.login_title}
                    </h1>
                  ) : (
                    <Skeleton className="mt-4 h-7 w-48" />
                  )}
                  {chrome?.tagline ? (
                    <p className="mt-4 max-w-[22rem] text-[14px] leading-relaxed text-[var(--trim-subtle)]">
                      {chrome.tagline}
                    </p>
                  ) : (
                    <>
                      <Skeleton className="mt-4 h-4 w-full max-w-[20rem]" />
                      <Skeleton className="mt-2 h-4 w-3/4 max-w-[16rem]" />
                    </>
                  )}
                  {/* Matches live login left rail: ← wordmark home control */}
                  <div className="mt-8 inline-flex h-9 items-center gap-2 rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel)] px-4 shadow-[var(--trim-card-shadow)]">
                    <span className="text-[13px] text-[var(--trim-subtle)]" aria-hidden>
                      ←
                    </span>
                    <TrimWordmark size="sm" alt={chrome?.brand || "Trim"} />
                  </div>
                </>
              ) : mode === "contact" ? (
                <>
                  {chrome?.nav_contact ? (
                    <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-[var(--trim-subtle)]">
                      {chrome.nav_contact}
                    </p>
                  ) : (
                    <Skeleton className="h-3 w-20" />
                  )}
                  {chrome?.contact_page_heading ? (
                    <h1 className="font-display mt-4 max-w-[22rem] text-[1.5rem] font-medium leading-[1.2] tracking-[-0.025em] text-[var(--trim-fg)] xl:text-[1.75rem]">
                      {chrome.contact_page_heading}
                    </h1>
                  ) : (
                    <>
                      <Skeleton className="mt-4 h-7 w-56" />
                      <Skeleton className="mt-2 h-7 w-36" />
                    </>
                  )}
                  {chrome?.contact_page_body ? (
                    <p className="mt-4 max-w-[22rem] text-[14px] leading-relaxed text-[var(--trim-subtle)] xl:text-[15px]">
                      {chrome.contact_page_body}
                    </p>
                  ) : (
                    <>
                      <Skeleton className="mt-4 h-4 w-full max-w-[20rem]" />
                      <Skeleton className="mt-2 h-4 w-3/4 max-w-[16rem]" />
                      <Skeleton className="mt-2 h-4 w-2/3 max-w-[14rem]" />
                    </>
                  )}
                </>
              ) : (
                <>
                  {chrome?.eyebrow ? (
                    <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-[var(--trim-subtle)]">
                      {chrome.eyebrow}
                    </p>
                  ) : (
                    <Skeleton className="h-3 w-28" />
                  )}
                  {chrome?.headline ? (
                    <h1 className="font-display mt-5 max-w-[22rem] text-[1.5rem] font-medium leading-[1.2] tracking-[-0.025em] text-[var(--trim-subtle)] xl:text-[1.75rem]">
                      {chrome.headline}
                    </h1>
                  ) : (
                    <>
                      <Skeleton className="mt-5 h-7 w-56" />
                      <Skeleton className="mt-2 h-7 w-40" />
                    </>
                  )}
                  {chrome?.tagline ? (
                    <p className="mt-4 max-w-[22rem] text-[14px] leading-relaxed text-[var(--trim-subtle)]">
                      {chrome.tagline}
                    </p>
                  ) : (
                    <>
                      <Skeleton className="mt-4 h-4 w-full max-w-[20rem]" />
                      <Skeleton className="mt-2 h-4 w-3/4 max-w-[16rem]" />
                    </>
                  )}
                  <div className="mt-8 flex flex-wrap gap-2.5">
                    {chrome?.cta_start ? (
                      <span className="inline-flex h-9 items-center rounded-md border border-[var(--trim-border)] px-5 text-[13px] text-[var(--trim-subtle)]">
                        {chrome.cta_start}
                      </span>
                    ) : (
                      <Skeleton className="h-9 w-28 rounded-md" />
                    )}
                    {chrome?.cta_demo || chrome?.cta_source ? (
                      <span className="inline-flex h-9 items-center rounded-md border border-[var(--trim-border)] px-4 text-[13px] text-[var(--trim-subtle)]">
                        {chrome.cta_demo || chrome.cta_source}
                      </span>
                    ) : (
                      <Skeleton className="h-9 w-24 rounded-md" />
                    )}
                  </div>
                </>
              )}
            </div>

            <footer className="border-t border-[var(--trim-border)] pt-5 text-[12px] text-[var(--trim-subtle)]">
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0 space-y-3">
                  <Skeleton className="h-3 w-40" />
                  <div className="flex flex-wrap gap-x-4 gap-y-2">
                    <Skeleton className="h-3 w-10" />
                    {chrome?.footer_privacy ? (
                      <span>{chrome.footer_privacy}</span>
                    ) : (
                      <Skeleton className="h-3 w-16" />
                    )}
                    {chrome?.footer_terms ? (
                      <span>{chrome.footer_terms}</span>
                    ) : (
                      <Skeleton className="h-3 w-12" />
                    )}
                    {chrome?.footer_github ? (
                      <span>{chrome.footer_github}</span>
                    ) : (
                      <Skeleton className="h-3 w-14" />
                    )}
                    {chrome?.nav_pricing ? (
                      <span>{chrome.nav_pricing}</span>
                    ) : (
                      <Skeleton className="h-3 w-14" />
                    )}
                  </div>
                </div>
                <Skeleton className="mt-0.5 h-8 w-8 rounded-md" />
              </div>
            </footer>
          </div>
        </aside>

        <div className="min-w-0 flex-1">
          {mode === "pricing" ? (
            <LandingPricingContentSkeleton cardCount={pricingCardCount} />
          ) : mode === "login" ? (
            <div className="flex min-h-[calc(100vh-4rem)] w-full items-center justify-center px-5 py-16 sm:px-8">
              <AuthPageSkeleton
                title={chrome?.login_title || undefined}
                providerSlots={providerSlots}
              />
            </div>
          ) : mode === "contact" ? (
            <LandingContactContentSkeleton chrome={chrome} />
          ) : (
            <LandingHomeContentSkeleton />
          )}
          <footer className="flex flex-wrap items-center gap-4 border-t border-[var(--trim-border)] px-5 py-8 lg:hidden">
            <Skeleton className="mr-auto h-3 w-32" />
            <Skeleton className="h-3 w-10" />
            <Skeleton className="h-3 w-12" />
            <Skeleton className="h-3 w-10" />
            <Skeleton className="h-3 w-14" />
            <Skeleton className="h-8 w-8 rounded-md" />
          </footer>
        </div>
      </div>
    </main>
  );
}

/** Right-rail home sections: hero window, stats, demo, install duo, flow, why, use. */
export function LandingHomeContentSkeleton() {
  return (
    <div className="bg-[var(--trim-bg)]">
      <div className="border-b border-[var(--trim-border)] px-5 py-10 lg:hidden">
        <Skeleton className="mx-auto h-3 w-28" />
        <Skeleton className="mx-auto mt-5 h-8 w-64" />
        <Skeleton className="mx-auto mt-4 h-4 w-72 max-w-full" />
        <div className="mt-8 flex justify-center gap-2">
          <Skeleton className="h-9 w-28 rounded-md" />
          <Skeleton className="h-9 w-24 rounded-md" />
        </div>
      </div>

      {/* Hero IDE/CLI window (LandingHeroStage) */}
      <div className="px-5 py-8 sm:px-8 xl:px-12 xl:py-10">
        <div className="overflow-hidden rounded-[10px] border border-[var(--trim-border)] bg-[var(--trim-panel)]">
          <div className="flex h-10 items-center gap-3 border-b border-[var(--trim-border)] px-3">
            <div className="flex gap-[6px]">
              <Skeleton className="h-[10px] w-[10px] rounded-full" />
              <Skeleton className="h-[10px] w-[10px] rounded-full" />
              <Skeleton className="h-[10px] w-[10px] rounded-full" />
            </div>
            <Skeleton className="mx-auto h-6 w-full max-w-sm rounded-md" />
            <div className="flex gap-0.5">
              <Skeleton className="h-6 w-10 rounded" />
              <Skeleton className="h-6 w-10 rounded" />
            </div>
          </div>
          <div className="flex h-9 items-center gap-3 border-b border-[var(--trim-border)] bg-[var(--trim-panel-2)] px-3">
            <Skeleton className="h-1.5 w-1.5 rounded-full" />
            <Skeleton className="h-3 w-12" />
            <Skeleton className="h-3 w-10" />
            <Skeleton className="h-3 w-3" />
            <Skeleton className="h-3 w-10" />
            <Skeleton className="h-5 w-16 rounded" />
            <Skeleton className="ml-auto h-3 w-10" />
          </div>
          <div className="grid min-h-[420px] lg:grid-cols-[1fr_1fr_0.85fr]">
            <div className="border-b border-[var(--trim-border)] lg:border-b-0 lg:border-r">
              <div className="flex h-7 items-center justify-between border-b border-[var(--trim-border)] px-3">
                <Skeleton className="h-3 w-36" />
                <Skeleton className="h-3 w-10" />
              </div>
              <div className="space-y-2 p-3">
                {shimmerKeys("before", 12).map((id) => (
                  <Skeleton key={id} className="h-3 w-full" />
                ))}
              </div>
            </div>
            <div className="border-b border-[var(--trim-border)] lg:border-b-0 lg:border-r">
              <div className="flex h-7 items-center justify-between border-b border-[var(--trim-border)] px-3">
                <Skeleton className="h-3 w-40" />
                <Skeleton className="h-3 w-10" />
              </div>
              <div className="space-y-2 p-3">
                {shimmerKeys("after", 10).map((id) => (
                  <Skeleton key={id} className="h-3 w-full" />
                ))}
              </div>
            </div>
            <div className="bg-[var(--trim-panel-2)]">
              <div className="flex h-7 items-center border-b border-[var(--trim-border)] px-3">
                <Skeleton className="h-3 w-28" />
              </div>
              <div className="space-y-2 p-3">
                {shimmerKeys("act", 10).map((id) => (
                  <Skeleton key={id} className="h-3 w-[90%]" />
                ))}
              </div>
            </div>
          </div>
          <div className="flex h-7 items-center gap-3 border-t border-[var(--trim-border)] px-3">
            <Skeleton className="h-1.5 w-1.5 rounded-full" />
            <Skeleton className="h-3 w-48" />
          </div>
        </div>
      </div>

      <div className="space-y-2 border-t border-[var(--trim-border)]">
        {/* Stats */}
        <section className="w-full px-5 py-16 sm:px-8 xl:px-12">
          <Skeleton className="h-7 w-48" />
          <Skeleton className="mt-3 h-4 w-72 max-w-full" />
          <div className="mt-10 grid gap-3 sm:grid-cols-3">
            {shimmerKeys("stat", 3).map((id) => (
              <div
                key={id}
                className="rounded-2xl border border-[var(--trim-border)] bg-[var(--trim-panel)] shadow-[var(--trim-card-shadow)] px-6 py-8"
              >
                <Skeleton className="h-10 w-20" />
                <Skeleton className="mt-3 h-4 w-28" />
                <Skeleton className="mt-2 h-3 w-36" />
              </div>
            ))}
          </div>
        </section>

        {/* Demo: scenario tabs + checklist/log + before/after */}
        <section className="w-full px-5 py-16 sm:px-8 xl:px-12">
          <Skeleton className="h-7 w-56" />
          <Skeleton className="mt-3 h-4 w-80 max-w-full" />
          <div className="mt-8 overflow-hidden rounded-[10px] border border-[var(--trim-border)] bg-[var(--trim-panel)]">
            <div className="flex items-center gap-0 border-b border-[var(--trim-border)] px-2">
              {shimmerKeys("tab", 4).map((id) => (
                <Skeleton key={id} className="m-1.5 h-7 w-24" />
              ))}
              <Skeleton className="ml-auto mr-2 hidden h-3 w-10 sm:block" />
            </div>
            <div className="grid lg:grid-cols-2">
              <div className="flex flex-col border-b border-[var(--trim-border)] lg:border-b-0 lg:border-r">
                <div className="space-y-2 border-b border-[var(--trim-border)] px-3.5 py-3">
                  <Skeleton className="h-3 w-24" />
                  <Skeleton className="h-4 w-56" />
                  <Skeleton className="h-3 w-48" />
                  <Skeleton className="h-3 w-44" />
                  <Skeleton className="h-3 w-40" />
                </div>
                <div className="space-y-1.5 px-3.5 py-3">
                  {shimmerKeys("log", 6).map((id) => (
                    <div key={id} className="flex items-start gap-2">
                      <Skeleton className="mt-1.5 h-1.5 w-1.5 rotate-45 border border-[var(--trim-border)] bg-transparent" />
                      <Skeleton className="h-3 flex-1" />
                    </div>
                  ))}
                </div>
                <div className="mt-auto space-y-2 border-t border-[var(--trim-border)] px-3.5 py-3">
                  <Skeleton className="h-3 w-64" />
                  <Skeleton className="h-5 w-20 rounded" />
                </div>
              </div>
              <div className="grid grid-rows-2">
                <div className="border-b border-[var(--trim-border)]">
                  <div className="flex items-center justify-between px-3.5 py-1.5">
                    <Skeleton className="h-3 w-14" />
                    <Skeleton className="h-3 w-20" />
                  </div>
                  <div className="space-y-2 px-3.5 pb-3">
                    {shimmerKeys("db", 5).map((id) => (
                      <Skeleton key={id} className="h-3 w-full" />
                    ))}
                  </div>
                </div>
                <div>
                  <div className="flex items-center justify-between px-3.5 py-1.5">
                    <Skeleton className="h-3 w-12" />
                    <Skeleton className="h-3 w-20" />
                  </div>
                  <div className="space-y-2 px-3.5 pb-3">
                    {shimmerKeys("da", 5).map((id) => (
                      <Skeleton key={id} className="h-3 w-full" />
                    ))}
                  </div>
                </div>
              </div>
            </div>
            <div className="flex h-7 items-center gap-2 border-t border-[var(--trim-border)] px-3">
              <Skeleton className="h-1.5 w-1.5 rounded-full" />
              <Skeleton className="h-3 w-40" />
              <Skeleton className="ml-auto hidden h-3 w-32 sm:block" />
            </div>
          </div>
        </section>

        {/* Install: CLI + IDE duo + snippet + CTAs */}
        <section className="px-5 py-16 sm:px-8 xl:px-12">
          <Skeleton className="h-7 w-72 max-w-full" />
          <Skeleton className="mt-2 h-4 w-96 max-w-full" />
          <div className="mt-8 grid gap-4 xl:grid-cols-2">
            {shimmerKeys("duo", 2).map((id) => (
              <div key={id}>
                <Skeleton className="mb-2 h-3 w-28" />
                <div className="overflow-hidden rounded-[10px] border border-[var(--trim-border)] bg-[var(--trim-panel)]">
                  <div className="flex h-10 items-center gap-2 border-b border-[var(--trim-border)] px-3">
                    <Skeleton className="h-3 w-40" />
                    <Skeleton className="ml-auto h-6 w-16" />
                  </div>
                  <div className="space-y-2 p-4">
                    {shimmerKeys(`${id}-line`, 8).map((lid) => (
                      <Skeleton key={lid} className="h-3 w-full" />
                    ))}
                  </div>
                </div>
              </div>
            ))}
          </div>
          <Skeleton className="mt-6 h-24 w-full rounded-none border border-[var(--trim-border)]" />
          <div className="mt-5 flex flex-wrap gap-2">
            <Skeleton className="h-8 w-28 rounded-md" />
            <Skeleton className="h-8 w-28 rounded-md" />
          </div>
        </section>

        {/* How it works / flow */}
        <section className="w-full px-5 py-16 sm:px-8 xl:px-12">
          <Skeleton className="h-7 w-40" />
          <Skeleton className="mt-3 h-4 w-96 max-w-full" />
          <ol className="mt-10 max-w-4xl space-y-0">
            {shimmerKeys("flow", 4).map((id) => (
              <li
                key={id}
                className="relative grid grid-cols-1 gap-3 border-l border-[var(--trim-border)] py-7 pl-8 md:grid-cols-[17.5rem_minmax(0,1fr)] md:gap-x-12"
              >
                <Skeleton className="absolute -left-1.5 top-[2.05rem] h-3 w-3 rounded-full" />
                <div>
                  <Skeleton className="h-3 w-14" />
                  <Skeleton className="mt-2 h-5 w-40" />
                </div>
                <div className="space-y-2 md:pt-[1.375rem]">
                  <Skeleton className="h-3 w-full" />
                  <Skeleton className="h-3 w-[90%]" />
                  <Skeleton className="h-3 w-[70%]" />
                </div>
              </li>
            ))}
          </ol>
        </section>

        {/* Why */}
        <section className="w-full px-5 py-16 sm:px-8 xl:px-12">
          <Skeleton className="h-7 w-36" />
          <Skeleton className="mt-3 h-4 w-72" />
          <div className="mt-10 grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {shimmerKeys("why", 6).map((id) => (
              <div key={id} className="border-l border-[var(--trim-border)] py-1 pl-5">
                <Skeleton className="h-4 w-32" />
                <Skeleton className="mt-2 h-3 w-full" />
                <Skeleton className="mt-1 h-3 w-[83%]" />
              </div>
            ))}
          </div>
        </section>

        {/* Use cases */}
        <section className="w-full px-5 py-16 sm:px-8 xl:px-12">
          <Skeleton className="h-7 w-44" />
          <Skeleton className="mt-3 h-4 w-80 max-w-full" />
          <ol className="mt-10 grid gap-4 md:grid-cols-3">
            {shimmerKeys("use", 3).map((id) => (
              <li
                key={id}
                className="rounded-2xl border border-[var(--trim-border)] bg-[var(--trim-panel)] shadow-[var(--trim-card-shadow)] p-6"
              >
                <Skeleton className="h-3 w-14" />
                <Skeleton className="mt-3 h-5 w-32" />
                <Skeleton className="mt-2 h-3 w-full" />
                <Skeleton className="mt-1 h-3 w-[90%]" />
                <Skeleton className="mt-1 h-3 w-[70%]" />
              </li>
            ))}
          </ol>
        </section>
      </div>
    </div>
  );
}

/** Feature-row widths mirror typical plan_catalog feature line lengths. */
const LANDING_PRICE_FEATURE_WIDTHS = [
  "w-full",
  "w-[92%]",
  "w-[88%]",
  "w-[95%]",
  "w-[80%]",
  "w-[90%]",
  "w-[72%]",
] as const;

/**
 * One landing pricing card shimmer - mirrors LandingPricing article:
 * title, optional unlimited badge, 2-line description, price + period,
 * 7 check+feature rows, full-width CTA (Button default h-9).
 */
export function LandingPricingCardSkeleton({
  showUnlimitedBadge = false,
  showPopularBadge = false,
}: {
  showUnlimitedBadge?: boolean;
  showPopularBadge?: boolean;
} = {}) {
  return (
    <div
      className={cn(
        "relative flex flex-col rounded-2xl border bg-[var(--trim-panel)] p-6 pt-7 shadow-[var(--trim-card-shadow)]",
        showPopularBadge
          ? "border-[var(--trim-ink)] ring-1 ring-[var(--trim-ink)]/20 lg:-translate-y-1 lg:shadow-[var(--trim-float-shadow)]"
          : "border-[var(--trim-border)]",
      )}
    >
      {showPopularBadge ? (
        <span className="absolute -top-3 left-1/2 z-10 -translate-x-1/2">
          <Skeleton className="h-6 w-20 rounded-md" />
        </span>
      ) : null}

      <div className="flex min-h-[1.5rem] items-start justify-between gap-2">
        <Skeleton className="h-7 w-24" />
      </div>

      {showUnlimitedBadge ? <Skeleton className="mt-3 h-6 w-44 max-w-full rounded-md" /> : null}

      <div className="mt-2 space-y-1.5">
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-[90%]" />
      </div>

      <div className="mt-6 flex items-baseline gap-2">
        <Skeleton className="h-9 w-28" />
        <Skeleton className="h-4 w-12" />
      </div>

      <ul className="mt-6 mb-0 flex-1 space-y-2">
        {LANDING_PRICE_FEATURE_WIDTHS.map((w) => (
          <li key={w} className="flex items-start gap-2">
            <Skeleton className="mt-0.5 h-4 w-4 shrink-0 rounded-sm" />
            <Skeleton className={cn("h-4", w)} />
          </li>
        ))}
      </ul>

      <Skeleton className="mt-8 h-9 w-full rounded-md" />
    </div>
  );
}

/** Contact right-rail: eyebrow, heading, body, email card, topic grid (matches ContactPageContent). */
export function LandingContactContentSkeleton({
  chrome,
}: {
  chrome?: Pick<
    LandingPageSkeletonChrome,
    "nav_contact" | "contact_page_heading" | "contact_page_body"
  >;
} = {}) {
  return (
    <div className="mx-auto w-full max-w-2xl px-5 py-12 sm:px-8 lg:px-10 lg:py-16">
      {chrome?.nav_contact ? (
        <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-[var(--trim-muted)]">
          {chrome.nav_contact}
        </p>
      ) : (
        <Skeleton className="h-3 w-20" />
      )}
      {chrome?.contact_page_heading ? (
        <h1 className="font-display mt-3 text-3xl font-semibold tracking-tight text-[var(--trim-fg)] sm:text-4xl">
          {chrome.contact_page_heading}
        </h1>
      ) : (
        <>
          <Skeleton className="mt-3 h-9 w-72 max-w-full sm:h-10" />
          <Skeleton className="mt-2 h-9 w-48 max-w-full sm:h-10" />
        </>
      )}
      {chrome?.contact_page_body ? (
        <p className="mt-4 text-base leading-relaxed text-[var(--trim-muted)]">
          {chrome.contact_page_body}
        </p>
      ) : (
        <div className="mt-4 space-y-2">
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-4/5" />
        </div>
      )}

      <div className="mt-8 rounded-xl border border-[var(--trim-border)] bg-[var(--trim-panel)] p-5 shadow-[var(--trim-card-shadow)]">
        <Skeleton className="h-3 w-24" />
        <div className="mt-2 flex items-center gap-2">
          <Skeleton className="h-4 w-4 shrink-0 rounded-sm" />
          <Skeleton className="h-6 w-56 max-w-full" />
        </div>
        <Skeleton className="mt-4 h-9 w-36 rounded-md" />
      </div>

      <div className="mt-10">
        <Skeleton className="h-4 w-40" />
        <ul className="mt-4 grid gap-3 sm:grid-cols-2">
          {shimmerKeys("contact-topic", 4).map((id) => (
            <li
              key={id}
              className="flex flex-col rounded-xl border border-[var(--trim-border)] bg-[var(--trim-panel)] p-4"
            >
              <Skeleton className="h-4 w-28" />
              <Skeleton className="mt-1.5 h-3 w-full" />
              <Skeleton className="mt-1.5 h-3 w-4/5" />
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}

/** Pricing right-rail: interval toggle + plan cards. */
export function LandingPricingContentSkeleton({ cardCount = 4 }: { cardCount?: number }) {
  const n = Math.max(0, cardCount);
  return (
    <section className="w-full px-5 py-16 sm:px-8 xl:px-12">
      <div className="mx-auto flex w-full max-w-6xl flex-col items-center text-center">
        <div className="max-w-2xl lg:hidden">
          <Skeleton className="mx-auto h-7 w-40" />
          <Skeleton className="mx-auto mt-3 h-4 w-72 max-w-full" />
        </div>
        <div className="mt-6 inline-flex items-center gap-3 rounded-md border border-[var(--trim-border)] bg-[var(--trim-panel)] px-3 py-2 shadow-[var(--trim-card-shadow)] lg:mt-0">
          <div className="space-y-1.5 text-left">
            <Skeleton className="h-4 w-28 sm:h-[1.25rem]" />
            <Skeleton className="h-3 w-36 sm:h-4" />
          </div>
          <Skeleton className="h-5 w-9 rounded-full" />
        </div>
      </div>
      <div className="mx-auto mt-10 grid w-full max-w-none gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {shimmerKeys("price-card", n).map((id, i) => (
          <LandingPricingCardSkeleton
            key={id}
            /* Catalog sort: Free, Pro, Team, Enterprise - Free+Enterprise unlimited, Pro popular. */
            showUnlimitedBadge={i === 0 || i === 3}
            showPopularBadge={i === 1}
          />
        ))}
      </div>
    </section>
  );
}

/** Legal page header: brand + Docs + auth chrome + theme (matches SiteNavHeader). */
export function LegalPageHeaderSkeleton({
  authSlot = "pending",
}: {
  authSlot?: PublicAuthSkeletonSlot;
} = {}) {
  return (
    <header className="border-b border-[var(--trim-border)] bg-[var(--trim-panel)]">
      <div className="mx-auto flex w-full max-w-[1400px] items-center justify-between px-6 py-5">
        <TrimWordmark size="lg" alt="Trim" />
        <div className="flex items-center gap-3">
          <Skeleton className="h-4 w-10" />
          <PublicAuthSlotSkeleton slot={authSlot} className="h-9 w-36" />
          <Skeleton className="h-8 w-8 rounded-md" />
        </div>
      </div>
    </header>
  );
}

/** Privacy / terms: title, updated, sections (+ optional TOC). */
export function LegalArticleSkeleton({
  sectionCount,
  embedded = false,
}: {
  /** From privacy_sections / terms_sections length when known; 0 while unknown. */
  sectionCount: number;
  /** When true, omit outer padding / right TOC (parent shell provides them). */
  embedded?: boolean;
}) {
  const n = sectionCount > 0 ? sectionCount : 8;
  const article = (
    <div className="min-w-0 space-y-8">
      {!embedded ? (
        <div className="xl:hidden">
          <Skeleton className="h-10 w-full rounded-lg" />
        </div>
      ) : (
        <div className="xl:hidden">
          <Skeleton className="h-10 w-full rounded-lg" />
        </div>
      )}
      <div className="space-y-3">
        <Skeleton className="h-3 w-16" />
        <Skeleton className="h-9 w-64 max-w-full" />
        <Skeleton className="h-4 w-40" />
      </div>
      {shimmerKeys("legal-section", n).map((id) => (
        <div key={id} className="space-y-3">
          <Skeleton className="h-6 w-48" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-[92%]" />
          <Skeleton className="h-4 w-[70%]" />
        </div>
      ))}
      <div className="mt-14 grid gap-3 border-t border-[var(--trim-border)] pt-8 sm:grid-cols-2">
        <Skeleton className="h-16 w-full rounded-lg" />
        <Skeleton className="h-16 w-full rounded-lg" />
      </div>
    </div>
  );

  if (embedded) return article;

  return (
    <div className="mx-auto grid w-full max-w-none gap-10 px-5 pb-24 pt-6 sm:px-8 lg:px-12 xl:grid-cols-[minmax(0,1fr)_220px] xl:px-16">
      {article}
      <aside className="hidden xl:block">
        <div className="sticky top-24 space-y-2">
          <Skeleton className="h-3 w-20" />
          <div className="space-y-2 border-l border-[var(--trim-border)] pl-3">
            {shimmerKeys("legal-toc", Math.min(n, 10)).map((id) => (
              <Skeleton key={id} className="h-3 w-28" />
            ))}
          </div>
        </div>
      </aside>
    </div>
  );
}

/** Docs article + TOC only (DocsShell stays mounted on soft-nav). */
export function DocsArticleSkeleton() {
  return (
    <div className="mx-auto grid w-full max-w-none gap-10 xl:grid-cols-[minmax(0,1fr)_220px]">
      <div className="min-w-0 space-y-5">
        <Skeleton className="mb-8 h-10 w-full rounded-lg xl:hidden" />
        <Skeleton className="h-3 w-28" />
        <Skeleton className="mt-3 h-10 w-72 max-w-full" />
        <Skeleton className="mt-3 h-4 w-full max-w-2xl" />
        <Skeleton className="mt-2 h-4 w-[83%] max-w-xl" />
        <div className="mt-10 space-y-4">
          <Skeleton className="h-6 w-48" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-[95%]" />
          <Skeleton className="h-4 w-[80%]" />
          <Skeleton className="mt-6 h-6 w-40" />
          <Skeleton className="h-32 w-full rounded-lg" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-[88%]" />
        </div>
        <div className="mt-14 grid gap-3 border-t border-[var(--trim-border)] pt-8 sm:grid-cols-2">
          <Skeleton className="h-16 w-full rounded-lg" />
          <Skeleton className="h-16 w-full rounded-lg" />
        </div>
      </div>
      <aside className="hidden self-start xl:block">
        <div className="sticky top-24 space-y-2">
          <Skeleton className="h-3 w-20" />
          <div className="space-y-2 border-l border-[var(--trim-border)] pl-3">
            {shimmerKeys("doc-toc", 5).map((id) => (
              <Skeleton key={id} className="h-3 w-28" />
            ))}
          </div>
        </div>
      </aside>
    </div>
  );
}

/** Docs shell: sticky header, accordion sidebar, article, on-this-page TOC. */
export function DocsPageSkeleton({
  authSlot = "pending",
}: {
  authSlot?: PublicAuthSkeletonSlot;
} = {}) {
  return (
    <div className="min-h-screen bg-[var(--trim-bg)] text-[var(--trim-fg)]">
      <header className="sticky top-0 z-40 border-b border-[var(--trim-border)] bg-[var(--trim-panel)]/95 backdrop-blur">
        <div className="mx-auto flex h-14 w-full max-w-none items-center gap-3 px-4 sm:px-6 lg:px-10">
          <Skeleton className="h-8 w-8 rounded-md lg:hidden" />
          <TrimWordmark size="md" alt="Trim" />
          <Skeleton className="hidden h-4 w-10 sm:block" />
          <div className="ml-auto flex items-center gap-2">
            <Skeleton className="hidden h-7 w-14 sm:block" />
            <Skeleton className="hidden h-7 w-14 md:block" />
            <PublicAuthSlotSkeleton
              slot={authSlot}
              className="hidden h-8 w-36 rounded-md sm:block"
            />
            <Skeleton className="h-8 w-8 rounded-md" />
          </div>
        </div>
      </header>
      <div className="mx-auto flex w-full max-w-none items-start">
        <aside className="hidden w-[260px] shrink-0 self-start border-r border-[var(--trim-border)] px-3 pb-8 pt-6 lg:sticky lg:top-14 lg:block lg:h-[calc(100vh-3.5rem)] lg:overflow-y-auto">
          {shimmerKeys("doc-nav", 9).map((id, i) => (
            <div key={id} className="border-b border-[var(--trim-border)] py-1">
              <div className="flex items-center justify-between px-2 py-2.5">
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-3.5 w-3.5" />
              </div>
              {i === 0 ? (
                <div className="space-y-1 pb-2 pl-3">
                  {shimmerKeys("doc-link", 3).map((lid) => (
                    <Skeleton key={lid} className="h-7 w-full rounded-md" />
                  ))}
                </div>
              ) : null}
            </div>
          ))}
        </aside>
        <div className="min-w-0 flex-1 px-4 py-8 sm:px-8 lg:px-12 xl:px-16">
          <DocsArticleSkeleton />
        </div>
      </div>
    </div>
  );
}

/** CLI browser auth page: eyebrow, title, key panel, copy/retry actions. */
export type CliAuthPageSkeletonChrome = {
  eyebrow?: string;
  title?: string;
  body?: string;
  copy_label?: string;
  issue_label?: string;
  footer?: string;
};

export function CliAuthPageSkeleton({
  chrome,
}: {
  chrome?: CliAuthPageSkeletonChrome;
} = {}) {
  return (
    <main className="flex min-h-screen items-center justify-center bg-[var(--trim-bg)] px-6">
      <div className="w-full max-w-lg rounded-lg border border-[var(--trim-border)] bg-[var(--trim-panel)] p-8">
        {chrome?.eyebrow ? (
          <p className="text-xs font-medium uppercase tracking-wider text-[var(--trim-subtle)]">
            {chrome.eyebrow}
          </p>
        ) : (
          <Skeleton className="h-3 w-16" />
        )}
        {chrome?.title ? (
          <h1 className="mt-2 text-2xl font-semibold text-[var(--trim-subtle)]">{chrome.title}</h1>
        ) : (
          <Skeleton className="mt-3 h-8 w-56" />
        )}
        {chrome?.body ? (
          <p className="mt-2 text-sm text-[var(--trim-subtle)]">{chrome.body}</p>
        ) : (
          <>
            <Skeleton className="mt-3 h-4 w-full max-w-md" />
            <Skeleton className="mt-2 h-4 w-3/4 max-w-sm" />
          </>
        )}
        <Skeleton className="mt-6 h-24 w-full rounded-md" />
        <div className="mt-4 flex gap-2">
          {chrome?.copy_label ? (
            <span className="inline-flex h-10 items-center rounded-md border border-[var(--trim-border)] px-4 text-sm text-[var(--trim-subtle)]">
              {chrome.copy_label}
            </span>
          ) : (
            <Skeleton className="h-10 w-28" />
          )}
          {chrome?.issue_label ? (
            <span className="inline-flex h-10 items-center rounded-md border border-[var(--trim-border)] px-4 text-sm text-[var(--trim-subtle)]">
              {chrome.issue_label}
            </span>
          ) : (
            <Skeleton className="h-10 w-32" />
          )}
        </div>
        {chrome?.footer ? (
          <p className="mt-4 text-xs text-[var(--trim-subtle)]">{chrome.footer}</p>
        ) : (
          <Skeleton className="mt-4 h-3 w-72" />
        )}
      </div>
    </main>
  );
}

/** Compact subscription status strip under the dashboard header actions. */
export function SubscriptionStatusSkeleton() {
  return (
    <div className="mt-4 flex flex-wrap items-center gap-3 rounded-lg border border-[var(--trim-border)] bg-[var(--trim-panel)] px-4 py-3 text-sm shadow-[var(--trim-card-shadow)]">
      <Skeleton className="h-4 w-28" />
      <Skeleton className="h-4 w-40" />
      <Skeleton className="h-4 w-24" />
    </div>
  );
}
