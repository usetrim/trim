"use client";

import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

const STORAGE_KEY = "trim.admin.step_up";
/** Matches server max TRIM_ADMIN_STEP_UP_TTL_SEC (12h operator session). */
const FALLBACK_TTL_SEC = 43200;

type PersistedSlice = {
  token: string;
  expiresAt: number;
};

type WindowStepUp = {
  token: string;
  expiresAt: number;
};

declare global {
  interface Window {
    /** Survives duplicate Zustand module instances under Next HMR. */
    __trimStepUp?: WindowStepUp;
  }
}

function writeEverywhere(token: string, expiresAt: number) {
  if (typeof window === "undefined") return;
  window.__trimStepUp = { token, expiresAt };
  try {
    sessionStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ state: { token, expiresAt }, version: 0 }),
    );
  } catch {
    /* private mode */
  }
}

function clearEverywhere() {
  if (typeof window === "undefined") return;
  window.__trimStepUp = undefined;
  try {
    sessionStorage.removeItem(STORAGE_KEY);
  } catch {
    /* private mode */
  }
}

function readSessionSync(): PersistedSlice | null {
  if (typeof window === "undefined") return null;
  const win = window.__trimStepUp;
  if (
    win &&
    typeof win.token === "string" &&
    win.token.trim() &&
    typeof win.expiresAt === "number" &&
    Date.now() < win.expiresAt
  ) {
    return { token: win.token.trim(), expiresAt: win.expiresAt };
  }
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as { state?: PersistedSlice } & PersistedSlice;
    const slice = parsed.state ?? parsed;
    if (
      typeof slice?.token !== "string" ||
      !slice.token.trim() ||
      typeof slice?.expiresAt !== "number" ||
      !Number.isFinite(slice.expiresAt) ||
      Date.now() >= slice.expiresAt
    ) {
      return null;
    }
    window.__trimStepUp = { token: slice.token.trim(), expiresAt: slice.expiresAt };
    return { token: slice.token.trim(), expiresAt: slice.expiresAt };
  } catch {
    return null;
  }
}

type StepUpState = {
  token: string;
  expiresAt: number;
  verifyOpen: boolean;
  setToken: (token: string, expiresInSec: number) => void;
  clear: () => void;
  isActive: () => boolean;
  activeToken: () => string;
  openVerify: () => void;
  closeVerify: () => void;
};

/**
 * Step-up elevation after one successful TOTP/passkey verify.
 * Writes window + sessionStorage synchronously so HMR/duplicate modules
 * and remounts still see the same elevation for TRIM_ADMIN_STEP_UP_TTL_SEC.
 */
export const useStepUpStore = create<StepUpState>()(
  persist(
    (set, get) => ({
      token: "",
      expiresAt: 0,
      verifyOpen: false,
      setToken: (token, expiresInSec) => {
        const next = String(token || "").trim();
        if (!next) {
          clearEverywhere();
          set({ token: "", expiresAt: 0, verifyOpen: false });
          return;
        }
        let ttl = Number(expiresInSec);
        if (!Number.isFinite(ttl) || ttl < 60) ttl = FALLBACK_TTL_SEC;
        if (ttl > FALLBACK_TTL_SEC) ttl = FALLBACK_TTL_SEC;
        const expiresAt = Date.now() + ttl * 1000;
        writeEverywhere(next, expiresAt);
        set({
          token: next,
          expiresAt,
          verifyOpen: false,
        });
      },
      clear: () => {
        clearEverywhere();
        set({ token: "", expiresAt: 0, verifyOpen: false });
      },
      isActive: () => Boolean(get().activeToken()),
      activeToken: () => {
        const { token, expiresAt } = get();
        if (Boolean(token.trim()) && Number.isFinite(expiresAt) && Date.now() < expiresAt) {
          writeEverywhere(token.trim(), expiresAt);
          return token.trim();
        }
        const synced = readSessionSync();
        if (!synced) return "";
        set({ token: synced.token, expiresAt: synced.expiresAt });
        return synced.token;
      },
      openVerify: () => {
        // Elevation token is optional under enroll_only CRUD; dialog still
        // auto-closes when TOTP/passkey status shows enrolled.
        set({ verifyOpen: true });
      },
      closeVerify: () => set({ verifyOpen: false }),
    }),
    {
      name: STORAGE_KEY,
      storage: createJSONStorage(() => sessionStorage),
      partialize: (s) => ({ token: s.token, expiresAt: s.expiresAt }),
      onRehydrateStorage: () => (state) => {
        if (!state) return;
        if (!state.token?.trim() || !state.expiresAt || Date.now() >= state.expiresAt) {
          state.token = "";
          state.expiresAt = 0;
          clearEverywhere();
          return;
        }
        writeEverywhere(state.token.trim(), state.expiresAt);
      },
    },
  ),
);
