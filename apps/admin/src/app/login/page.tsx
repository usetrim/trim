import { AuthPageSkeleton } from "@/components/skeletons/page-skeletons";
import { allowedAuthProvidersFromEnv } from "@/lib/auth-providers";
import { Suspense } from "react";
import LoginPage from "./login-client";

export default function Page() {
  return (
    <Suspense fallback={<AuthPageSkeleton providerSlots={allowedAuthProvidersFromEnv().length} />}>
      <LoginPage />
    </Suspense>
  );
}
