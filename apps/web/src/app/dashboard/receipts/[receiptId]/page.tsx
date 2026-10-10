"use client";

import { ReceiptView } from "@/components/billing/receipt-view";
import { useDashboardSession } from "@/hooks/use-dashboard-session";
import { useParams } from "next/navigation";
import { Suspense } from "react";

function ReceiptPageInner() {
  const params = useParams<{ receiptId: string }>();
  const { accessToken } = useDashboardSession();
  return <ReceiptView receiptId={params.receiptId} accessToken={accessToken} />;
}

export default function ReceiptPage() {
  return (
    <Suspense fallback={null}>
      <ReceiptPageInner />
    </Suspense>
  );
}
