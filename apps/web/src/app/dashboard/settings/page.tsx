"use client";

import { SettingsClient } from "@/components/settings/settings-client";
import { useDashboardSession } from "@/hooks/use-dashboard-session";
import { Suspense } from "react";

function SettingsBody() {
  const { accessToken } = useDashboardSession();
  return <SettingsClient accessToken={accessToken} />;
}

export default function SettingsPage() {
  return (
    <Suspense fallback={null}>
      <SettingsBody />
    </Suspense>
  );
}
