"use client";

import { releaseBodyPointerEvents } from "@/hooks/use-deferred-dialog-selection";
import { useRouter } from "next/navigation";
import { useCallback } from "react";

/**
 * Navigate to a notification href even when already on that URL.
 * Soft-nav Link to the same path+query does nothing, which froze deep-link
 * dialogs after the first open until a full refresh.
 */
export function useNotificationHrefNavigate() {
  const router = useRouter();

  return useCallback(
    (href: string) => {
      const target = href.trim();
      if (!target.startsWith("/") || target.startsWith("//")) return;

      releaseBodyPointerEvents();

      const current =
        typeof window !== "undefined" ? `${window.location.pathname}${window.location.search}` : "";

      if (current === target) {
        // Replay: clear query then re-apply so destination deep-link effects re-run.
        try {
          const u = new URL(target, window.location.origin);
          router.replace(u.pathname);
          window.setTimeout(() => {
            router.push(target);
            releaseBodyPointerEvents();
          }, 0);
        } catch {
          router.push(target);
        }
        return;
      }

      router.push(target);
    },
    [router],
  );
}
