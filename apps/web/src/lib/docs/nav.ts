export type DocNavItem = {
  slug: string;
  title: string;
};

export type DocNavSection = {
  id: string;
  title: string;
  items: DocNavItem[];
};

export type DocHeading = {
  id: string;
  title: string;
};

export type DocPage = {
  slug: string;
  sectionId: string;
  title: string;
  description: string;
  headings: DocHeading[];
};

/** Sidebar IA mirrors Cursor / Vercel / Next.js: Get started, Concepts, product surfaces, then ops. */
export const DOC_NAV: DocNavSection[] = [
  {
    id: "get-started",
    title: "Get started",
    items: [
      { slug: "", title: "Introduction" },
      { slug: "quickstart", title: "Quickstart" },
      { slug: "installation", title: "Installation" },
      { slug: "uninstall", title: "Uninstall" },
    ],
  },
  {
    id: "concepts",
    title: "Concepts",
    items: [
      { slug: "concepts/overview", title: "How Trim works" },
      { slug: "concepts/always-on", title: "Always-on with IDE" },
      { slug: "concepts/fast-vs-deep", title: "Fast Mode and Deep Mode" },
      { slug: "concepts/privacy", title: "Privacy model" },
      { slug: "concepts/metering", title: "Metering and quotas" },
    ],
  },
  {
    id: "cli",
    title: "CLI",
    items: [
      { slug: "cli/overview", title: "CLI overview" },
      { slug: "cli/start", title: "trim start" },
      { slug: "cli/autostart", title: "trim autostart" },
      { slug: "cli/login", title: "trim login" },
      { slug: "cli/status", title: "trim status" },
      { slug: "cli/stats", title: "trim stats" },
      { slug: "cli/compress", title: "trim compress" },
      { slug: "cli/config", title: "Configuration" },
      { slug: "cli/history-keep-turns", title: "history_keep_turns" },
    ],
  },
  {
    id: "ide",
    title: "IDE",
    items: [
      { slug: "ide/connect", title: "Connect any IDE" },
      { slug: "ide/cursor", title: "Cursor setup" },
      { slug: "ide/vscode", title: "VS Code setup" },
      { slug: "ide/continue", title: "Continue setup" },
      { slug: "ide/windsurf", title: "Windsurf setup" },
      { slug: "ide/zed", title: "Zed setup" },
      { slug: "ide/jetbrains", title: "JetBrains setup" },
      { slug: "ide/aider", title: "Aider and shell clients" },
      { slug: "ide/extension", title: "Trim IDE extension" },
      { slug: "ide/openai-compatible", title: "OpenAI-compatible clients" },
      { slug: "ide/claude-code", title: "Claude Code" },
      { slug: "ide/provider-adapters", title: "Provider adapters" },
      { slug: "ide/troubleshoot-base-url", title: "Base URL troubleshooting" },
      { slug: "ide/workflow", title: "Day-to-day workflow" },
    ],
  },
  {
    id: "dashboard",
    title: "Dashboard",
    items: [
      { slug: "dashboard/overview", title: "Dashboard overview" },
      { slug: "dashboard/usage", title: "Usage and traces" },
      { slug: "dashboard/billing", title: "Plans and billing" },
      { slug: "dashboard/keys", title: "Settings and API keys" },
      { slug: "dashboard/team", title: "Team and seats" },
      { slug: "dashboard/preferences", title: "Preferences" },
    ],
  },
  {
    id: "admin",
    title: "Admin console",
    items: [
      { slug: "admin/overview", title: "Admin overview" },
      { slug: "admin/users", title: "Users" },
      { slug: "admin/rbac", title: "RBAC" },
      { slug: "admin/billing", title: "Billing and catalog" },
      { slug: "admin/revenue", title: "Sales and revenue" },
      { slug: "admin/product", title: "Product" },
      { slug: "admin/chrome", title: "Chrome" },
      { slug: "admin/auth", title: "Auth settings" },
      { slug: "admin/denylist", title: "Denylist" },
      { slug: "admin/compliance", title: "Compliance" },
      { slug: "admin/audit", title: "Audit" },
      { slug: "admin/observability", title: "Observability" },
      { slug: "admin/segments", title: "Segments" },
      { slug: "admin/enterprise", title: "Enterprise" },
      { slug: "admin/email", title: "Email templates" },
      { slug: "admin/distribution", title: "Distribution" },
      { slug: "admin/break-glass", title: "Break-glass" },
    ],
  },
  {
    id: "api",
    title: "API",
    items: [
      { slug: "api/overview", title: "API overview" },
      { slug: "api/auth", title: "Authentication" },
      { slug: "api/proxy", title: "Chat completions proxy" },
      { slug: "api/errors", title: "Errors and quotas" },
    ],
  },
  {
    id: "guides",
    title: "Guides",
    items: [
      { slug: "guides/troubleshooting", title: "Troubleshooting" },
      { slug: "guides/faq", title: "FAQ" },
    ],
  },
  {
    id: "security",
    title: "Security",
    items: [
      { slug: "security/overview", title: "Security overview" },
      { slug: "security/device-binding", title: "Device binding" },
      { slug: "security/fraud", title: "Abuse prevention" },
      { slug: "security/telemetry", title: "Telemetry" },
    ],
  },
  {
    id: "self-hosting",
    title: "Self-hosting",
    items: [
      { slug: "self-hosting/overview", title: "Self-hosting overview" },
      { slug: "self-hosting/environment", title: "Environment" },
      { slug: "self-hosting/deploy", title: "Deploy the API" },
    ],
  },
];

export function flattenDocNav(): DocNavItem[] {
  return DOC_NAV.flatMap((section) => section.items);
}

export function docHref(slug: string) {
  return slug ? `/docs/${slug}` : "/docs";
}

export function findDocSectionId(slug: string): string | null {
  for (const section of DOC_NAV) {
    if (section.items.some((item) => item.slug === slug)) return section.id;
  }
  return null;
}

export function adjacentDocs(slug: string): {
  prev: DocNavItem | null;
  next: DocNavItem | null;
} {
  const flat = flattenDocNav();
  const index = flat.findIndex((item) => item.slug === slug);
  if (index < 0) return { prev: null, next: null };
  return {
    prev: index > 0 ? flat[index - 1] : null,
    next: index < flat.length - 1 ? flat[index + 1] : null,
  };
}
