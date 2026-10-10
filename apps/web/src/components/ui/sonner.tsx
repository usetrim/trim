"use client";

import { useTheme } from "next-themes";
import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { Toaster as Sonner, type ToasterProps } from "sonner";

/**
 * Stacking layers (project-wide):
 * - mobile nav / soft overlays: z-20
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
 * Theme-aware toast host. Avoids Sonner richColors (blue/green/red invent accents).
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
  const { resolvedTheme } = useTheme();
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
      theme={(resolvedTheme === "light" ? "light" : "dark") as ToasterProps["theme"]}
      className="toaster group"
      visibleToasts={5}
      toastOptions={{
        classNames: {
          toast:
            "group toast trim-float group-[.toaster]:bg-[var(--trim-panel)] group-[.toaster]:text-[var(--trim-fg)] group-[.toaster]:border-[var(--trim-border-strong)] group-[.toaster]:opacity-100",
          description: "group-[.toast]:text-[var(--trim-muted)]",
          actionButton:
            "group-[.toast]:bg-[var(--trim-ink)] group-[.toast]:text-[var(--trim-ink-inverse)]",
          cancelButton:
            "group-[.toast]:border group-[.toast]:border-[var(--trim-border)] group-[.toast]:bg-[var(--trim-panel)] group-[.toast]:text-[var(--trim-muted)]",
          success:
            "group-[.toaster]:bg-[var(--trim-panel)] group-[.toaster]:text-[var(--trim-fg)] group-[.toaster]:border-[var(--trim-border-strong)]",
          error:
            "group-[.toaster]:bg-[var(--trim-panel)] group-[.toaster]:text-[var(--trim-fg)] group-[.toaster]:border-[var(--trim-border-strong)]",
          warning:
            "group-[.toaster]:bg-[var(--trim-panel)] group-[.toaster]:text-[var(--trim-fg)] group-[.toaster]:border-[var(--trim-border-strong)]",
          info: "group-[.toaster]:bg-[var(--trim-panel)] group-[.toaster]:text-[var(--trim-fg)] group-[.toaster]:border-[var(--trim-border-strong)]",
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
