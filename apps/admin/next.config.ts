import type { NextConfig } from "next";

// Optional CDN / static asset host. Empty = relative assets (no invent CDN URL).
const assetPrefix = (process.env.NEXT_PUBLIC_ASSET_PREFIX || "").trim();

/**
 * Next 15 defaults dynamic router-cache staleTime to 0s, so every soft-nav
 * re-fetches the page RSC (loading.tsx flash + client remount). Sidebar links
 * use prefetch (static window once), then fall back to dynamic - set dynamic
 * to match React Query staleTime so revisits reuse the segment; mutations still
 * refresh via invalidateQueries.
 */
const nextConfig: NextConfig = {
  reactStrictMode: true,
  ...(assetPrefix ? { assetPrefix } : {}),
  // SSG can wait on public chrome fetch; keep above AbortSignal timeout (8s) with headroom.
  staticPageGenerationTimeout: 120,
  experimental: {
    // Cap SSG parallelism so local API is not stampeded during next build.
    cpus: 2,
    staleTimes: {
      dynamic: 300,
      static: 300,
    },
  },
  async redirects() {
    return [
      { source: "/billing/receipts", destination: "/sales/receipts", permanent: false },
      { source: "/billing/subscriptions", destination: "/sales/subscriptions", permanent: false },
      { source: "/billing/credits", destination: "/sales/credits", permanent: false },
      { source: "/enterprise", destination: "/sales/enterprise", permanent: false },
    ];
  },
};

export default nextConfig;
