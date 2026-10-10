"use client";

import { useThemeStore } from "@/stores/theme";
import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { Toaster as Sonner, type ToasterProps } from "sonner";

/**
 * Stacking layers (project-wide):
 * - dialog / alert / sheet / popover / select: z-100
 * - nested calendar in dialogs: z-110
 * - toasts: z-9999 via #trim-toast-host on document.body
 *
 * One host for every CRUD toast. Keep host as last body child when Radix
 * portals mount - do not remount Sonner (move host in place only).
 */
const TOAST_Z_INDEX = 9999;
const TOAST_HOST_ID = "trim-toast-host";

function ensureToastHost(): HTMLElement | null {
  if (typeof document === "undefined") return null;
  let host = document.getElementById(TOAST_HOST_ID);
  if (!host) {
    host = document.createElement("div");
    host.id = TOAST_HOST_ID;
    host.setAttribute("data-trim-toast-host", "");
    document.body.appendChild(host);
  } else if (document.body.lastElementChild !== host) {
    document.body.appendChild(host);
  }
  return host;
}

/**
 * Semantic toast host. Avoids Sonner richColors (blue/green/red invent accents).
 * Theme follows admin light/dark store (dark until hydrated).
 * Desktop: top-right. Mobile (≤600px): Sonner full-bleed + CSS in globals.css -
 * do not force left:auto globally or mobile layout breaks.
 */
export function Toaster({
  position = "top-right",
  offset = { top: "1rem", right: "1rem" },
  mobileOffset = {
    top: "0.75rem",
    left: "0.75rem",
    right: "0.75rem",
  },
  style,
  ...props
}: ToasterProps) {
  const mode = useThemeStore((s) => s.mode);
  const theme: ToasterProps["theme"] = mode === "light" ? "light" : "dark";
  const [host, setHost] = useState<HTMLElement | null>(null);

  useEffect(() => {
    const el = ensureToastHost();
    setHost(el);
    let raf = 0;
    const mo = new MutationObserver(() => {
      // Reorder only - never setState (avoids remounting Sonner on every modal open).
      cancelAnimationFrame(raf);
      raf = requestAnimationFrame(() => {
        ensureToastHost();
      });
    });
    mo.observe(document.body, { childList: true });
    return () => {
      mo.disconnect();
      cancelAnimationFrame(raf);
    };
  }, []);

  if (!host) return null;

  return createPortal(
    <Sonner
      theme={theme}
      className="toaster group"
      visibleToasts={5}
      toastOptions={{
        classNames: {
          toast:
            "group toast group-[.toaster]:bg-background group-[.toaster]:text-foreground group-[.toaster]:border-border group-[.toaster]:shadow-lg group-[.toaster]:opacity-100 dark:group-[.toaster]:bg-popover dark:group-[.toaster]:text-popover-foreground dark:group-[.toaster]:border-border",
          description: "group-[.toast]:text-muted-foreground",
          actionButton: "group-[.toast]:bg-accent group-[.toast]:text-accent-foreground",
          cancelButton:
            "group-[.toast]:bg-muted group-[.toast]:text-muted-foreground dark:group-[.toast]:bg-accent dark:group-[.toast]:text-foreground",
          success: "group-[.toaster]:border-border",
          error: "group-[.toaster]:border-border",
          warning: "group-[.toaster]:border-border",
          info: "group-[.toaster]:border-border",
        },
      }}
      {...props}
      position={position}
      offset={offset}
      mobileOffset={mobileOffset}
      style={{
        ...(typeof style === "object" && style ? style : {}),
        zIndex: TOAST_Z_INDEX,
      }}
    />,
    host,
  );
}
