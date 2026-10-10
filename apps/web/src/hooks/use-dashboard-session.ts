"use client";

import { createContext, createElement, useContext, type ReactNode } from "react";

export type DashboardSession = {
  accessToken: string;
  userId?: string;
  email?: string;
  fullName?: string;
  avatarUrl?: string;
};

const DashboardSessionContext = createContext<DashboardSession | null>(null);

export function DashboardSessionProvider({
  session,
  children,
}: {
  /** Null while DashboardGate settles or after sign-out bounce. */
  session: DashboardSession | null;
  children: ReactNode;
}) {
  return createElement(DashboardSessionContext.Provider, { value: session }, children);
}

export function useDashboardSession(): DashboardSession {
  const ctx = useContext(DashboardSessionContext);
  if (!ctx) {
    throw new Error("useDashboardSession requires a settled DashboardGate session");
  }
  return ctx;
}

/** Optional read while the gate is still settling (shell stays mounted). */
export function useDashboardSessionOptional(): DashboardSession | null {
  return useContext(DashboardSessionContext);
}
