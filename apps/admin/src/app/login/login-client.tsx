"use client";

import { OAuthProviderIcon } from "@/components/auth/oauth-provider-icon";
import { AuthPageSkeleton } from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { useAuthProvidersPublic } from "@/hooks/queries/auth";
import { authProviderSlotCount } from "@/lib/auth-providers";
import { buildOAuthProviderOptions } from "@/lib/oauth-provider-options";
import { createClient } from "@/lib/supabase/client";
import type { AuthProvidersPayload } from "@/types/admin";
import type { Provider } from "@supabase/supabase-js";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";

type LoginProviderItem = {
  id: string;
  display_name: string;
  action_label: string;
  pending_label: string;
};

function resolveLoginError(errorParam: string | null, chrome: AuthProvidersPayload): string {
  if (!errorParam) return "";
  if (errorParam === "provider_not_allowed" || errorParam === "provider_missing") {
    return chrome.login_provider_disabled_message || "";
  }
  if (errorParam === "missing_supabase_env") {
    return chrome.login_missing_supabase_env || "";
  }
  if (errorParam === "oauth_exchange_failed") {
    return chrome.login_oauth_exchange_failed || "";
  }
  if (errorParam === "login_default_next_missing") {
    return chrome.admin_login_default_next_missing || chrome.login_failed_message || "";
  }
  if (errorParam === "oauth_email_missing") {
    return chrome.login_oauth_email_missing || chrome.login_failed_message || "";
  }
  if (errorParam === "oauth_email_denied") {
    return chrome.login_oauth_email_denied || chrome.login_failed_message || "";
  }
  if (errorParam === "auth_providers_unavailable") {
    return (
      chrome.login_auth_providers_unavailable ||
      chrome.providers_empty_message ||
      chrome.login_failed_message ||
      ""
    );
  }
  if (errorParam === "admin_forbidden") {
    return chrome.admin_forbidden_page || chrome.login_failed_message || "";
  }
  if (errorParam === "admin_credentials") {
    return chrome.admin_credentials_revoked || chrome.login_failed_message || "";
  }
  if (errorParam === "admin_bootstrap") {
    return (
      chrome.admin_bootstrap_required ||
      chrome.login_auth_providers_unavailable ||
      chrome.login_failed_message ||
      ""
    );
  }
  if (errorParam === "session_rejected") {
    return chrome.login_oauth_exchange_failed || chrome.login_failed_message || "";
  }
  return chrome.login_failed_message || "";
}

export default function LoginPage() {
  const params = useSearchParams();
  const router = useRouter();
  const nextFromQuery = (params.get("next") || "").trim();
  const errorParam = params.get("error");
  const [pendingProvider, setPendingProvider] = useState<string | null>(null);
  const providersQuery = useAuthProvidersPublic();
  const toastedRef = useRef<string | null>(null);

  // Never invent /. Query wins; else backend ADMIN_LOGIN_DEFAULT_NEXT only.
  const nextPath = nextFromQuery || (providersQuery.data?.admin_login_default_next || "").trim();

  const providerItems = useMemo(() => {
    const items = providersQuery.data?.items ?? [];
    return items.filter(
      (item): item is LoginProviderItem =>
        Boolean(item?.id?.trim()) &&
        Boolean(item?.display_name?.trim()) &&
        Boolean(item?.action_label?.trim()) &&
        Boolean(item?.pending_label?.trim()),
    );
  }, [providersQuery.data?.items]);

  // Admin title only; fail closed if unset (no web login_title invent).
  const loginTitle = (providersQuery.data?.admin_login_title || "").trim();

  // Surface auth failures as toasts; keep social buttons visible (no dead-end page).
  useEffect(() => {
    if (!providersQuery.data || !errorParam) return;
    const msg = resolveLoginError(errorParam, providersQuery.data).trim();
    if (!msg) return;
    if (toastedRef.current === errorParam) return;
    toastedRef.current = errorParam;
    toast.error(msg);
    // Drop ?error= so refresh does not re-toast; keep social buttons.
    const next = new URLSearchParams(params.toString());
    next.delete("error");
    const q = next.toString();
    const loginPath = (providersQuery.data?.site?.path_login || "").trim() || "/login";
    router.replace(q ? `${loginPath}?${q}` : loginPath);
  }, [providersQuery.data, errorParam, params, router]);

  useEffect(() => {
    if (!providersQuery.isError) return;
    const msg = ((providersQuery.error as Error)?.message || "").trim();
    if (!msg) return;
    toast.error(msg);
  }, [providersQuery.isError, providersQuery.error]);

  async function signIn(providerId: string) {
    setPendingProvider(providerId);
    try {
      if (!nextPath) {
        throw new Error(
          providersQuery.data?.admin_login_default_next_missing ||
            providersQuery.data?.login_failed_message ||
            "",
        );
      }
      const supabase = createClient();
      const origin = window.location.origin;
      const callbackPath = (providersQuery.data?.site?.path_auth_callback || "").trim();
      if (
        !callbackPath ||
        !callbackPath.startsWith("/") ||
        callbackPath.startsWith("//") ||
        nextPath === callbackPath
      ) {
        throw new Error(
          providersQuery.data?.admin_login_default_next_missing ||
            providersQuery.data?.login_failed_message ||
            "",
        );
      }
      const options = buildOAuthProviderOptions(
        providerId,
        providersQuery.data as AuthProvidersPayload | undefined,
        `${origin}${callbackPath}?next=${encodeURIComponent(nextPath)}`,
        providersQuery.data?.login_failed_message || "",
      );
      const { data, error: oauthError } = await supabase.auth.signInWithOAuth({
        provider: providerId as Provider,
        options: { ...options, skipBrowserRedirect: true },
      });
      if (oauthError) {
        throw new Error(providersQuery.data?.login_failed_message || "");
      }
      const url = (data?.url || "").trim();
      if (!url) {
        throw new Error(providersQuery.data?.login_failed_message || "");
      }
      // Explicit navigation - never leave the button stuck on pending.
      window.location.assign(url);
    } catch (e) {
      const msg = e instanceof Error ? e.message : providersQuery.data?.login_failed_message || "";
      if (msg) toast.error(msg);
      setPendingProvider(null);
    }
  }

  const providerSlotHint = providersQuery.data?.items?.length;

  if (providersQuery.isLoading) {
    return <AuthPageSkeleton title="" providerSlots={authProviderSlotCount(providerSlotHint)} />;
  }

  return (
    <main className="mx-auto flex min-h-[70vh] w-full max-w-md flex-col justify-center gap-6 px-4 py-16">
      <div className="space-y-2 text-center">
        <h1 className="text-2xl font-semibold tracking-tight text-foreground">{loginTitle}</h1>
      </div>
      {providerItems.length === 0 ? (
        <p className="text-center text-sm text-muted-foreground">
          {providersQuery.data?.providers_empty_message || ""}
        </p>
      ) : null}
      <div className="flex flex-col gap-3">
        {providerItems.map((item) => (
          <Button
            key={item.id}
            type="button"
            variant="secondary"
            className="w-full"
            isLoading={pendingProvider === item.id}
            pendingLabel={item.pending_label}
            disabled={pendingProvider !== null}
            onClick={() => void signIn(item.id)}
          >
            <OAuthProviderIcon provider={item.id} />
            {item.action_label}
          </Button>
        ))}
      </div>
    </main>
  );
}
