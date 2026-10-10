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
import type { ReactNode } from "react";

type ColumnSpec = { label: string; width?: string };

function shimmerKeys(prefix: string, count: number): string[] {
  return Array.from({ length: Math.max(0, count) }, (_, i) => `${prefix}-${i}`);
}

export function MetricValueSkeleton({ className }: { className?: string }) {
  return <Skeleton aria-hidden className={className ?? "h-8 w-24"} />;
}

export function MetricCardsSkeleton({
  labels,
}: {
  labels: Array<string | undefined>;
}) {
  if (!labels.length) return null;
  // Match live admin grids: 5 KPI tiles use 3 cols (observability); 4 use 4 cols (overview).
  const gridClass =
    labels.length <= 2
      ? "grid gap-3 sm:grid-cols-2"
      : labels.length === 4
        ? "grid gap-3 sm:grid-cols-2 lg:grid-cols-4"
        : labels.length <= 5
          ? "grid gap-3 sm:grid-cols-2 lg:grid-cols-3"
          : "grid gap-3 sm:grid-cols-2 lg:grid-cols-4";
  const cards = shimmerKeys("metric", labels.length).map((key, i) => ({
    key,
    label: labels[i],
  }));
  return (
    <div className={gridClass}>
      {cards.map(({ key, label }) => (
        <Card key={key}>
          <CardHeader>
            {label ? (
              <CardTitle className="text-sm font-medium text-muted-foreground">{label}</CardTitle>
            ) : (
              <Skeleton className="h-4 w-24" />
            )}
          </CardHeader>
          <CardContent>
            <MetricValueSkeleton />
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

export function PaginationBarSkeleton({ className }: { className?: string }) {
  return (
    <div className={className ?? "flex flex-wrap items-center gap-2"}>
      <Skeleton className="mr-auto h-3 w-24" />
      <Skeleton className="h-8 w-12" />
      <Skeleton className="h-8 w-14" />
      <Skeleton className="h-8 w-16" />
      <Skeleton className="h-8 w-10" />
    </div>
  );
}

export function FilterBarSkeleton({
  count = 4,
  leadSearch = true,
  wideIndexes,
}: {
  count?: number;
  /** When false, every slot uses select/text width (audit, distribution). */
  leadSearch?: boolean;
  /** Indexes that should match dateRange / auto width (segments created range). */
  wideIndexes?: number[];
}) {
  const n = Math.min(8, Math.max(0, count));
  if (n < 1) return null;
  const wide = new Set(wideIndexes ?? []);
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-end">
      {shimmerKeys("filter", n).map((id, i) => {
        const widthClass =
          leadSearch && i === 0
            ? "w-full sm:min-w-[16rem] sm:flex-1 sm:max-w-md"
            : wide.has(i)
              ? "w-full sm:w-auto sm:min-w-[14rem]"
              : "w-full sm:w-44";
        return (
          <div key={id} className={`space-y-1.5 ${widthClass}`}>
            <Skeleton className="h-3 w-16" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-5 w-28" />
          </div>
        );
      })}
    </div>
  );
}

export function FormPageSkeleton({
  title,
  fields,
  hideTitle = false,
}: {
  title?: string;
  fields: number;
  /** When composing under another page skeleton that already shows the title. */
  hideTitle?: boolean;
}) {
  const n = fields > 0 ? fields : 0;
  return (
    <div className="space-y-4">
      {hideTitle ? null : title ? (
        <h1 className="text-xl font-semibold tracking-tight text-muted-foreground">{title}</h1>
      ) : (
        <Skeleton className="h-7 w-48" />
      )}
      <Card>
        <CardContent className="grid gap-4 pt-6 sm:grid-cols-2 xl:grid-cols-3">
          {shimmerKeys("field", n).map((id) => (
            <div key={id} className="space-y-1.5">
              <Skeleton className="h-3 w-24" />
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-5 w-40" />
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}

/** Tabs strip shimmer (denylist / RBAC style pages). */
export function TabsSkeleton({ count = 3 }: { count?: number }) {
  const n = Math.max(1, Math.min(5, count));
  return (
    <div className="inline-flex h-9 items-center gap-1 rounded-lg bg-accent p-1">
      {shimmerKeys("tab", n).map((id) => (
        <Skeleton key={id} className="h-7 w-20 rounded-md" />
      ))}
    </div>
  );
}

/**
 * Page shape: optional tabs + optional create toolbar / inline form + table.
 * Modal-first CRUD pages pass formFields={0} so only create-button shimmer shows.
 */
export function FormPlusTableSkeleton({
  title,
  columns,
  rows,
  formFields = 2,
  toolbarButtons,
  showTabs = false,
  tabCount = 3,
  showPagination = true,
  showFilters = false,
  filterCount = 1,
  formFirst = true,
  filtersBeforeForm = false,
}: {
  title?: string;
  columns: ColumnSpec[];
  rows: number;
  /** 0 = create-button toolbar only (dialogs hold the form). */
  formFields?: number;
  /** When formFields is 0, how many toolbar button shimmers (plans sync+create). */
  toolbarButtons?: number;
  showTabs?: boolean;
  tabCount?: number;
  showPagination?: boolean;
  showFilters?: boolean;
  /** Filter shimmer count when showFilters (list pages with one search default to 1). */
  filterCount?: number;
  /** When false, table (with optional filters) renders above the form card. */
  formFirst?: boolean;
  /** Live denylist shape: filters then create toolbar then table. */
  filtersBeforeForm?: boolean;
}) {
  const buttonCount = Math.max(0, Math.min(4, toolbarButtons ?? 1));
  const form =
    formFields > 0 ? (
      <Card>
        <CardContent className="space-y-4 pt-6">
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {shimmerKeys("form", formFields).map((id) => (
              <div key={id} className="space-y-1.5">
                <Skeleton className="h-3 w-24" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-5 w-40" />
              </div>
            ))}
          </div>
          <Skeleton className="h-9 w-24" />
        </CardContent>
      </Card>
    ) : buttonCount > 0 ? (
      <div className="flex flex-wrap gap-2">
        {shimmerKeys("toolbar-btn", buttonCount).map((id) => (
          <Skeleton key={id} className="h-9 w-28" />
        ))}
      </div>
    ) : null;
  const table = (
    <DataTableSkeleton
      title=""
      columns={columns}
      rows={rows}
      showFilters={showFilters && !filtersBeforeForm}
      filterCount={filterCount}
      showPagination={showPagination}
    />
  );
  const filters =
    showFilters && filtersBeforeForm ? <FilterBarSkeleton count={filterCount} /> : null;
  return (
    <div className="space-y-4">
      {title ? (
        <h1 className="text-xl font-semibold tracking-tight text-muted-foreground">{title}</h1>
      ) : null}
      {showTabs ? <TabsSkeleton count={tabCount} /> : null}
      {filtersBeforeForm ? (
        <>
          {filters}
          {form}
          {table}
        </>
      ) : formFirst ? (
        <>
          {form}
          {table}
        </>
      ) : (
        <>
          {table}
          {form}
        </>
      )}
    </div>
  );
}

/** Usage stacked chart + LOC heatmap card shimmers (matches dashboard/observability charts). */
export function DashboardChartsSkeleton() {
  return (
    <>
      <Card>
        <CardContent className="space-y-4 pt-6">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
            <div className="space-y-2">
              <Skeleton className="h-7 w-48" />
              <Skeleton className="h-4 w-64" />
            </div>
            <Skeleton className="h-9 w-[140px]" />
          </div>
          <Skeleton className="h-40 w-full rounded-md sm:h-56" />
        </CardContent>
      </Card>
      <Card>
        <CardContent className="space-y-4 pt-6">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="space-y-2">
              <Skeleton className="h-4 w-40" />
              <Skeleton className="h-9 w-24" />
            </div>
            <Skeleton className="h-10 w-56 rounded-lg" />
          </div>
          <Skeleton className="h-[110px] w-full max-w-xl rounded-md" />
        </CardContent>
      </Card>
    </>
  );
}

/** Overview soft-nav: KPI strip, usage chart, heatmap, then three ops tables. */
export function OpsOverviewSkeleton({
  title,
  metricCount = 4,
  columns,
  rows,
  showBrand = true,
  showCharts = true,
  showOpsTables = true,
}: {
  title?: string;
  metricCount?: number;
  /** Kept for call-site compat; overview tables are two-column ops lists. */
  columns?: ColumnSpec[];
  rows?: number;
  showBrand?: boolean;
  showCharts?: boolean;
  showOpsTables?: boolean;
}) {
  const tableRows = rows && rows > 0 ? Math.min(rows, 6) : 4;
  const tableCols = columns && columns.length > 0 ? columns : [{ label: "" }, { label: "" }];
  return (
    <div className="space-y-6">
      {showBrand ? (
        title ? (
          <div>
            <h1 className="text-xl font-semibold tracking-tight text-muted-foreground">{title}</h1>
            <Skeleton className="mt-2 h-4 w-56" />
          </div>
        ) : (
          <div className="space-y-2">
            <Skeleton className="h-7 w-40" />
            <Skeleton className="h-4 w-56" />
          </div>
        )
      ) : null}
      {metricCount > 0 ? (
        <MetricCardsSkeleton labels={Array.from({ length: metricCount }, () => undefined)} />
      ) : null}
      {showCharts ? <DashboardChartsSkeleton /> : null}
      {showOpsTables
        ? shimmerKeys("ops-table", 3).map((id) => (
            <Card key={id}>
              <CardHeader className="pb-2">
                <Skeleton className="h-4 w-32" />
              </CardHeader>
              <CardContent>
                <DataTableSkeleton
                  title=""
                  columns={tableCols}
                  rows={tableRows}
                  showFilters={false}
                  showPagination={false}
                />
              </CardContent>
            </Card>
          ))
        : null}
    </div>
  );
}

export function DataTableSkeleton({
  title,
  columns,
  rows,
  showPagination = true,
  showFilters = true,
  filterCount,
  filterLeadSearch = true,
  filterWideIndexes,
}: {
  title: string;
  columns: ColumnSpec[];
  rows: number;
  showPagination?: boolean;
  /** List pages default to a filter bar shimmer above the table. */
  showFilters?: boolean;
  filterCount?: number;
  /** When false, filter shimmer slots are equal select/text width. */
  filterLeadSearch?: boolean;
  /** Match dateRange-width filter slots (see FilterBarSkeleton). */
  filterWideIndexes?: number[];
}) {
  const n = rows > 0 ? rows : 8;
  // Keep caller column count stable while chrome labels are still empty.
  // Never invent a different column count (was hardcoding 5).
  const resolvedColumns =
    columns.length > 0 ? columns : [{ label: "" }, { label: "" }, { label: "" }];
  const cells = resolvedColumns.map((col, i) => ({ ...col, id: `col-${i}` }));
  return (
    <div className="space-y-4">
      {title ? (
        <h1 className="text-xl font-semibold tracking-tight text-muted-foreground">{title}</h1>
      ) : null}
      {showFilters ? (
        <FilterBarSkeleton
          count={filterCount}
          leadSearch={filterLeadSearch}
          wideIndexes={filterWideIndexes}
        />
      ) : null}
      <div className="overflow-x-auto rounded-md border border-border bg-card">
        <Table>
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              {cells.map((col) => (
                <TableHead key={col.id}>
                  {col.label ? col.label : <Skeleton className="h-4 w-16" />}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {shimmerKeys("row", n).map((rowId) => (
              <TableRow key={rowId}>
                {cells.map((col) => (
                  <TableCell key={`${rowId}-${col.id}`}>
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

/** Two stacked list sections (chrome messages+legal, receipts+disputes). */
export function DualTableSkeleton({
  title,
  primaryColumns,
  secondaryColumns,
  rows,
  showFilters = true,
  filterCount = 1,
  showSecondaryFilters = false,
  midToolbarButtons = 0,
  showTabs = false,
  tabCount = 3,
}: {
  title?: string;
  primaryColumns: ColumnSpec[];
  secondaryColumns: ColumnSpec[];
  rows: number;
  showFilters?: boolean;
  filterCount?: number;
  /** Secondary list usually has no filter bar (legal docs, disputes). */
  showSecondaryFilters?: boolean;
  /** Optional create/action button between primary and secondary (receipts dispute). */
  midToolbarButtons?: number;
  showTabs?: boolean;
  tabCount?: number;
}) {
  const midCount = Math.max(0, Math.min(3, midToolbarButtons));
  return (
    <div className="space-y-6">
      {title ? (
        <h1 className="text-xl font-semibold tracking-tight text-muted-foreground">{title}</h1>
      ) : null}
      {showTabs ? <TabsSkeleton count={tabCount} /> : null}
      <DataTableSkeleton
        title=""
        columns={primaryColumns}
        rows={rows}
        showFilters={showFilters}
        filterCount={filterCount}
      />
      {midCount > 0 ? (
        <div className="flex flex-wrap gap-2">
          {shimmerKeys("mid-toolbar", midCount).map((id) => (
            <Skeleton key={id} className="h-9 w-28" />
          ))}
        </div>
      ) : null}
      {secondaryColumns.length > 0 ? (
        <DataTableSkeleton
          title=""
          columns={secondaryColumns}
          rows={Math.min(rows, 6)}
          showFilters={showSecondaryFilters}
          filterCount={1}
        />
      ) : null}
    </div>
  );
}

/** Metrics strip + table (observability style, no huge chart block). */
export function MetricsPlusTableSkeleton({
  title,
  metricCount = 4,
  leadingCards = 0,
  breakdownCards = 0,
  columns,
  rows,
  showFilters = false,
  filterCount = 1,
}: {
  title?: string;
  metricCount?: number;
  /** Optional top row (e.g. CLI summary cards) without inventing labels. */
  leadingCards?: number;
  /** Optional full-width card blocks with inner table shimmer (breakdown / heatmap). */
  breakdownCards?: number;
  columns: ColumnSpec[];
  rows: number;
  showFilters?: boolean;
  filterCount?: number;
}) {
  return (
    <div className="space-y-6">
      {title ? (
        <h1 className="text-xl font-semibold tracking-tight text-muted-foreground">{title}</h1>
      ) : null}
      {leadingCards > 0 ? (
        <MetricCardsSkeleton labels={Array.from({ length: leadingCards }, () => undefined)} />
      ) : null}
      <MetricCardsSkeleton labels={Array.from({ length: metricCount }, () => undefined)} />
      {breakdownCards > 0
        ? shimmerKeys("obs-breakdown", breakdownCards).map((id) => (
            <Card key={id} className="bg-card">
              <CardHeader className="pb-2">
                <Skeleton className="h-4 w-32" />
              </CardHeader>
              <CardContent>
                <DataTableSkeleton
                  title=""
                  columns={[{ label: "" }, { label: "" }]}
                  rows={3}
                  showFilters={false}
                  showPagination={false}
                />
              </CardContent>
            </Card>
          ))
        : null}
      <DataTableSkeleton
        title=""
        columns={columns}
        rows={rows}
        showFilters={showFilters}
        filterCount={filterCount}
      />
    </div>
  );
}

/** Form fields then a table (compliance retention + access review). */
export function FormThenTableSkeleton({
  title,
  fields,
  columns,
  rows,
  showFilters = true,
}: {
  title?: string;
  fields: number;
  columns: ColumnSpec[];
  rows: number;
  showFilters?: boolean;
}) {
  return (
    <div className="space-y-4">
      <FormPageSkeleton title={title} fields={fields} />
      <DataTableSkeleton
        title=""
        columns={columns}
        rows={rows}
        showFilters={showFilters}
        filterCount={1}
      />
    </div>
  );
}

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
    <main className="mx-auto flex min-h-[70vh] w-full max-w-md flex-col justify-center gap-6 px-4 py-16">
      <div className="space-y-2 text-center">
        {title ? (
          <h1 className="text-2xl font-semibold tracking-tight text-muted-foreground">{title}</h1>
        ) : (
          <Skeleton className="mx-auto h-8 w-56" />
        )}
      </div>
      {slots.length > 0 ? (
        <div className="flex flex-col gap-3">
          {slots.map((id) => (
            <div
              key={id}
              className="bg-secondary flex h-9 w-full items-center justify-center gap-2 rounded-md px-4 shadow-sm"
            >
              <Skeleton className="h-4 w-4 shrink-0 rounded-sm" />
              <Skeleton className="h-4 w-44" />
            </div>
          ))}
        </div>
      ) : null}
    </main>
  );
}

/**
 * Chrome-only shell while AdminGate settles-(sidebar + top bar).
 * Keep main empty (or pass route children) - a fake page body here flashes
 * as the wrong skeleton before the real OpsOverview / DataTable shimmer.
 */
export function AdminShellSkeleton({ children }: { children?: ReactNode }) {
  return (
    <div className="flex min-h-screen bg-background text-foreground">
      <aside className="sticky top-0 hidden h-screen w-60 shrink-0 flex-col self-start overflow-hidden border-r border-border bg-card md:flex">
        <div className="space-y-2 border-b border-border px-4 py-4">
          <TrimWordmark size="sm" alt="Trim" />
          <Skeleton className="h-3 w-40" />
          <Skeleton className="h-3 w-20" />
        </div>
        <div className="flex flex-col gap-1 p-2">
          {shimmerKeys("nav", 10).map((id) => (
            <Skeleton key={id} className="h-9 w-full rounded-md" />
          ))}
        </div>
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <div className="sticky top-0 z-40 flex h-14 shrink-0 items-center gap-2 border-b border-border bg-card/95 px-3 backdrop-blur supports-[backdrop-filter]:bg-card/80 sm:px-6">
          <Skeleton className="h-9 w-9 shrink-0 rounded-md md:hidden" />
          <div className="md:hidden">
            <TrimWordmark size="sm" alt="Trim" />
          </div>
          {/* Cold boot: theme (icon h-9) + bell (sm h-8) + avatar menu (h-9). */}
          <div className="ml-auto flex items-center gap-2">
            <Skeleton className="h-9 w-9 shrink-0 rounded-md" />
            <Skeleton className="h-8 w-8 shrink-0 rounded-md" />
            <Skeleton className="h-9 w-36 shrink-0 rounded-md" />
          </div>
        </div>
        <div className="flex-1 p-3 sm:p-4 md:p-6">
          <div className="mx-auto w-full max-w-7xl">{children}</div>
        </div>
      </div>
    </div>
  );
}

export type ReceiptSkeletonChrome = {
  col_description?: string;
  col_product?: string;
  col_qty?: string;
  col_unit?: string;
  col_tax_rate?: string;
  col_amount?: string;
  section_bill_to?: string;
  section_invoice_from?: string;
  section_invoice_details?: string;
  section_transaction?: string;
  section_tax_breakdown?: string;
  section_payment?: string;
  footer?: string;
};

/**
 * Admin receipt detail shimmer - mirrors billing ReceiptView tax-invoice article.
 */
export function ReceiptDetailSkeleton({
  chrome,
  lineRows = 1,
}: {
  chrome?: ReceiptSkeletonChrome;
  lineRows?: number;
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
      <article className="bg-[var(--trim-panel)] text-[var(--trim-fg)]">
        <header className="bg-[var(--trim-panel-2)] px-5 py-6 sm:px-8 sm:py-7">
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
              <h2 className="text-[13px] font-bold">{chrome.section_bill_to}</h2>
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
              <h2 className="text-[13px] font-bold">{chrome.section_invoice_from}</h2>
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
            <h2 className="text-[13px] font-bold">{chrome.section_invoice_details}</h2>
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
            <h2 className="mb-4 text-[13px] font-bold">{chrome.section_transaction}</h2>
          ) : (
            <Skeleton className="mb-4 h-4 w-24" />
          )}
          <div className="overflow-x-auto">
            <table className="w-full min-w-[520px] border-collapse text-left text-[13px]">
              <thead>
                <tr className="border-b border-[var(--trim-border)]">
                  <th className="pb-2.5 pr-3 text-[12px] font-bold">
                    {productCol || <Skeleton className="inline-block h-3 w-16" />}
                  </th>
                  <th className="px-2 pb-2.5 text-right text-[12px] font-bold">
                    {chrome?.col_qty || <Skeleton className="ml-auto inline-block h-3 w-8" />}
                  </th>
                  <th className="px-2 pb-2.5 text-right text-[12px] font-bold">
                    {chrome?.col_unit || <Skeleton className="ml-auto inline-block h-3 w-14" />}
                  </th>
                  <th className="px-2 pb-2.5 text-right text-[12px] font-bold">
                    {chrome?.col_tax_rate || <Skeleton className="ml-auto inline-block h-3 w-14" />}
                  </th>
                  <th className="pb-2.5 pl-2 text-right text-[12px] font-bold">
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
            <div className="w-full max-w-[240px] space-y-0">
              {[0, 1, 2, 3].map((i) => (
                <div
                  key={i}
                  className="flex items-center justify-between gap-8 border-b border-[var(--trim-border)] py-2"
                >
                  <Skeleton className={`h-4 ${i === 3 ? "w-24" : "w-16"}`} />
                  <Skeleton className={`h-4 ${i === 3 ? "w-14" : "w-12"}`} />
                </div>
              ))}
            </div>
          </div>
          <div className="mt-8 max-w-[240px]">
            {chrome?.section_tax_breakdown ? (
              <h2 className="text-[13px] font-bold">{chrome.section_tax_breakdown}</h2>
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

        <footer className="mt-4 bg-[var(--trim-panel-2)] px-5 py-7 text-center sm:px-8">
          <div className="mx-auto flex max-w-lg flex-col items-center gap-3">
            {chrome?.footer ? (
              <p className="text-xs text-muted-foreground">{chrome.footer}</p>
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
