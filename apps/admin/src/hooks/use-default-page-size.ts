"use client";

import { useAuthProviders } from "@/hooks/queries/auth";
import { useEffect, useState } from "react";

/**
 * Stable default page size from public auth-providers.
 * Initialize from warm cache on soft-nav remount; never flicker back to 0
 * (that tore down list pages / filters mid-typing).
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
