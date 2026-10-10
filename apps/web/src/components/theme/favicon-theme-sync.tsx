"use client";

import { useEffect } from "react";

/**
 * Tab icons follow painted app theme (`html.dark`) only.
 * Only mutates nodes we own (`data-trim-favicon` / `data-trim-theme-color`).
 * Never remove Next metadata/viewport head tags - that races React soft-nav
 * and throws `Cannot read properties of null (reading 'removeChild')`.
 */
export function FaviconThemeSync() {
  useEffect(() => {
    let token = 0;
    let applying = false;

    const clearOurs = () => {
      for (const el of document.head.querySelectorAll(
        "[data-trim-favicon],[data-trim-theme-color]",
      )) {
        el.remove();
      }
    };

    const addLink = (attrs: Record<string, string>) => {
      const el = document.createElement("link");
      el.setAttribute("data-trim-favicon", "1");
      for (const [k, v] of Object.entries(attrs)) {
        el.setAttribute(k, v);
      }
      document.head.appendChild(el);
    };

    const apply = () => {
      if (applying) return;
      applying = true;
      try {
        token += 1;
        const v = `v=9-${token}`;
        const dark = document.documentElement.classList.contains("dark");
        const theme = dark ? "dark" : "light";

        clearOurs();

        addLink({
          rel: "icon",
          type: "image/x-icon",
          href: `/icons/favicon-${theme}.ico?${v}`,
        });
        addLink({
          rel: "icon",
          type: "image/png",
          sizes: "16x16",
          href: `/icons/icon-16-${theme}.png?${v}`,
        });
        addLink({
          rel: "icon",
          type: "image/png",
          sizes: "32x32",
          href: `/icons/icon-32-${theme}.png?${v}`,
        });
        addLink({
          rel: "icon",
          type: "image/png",
          sizes: "48x48",
          href: `/icons/icon-48-${theme}.png?${v}`,
        });
        addLink({
          rel: "apple-touch-icon",
          sizes: "180x180",
          href: `/icons/apple-touch-icon.png?${v}`,
        });

        const themeMeta = document.createElement("meta");
        themeMeta.setAttribute("data-trim-theme-color", "1");
        themeMeta.name = "theme-color";
        themeMeta.content = dark ? "#000000" : "#ffffff";
        document.head.appendChild(themeMeta);
      } finally {
        applying = false;
      }
    };

    apply();
    const obs = new MutationObserver(() => {
      // Class toggles only; never observe head (our own appends would loop).
      apply();
    });
    obs.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["class"],
    });
    return () => {
      obs.disconnect();
      clearOurs();
    };
  }, []);

  return null;
}
