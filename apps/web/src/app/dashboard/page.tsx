"use client";

import { DashboardClient } from "@/components/dashboard/dashboard-client";
import { useDashboardSession } from "@/hooks/use-dashboard-session";
import { Suspense } from "react";

function DashboardHomeInner() {
  const { accessToken, userId, email, fullName } = useDashboardSession();
  return (
    <DashboardClient
      accessToken={accessToken}
      userId={userId}
      email={email}
      fullName={fullName}
      sessionReady
    />
  );
}

export default function DashboardHomePage() {
  return (
    <Suspense fallback={null}>
      <DashboardHomeInner />
    </Suspense>
  );
}
