"use client";

import { EnterpriseInquiriesClient } from "@/components/dashboard/enterprise-inquiries-client";
import { useDashboardSession } from "@/hooks/use-dashboard-session";
import { Suspense } from "react";

function EnterpriseBody() {
  const { accessToken, userId, email } = useDashboardSession();
  return <EnterpriseInquiriesClient accessToken={accessToken} userId={userId} email={email} />;
}

export default function EnterprisePage() {
  return (
    <Suspense fallback={null}>
      <EnterpriseBody />
    </Suspense>
  );
}
