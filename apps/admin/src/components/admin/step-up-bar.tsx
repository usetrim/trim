"use client";

import { TotpQr } from "@/components/admin/totp-qr";
import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  useStepUp,
  useTOTPBegin,
  useTOTPConfirm,
  useWebAuthnAssertBegin,
  useWebAuthnAssertFinish,
  useWebAuthnRegisterBegin,
  useWebAuthnRegisterFinish,
} from "@/hooks/mutations/step-up";
import { useAdminNavChrome } from "@/hooks/queries/chrome";
import { useAdminTOTPStatus } from "@/hooks/queries/totp";
import { useAdminWebAuthnStatus } from "@/hooks/queries/webauthn";
import { useAccessToken } from "@/hooks/use-access-token";
import { adminQk } from "@/lib/query-keys";
import { assertPasskey, createPasskey, webauthnSupported } from "@/lib/webauthn-browser";
import { useStepUpStore } from "@/stores/step-up";
import type { ApiError } from "@/types/admin";
import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
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

type AuthPanel = "verify" | "methods";

/**
 * Auth-only step-up enroll chrome (TOTP / passkey register).
 * Write pages do not mount this; they open StepUpVerifyDialog via requireStepUp.
 * Pass allowEnroll on Auth settings only.
 */
export function StepUpBar({
  token,
  allowEnroll = false,
}: {
  token: string;
  /** Full enroll UI. Keep false on list/write pages so passkey sprawl does not cover every page. */
  allowEnroll?: boolean;
}) {
  const access = useAccessToken(token);
  const qc = useQueryClient();
  const stepUpToken = useStepUpStore((s) => s.token);
  const stepUpExpiresAt = useStepUpStore((s) => s.expiresAt);
  const active =
    Boolean(useStepUpStore.getState().activeToken()) ||
    (Boolean(stepUpToken.trim()) && Date.now() < stepUpExpiresAt);
  const chrome = useAdminNavChrome(access);
  const totp = useAdminTOTPStatus(access);
  const webauthn = useAdminWebAuthnStatus(access);
  const stepUp = useStepUp(access);
  const begin = useTOTPBegin(access);
  const confirm = useTOTPConfirm(access);
  const waRegBegin = useWebAuthnRegisterBegin(access);
  const waRegFinish = useWebAuthnRegisterFinish(access);
  const waAssertBegin = useWebAuthnAssertBegin(access);
  const waAssertFinish = useWebAuthnAssertFinish(access);
  const [code, setCode] = useState("");
  const [codeError, setCodeError] = useState("");
  const [factorError, setFactorError] = useState("");
  const [secret, setSecret] = useState("");
  const [otpauth, setOtpauth] = useState("");
  const [passkeyName, setPasskeyName] = useState("");
  const [waBusy, setWaBusy] = useState(false);
  const [panel, setPanel] = useState<AuthPanel>("methods");

  const totpEnrolled = Boolean(totp.data?.enrolled);
  const webauthnEnrolled = Boolean(webauthn.data?.enrolled);
  const factorEnrolled = totpEnrolled || webauthnEnrolled;
  const verifyTitle = chrome.data?.ADMIN_TOTP_VERIFY_TITLE?.trim() || "";
  const enrollTitle = chrome.data?.ADMIN_TOTP_ENROLL_TITLE?.trim() || "";
  const enrollIntro = chrome.data?.ADMIN_TOTP_ENROLL_INTRO?.trim() || "";
  const totpStep1 = chrome.data?.ADMIN_TOTP_STEP_1?.trim() || "";
  const totpStep2 = chrome.data?.ADMIN_TOTP_STEP_2?.trim() || "";
  const totpStep3 = chrome.data?.ADMIN_TOTP_STEP_3?.trim() || "";
  const codeLabel = chrome.data?.ADMIN_TOTP_CODE_LABEL?.trim() || "";
  const codePlaceholder = chrome.data?.ADMIN_TOTP_CODE_PLACEHOLDER?.trim() || "";
  const secretLabel = chrome.data?.ADMIN_TOTP_SECRET_LABEL?.trim() || "";
  const secretHint = chrome.data?.ADMIN_TOTP_SECRET_HINT?.trim() || "";
  const qrCaption = chrome.data?.ADMIN_TOTP_QR_CAPTION?.trim() || "";
  const beginLabel = chrome.data?.ADMIN_TOTP_BEGIN?.trim() || "";
  const confirmLabel = chrome.data?.ADMIN_TOTP_CONFIRM?.trim() || "";
  const verifyLabel = chrome.data?.ADMIN_STEP_UP_VERIFY?.trim() || "";
  const pending = chrome.data?.ADMIN_PENDING_SAVING?.trim() || "";
  const requiredMsg = chrome.data?.ADMIN_STEP_UP_FACTOR_REQUIRED?.trim() || "";
  const enrollChoice = chrome.data?.ADMIN_STEP_UP_ENROLL_CHOICE?.trim() || "";
  const methodsTitle = chrome.data?.ADMIN_STEP_UP_METHODS_TITLE?.trim() || "";
  const methodsSubtitle = chrome.data?.ADMIN_STEP_UP_METHODS_SUBTITLE?.trim() || "";
  const methodEnrolled = chrome.data?.ADMIN_STEP_UP_METHOD_ENROLLED?.trim() || "";
  const showVerifyLabel = chrome.data?.ADMIN_STEP_UP_SHOW_VERIFY?.trim() || "";
  const openVerifyLabel = chrome.data?.ADMIN_STEP_UP_OPEN_VERIFY?.trim() || "";
  const activeLabel = chrome.data?.ADMIN_STEP_UP_ACTIVE?.trim() || "";
  const waEnrollTitle = chrome.data?.ADMIN_WEBAUTHN_ENROLL_TITLE?.trim() || "";
  const waEnrollIntro = chrome.data?.ADMIN_WEBAUTHN_ENROLL_INTRO?.trim() || "";
  const waStep1 = chrome.data?.ADMIN_WEBAUTHN_STEP_1?.trim() || "";
  const waStep2 = chrome.data?.ADMIN_WEBAUTHN_STEP_2?.trim() || "";
  const waVerifyTitle = chrome.data?.ADMIN_WEBAUTHN_VERIFY_TITLE?.trim() || "";
  const waBeginReg = chrome.data?.ADMIN_WEBAUTHN_BEGIN_REGISTER?.trim() || "";
  const waBeginAssert = chrome.data?.ADMIN_WEBAUTHN_BEGIN_ASSERT?.trim() || "";
  const waNameLabel = chrome.data?.ADMIN_WEBAUTHN_NAME_LABEL?.trim() || "";
  const waNamePlaceholder = chrome.data?.ADMIN_WEBAUTHN_NAME_PLACEHOLDER?.trim() || "";
  const waNameHint = chrome.data?.ADMIN_WEBAUTHN_NAME_HINT?.trim() || "";
  const waUnsupported = chrome.data?.ADMIN_WEBAUTHN_BROWSER_UNSUPPORTED?.trim() || "";
  const waRpMissing = chrome.data?.ADMIN_WEBAUTHN_RP_MISSING?.trim() || "";
  const verifyHint = chrome.data?.ADMIN_STEP_UP_VERIFY_HINT?.trim() || "";
  const stepUpSuccess = chrome.data?.ADMIN_STEP_UP_SUCCESS?.trim() || "";
  const totpEnrollSuccess = chrome.data?.ADMIN_TOTP_ENROLL_SUCCESS?.trim() || "";
  const waEnrollSuccess = chrome.data?.ADMIN_WEBAUTHN_ENROLL_SUCCESS?.trim() || "";
  const rpReady = Boolean(webauthn.data?.rp_configured);
  const browserOK = webauthnSupported();
  const totpSetupStarted = Boolean(secret);

  // Write pages: hide bar while step-up is already unlocked.
  // Auth page: keep methods visible; verify is opt-in and hidden while active.
  if (active && !allowEnroll) return null;

  const runRegisterPasskey = async () => {
    if (!browserOK || !rpReady || !pending) return;
    setWaBusy(true);
    setFactorError("");
    try {
      const options = await waRegBegin.mutateAsync();
      const credential = await createPasskey(options);
      await waRegFinish.mutateAsync({
        credential,
        friendlyName: passkeyName.trim() || undefined,
      });
      setPasskeyName("");
      // Elevate immediately after enroll so the next write does not re-prompt.
      try {
        const assertOpts = await waAssertBegin.mutateAsync();
        const assertion = await assertPasskey(assertOpts);
        await waAssertFinish.mutateAsync(assertion);
        toastSuccess(stepUpSuccess || waEnrollSuccess);
        setPanel("methods");
      } catch {
        toastSuccess(waEnrollSuccess);
      }
    } catch (err) {
      reportApiError(err, setFactorError);
    } finally {
      setWaBusy(false);
    }
  };

  const runAssertPasskey = async () => {
    if (!browserOK || !pending) return;
    setWaBusy(true);
    setFactorError("");
    try {
      const options = await waAssertBegin.mutateAsync();
      const credential = await assertPasskey(options);
      await waAssertFinish.mutateAsync(credential);
      toastSuccess(stepUpSuccess);
      setPanel("methods");
    } catch (err) {
      reportApiError(err, setFactorError);
    } finally {
      setWaBusy(false);
    }
  };

  const showTotpVerify = totpEnrolled && Boolean(codeLabel && verifyLabel && pending);
  const showPasskeyVerify = webauthnEnrolled && Boolean(waBeginAssert && pending && browserOK);
  const canVerify = showTotpVerify || showPasskeyVerify;
  // Auth home after enroll is methods (status + add other factor). Verify is opt-in.
  const hasMethodsNav = Boolean(openVerifyLabel && showVerifyLabel);
  const methodsVisible =
    allowEnroll &&
    (!factorEnrolled || active || panel === "methods" || !canVerify || !hasMethodsNav);
  const verifyVisible =
    factorEnrolled &&
    canVerify &&
    !active &&
    (!allowEnroll || (panel === "verify" && hasMethodsNav));

  const methodsBody = methodsVisible ? (
    <div className="space-y-5">
      {active && activeLabel ? (
        <p className="rounded-md border border-border/80 bg-background/40 px-3 py-2 text-sm text-foreground">
          {activeLabel}
        </p>
      ) : null}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="space-y-1.5">
          {factorEnrolled && methodsTitle ? (
            <p className="font-medium text-foreground">{methodsTitle}</p>
          ) : null}
          {!factorEnrolled && requiredMsg ? (
            <p className="font-medium text-foreground">{requiredMsg}</p>
          ) : null}
          {factorEnrolled && methodsSubtitle ? (
            <p className="text-muted-foreground">{methodsSubtitle}</p>
          ) : null}
          {!factorEnrolled && enrollChoice ? (
            <p className="text-muted-foreground">{enrollChoice}</p>
          ) : null}
        </div>
        {factorEnrolled && canVerify && !active && openVerifyLabel ? (
          <Button type="button" size="sm" variant="outline" onClick={() => setPanel("verify")}>
            {openVerifyLabel}
          </Button>
        ) : null}
      </div>

      {!totpEnrolled ? (
        <section className="space-y-3 rounded-md border border-border/80 bg-background/40 p-4">
          <div className="space-y-1">
            {enrollTitle ? (
              <h2 className="text-sm font-semibold text-foreground">{enrollTitle}</h2>
            ) : null}
            {enrollIntro ? <p className="text-muted-foreground">{enrollIntro}</p> : null}
          </div>

          <ol className="list-decimal space-y-3 pl-5 text-foreground marker:font-medium marker:text-muted-foreground">
            {totpStep1 ? (
              <li className="space-y-2 pl-1">
                <p>{totpStep1}</p>
                {beginLabel && pending ? (
                  <Button
                    type="button"
                    size="sm"
                    variant="secondary"
                    isLoading={begin.isPending}
                    pendingLabel={pending}
                    disabled={totpSetupStarted}
                    onClick={() =>
                      void begin.mutate(undefined, {
                        onSuccess: (data) => {
                          setSecret(data.secret || "");
                          setOtpauth(data.otpauth_url || "");
                        },
                        onError: (err) => {
                          if (apiErrorCode(err) === "ADMIN_TOTP_ALREADY_ENROLLED") {
                            void qc.invalidateQueries({ queryKey: adminQk.totp });
                            return;
                          }
                          reportApiError(err);
                        },
                      })
                    }
                  >
                    {beginLabel}
                  </Button>
                ) : null}
                {begin.isError && (begin.error as Error)?.message?.trim() ? (
                  <p className="text-sm text-destructive">
                    {(begin.error as Error).message.trim()}
                  </p>
                ) : null}
              </li>
            ) : beginLabel && pending ? (
              <li className="space-y-2 pl-1">
                <Button
                  type="button"
                  size="sm"
                  variant="secondary"
                  isLoading={begin.isPending}
                  pendingLabel={pending}
                  disabled={totpSetupStarted}
                  onClick={() =>
                    void begin.mutate(undefined, {
                      onSuccess: (data) => {
                        setSecret(data.secret || "");
                        setOtpauth(data.otpauth_url || "");
                      },
                      onError: (err) => {
                        if (apiErrorCode(err) === "ADMIN_TOTP_ALREADY_ENROLLED") {
                          void qc.invalidateQueries({ queryKey: adminQk.totp });
                          return;
                        }
                        reportApiError(err);
                      },
                    })
                  }
                >
                  {beginLabel}
                </Button>
              </li>
            ) : null}

            {totpSetupStarted && (totpStep2 || otpauth || secret) ? (
              <li className="space-y-3 pl-1">
                {totpStep2 ? <p>{totpStep2}</p> : null}
                <div className="flex flex-col gap-3 sm:flex-row sm:items-start">
                  {otpauth ? (
                    <div className="space-y-1.5">
                      <TotpQr otpauthUrl={otpauth} alt={qrCaption || secretLabel} />
                      {qrCaption ? (
                        <p className="max-w-[12rem] text-xs text-muted-foreground">{qrCaption}</p>
                      ) : null}
                    </div>
                  ) : null}
                  {secret && secretLabel ? (
                    <div className="min-w-0 flex-1 space-y-1.5">
                      <p className="text-xs font-medium text-foreground">{secretLabel}</p>
                      <p className="break-all rounded-md border border-border bg-muted/40 px-3 py-2 font-mono text-xs text-foreground">
                        {secret}
                      </p>
                      {secretHint ? (
                        <p className="text-xs text-muted-foreground">{secretHint}</p>
                      ) : null}
                    </div>
                  ) : null}
                </div>
              </li>
            ) : null}

            {totpSetupStarted && codeLabel ? (
              <li className="space-y-2 pl-1">
                {totpStep3 ? <p>{totpStep3}</p> : null}
                <div className="flex max-w-lg flex-wrap items-end gap-2">
                  <Field id="step-up-enroll-code" label={codeLabel} className="min-w-0 flex-1">
                    <Input
                      id="step-up-enroll-code"
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
                  {confirmLabel && pending ? (
                    <Button
                      type="button"
                      size="sm"
                      isLoading={confirm.isPending}
                      pendingLabel={pending}
                      disabled={code.trim().length !== 6}
                      onClick={() => {
                        const enrollCode = code.trim();
                        void confirm.mutate(enrollCode, {
                          onSuccess: () => {
                            // Server mints step_up_token on enroll confirm (same proof).
                            toastSuccess(stepUpSuccess || totpEnrollSuccess);
                            setCode("");
                            setCodeError("");
                            setSecret("");
                            setOtpauth("");
                            setPanel("methods");
                          },
                          onError: (err) => reportApiError(err, setCodeError),
                        });
                      }}
                    >
                      {confirmLabel}
                    </Button>
                  ) : null}
                </div>
                {codeError ? <p className="text-sm text-destructive">{codeError}</p> : null}
              </li>
            ) : null}
          </ol>
        </section>
      ) : (
        <section className="space-y-1 rounded-md border border-border/80 bg-background/40 p-4">
          {enrollTitle ? (
            <h2 className="text-sm font-semibold text-foreground">{enrollTitle}</h2>
          ) : null}
          {methodEnrolled ? (
            <p className="text-sm text-muted-foreground">{methodEnrolled}</p>
          ) : null}
        </section>
      )}

      {!webauthnEnrolled ? (
        <section className="space-y-3 rounded-md border border-border/80 bg-background/40 p-4">
          <div className="space-y-1">
            {waEnrollTitle ? (
              <h2 className="text-sm font-semibold text-foreground">{waEnrollTitle}</h2>
            ) : null}
            {waEnrollIntro ? <p className="text-muted-foreground">{waEnrollIntro}</p> : null}
          </div>

          {!browserOK && waUnsupported ? (
            <p className="text-muted-foreground">{waUnsupported}</p>
          ) : null}
          {browserOK && !rpReady && waRpMissing ? (
            <p className="text-muted-foreground">{waRpMissing}</p>
          ) : null}

          {browserOK && rpReady ? (
            <ol className="list-decimal space-y-3 pl-5 text-foreground marker:font-medium marker:text-muted-foreground">
              {waStep1 && waNameLabel ? (
                <li className="space-y-2 pl-1">
                  <p>{waStep1}</p>
                  <Field
                    id="step-up-passkey-name"
                    label={waNameLabel}
                    description={waNameHint || undefined}
                    className="max-w-md"
                  >
                    <Input
                      id="step-up-passkey-name"
                      value={passkeyName}
                      onChange={(e) => setPasskeyName(e.target.value)}
                      placeholder={waNamePlaceholder || waNameLabel}
                      autoComplete="off"
                    />
                  </Field>
                </li>
              ) : null}
              {waStep2 && waBeginReg && pending ? (
                <li className="space-y-2 pl-1">
                  <p>{waStep2}</p>
                  <Button
                    type="button"
                    size="sm"
                    variant="secondary"
                    isLoading={waBusy || waRegBegin.isPending || waRegFinish.isPending}
                    pendingLabel={pending}
                    onClick={() => void runRegisterPasskey()}
                  >
                    {waBeginReg}
                  </Button>
                  {factorError ? <p className="text-sm text-destructive">{factorError}</p> : null}
                </li>
              ) : null}
            </ol>
          ) : null}
        </section>
      ) : (
        <section className="space-y-1 rounded-md border border-border/80 bg-background/40 p-4">
          {waEnrollTitle ? (
            <h2 className="text-sm font-semibold text-foreground">{waEnrollTitle}</h2>
          ) : null}
          {methodEnrolled ? (
            <p className="text-sm text-muted-foreground">{methodEnrolled}</p>
          ) : null}
        </section>
      )}
    </div>
  ) : null;

  const verifyBody = verifyVisible ? (
    <div className="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-end">
      <div className="flex w-full flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
        <div className="space-y-1">
          {verifyTitle || waVerifyTitle ? (
            <p className="text-sm font-medium text-foreground">{verifyTitle || waVerifyTitle}</p>
          ) : null}
          {verifyHint ? <p className="text-muted-foreground">{verifyHint}</p> : null}
        </div>
        {allowEnroll && showVerifyLabel ? (
          <Button type="button" size="sm" variant="outline" onClick={() => setPanel("methods")}>
            {showVerifyLabel}
          </Button>
        ) : null}
      </div>
      {showTotpVerify ? (
        <div className="flex w-full min-w-0 flex-col gap-2 sm:max-w-xs sm:flex-1">
          <div className="flex flex-wrap items-end gap-2">
            <Field id="step-up-verify-code" label={codeLabel} className="min-w-[10rem] flex-1">
              <Input
                id="step-up-verify-code"
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
            <Button
              type="button"
              size="sm"
              variant="secondary"
              isLoading={stepUp.isPending}
              pendingLabel={pending}
              disabled={code.trim().length !== 6}
              onClick={() =>
                void stepUp.mutate(code.trim(), {
                  onSuccess: () => {
                    setCode("");
                    setCodeError("");
                    setPanel("methods");
                    toastSuccess(stepUpSuccess);
                  },
                  onError: (err) => reportApiError(err, setCodeError),
                })
              }
            >
              {verifyLabel}
            </Button>
          </div>
          {codeError ? <p className="text-sm text-destructive">{codeError}</p> : null}
        </div>
      ) : null}
      {showPasskeyVerify ? (
        <div className="flex flex-col gap-2">
          <Button
            type="button"
            size="sm"
            variant="outline"
            isLoading={waBusy || waAssertBegin.isPending || waAssertFinish.isPending}
            pendingLabel={pending}
            onClick={() => void runAssertPasskey()}
          >
            {waBeginAssert}
          </Button>
          {factorError ? <p className="text-sm text-destructive">{factorError}</p> : null}
        </div>
      ) : null}
    </div>
  ) : null;

  if (!methodsVisible && !verifyVisible) return null;

  return (
    <div className="rounded-md border border-border bg-card px-4 py-4 text-sm">
      {verifyVisible ? verifyBody : null}
      {methodsVisible ? methodsBody : null}
    </div>
  );
}
