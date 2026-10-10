"use client";

import { CliAuthPageSkeleton } from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { useAuthProviders } from "@/hooks/queries/auth";
import { qk } from "@/lib/query-keys";
import { createClient } from "@/lib/supabase/client";
import { toastApiError, toastApiSuccess } from "@/lib/toast-api";
import { useQueryClient } from "@tanstack/react-query";
import { useSearchParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

function apiBase(): string {
  const base = process.env.NEXT_PUBLIC_API_URL;
  if (!base) {
    const missing = process.env.NEXT_PUBLIC_API_URL_MISSING?.trim();
    throw new Error(missing || "");
  }
  return base.replace(/\/$/, "");
}

type KeyLabels = {
  issue_action_label: string;
  issue_pending_label: string;
  issue_retry_action_label: string;
  issue_another_action_label: string;
  copy_action_label: string;
  copy_pending_label: string;
  copied_action_label: string;
};

export default function CLIAuthPage() {
  const params = useSearchParams();
  const hardware = params.get("hardware") ?? "";
  const authProviders = useAuthProviders();
  const qc = useQueryClient();
  const [apiKey, setApiKey] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [issuing, setIssuing] = useState(false);
  const [copied, setCopied] = useState(false);
  const [copying, setCopying] = useState(false);
  const [keyLabels, setKeyLabels] = useState<KeyLabels | null>(null);

  const cliLabels = authProviders.data?.cli ?? null;
  const labels: KeyLabels | null = keyLabels ?? cliLabels ?? null;

  const issueKey = useCallback(async () => {
    setLoading(true);
    setIssuing(true);
    setError(null);
    try {
      const supabase = createClient();
      const { data } = await supabase.auth.getSession();
      const session = data.session;
      if (!session?.access_token) {
        const appUrl = process.env.NEXT_PUBLIC_APP_URL;
        if (!appUrl) {
          throw new Error(authProviders.data?.login_env_app_url_missing || "");
        }
        const loginHref = (authProviders.data?.site?.nav_sign_in_href || "").trim();
        const cliAuthPath = (authProviders.data?.site?.path_cli_auth || "").trim();
        if (!loginHref || !cliAuthPath) {
          throw new Error(
            authProviders.data?.login_default_next_missing ||
              authProviders.data?.login_failed_message ||
              "",
          );
        }
        const next = encodeURIComponent(
          `${cliAuthPath}${hardware ? `?hardware=${encodeURIComponent(hardware)}` : ""}`,
        );
        window.location.href = `${appUrl}${loginHref}?next=${next}`;
        return;
      }

      const res = await fetch(`${apiBase()}/api/v1/me/api-keys`, {
        method: "POST",
        headers: {
          Accept: "application/json",
          "Content-Type": "application/json",
          Authorization: `Bearer ${session.access_token}`,
        },
        body: JSON.stringify({
          hardware_uuid: hardware,
          // agent_id omitted: key usable from CLI and IDE; X-Trim-Agent-Id still required per request
        }),
        signal: AbortSignal.timeout(60_000),
      });
      if (!res.ok) {
        let apiMessage = "";
        try {
          const parsed = (await res.json()) as {
            error?: string;
            message?: string;
          };
          apiMessage =
            (typeof parsed.error === "string" && parsed.error.trim()) ||
            (typeof parsed.message === "string" && parsed.message.trim()) ||
            "";
        } catch {
          apiMessage = "";
        }
        throw new Error(apiMessage);
      }
      const body = (await res.json()) as {
        api_key: string;
        message?: string;
        action_label: string;
        pending_label: string;
        retry_action_label: string;
        issue_another_action_label: string;
        copy_action_label: string;
        copy_pending_label: string;
        copied_action_label: string;
      };
      setApiKey(body.api_key);
      setKeyLabels({
        issue_action_label: body.action_label,
        issue_pending_label: body.pending_label,
        issue_retry_action_label: body.retry_action_label,
        issue_another_action_label: body.issue_another_action_label,
        copy_action_label: body.copy_action_label,
        copy_pending_label: body.copy_pending_label,
        copied_action_label: body.copied_action_label,
      });
      toastApiSuccess(body);
      await qc.invalidateQueries({ queryKey: qk.apiKeys });
    } catch (e) {
      const msg =
        e instanceof Error ? e.message : authProviders.data?.cli?.issue_failed_message || "";
      setError(msg);
      toastApiError(new Error(msg));
    } finally {
      setLoading(false);
      setIssuing(false);
    }
  }, [
    hardware,
    authProviders.data?.cli?.issue_failed_message,
    authProviders.data?.login_env_app_url_missing,
    authProviders.data?.login_default_next_missing,
    authProviders.data?.login_failed_message,
    authProviders.data?.site?.nav_sign_in_href,
    authProviders.data?.site?.path_cli_auth,
    qc,
  ]);

  useEffect(() => {
    void issueKey();
  }, [issueKey]);

  async function copyKey() {
    if (!apiKey) return;
    setCopying(true);
    try {
      await navigator.clipboard.writeText(apiKey);
      setCopied(true);
      const resetRaw = authProviders.data?.cli?.copied_reset_ms?.trim() || "";
      const resetMs = Number.parseInt(resetRaw, 10);
      if (Number.isFinite(resetMs) && resetMs > 0) {
        window.setTimeout(() => setCopied(false), resetMs);
      } else {
        setCopied(false);
      }
    } finally {
      setCopying(false);
    }
  }

  if ((loading && !error) || authProviders.isLoading) {
    return (
      <CliAuthPageSkeleton
        chrome={{
          eyebrow: authProviders.data?.cli?.eyebrow,
          title: authProviders.data?.cli?.title,
          body: authProviders.data?.cli?.body,
          copy_label: authProviders.data?.cli?.copy_action_label,
          issue_label: authProviders.data?.cli?.issue_another_action_label,
          footer: authProviders.data?.cli?.footer,
        }}
      />
    );
  }

  const page = authProviders.data?.cli;

  return (
    <main className="flex min-h-screen items-center justify-center bg-[var(--trim-bg)] px-6">
      <div className="w-full max-w-lg rounded-lg border border-[var(--trim-border)] bg-[var(--trim-panel)] p-8">
        <p className="text-xs font-medium uppercase tracking-wider text-[var(--trim-muted)]">
          {page?.eyebrow || ""}
        </p>
        <h1 className="mt-2 text-2xl font-semibold text-[var(--trim-fg)]">{page?.title || ""}</h1>
        <p className="mt-2 text-sm text-[var(--trim-muted)]">{page?.body || ""}</p>

        {error ? (
          <div className="mt-6 space-y-3">
            <p className="text-sm text-destructive">{error}</p>
            <Button
              variant="outline"
              disabled={!labels?.issue_retry_action_label}
              isLoading={issuing}
              pendingLabel={labels?.issue_pending_label || undefined}
              onClick={() => void issueKey()}
            >
              {labels?.issue_retry_action_label}
            </Button>
          </div>
        ) : null}

        {!error && apiKey ? (
          <div className="mt-6 space-y-4">
            <div className="break-all rounded-md border border-[var(--trim-border)] bg-[var(--trim-code)] px-4 py-3 font-mono text-sm text-[var(--trim-fg)]">
              {apiKey}
            </div>
            <div className="flex gap-2">
              <Button
                disabled={!labels?.copy_action_label}
                isLoading={copying}
                pendingLabel={labels?.copy_pending_label || undefined}
                onClick={() => void copyKey()}
              >
                {copied ? labels?.copied_action_label : labels?.copy_action_label}
              </Button>
              <Button
                variant="outline"
                disabled={!labels?.issue_another_action_label}
                isLoading={issuing}
                pendingLabel={labels?.issue_pending_label || undefined}
                onClick={() => void issueKey()}
              >
                {labels?.issue_another_action_label}
              </Button>
            </div>
            <p className="text-xs text-[var(--trim-muted)]">{page?.footer || ""}</p>
          </div>
        ) : null}
      </div>
    </main>
  );
}
