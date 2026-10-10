"use client";

import {
  DataTableSkeleton,
  DualTableSkeleton,
  FormPageSkeleton,
  OpsOverviewSkeleton,
  TabsSkeleton,
} from "@/components/skeletons/page-skeletons";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useDefaultPageSize } from "@/hooks/use-default-page-size";
import { usePathname } from "next/navigation";

/**
 * Route-faithful body shimmer while AdminGate settles.
 * Matches the page clients' boot skeletons so cold login does not flash a
 * generic wrong body before the real page shimmer mounts.
 */
export function AdminBootSkeleton() {
  const pathname = usePathname();
  const pageSize = useDefaultPageSize();
  const rows = pageSize > 0 ? pageSize : 8;

  if (pathname === "/" || pathname === "") {
    return <OpsOverviewSkeleton metricCount={12} rows={4} />;
  }

  if (pathname.startsWith("/billing/settings") || pathname.startsWith("/product")) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-7 w-40" />
        <TabsSkeleton count={pathname.startsWith("/product") ? 5 : 4} />
        <FormPageSkeleton hideTitle fields={pathname.startsWith("/product") ? 6 : 8} />
        {pathname.startsWith("/product") ? (
          <DataTableSkeleton
            title=""
            columns={[{ label: "" }, { label: "" }]}
            rows={6}
            showFilters={false}
          />
        ) : null}
      </div>
    );
  }

  if (pathname.startsWith("/auth")) {
    return (
      <div className="space-y-4">
        <Card>
          <CardContent className="pt-6">
            <Skeleton className="h-24 w-full rounded-md" />
          </CardContent>
        </Card>
        <DualTableSkeleton
          primaryColumns={[{ label: "" }, { label: "" }]}
          secondaryColumns={[{ label: "" }, { label: "" }]}
          rows={6}
          showFilters={false}
        />
      </div>
    );
  }

  // Detail routes and most console lists share a table-shaped boot.
  return (
    <div className="space-y-4">
      <Skeleton className="h-7 w-40" />
      <DataTableSkeleton
        title=""
        columns={[{ label: "" }, { label: "" }, { label: "" }, { label: "" }, { label: "" }]}
        rows={rows}
        filterCount={pathname.startsWith("/users/") ? 0 : 4}
        showFilters={!pathname.startsWith("/users/")}
      />
    </div>
  );
}
