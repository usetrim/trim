"use client";

import { ThemeProvider as NextThemesProvider } from "next-themes";
import type { ReactNode } from "react";

export function ThemeProvider({ children }: { children: ReactNode }) {
  return (
    <NextThemesProvider
      attribute="class"
      // First visit follows OS via enableSystem; toggle only stores light|dark.
      defaultTheme="system"
      enableSystem
      disableTransitionOnChange
      storageKey="trim-theme"
    >
      {children}
    </NextThemesProvider>
  );
}
