import { adminFetch } from "@/lib/admin-api/client";
import { downloadRowsAsXlsx } from "@/lib/export-xlsx";
import { invalidateAfterUserMutation } from "@/lib/query-keys";
import type { CreditGrantBody, UserQuotaBody, UserStatusBody } from "@/types/admin";
import { useMutation, useQueryClient } from "@tanstack/react-query";

function gdprExportRows(data: {
  export?: unknown;
  generated_at?: string;
}): Array<Record<string, unknown>> {
  const payload = data.export;
  if (Array.isArray(payload)) {
    return payload.filter(
      (row): row is Record<string, unknown> =>
        Boolean(row) && typeof row === "object" && !Array.isArray(row),
    );
  }
  if (payload && typeof payload === "object") {
    return Object.entries(payload as Record<string, unknown>).map(([key, value]) => ({
      key,
      value:
        value != null && typeof value === "object" ? JSON.stringify(value) : (value as unknown),
    }));
  }
  if (data.generated_at) {
    return [{ generated_at: data.generated_at }];
  }
  return [];
}

export function useAdminUserStatus(token: string, userId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: UserStatusBody) =>
      adminFetch(`/api/v1/admin/users/${userId}/status`, {
        method: "POST",
        token,
        body: JSON.stringify(body),
      }),
    onSuccess: async () => {
      await invalidateAfterUserMutation(qc, userId);
    },
  });
}

export function useAdminUserNotes(token: string, userId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (notes: string) =>
      adminFetch(`/api/v1/admin/users/${userId}/notes`, {
        method: "PATCH",
        token,
        body: JSON.stringify({ notes }),
      }),
    onSuccess: async () => {
      await invalidateAfterUserMutation(qc, userId);
    },
  });
}

export function useAdminUserQuota(token: string, userId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: UserQuotaBody) =>
      adminFetch(`/api/v1/admin/users/${userId}/quota`, {
        method: "POST",
        token,
        body: JSON.stringify(body),
      }),
    onSuccess: async () => {
      await invalidateAfterUserMutation(qc, userId);
    },
  });
}

export function useAdminUserRevokeKeys(token: string, userId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () =>
      adminFetch(`/api/v1/admin/users/${userId}/revoke-keys`, {
        method: "POST",
        token,
      }),
    onSuccess: async () => {
      await invalidateAfterUserMutation(qc, userId);
    },
  });
}

export function useAdminUserForceLogout(token: string, userId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (reason: string) =>
      adminFetch(`/api/v1/admin/users/${userId}/force-logout`, {
        method: "POST",
        token,
        body: JSON.stringify({ reason }),
      }),
    onSuccess: async () => {
      await invalidateAfterUserMutation(qc, userId);
    },
  });
}

export function useAdminUserGDPRExport(token: string, userId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (reason: string) => {
      const data = await adminFetch<{
        message?: string;
        export?: unknown;
        generated_at?: string;
        filename?: string;
      }>(`/api/v1/admin/users/${userId}/gdpr-export`, {
        method: "POST",
        token,
        body: JSON.stringify({ reason }),
      });
      const nameFromApi = (data.filename || "").trim();
      // Server filename only (site_messages). Fail closed; never invent a download name.
      if (!nameFromApi) {
        throw new Error("");
      }
      await downloadRowsAsXlsx(gdprExportRows(data), nameFromApi, "GDPR");
      return data;
    },
    onSuccess: async () => {
      await invalidateAfterUserMutation(qc, userId);
    },
  });
}

export function useAdminUserGDPRErase(token: string, userId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (reason: string) =>
      adminFetch(`/api/v1/admin/users/${userId}/gdpr-erase`, {
        method: "POST",
        token,
        body: JSON.stringify({ reason }),
      }),
    onSuccess: async () => {
      await invalidateAfterUserMutation(qc, userId);
    },
  });
}

export function useCreditGrant(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: CreditGrantBody) =>
      adminFetch("/api/v1/admin/billing/credits", {
        method: "POST",
        token,
        body: JSON.stringify(body),
      }),
    onSuccess: async (_data, vars) => {
      await invalidateAfterUserMutation(qc, vars.user_id);
    },
  });
}
