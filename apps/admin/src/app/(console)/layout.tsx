import { AdminPage } from "@/components/admin/admin-page";

/**
 * Persistent operator shell. Auth is enforced by middleware + AdminGate.
 * Do NOT await cookies/session here - that re-runs the layout RSC on every
 * soft navigation and remounts the whole client tree (looks like a full refresh).
 */
export default function ConsoleLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <AdminPage>{children}</AdminPage>;
}
