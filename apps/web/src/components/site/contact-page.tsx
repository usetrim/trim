"use client";

import { LandingContactContentSkeleton } from "@/components/skeletons/page-skeletons";
import { useAuthProviders } from "@/hooks/queries/auth";
import type { AuthProvidersSiteChrome } from "@/types/auth";
import { Mail } from "lucide-react";

function buildMailto(email: string, subject: string, body: string): string {
  const parts: string[] = [];
  if (subject) parts.push(`subject=${encodeURIComponent(subject)}`);
  if (body) parts.push(`body=${encodeURIComponent(body)}`);
  return parts.length ? `mailto:${email}?${parts.join("&")}` : `mailto:${email}`;
}

type Topic = {
  key: string;
  label: string;
  desc: string;
  subject: string;
  body: string;
};

function topicsFromSite(site: AuthProvidersSiteChrome): Topic[] {
  const rows: Array<[string, string?, string?, string?, string?]> = [
    [
      "general",
      site.contact_topic_general_label,
      site.contact_topic_general_desc,
      site.contact_topic_general_subject,
      site.contact_topic_general_body,
    ],
    [
      "billing",
      site.contact_topic_billing_label,
      site.contact_topic_billing_desc,
      site.contact_topic_billing_subject,
      site.contact_topic_billing_body,
    ],
    [
      "sales",
      site.contact_topic_sales_label,
      site.contact_topic_sales_desc,
      site.contact_topic_sales_subject,
      site.contact_topic_sales_body,
    ],
    [
      "security",
      site.contact_topic_security_label,
      site.contact_topic_security_desc,
      site.contact_topic_security_subject,
      site.contact_topic_security_body,
    ],
  ];
  return rows
    .map(([key, label, desc, subject, body]) => ({
      key,
      label: (label || "").trim(),
      desc: (desc || "").trim(),
      subject: (subject || "").trim(),
      body: (body || "").trim(),
    }))
    .filter((t) => t.label && t.subject);
}

export function ContactPageContent() {
  const chrome = useAuthProviders();
  const site = chrome.data?.site;
  const email = (chrome.data?.support_email || "").trim();

  if (chrome.isPending || !site) {
    return <LandingContactContentSkeleton />;
  }

  const heading = site.contact_page_heading?.trim() || "";
  const body = site.contact_page_body?.trim() || "";
  const emailLabel = site.contact_email_label?.trim() || "";
  const openCta = site.contact_open_mail_cta?.trim() || "";
  const topicsHeading = site.contact_topics_heading?.trim() || "";
  const topics = topicsFromSite(site);

  if (!heading || !body || !email || !emailLabel) {
    return null;
  }

  const plainMailto = buildMailto(email, "", "");

  return (
    <div className="mx-auto w-full max-w-2xl px-5 py-12 sm:px-8 lg:px-10 lg:py-16">
      <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-[var(--trim-muted)]">
        {site.nav_contact}
      </p>
      <h1 className="font-display mt-3 text-3xl font-semibold tracking-tight text-[var(--trim-fg)] sm:text-4xl">
        {heading}
      </h1>
      <p className="mt-4 text-base leading-relaxed text-[var(--trim-muted)]">{body}</p>

      <div className="mt-8 rounded-xl border border-[var(--trim-border)] bg-[var(--trim-panel)] p-5 shadow-[var(--trim-card-shadow)]">
        <p className="text-xs font-medium uppercase tracking-[0.14em] text-[var(--trim-muted)]">
          {emailLabel}
        </p>
        <a
          href={plainMailto}
          className="mt-2 inline-flex items-center gap-2 text-lg font-medium text-[var(--trim-fg)] underline underline-offset-4 transition hover:opacity-90"
        >
          <Mail className="h-4 w-4 shrink-0 opacity-70" aria-hidden />
          {email}
        </a>
        {openCta ? (
          <div className="mt-4">
            <a
              href={plainMailto}
              className="inline-flex h-9 items-center justify-center rounded-md bg-[var(--trim-ink)] px-4 text-sm font-medium text-[var(--trim-ink-inverse)] transition hover:opacity-90"
            >
              {openCta}
            </a>
          </div>
        ) : null}
      </div>

      {topicsHeading && topics.length > 0 ? (
        <div className="mt-10">
          <h2 className="text-sm font-medium text-[var(--trim-fg)]">{topicsHeading}</h2>
          <ul className="mt-4 grid gap-3 sm:grid-cols-2">
            {topics.map((topic) => (
              <li key={topic.key}>
                <a
                  href={buildMailto(email, topic.subject, topic.body)}
                  className="flex h-full flex-col rounded-xl border border-[var(--trim-border)] bg-[var(--trim-panel)] p-4 transition hover:border-[var(--trim-border-strong)] hover:bg-[var(--trim-hover)]"
                >
                  <span className="text-sm font-semibold text-[var(--trim-fg)]">{topic.label}</span>
                  {topic.desc ? (
                    <span className="mt-1.5 text-[13px] leading-relaxed text-[var(--trim-muted)]">
                      {topic.desc}
                    </span>
                  ) : null}
                </a>
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </div>
  );
}
