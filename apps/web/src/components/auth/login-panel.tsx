"use client";

import { OAuthProviderIcon } from "@/components/auth/oauth-provider-icon";
import { AuthPageSkeleton } from "@/components/skeletons/page-skeletons";
import { Button } from "@/components/ui/button";
import { useAuthProviders } from "@/hooks/queries/auth";
import { allowedAuthProvidersFromEnv } from "@/lib/auth-providers";
import { buildOAuthProviderOptions } from "@/lib/oauth-provider-options";
import { createClient } from "@/lib/supabase/client";
import { cn } from "@/lib/utils";
import type { AuthProvidersSiteChrome } from "@/types/auth";
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

type Chrome = NonNullable<ReturnType<typeof useAuthProviders>["data"]>;

function resolveLoginError(errorParam: string | null, chrome: Chrome): string {
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
    return chrome.login_default_next_missing || chrome.login_failed_message || "";
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
  if (errorParam === "session_rejected") {
    return chrome.login_oauth_exchange_failed || chrome.login_failed_message || "";
  }
  return chrome.login_failed_message || "";
}

/** Sign-in panel - use embedded inside marketing shell right rail. */
export function LoginPanel({ embedded = false }: { embedded?: boolean }) {
  const params = useSearchParams();
  const router = useRouter();
  const nextFromQuery = (params.get("next") || "").trim();
  const errorParam = params.get("error");
  const [pendingProvider, setPendingProvider] = useState<string | null>(null);
  const providersQuery = useAuthProviders();
  const toastedRef = useRef<string | null>(null);

  const nextPath = nextFromQuery || (providersQuery.data?.login_default_next || "").trim();

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

  const loginTitle = providersQuery.data?.login_title || "";
  const site = providersQuery.data?.site as AuthProvidersSiteChrome | undefined;

  useEffect(() => {
    if (!providersQuery.data || !errorParam) return;
    const msg = resolveLoginError(errorParam, providersQuery.data).trim();
    if (!msg) return;
    if (toastedRef.current === errorParam) return;
    toastedRef.current = errorParam;
    toast.error(msg);
    const next = new URLSearchParams(params.toString());
    next.delete("error");
    const q = next.toString();
    const loginPath = (site?.path_login || site?.nav_sign_in_href || "/login").trim();
    router.replace(q ? `${loginPath}?${q}` : loginPath);
  }, [providersQuery.data, errorParam, params, router, site?.path_login, site?.nav_sign_in_href]);

  async function signIn(provider: string) {
    setPendingProvider(provider);
    try {
      if (!nextPath) {
        throw new Error(
          providersQuery.data?.login_default_next_missing ||
            providersQuery.data?.login_failed_message ||
            "",
        );
      }
      const supabase = createClient();
      // Same-origin redirectTo keeps the PKCE code_verifier cookie on this host/port.
      const origin = window.location.origin;
      const callbackPath = (providersQuery.data?.site?.path_auth_callback || "").trim();
      if (
        !callbackPath ||
        !callbackPath.startsWith("/") ||
        callbackPath.startsWith("//") ||
        nextPath === callbackPath
      ) {
        throw new Error(
          providersQuery.data?.login_default_next_missing ||
            providersQuery.data?.login_failed_message ||
            "",
        );
      }
      const options = buildOAuthProviderOptions(
        provider,
        providersQuery.data,
        `${origin}${callbackPath}?next=${encodeURIComponent(nextPath)}`,
        providersQuery.data?.login_failed_message || "",
      );
      const { data, error: authError } = await supabase.auth.signInWithOAuth({
        provider: provider as Provider,
        options: { ...options, skipBrowserRedirect: true },
      });
      if (authError) {
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

  if (providersQuery.isPending && !providersQuery.data) {
    return (
      <div className="flex min-h-[calc(100vh-4rem)] w-full items-center justify-center px-5 py-16 sm:px-8">
        <AuthPageSkeleton title={undefined} providerSlots={allowedAuthProvidersFromEnv().length} />
      </div>
    );
  }

  const card = (
    <>
      <div className="space-y-2 text-center">
        <h1
          className={cn(
            "font-semibold tracking-tight text-[var(--trim-fg)]",
            embedded ? "font-display text-xl lg:text-2xl" : "text-2xl",
          )}
        >
          {loginTitle}
        </h1>
      </div>
      {providerItems.length === 0 ? (
        <p className="text-center text-sm text-[var(--trim-muted)]">
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
    </>
  );

  if (embedded) {
    return (
      <div className="flex min-h-[calc(100vh-4rem)] w-full items-center justify-center px-5 py-16 sm:px-8">
        <div className="flex w-full max-w-sm flex-col gap-6">{card}</div>
      </div>
    );
  }

  return (
    <main className="mx-auto flex min-h-[70vh] w-full max-w-md flex-col justify-center gap-6 px-4 py-16">
      {card}
    </main>
  );
}
