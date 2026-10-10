"use client";

import { ReceiptView } from "@/components/billing/receipt-view";
import { useAdminToken } from "@/hooks/use-admin-token";
import { useParams } from "next/navigation";
import { Suspense } from "react";

function AdminReceiptInner() {
  const params = useParams<{ id: string }>();
  const token = useAdminToken();
  return <ReceiptView receiptId={params.id} accessToken={token} listHref="/billing/receipts" />;
}

export default function AdminReceiptPage() {
  return (
    <Suspense fallback={null}>
      <AdminReceiptInner />
    </Suspense>
  );
}
