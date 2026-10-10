import { apiFetch, requireAccessToken } from "@/lib/api/client";
import { invalidateAccount, qk } from "@/lib/query-keys";
import type {
  AvatarSyncResponse,
  DeleteAccountResponse,
  PatchPreferencesBody,
  PreferencesResponse,
} from "@/types/me";
import { useMutation, useQueryClient } from "@tanstack/react-query";

export function useAvatarSync(token: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      requireAccessToken(token);
      return apiFetch<AvatarSyncResponse>("/api/v1/me/avatar/sync", {
        method: "POST",
        token,
      });
    },
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: qk.preferences }),
        qc.invalidateQueries({ queryKey: qk.subscription }),
        qc.invalidateQueries({ queryKey: qk.quota }),
      ]);
    },
  });
}

export function useDeleteAccount(token: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      requireAccessToken(token);
      return apiFetch<DeleteAccountResponse>("/api/v1/me/account", {
        method: "DELETE",
        token,
      });
    },
    onSuccess: async () => {
      await invalidateAccount(qc);
      qc.clear();
    },
  });
}

export function usePatchPreferences(token: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: PatchPreferencesBody) => {
      requireAccessToken(token);
      return apiFetch<PreferencesResponse>("/api/v1/me/preferences", {
        method: "PATCH",
        token,
        body: JSON.stringify(body),
      });
    },
    onSuccess: async (data) => {
      qc.setQueryData(qk.preferences, data);
      await qc.invalidateQueries({ queryKey: qk.preferences });
    },
  });
}

export function useBulkDeleteEvents(token: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (ids: string[]) => {
      requireAccessToken(token);
      return apiFetch<{ deleted?: number; message?: string }>("/api/v1/me/events", {
        method: "DELETE",
        token,
        body: JSON.stringify({ ids }),
      });
    },
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: qk.events }),
        qc.invalidateQueries({ queryKey: qk.eventStats }),
      ]);
    },
  });
}

export function useBulkDeleteEnterpriseInquiries(token: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (ids: string[]) => {
      requireAccessToken(token);
      return apiFetch<{ deleted?: number; message?: string }>("/api/v1/me/enterprise-inquiries", {
        method: "DELETE",
        token,
        body: JSON.stringify({ ids }),
      });
    },
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: qk.enterpriseInquiries });
    },
  });
}
