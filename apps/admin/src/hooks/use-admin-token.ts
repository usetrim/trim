"use client";

import { useAccessToken } from "@/hooks/use-access-token";
import { createContext, createElement, useContext, type ReactNode } from "react";

const AdminTokenContext = createContext<string | null>(null);

/** Provides the gate-verified access token to console pages (avoids per-page SSR token fetch). */
export function AdminTokenProvider({
  token,
  children,
}: {
  token: string;
  children: ReactNode;
}) {
  return createElement(AdminTokenContext.Provider, { value: token }, children);
}

/**
 * Prefer shell token from AdminGate. When inside the shell, do not open a second
 * Supabase auth subscription on every page remount (that flickered list queries
 * and felt like a full refresh while typing / soft-navigating).
 */
export function useAdminToken(initialToken?: string): string {
  const fromShell = useContext(AdminTokenContext);
  const inShell = fromShell !== null;
  const browser = useAccessToken(inShell ? undefined : initialToken, !inShell);
  if (inShell) return fromShell.trim();
  return browser;
}
