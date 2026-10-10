import { adminFetch } from "@/lib/admin-api/client";
import { withQuery } from "@/lib/admin-api/query-string";
import { adminQk } from "@/lib/query-keys";
import { useStepUpStore } from "@/stores/step-up";
import type { StepUpResponse, TOTPBeginResponse, WebAuthnOptionsPayload } from "@/types/admin";
import { useMutation, useQueryClient } from "@tanstack/react-query";

function applyStepUpToken(data: unknown) {
  const raw = (data ?? {}) as Record<string, unknown>;
  const tok = String(raw.step_up_token || raw.token || "").trim();
  const ttl = Number(raw.expires_in_sec ?? raw.expires_in ?? 0);
  if (!tok) return;
  useStepUpStore.getState().setToken(tok, ttl);
}

export function useStepUp(token: string) {
  return useMutation({
    mutationFn: (code: string) =>
      adminFetch<StepUpResponse>("/api/v1/admin/auth/step-up", {
        method: "POST",
        token,
        body: JSON.stringify({ code }),
      }),
    onSuccess: (data) => {
      applyStepUpToken(data);
    },
    meta: { skipToast: true },
  });
}

export function useTOTPBegin(token: string) {
  return useMutation({
    mutationFn: () =>
      adminFetch<TOTPBeginResponse>("/api/v1/admin/auth/totp/begin", {
        method: "POST",
        token,
      }),
    meta: { skipToast: true },
  });
}

export function useTOTPConfirm(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (code: string) =>
      adminFetch<StepUpResponse>("/api/v1/admin/auth/totp/confirm", {
        method: "POST",
        token,
        body: JSON.stringify({ code }),
      }),
    onSuccess: async (data) => {
      // Enroll proof also elevates - one TOTP entry unlocks the session TTL.
      applyStepUpToken(data);
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.me }),
        qc.invalidateQueries({ queryKey: adminQk.totp }),
      ]);
    },
    meta: { skipToast: true },
  });
}

export function useTOTPDisable(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (code: string) =>
      adminFetch<{ message?: string }>("/api/v1/admin/auth/totp/disable", {
        method: "POST",
        token,
        body: JSON.stringify({ code }),
      }),
    onSuccess: async () => {
      useStepUpStore.getState().clear();
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.me }),
        qc.invalidateQueries({ queryKey: adminQk.totp }),
      ]);
    },
  });
}

export function useWebAuthnRegisterBegin(token: string) {
  return useMutation({
    mutationFn: () =>
      adminFetch<WebAuthnOptionsPayload>("/api/v1/admin/auth/webauthn/register/begin", {
        method: "POST",
        token,
      }),
    meta: { skipToast: true },
  });
}

export function useWebAuthnRegisterFinish(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: {
      credential: Record<string, unknown>;
      friendlyName?: string;
    }) =>
      adminFetch<{ id: string } & StepUpResponse>(
        withQuery("/api/v1/admin/auth/webauthn/register/finish", {
          friendly_name: input.friendlyName || undefined,
        }),
        {
          method: "POST",
          token,
          body: JSON.stringify(input.credential),
        },
      ),
    onSuccess: async (data) => {
      applyStepUpToken(data);
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.me }),
        qc.invalidateQueries({ queryKey: adminQk.webauthn }),
      ]);
    },
    meta: { skipToast: true },
  });
}

export function useWebAuthnAssertBegin(token: string) {
  return useMutation({
    mutationFn: () =>
      adminFetch<WebAuthnOptionsPayload>("/api/v1/admin/auth/webauthn/assert/begin", {
        method: "POST",
        token,
      }),
    meta: { skipToast: true },
  });
}

export function useWebAuthnAssertFinish(token: string) {
  return useMutation({
    mutationFn: (credential: Record<string, unknown>) =>
      adminFetch<StepUpResponse>("/api/v1/admin/auth/webauthn/assert/finish", {
        method: "POST",
        token,
        body: JSON.stringify(credential),
      }),
    onSuccess: (data) => {
      applyStepUpToken(data);
    },
    meta: { skipToast: true },
  });
}

export function useWebAuthnRemove(token: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      adminFetch<{ message?: string }>(`/api/v1/admin/auth/webauthn/${encodeURIComponent(id)}`, {
        method: "DELETE",
        token,
      }),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: adminQk.me }),
        qc.invalidateQueries({ queryKey: adminQk.webauthn }),
      ]);
    },
  });
}
