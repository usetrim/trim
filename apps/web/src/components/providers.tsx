"use client";

import { ThemeProvider } from "@/components/theme/theme-provider";
import { ThemeAttrSync } from "@/components/theme/theme-attr-sync";
import { FaviconThemeSync } from "@/components/theme/favicon-theme-sync";
import { Toaster } from "@/components/ui/sonner";
import { toastApiError, toastApiSuccess } from "@/lib/toast-api";
import {
  MutationCache,
  QueryClient,
  QueryClientProvider,
  useMutation,
} from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { toast } from "sonner";

/** Dev-only bridges for live toast verification. */
function DevToastBridge() {
  const probe = useMutation({
    mutationFn: async () => ({ message: "MutationCache smoke OK" }),
  });

  useEffect(() => {
    if (process.env.NODE_ENV !== "development") return;
    const w = window as Window & {
      __trimToast?: typeof toast;
      __trimMutationToast?: () => void;
    };
    w.__trimToast = toast;
    w.__trimMutationToast = () => {
      probe.mutate();
    };
    return () => {
      w.__trimToast = undefined;
      w.__trimMutationToast = undefined;
    };
  }, [probe]);

  return null;
}

export function Providers({ children }: { children: React.ReactNode }) {
  const [client] = useState(
    () =>
      new QueryClient({
        mutationCache: new MutationCache({
          onSuccess: (data, _vars, _ctx, mutation) => {
            if (mutation.meta?.skipToast) return;
            toastApiSuccess(data);
          },
          onError: (error, _vars, _ctx, mutation) => {
            if (mutation.meta?.skipToast) return;
            toastApiError(error);
          },
        }),
        defaultOptions: {
          queries: {
            // Soft-nav reuses warm cache for 5m; mutations call invalidateQueries.
            // Next.js experimental.staleTimes (same window) stops RSC remount churn.
            staleTime: 5 * 60_000,
            gcTime: 30 * 60_000,
            retry: 1,
            refetchOnWindowFocus: false,
            refetchOnReconnect: true,
            // Remount refetch only when stale (after invalidate / TTL) - not every nav.
            refetchOnMount: true,
            // Do NOT set placeholderData globally - list hooks opt in with keepPreviousData.
          },
          mutations: {
            retry: 0,
          },
        },
      }),
  );

  return (
    <ThemeProvider>
      <ThemeAttrSync />
      <FaviconThemeSync />
      <QueryClientProvider client={client}>
        <DevToastBridge />
        {children}
        <Toaster closeButton />
      </QueryClientProvider>
    </ThemeProvider>
  );
}
