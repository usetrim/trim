"use client";

import { useAuthProviders } from "@/hooks/queries/auth";
import { useEffect, useState } from "react";

/**
 * Stable default page size from public auth-providers.
 * Soft-nav remounts keep the last known size so list/skeleton rows do not
 * collapse to 0 (partial shimmer) while chrome revalidates.
 */
export function useDefaultPageSize(): number {
  const providers = useAuthProviders();
  const size = providers.data?.default_page_size;
  const resolved = typeof size === "number" && size > 0 ? size : 0;
  const [stable, setStable] = useState(resolved);

  useEffect(() => {
    if (resolved > 0) setStable(resolved);
  }, [resolved]);

  return resolved > 0 ? resolved : stable;
}

/** Skeleton/table placeholder rows while page size is still unknown (admin parity). */
export function skeletonPageRows(pageSize: number, fallback = 8): number {
  return pageSize > 0 ? pageSize : fallback;
}
