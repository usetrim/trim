import { Suspense } from "react";
import CLIAuthPage from "./cli-auth-client";
import { CliAuthSuspenseFallback } from "./cli-auth-fallback";

export default function Page() {
  return (
    <Suspense fallback={<CliAuthSuspenseFallback />}>
      <CLIAuthPage />
    </Suspense>
  );
}
