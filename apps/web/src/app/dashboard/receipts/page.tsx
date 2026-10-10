"use client";

import { ReceiptsListClient } from "@/components/dashboard/receipts-list-client";
import { useDashboardSession } from "@/hooks/use-dashboard-session";
import { Suspense } from "react";

function ReceiptsListBody() {
  const { accessToken } = useDashboardSession();
  return <ReceiptsListClient accessToken={accessToken} />;
}

export default function ReceiptsListPage() {
  return (
    <Suspense fallback={null}>
      <ReceiptsListBody />
    </Suspense>
  );
}
