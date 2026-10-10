"use client";

import { TotpQr } from "@/components/admin/totp-qr";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  useTOTPBegin,
  useTOTPConfirm,
  useWebAuthnRegisterBegin,
  useWebAuthnRegisterFinish,
} from "@/hooks/mutations/step-up";
import { useAdminNavChrome } from "@/hooks/queries/chrome";
import { useAdminTOTPStatus } from "@/hooks/queries/totp";
import { useAdminWebAuthnStatus } from "@/hooks/queries/webauthn";
import { useAccessToken } from "@/hooks/use-access-token";
import { adminQk } from "@/lib/query-keys";
import { createPasskey, webauthnSupported } from "@/lib/webauthn-browser";
import { useStepUpStore } from "@/stores/step-up";
import type { ApiError } from "@/types/admin";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { toast } from "sonner";

function apiErrorMessage(err: unknown): string {
  return err instanceof Error ? err.message.trim() : "";
}

function apiErrorCode(err: unknown): string {
  const code = (err as ApiError | undefined)?.body?.code;
  return typeof code === "string" ? code.trim() : "";
}

function reportApiError(err: unknown, setInline?: (msg: string) => void) {
  const msg = apiErrorMessage(err);
  if (!msg) return;
  toast.error(msg);
  setInline?.(msg);
}

function toastSuccess(msg: string) {
  const t = msg.trim();
  if (t) toast.success(t);
}

/**
 * Shell dialog when writes require MFA enrollment.
 * Once TOTP or passkey is enrolled, CRUD does not re-prompt - this dialog closes.
 */
