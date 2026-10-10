import { LoginPanel } from "@/components/auth/login-panel";
import { AuthPageSkeleton } from "@/components/skeletons/page-skeletons";
import { allowedAuthProvidersFromEnv } from "@/lib/auth-providers";
import { Suspense } from "react";

export default function Page() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-[50vh] items-center justify-center px-5 py-16">
          <AuthPageSkeleton providerSlots={allowedAuthProvidersFromEnv().length} />
        </div>
      }
    >
      <LoginPanel embedded />
    </Suspense>
  );
}
