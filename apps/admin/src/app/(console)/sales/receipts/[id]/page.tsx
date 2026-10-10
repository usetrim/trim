"use client";

import { ReceiptView } from "@/components/billing/receipt-view";
import { useAdminToken } from "@/hooks/use-admin-token";
import { useParams } from "next/navigation";
import { Suspense } from "react";

function SalesReceiptInner() {
  const params = useParams<{ id: string }>();
  const token = useAdminToken();
  return <ReceiptView receiptId={params.id} accessToken={token} listHref="/sales/receipts" />;
}

export default function SalesReceiptPage() {
  return (
    <Suspense fallback={null}>
      <SalesReceiptInner />
    </Suspense>
  );
}
