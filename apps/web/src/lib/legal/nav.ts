/**
 * Legal left-nav IA mirrors DocsSidebar: exclusive accordion categories with
 * in-page section links. Group titles are presentation chrome; section bodies
 * still come only from site_legal_sections.
 */

export type LegalKind = "privacy" | "terms";

export type LegalNavItem = {
  id: string;
  title: string;
  index: number;
};

export type LegalNavSection = {
  id: string;
  title: string;
  items: LegalNavItem[];
};

type LegalSectionLike = {
  heading?: string;
};

/** Stable DOM / hash id - kind-prefixed so dual-mounted Privacy+Terms never collide. */
export function legalSectionId(kind: LegalKind, heading: string | undefined, index: number) {
  const base = (heading || "introduction")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "");
  return `${kind}-${base || "section"}-${index}`;
}

export function legalBasePath(kind: LegalKind) {
  return kind === "privacy" ? "/privacy" : "/terms";
}

/** Ordered category → exact section headings (as stored in site_legal_sections). */
const PRIVACY_GROUPS: { id: string; title: string; headings: string[] }[] = [
  {
    id: "privacy-overview",
    title: "Overview",
    headings: [
      "Introduction",
      "Definitions",
      "Who we are and how to contact us",
      "Support and how to reach us",
      "Scope and roles",
    ],
  },
  {
    id: "privacy-data",
    title: "Data we collect",
    headings: [
      "Account Data and Service Data",
      "Categories of information we collect",
      "Sensitive personal information",
      "Information we do not need for local compression",
      "Sources of information",
    ],
  },
  {
    id: "privacy-use",
    title: "How we use data",
    headings: [
      "How we use information and lawful bases",
      "Payments and Merchant of Record",
      "Cookies and similar technologies",
      "Marketing communications",
    ],
  },
  {
    id: "privacy-sharing",
    title: "Sharing and transfers",
    headings: [
      "When we share information",
      "Subprocessors and service providers",
      "International transfers",
      "EU, UK, and other representatives",
    ],
  },
  {
    id: "privacy-retention",
    title: "Retention and security",
    headings: [
      "Retention",
      "Retention criteria by category",
      "Security",
      "Security incidents",
      "Automated processing",
      "Children",
    ],
  },
  {
    id: "privacy-rights",
    title: "Your rights",
    headings: [
      "Your privacy rights",
      "How to submit a privacy request",
      "Appeals and complaints",
      "Enterprise data processing terms",
      "California and similar US state notices",
      "Do not sell or share; advertising",
    ],
  },
  {
    id: "privacy-other",
    title: "Other",
    headings: [
      "Business transfers",
      "Upstream AI providers",
      "Links to other websites",
      "Changes to this policy",
      "Contact",
    ],
  },
];

const TERMS_GROUPS: { id: string; title: string; headings: string[] }[] = [
  {
    id: "terms-basics",
    title: "Agreement",
    headings: ["Agreement", "Acceptance", "Definitions", "Privacy Policy"],
  },
  {
    id: "terms-service",
    title: "The Service",
    headings: ["The Service", "License and access", "Eligibility and accounts", "Acceptable use"],
  },
  {
    id: "terms-billing",
    title: "Billing and access",
    headings: [
      "Plans, billing, upgrades, and cancellations",
      "Refunds and payment disputes",
      "Trials, renewals, and taxes",
      "Billing FAQ",
      "Teams and seats",
      "API keys and client access",
    ],
  },
  {
    id: "terms-content",
    title: "Content and IP",
    headings: [
      "Your content and feedback",
      "AI outputs and professional advice",
      "Beta and experimental features",
      "Confidentiality",
      "Intellectual property",
      "Copyright complaints (DMCA)",
      "Open source and self-hosting",
      "Customer content ownership",
      "Open-source contributions",
    ],
  },
  {
    id: "terms-liability",
    title: "Risk and liability",
    headings: [
      "Third-party services",
      "Availability and no SLA",
      "Disclaimer of warranties",
      "Limitation of liability",
      "Indemnity",
      "Mutual responsibility for AI use",
    ],
  },
  {
    id: "terms-legal",
    title: "Legal terms",
    headings: [
      "Suspension and termination",
      "Export controls and sanctions",
      "Governing law and disputes",
      "Notices",
      "General",
      "Changes",
      "Support",
      "Contact",
    ],
  },
];

function normalizeHeading(heading: string | undefined) {
  return (heading || "Introduction").trim();
}

/**
 * Map live DB sections into accordion categories. Unknown headings land in a
 * trailing "More" group so nav never silently drops content.
 */
export function buildLegalNav(kind: LegalKind, sections: LegalSectionLike[]): LegalNavSection[] {
  const groups = kind === "privacy" ? PRIVACY_GROUPS : TERMS_GROUPS;
  const byHeading = new Map<string, { index: number; title: string }>();

  sections.forEach((section, index) => {
    const title = normalizeHeading(section.heading);
    if (!byHeading.has(title)) {
      byHeading.set(title, { index, title });
    }
  });

  const used = new Set<string>();
  const nav: LegalNavSection[] = [];

  for (const group of groups) {
    const items: LegalNavItem[] = [];
    for (const heading of group.headings) {
      const match = byHeading.get(heading);
      if (!match) continue;
      used.add(heading);
      items.push({
        id: legalSectionId(kind, sections[match.index]?.heading, match.index),
        title: match.title,
        index: match.index,
      });
    }
    if (items.length > 0) {
      nav.push({ id: group.id, title: group.title, items });
    }
  }

  const leftovers: LegalNavItem[] = [];
  sections.forEach((section, index) => {
    const title = normalizeHeading(section.heading);
    if (used.has(title)) return;
    leftovers.push({
      id: legalSectionId(kind, section.heading, index),
      title,
      index,
    });
  });
  if (leftovers.length > 0) {
    nav.push({ id: `${kind}-more`, title: "More", items: leftovers });
  }

  return nav;
}

export function findLegalSectionGroupId(nav: LegalNavSection[], sectionId: string | undefined) {
  if (!sectionId) return nav[0]?.id || "";
  return (
    nav.find((section) => section.items.some((item) => item.id === sectionId))?.id ||
    nav[0]?.id ||
    ""
  );
}

export function parseLegalPath(pathname: string): { kind: LegalKind; slug: string } | null {
  if (pathname === "/privacy" || pathname.startsWith("/privacy/")) {
    const rest = pathname === "/privacy" ? "" : pathname.slice("/privacy/".length);
    return { kind: "privacy", slug: rest.replace(/\/$/, "") };
  }
  if (pathname === "/terms" || pathname.startsWith("/terms/")) {
    const rest = pathname === "/terms" ? "" : pathname.slice("/terms/".length);
    return { kind: "terms", slug: rest.replace(/\/$/, "") };
  }
  return null;
}
