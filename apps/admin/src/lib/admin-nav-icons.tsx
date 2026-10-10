import type { LucideIcon } from "lucide-react";
import {
  Activity,
  BadgeCheck,
  Ban,
  Building2,
  Coins,
  Filter,
  KeyRound,
  LayoutDashboard,
  LineChart,
  Mail,
  Package,
  Palette,
  Receipt,
  Repeat,
  ScrollText,
  Settings2,
  Share2,
  Shield,
  Siren,
  Sparkles,
  Users,
} from "lucide-react";

/**
 * Code-only nav icons keyed by admin_nav_items.id.
 * Icons are never stored in the database (labels/hrefs/permissions stay DB-driven).
 * Unknown ids render with no icon (fail closed - no invent default glyph).
 */
export const ADMIN_NAV_ICONS: Readonly<Record<string, LucideIcon>> = {
  dashboard: LayoutDashboard,
  users: Users,
  segments: Filter,
  revenue: LineChart,
  receipts: Receipt,
  subscriptions: Repeat,
  credits: Coins,
  enterprise: Building2,
  plans: Package,
  billing_settings: Settings2,
  rbac: Shield,
  auth: KeyRound,
  denylist: Ban,
  break_glass: Siren,
  product: Sparkles,
  distribution: Share2,
  email: Mail,
  chrome: Palette,
  observability: Activity,
  audit: ScrollText,
  compliance: BadgeCheck,
};

export function adminNavIcon(id: string | undefined | null): LucideIcon | null {
  const key = typeof id === "string" ? id.trim() : "";
  if (!key) return null;
  return ADMIN_NAV_ICONS[key] ?? null;
}
