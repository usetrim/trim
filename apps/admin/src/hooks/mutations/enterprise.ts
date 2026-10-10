import { adminFetch } from "@/lib/admin-api/client";
import { adminQk } from "@/lib/query-keys";
import { useMutation, useQueryClient } from "@tanstack/react-query";

const refetchActive = { refetchType: "active" as const };

export function usePatchEnterpriseInquiry(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      status,
      contract_notes,
      offered_seat_quantity,
    }: {
      id: string;
      status?: string;
      contract_notes?: string;
      offered_seat_quantity?: number;
    }) =>
      adminFetch(`/api/v1/admin/enterprise/inquiries/${id}`, {
        method: "PATCH",
        token,
        body: JSON.stringify({
          ...(status ? { status } : {}),
          contract_notes,
          offered_seat_quantity,
        }),
      }),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.enterprise, ...refetchActive }),
        qc.invalidateQueries({ queryKey: adminQk.audit, ...refetchActive }),
      ]);
    },
  });
}

export function useOfferEnterpriseInquiry(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      adminFetch<{
        id?: string;
        status?: string;
        message?: string;
      }>(`/api/v1/admin/enterprise/inquiries/${encodeURIComponent(id)}/offer`, {
        method: "POST",
        token,
      }),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.enterprise, ...refetchActive }),
        qc.invalidateQueries({ queryKey: adminQk.users, ...refetchActive }),
        qc.invalidateQueries({ queryKey: adminQk.subscriptions, ...refetchActive }),
        qc.invalidateQueries({ queryKey: adminQk.segmentsEnterprises, ...refetchActive }),
        qc.invalidateQueries({ queryKey: adminQk.dashboard, ...refetchActive }),
        qc.invalidateQueries({ queryKey: adminQk.audit, ...refetchActive }),
      ]);
    },
  });
}

/** @deprecated Use useOfferEnterpriseInquiry - Activate is an offer alias only. */
export const useActivateEnterpriseInquiry = useOfferEnterpriseInquiry;
