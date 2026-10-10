"use client";

import { ThemeApplier } from "@/components/theme/theme-applier";
import { FaviconThemeSync } from "@/components/theme/favicon-theme-sync";
import { Toaster } from "@/components/ui/sonner";
import { adminQk } from "@/lib/query-keys";
import { toastApiError, toastApiSuccess } from "@/lib/toast-api";
import { useStepUpStore } from "@/stores/step-up";
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
  const [client] = useState(() => {
    const qc = new QueryClient({
      mutationCache: new MutationCache({
        onSuccess: (data, _vars, _ctx, mutation) => {
          if (mutation.meta?.skipToast) return;
          toastApiSuccess(data);
        },
        onError: (error, _vars, _ctx, mutation) => {
          if (mutation.meta?.skipToast) return;
          const err = error as { body?: { code?: unknown }; status?: number };
          const code = typeof err.body?.code === "string" ? err.body.code : "";
          // Enroll-only policy: never toast per-action step-up copy.
          if (code === "ADMIN_STEP_UP_REQUIRED") {
            return;
          }
          if (code === "ADMIN_STEP_UP_FACTOR_REQUIRED") {
            // Unenrolled only - open enroll dialog; dialog self-closes once enrolled.
            useStepUpStore.getState().openVerify();
            // Refresh factor status so an already-enrolled session closes immediately.
            void qc.invalidateQueries({ queryKey: adminQk.totp });
            void qc.invalidateQueries({ queryKey: adminQk.webauthn });
            return;
          }
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
    });
    return qc;
  });

  return (
    <QueryClientProvider client={client}>
      <ThemeApplier />
      <FaviconThemeSync />
      <DevToastBridge />
      {children}
      <Toaster closeButton />
    </QueryClientProvider>
  );
}
