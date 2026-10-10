import { DashboardPage } from "@/components/dashboard/dashboard-page";
import type { ReactNode } from "react";

/**
 * Shared dashboard layout: sidebar shell persists; page segments soft-nav inside.
 */
export default function DashboardLayout({ children }: { children: ReactNode }) {
  return <DashboardPage>{children}</DashboardPage>;
}
