"use client";

import { DashboardGate, DashboardMain } from "@/components/dashboard/dashboard-gate";
import { DashboardShell } from "@/components/dashboard/dashboard-shell";
import type { ReactNode } from "react";

/**
 * Persistent customer console. Gate + shell stay mounted across soft-nav and
 * hard refresh; only the body swaps BootSkeleton → page (no ShellSkeleton remount).
 */
export function DashboardPage({ children }: { children: ReactNode }) {
  return (
    <DashboardGate>
      <DashboardShell>
        <DashboardMain>{children}</DashboardMain>
      </DashboardShell>
    </DashboardGate>
  );
}
