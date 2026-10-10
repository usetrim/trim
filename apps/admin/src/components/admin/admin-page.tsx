"use client";

import { AdminGate, useAdminSession } from "@/components/admin/admin-gate";
import { AdminShell } from "@/components/layout/admin-shell";
import { AdminTokenProvider } from "@/hooks/use-admin-token";
import type { ReactNode } from "react";

function AdminAuthedShell({ children }: { children: ReactNode }) {
  const { token, me } = useAdminSession();
  return (
    <AdminTokenProvider token={token}>
      <AdminShell token={token} me={me}>
        {children}
      </AdminShell>
    </AdminTokenProvider>
  );
}

/**
 * Persistent console chrome. Gate + shell stay mounted across soft-nav;
 * only `children` (the page) swaps.
 */
export function AdminPage({
  initialToken,
  children,
}: {
  initialToken?: string;
  children: ReactNode;
}) {
  return (
    <AdminGate initialToken={initialToken}>
      <AdminAuthedShell>{children}</AdminAuthedShell>
    </AdminGate>
  );
}
