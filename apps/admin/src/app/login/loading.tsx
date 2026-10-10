import { AuthPageSkeleton } from "@/components/skeletons/page-skeletons";
import { allowedAuthProvidersFromEnv } from "@/lib/auth-providers";

/**
 * Keep a matching AuthPageSkeleton only for cold load of /login (no console shell).
 * Provider slot count matches login-client env hint to avoid a second layout flash.
 */
export default function Loading() {
  return <AuthPageSkeleton providerSlots={allowedAuthProvidersFromEnv().length} />;
}
