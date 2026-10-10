import type { NextConfig } from "next";

// Optional CDN / static asset host. Empty = relative assets (no invent CDN URL).
const assetPrefix = (process.env.NEXT_PUBLIC_ASSET_PREFIX || "").trim();

/**
 * Next 15 defaults dynamic router-cache staleTime to 0s, so every soft-nav
 * re-fetches the page RSC. Align with React Query (5m): revisit reuses the
 * segment; billing/workspace mutations refresh via invalidateQueries.
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
};

export default nextConfig;
