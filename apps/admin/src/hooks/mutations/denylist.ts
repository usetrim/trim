import { adminFetch } from "@/lib/admin-api/client";
import { adminQk } from "@/lib/query-keys";
import type { DenylistAsnBody, DenylistEmailBody, DenylistIpBody } from "@/types/admin";
import { useMutation, useQueryClient } from "@tanstack/react-query";

export function useAddEmailDenylist(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: DenylistEmailBody) =>
      adminFetch("/api/v1/admin/denylist/email", {
        method: "POST",
        token,
        body: JSON.stringify(body),
      }),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.denylistEmail }),
        qc.invalidateQueries({ queryKey: adminQk.audit }),
      ]);
    },
  });
}

export function useAddIpDenylist(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: DenylistIpBody) =>
      adminFetch("/api/v1/admin/denylist/ip", {
        method: "POST",
        token,
        body: JSON.stringify(body),
      }),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.denylistIp }),
        qc.invalidateQueries({ queryKey: adminQk.audit }),
      ]);
    },
  });
}

export function useAddAsnDenylist(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: DenylistAsnBody) =>
      adminFetch("/api/v1/admin/denylist/asn", {
        method: "POST",
        token,
        body: JSON.stringify(body),
      }),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.denylistAsn }),
        qc.invalidateQueries({ queryKey: adminQk.audit }),
      ]);
    },
  });
}

export function useRemoveEmailDenylist(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (domain: string) =>
      adminFetch(`/api/v1/admin/denylist/email/${encodeURIComponent(domain)}`, {
        method: "DELETE",
        token,
      }),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.denylistEmail }),
        qc.invalidateQueries({ queryKey: adminQk.audit }),
      ]);
    },
  });
}

export function useRemoveIpDenylist(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cidr: string) =>
      adminFetch(`/api/v1/admin/denylist/ip/${encodeURIComponent(cidr)}`, {
        method: "DELETE",
        token,
      }),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.denylistIp }),
        qc.invalidateQueries({ queryKey: adminQk.audit }),
      ]);
    },
  });
}

export function useRemoveAsnDenylist(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (asn: number) =>
      adminFetch(`/api/v1/admin/denylist/asn/${asn}`, {
        method: "DELETE",
        token,
      }),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.denylistAsn }),
        qc.invalidateQueries({ queryKey: adminQk.audit }),
      ]);
    },
  });
}