export function StepUpVerifyDialog({ token }: { token: string }) {
  const access = useAccessToken(token);
  const qc = useQueryClient();
  const open = useStepUpStore((s) => s.verifyOpen);
  const closeVerify = useStepUpStore((s) => s.closeVerify);
  const chrome = useAdminNavChrome(access);
  const totp = useAdminTOTPStatus(access);
  const webauthn = useAdminWebAuthnStatus(access);
  const totpBegin = useTOTPBegin(access);
  const totpConfirm = useTOTPConfirm(access);
  const waRegBegin = useWebAuthnRegisterBegin(access);
  const waRegFinish = useWebAuthnRegisterFinish(access);
  const [code, setCode] = useState("");
  const [codeError, setCodeError] = useState("");
  const [factorError, setFactorError] = useState("");
  const [secret, setSecret] = useState("");
  const [otpauth, setOtpauth] = useState("");
  const [waBusy, setWaBusy] = useState(false);

  const statusPending = totp.isPending || webauthn.isPending;
  const totpEnrolled = Boolean(totp.data?.enrolled);
  const webauthnEnrolled = Boolean(webauthn.data?.enrolled);
  const factorEnrolled = totpEnrolled || webauthnEnrolled;
  const totpSetupStarted = Boolean(secret || otpauth);

  const enrollTitle = chrome.data?.ADMIN_TOTP_ENROLL_TITLE?.trim() || "";
  const codeLabel = chrome.data?.ADMIN_TOTP_CODE_LABEL?.trim() || "";
  const confirmLabel = chrome.data?.ADMIN_TOTP_CONFIRM?.trim() || "";
  const beginLabel = chrome.data?.ADMIN_TOTP_BEGIN?.trim() || "";
  const pending = chrome.data?.ADMIN_PENDING_SAVING?.trim() || "";
  const requiredMsg = chrome.data?.ADMIN_STEP_UP_FACTOR_REQUIRED?.trim() || "";
  const enrollChoice = chrome.data?.ADMIN_STEP_UP_ENROLL_CHOICE?.trim() || "";
  const stepUpSuccess = chrome.data?.ADMIN_STEP_UP_SUCCESS?.trim() || "";
  const totpEnrollSuccess = chrome.data?.ADMIN_TOTP_ENROLL_SUCCESS?.trim() || "";
  const codePlaceholder = chrome.data?.ADMIN_TOTP_CODE_PLACEHOLDER?.trim() || "";
  const secretLabel = chrome.data?.ADMIN_TOTP_SECRET_LABEL?.trim() || "";
  const secretHint = chrome.data?.ADMIN_TOTP_SECRET_HINT?.trim() || "";
  const qrCaption = chrome.data?.ADMIN_TOTP_QR_CAPTION?.trim() || "";
  const waEnrollTitle = chrome.data?.ADMIN_WEBAUTHN_ENROLL_TITLE?.trim() || "";
  const waBeginReg = chrome.data?.ADMIN_WEBAUTHN_BEGIN_REGISTER?.trim() || "";
  const waEnrollSuccess = chrome.data?.ADMIN_WEBAUTHN_ENROLL_SUCCESS?.trim() || "";
  const closeLabel =
    chrome.data?.ADMIN_DIALOG_CLOSE?.trim() || chrome.data?.ADMIN_CLOSE?.trim() || "";
  const browserOK = webauthnSupported();
  const title = enrollTitle || waEnrollTitle || requiredMsg;

  // Enrolled = CRUD unlocked. Never keep a verify wall open after enrollment.
  useEffect(() => {
    if (factorEnrolled && open) closeVerify();
  }, [factorEnrolled, open, closeVerify]);

  const resetLocal = () => {
    setCode("");
    setCodeError("");
    setFactorError("");
    setSecret("");
    setOtpauth("");
  };

  const handleTotpBegin = async () => {
    setFactorError("");
    try {
      const data = await totpBegin.mutateAsync();
      setSecret(data.secret || "");
      setOtpauth(data.otpauth_url || "");
    } catch (err) {
      // Stale client status: server already enrolled - refresh and close.
      if (apiErrorCode(err) === "ADMIN_TOTP_ALREADY_ENROLLED") {
        await qc.invalidateQueries({ queryKey: adminQk.totp });
        resetLocal();
        closeVerify();
        return;
      }
      reportApiError(err, setFactorError);
    }
  };

  const runRegisterPasskey = async () => {
    if (!browserOK || !pending) return;
    setWaBusy(true);
    setFactorError("");
    try {
      const options = await waRegBegin.mutateAsync();
      const credential = await createPasskey(options);
      await waRegFinish.mutateAsync({ credential });
      toastSuccess(stepUpSuccess || waEnrollSuccess);
      resetLocal();
      closeVerify();
    } catch (err) {
      reportApiError(err, setFactorError);
    } finally {
      setWaBusy(false);
    }
  };

  // Wait for factor status before showing enroll chrome (avoids flash for enrolled admins).
  const dialogOpen = open && !factorEnrolled && !statusPending;

  return (
    <Dialog
      open={dialogOpen}
      onOpenChange={(next) => {
        if (!next) {
          resetLocal();
          closeVerify();
        }
      }}
    >
      <DialogContent closeLabel={closeLabel} className="max-w-md">
        <DialogHeader>{title ? <DialogTitle>{title}</DialogTitle> : null}</DialogHeader>
        <div className="space-y-4">
          {requiredMsg ? <p className="text-sm text-muted-foreground">{requiredMsg}</p> : null}
          {enrollChoice ? <p className="text-sm text-muted-foreground">{enrollChoice}</p> : null}
          {!totpEnrolled && beginLabel && pending ? (
            <Button
              type="button"
              variant="secondary"
              isLoading={totpBegin.isPending}
              pendingLabel={pending}
              disabled={totpSetupStarted}
              onClick={() => void handleTotpBegin()}
            >
              {beginLabel}
            </Button>
          ) : null}
          {!totpEnrolled && totpSetupStarted ? (
            <div className="space-y-3">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-start">
                {otpauth ? (
                  <div className="space-y-1.5">
                    <TotpQr otpauthUrl={otpauth} alt={qrCaption || secretLabel} />
                    {qrCaption ? (
                      <p className="max-w-[12rem] text-xs text-muted-foreground">{qrCaption}</p>
                    ) : null}
                  </div>
                ) : null}
                {secret ? (
                  <div className="min-w-0 flex-1 space-y-1.5">
                    {secretLabel ? (
                      <p className="text-xs font-medium text-foreground">{secretLabel}</p>
                    ) : null}
                    <p className="break-all rounded-md border border-border bg-muted/40 px-3 py-2 font-mono text-xs text-foreground">
                      {secret}
                    </p>
                    {secretHint ? (
                      <p className="text-xs text-muted-foreground">{secretHint}</p>
                    ) : null}
                  </div>
                ) : null}
              </div>
              {codeLabel ? (
                <Field id="step-up-dialog-enroll-code" label={codeLabel}>
                  <Input
                    id="step-up-dialog-enroll-code"
                    value={code}
                    onChange={(e) => {
                      setCodeError("");
                      setCode(e.target.value.replace(/\D/g, "").slice(0, 6));
                    }}
                    placeholder={codePlaceholder || "000000"}
                    inputMode="numeric"
                    autoComplete="one-time-code"
                    maxLength={6}
                    aria-invalid={Boolean(codeError)}
                  />
                </Field>
              ) : null}
              {codeError ? <p className="text-sm text-destructive">{codeError}</p> : null}
            </div>
          ) : null}
          {factorError ? <p className="text-sm text-destructive">{factorError}</p> : null}
        </div>
        <DialogFooter>
          {!totpEnrolled && confirmLabel && pending ? (
            <Button
              type="button"
              isLoading={totpConfirm.isPending}
              pendingLabel={pending}
              disabled={code.trim().length !== 6}
              onClick={() =>
                void totpConfirm.mutate(code.trim(), {
                  onSuccess: () => {
                    toastSuccess(stepUpSuccess || totpEnrollSuccess);
                    resetLocal();
                    closeVerify();
                  },
                  onError: (err) => reportApiError(err, setCodeError),
                })
              }
            >
              {confirmLabel}
            </Button>
          ) : null}
          {!webauthnEnrolled && browserOK && waBeginReg && pending ? (
            <Button
              type="button"
              variant="outline"
              isLoading={waBusy || waRegBegin.isPending || waRegFinish.isPending}
              pendingLabel={pending}
              onClick={() => void runRegisterPasskey()}
            >
              {waBeginReg}
            </Button>
          ) : null}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
