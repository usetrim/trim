"use client";

import { adminFetch } from "@/lib/admin-api/client";
import { useAdminNavChrome } from "@/hooks/queries/chrome";
import { useAdminToken } from "@/hooks/use-admin-token";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo, useState } from "react";

type AdapterRow = {
  id: string;
  enabled: boolean;
  sort_order: number;
  dialect: string;
  door_label: string;
  match_model_prefixes: string[];
  model_aliases: Record<string, string>;
  upstream_kind: string;
  upstream_path: string;
  upstream_base_url?: string;
  auth_mode: string;
  anthropic_version?: string;
  anthropic_workspace_id?: string;
  default_max_tokens?: number;
  require_alias: boolean;
};

type AliasRow = {
  client_model: string;
  upstream_model: string;
  upstream_host_contains: string;
  enabled: boolean;
};

export function ProviderAdaptersClient() {
  const token = useAdminToken();
  const chrome = useAdminNavChrome(token);
  const qc = useQueryClient();
  const title =
    chrome.data?.ADMIN_PROVIDER_ADAPTERS_TITLE?.trim() ||
    chrome.data?.ADMIN_NAV_SETTINGS_BILLING?.trim() ||
    "Provider adapters";
  const desc = chrome.data?.ADMIN_PROVIDER_ADAPTERS_DESC?.trim() || "";

  const adapters = useQuery({
    queryKey: ["admin", "provider-adapters"],
    enabled: Boolean(token),
    queryFn: () =>
      adminFetch<{ items: AdapterRow[]; page_title?: string; description?: string }>(
        "/api/v1/admin/billing/provider-adapters",
        { token },
      ),
  });
  const aliases = useQuery({
    queryKey: ["admin", "openai-model-aliases"],
    enabled: Boolean(token),
    queryFn: () =>
      adminFetch<{ items: AliasRow[]; page_title?: string; description?: string }>(
        "/api/v1/admin/billing/openai-model-aliases",
        { token },
      ),
  });

  const [adapterJSON, setAdapterJSON] = useState("");
  const [aliasJSON, setAliasJSON] = useState("");
  const selectedAdapter = useMemo(() => adapters.data?.items?.[0], [adapters.data]);

  const saveAdapter = useMutation({
    mutationFn: async () => {
      const body = JSON.parse(adapterJSON || "{}") as AdapterRow;
      return adminFetch("/api/v1/admin/billing/provider-adapters", {
        token,
        method: "PUT",
        body: JSON.stringify(body),
      });
    },
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: ["admin", "provider-adapters"] });
    },
  });
  const saveAlias = useMutation({
    mutationFn: async () => {
      const body = JSON.parse(aliasJSON || "{}") as AliasRow;
      return adminFetch("/api/v1/admin/billing/openai-model-aliases", {
        token,
        method: "PUT",
        body: JSON.stringify(body),
      });
    },
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: ["admin", "openai-model-aliases"] });
    },
  });

  const heading = adapters.data?.page_title?.trim() || title;
  const description = adapters.data?.description?.trim() || desc;

  return (
    <div className="space-y-8">
      <div className="space-y-1">
        {heading ? <h1 className="text-xl font-semibold tracking-tight">{heading}</h1> : null}
        {description ? <p className="text-muted-foreground text-sm">{description}</p> : null}
      </div>

      <section className="space-y-3 rounded-lg border p-4">
        <h2 className="text-sm font-semibold">provider_adapters</h2>
        <p className="text-muted-foreground text-sm">
          PUT a full adapter JSON row (id, enabled, sort_order, dialect, door_label,
          match_model_prefixes, model_aliases, upstream_kind, upstream_path, upstream_base_url,
          auth_mode, anthropic_version, default_max_tokens, require_alias). dialect openai_compat
          routes same-shape hosts by model prefix via upstream_base_url (no daily
          UPSTREAM_OPENAI_URL flips). Synced to CLI via preferences. Org-scoped Anthropic keys use
          client anthropic-workspace-id or TRIM_ANTHROPIC_WORKSPACE_ID (not Admin defaults).
        </p>
        <ul className="text-sm list-disc pl-5 space-y-1">
          {(adapters.data?.items || []).map((a) => (
            <li key={a.id}>
              <button
                type="button"
                className="underline"
                onClick={() => setAdapterJSON(JSON.stringify(a, null, 2))}
              >
                {a.id}
              </button>{" "}
              - {a.enabled ? "enabled" : "disabled"} - {a.dialect} - door={a.door_label}
            </li>
          ))}
        </ul>
        {!adapterJSON && selectedAdapter ? (
          <button
            type="button"
            className="text-sm underline"
            onClick={() => setAdapterJSON(JSON.stringify(selectedAdapter, null, 2))}
          >
            Load first adapter into editor
          </button>
        ) : null}
        <textarea
          className="min-h-64 w-full rounded border bg-background p-2 font-mono text-xs"
          value={adapterJSON}
          onChange={(e) => setAdapterJSON(e.target.value)}
          placeholder='{"id":"anthropic",...}'
        />
        <button
          type="button"
          className="rounded bg-primary px-3 py-1.5 text-sm text-primary-foreground disabled:opacity-50"
          disabled={!adapterJSON || saveAdapter.isPending}
          onClick={() => saveAdapter.mutate()}
        >
          Save adapter
        </button>
        {saveAdapter.isError ? (
          <p className="text-destructive text-sm">{(saveAdapter.error as Error).message}</p>
        ) : null}
      </section>

      <section className="space-y-3 rounded-lg border p-4">
        <h2 className="text-sm font-semibold">
          {aliases.data?.page_title || "openai_model_aliases"}
        </h2>
        <p className="text-muted-foreground text-sm">
          {aliases.data?.description ||
            "Exact client_model → upstream_model rewrites before routing (host-scoped when upstream_host_contains is set)."}
        </p>
        <ul className="text-sm list-disc pl-5 space-y-1">
          {(aliases.data?.items || []).map((a) => (
            <li key={a.client_model}>
              <button
                type="button"
                className="underline"
                onClick={() => setAliasJSON(JSON.stringify(a, null, 2))}
              >
                {a.client_model}
              </button>{" "}
              → {a.upstream_model}
              {a.upstream_host_contains ? ` @ ${a.upstream_host_contains}` : ""}
            </li>
          ))}
        </ul>
        <textarea
          className="min-h-40 w-full rounded border bg-background p-2 font-mono text-xs"
          value={aliasJSON}
          onChange={(e) => setAliasJSON(e.target.value)}
          placeholder='{"client_model":"trim-gemini-flash","upstream_model":"gemini-flash-latest","upstream_host_contains":"generativelanguage.googleapis.com","enabled":true}'
        />
        <button
          type="button"
          className="rounded bg-primary px-3 py-1.5 text-sm text-primary-foreground disabled:opacity-50"
          disabled={!aliasJSON || saveAlias.isPending}
          onClick={() => saveAlias.mutate()}
        >
          Save alias
        </button>
        {saveAlias.isError ? (
          <p className="text-destructive text-sm">{(saveAlias.error as Error).message}</p>
        ) : null}
      </section>
    </div>
  );
}
