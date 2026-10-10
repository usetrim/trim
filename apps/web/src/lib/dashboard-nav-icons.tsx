import type { LucideIcon } from "lucide-react";
import { Activity, Building2, LayoutDashboard, Receipt, Settings, Users } from "lucide-react";

/**
 * Code-only customer console nav icons keyed by stable nav id.
 * Labels/hrefs stay site_messages / subscription chrome; icons are never inventing
 * from display text. Unknown ids → no icon (fail closed).
 */
export const DASHBOARD_NAV_ICONS: Readonly<Record<string, LucideIcon>> = {
  dashboard: LayoutDashboard,
  traces: Activity,
  receipts: Receipt,
  enterprise: Building2,
  team: Users,
  settings: Settings,
};

export function dashboardNavIcon(id: string | undefined | null): LucideIcon | null {
  const key = typeof id === "string" ? id.trim() : "";
  if (!key) return null;
  return DASHBOARD_NAV_ICONS[key] ?? null;
}
