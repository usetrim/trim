import { applyDocumentDarkClass, resolvePrefersDark } from "@/lib/theme-dom";
import { create } from "zustand";

/** App theme is binary. "system" is only resolved once at hydrate for legacy storage. */
export type ThemeMode = "light" | "dark";

type ThemeState = {
  mode: ThemeMode | null;
  setMode: (mode: ThemeMode) => void;
  hydrate: () => void;
};

const STORAGE_KEY = "trim-admin-theme";

function applyTheme(mode: ThemeMode) {
  applyDocumentDarkClass(mode === "dark", mode);
}

function readInitialMode(): ThemeMode {
  if (typeof localStorage === "undefined") {
    return resolvePrefersDark() ? "dark" : "light";
  }
  const raw = localStorage.getItem(STORAGE_KEY);
  if (raw === "light" || raw === "dark") return raw;
  // Legacy "system" or missing → resolve OS once, then persist concrete mode.
  const resolved: ThemeMode = resolvePrefersDark() ? "dark" : "light";
  try {
    localStorage.setItem(STORAGE_KEY, resolved);
  } catch {
    /* ignore */
  }
  return resolved;
}

export const useThemeStore = create<ThemeState>((set, get) => ({
  mode: null,
  setMode: (mode) => {
    if (typeof localStorage !== "undefined") {
      localStorage.setItem(STORAGE_KEY, mode);
    }
    applyTheme(mode);
    set({ mode });
  },
  hydrate: () => {
    if (get().mode) return;
    const mode = readInitialMode();
    applyTheme(mode);
    set({ mode });
  },
}));
