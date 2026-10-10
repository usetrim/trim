"use client";

import { TracesClient } from "@/components/dashboard/traces-client";
import { useDashboardSession } from "@/hooks/use-dashboard-session";
import { Suspense } from "react";

function TracesBody() {
  const { accessToken } = useDashboardSession();
  return <TracesClient accessToken={accessToken} />;
}

export default function TracesPage() {
  return (
    <Suspense fallback={null}>
      <TracesBody />
    </Suspense>
  );
}
