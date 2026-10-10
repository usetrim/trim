"use client";

import {
  DashboardPageSkeleton,
  DedicatedListPageSkeleton,
  ReceiptDetailSkeleton,
  SettingsPageSkeleton,
  TeamPageSkeleton,
} from "@/components/skeletons/page-skeletons";
import { useAuthProviders } from "@/hooks/queries/auth";
import { skeletonPageRows, useDefaultPageSize } from "@/hooks/use-default-page-size";
import { authProviderSlotCount } from "@/lib/auth-providers";
import {
  enterpriseListSkeletonProps,
  receiptsListSkeletonProps,
  tracesListSkeletonProps,
} from "@/lib/dedicated-list-skeletons";
import {
  dashboardSkeletonChrome,
  receiptSkeletonChrome,
  settingsSkeletonChrome,
  teamSkeletonChrome,
} from "@/lib/skeleton-chrome";
import { usePathname } from "next/navigation";

/**
 * Route-faithful body shimmer while session settles / page data loads.
 * Must match page clients' DedicatedListPageSkeleton props exactly.
 */
export function DashboardBootSkeleton() {
  const pathname = usePathname();
  const authProviders = useAuthProviders();
  const pageSize = useDefaultPageSize();
  const rows = skeletonPageRows(pageSize);
  const site = authProviders.data?.site;
  const providerSlots = authProviderSlotCount(
    authProviders.data?.provider_count ?? authProviders.data?.items?.length,
  );

  if (pathname.includes("/team")) {
    return (
      <TeamPageSkeleton
        chrome={teamSkeletonChrome(site)}
        workspaceRows={rows}
        memberRows={rows}
        embedded
      />
    );
  }
  if (pathname.includes("/settings")) {
    return (
      <SettingsPageSkeleton
        chrome={settingsSkeletonChrome(site)}
        keyRows={rows}
        providerSlots={providerSlots}
        identityRows={Math.max(1, providerSlots || 1)}
        embedded
      />
    );
  }
  // Receipt detail before list: /dashboard/receipts/{id}
  if (/\/receipts\/[^/]+/.test(pathname)) {
    return <ReceiptDetailSkeleton chrome={receiptSkeletonChrome(site)} lineRows={0} />;
  }
  if (pathname.includes("/receipts")) {
    return <DedicatedListPageSkeleton {...receiptsListSkeletonProps(site, rows)} />;
  }
  if (pathname.includes("/traces")) {
    return <DedicatedListPageSkeleton {...tracesListSkeletonProps(site, rows)} />;
  }
  if (pathname.includes("/enterprise")) {
    return <DedicatedListPageSkeleton {...enterpriseListSkeletonProps(site, rows)} />;
  }

  return (
    <DashboardPageSkeleton
      chrome={dashboardSkeletonChrome(site, {
        primary: true,
        topup: true,
        portal: false,
        team: false,
        settings: false,
        notifications: false,
      })}
      tableRows={rows}
      embedded
    />
  );
}
