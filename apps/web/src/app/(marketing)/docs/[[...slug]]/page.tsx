import { DocBody } from "@/components/docs/doc-body";
import { DocsPager } from "@/components/docs/docs-pager";
import { DocsToc } from "@/components/docs/docs-toc";
import { allDocSlugs, getDocPage } from "@/lib/docs/content";
import { flattenDocNav } from "@/lib/docs/nav";
import type { Metadata } from "next";
import { notFound } from "next/navigation";

type Props = { params: Promise<{ slug?: string[] }> };

function slugFromParams(slug?: string[]) {
  if (!slug || slug.length === 0) return "";
  return slug.join("/");
}

export function generateStaticParams() {
  return flattenDocNav().map((item) => ({
    slug: item.slug ? item.slug.split("/") : undefined,
  }));
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { slug: parts } = await params;
  const slug = slugFromParams(parts);
  const page = getDocPage(slug);
  if (!page) {
    return { robots: { index: false, follow: true } };
  }
  return {
    title: page.title,
    description: page.description,
  };
}

export default async function DocsPage({ params }: Props) {
  const { slug: parts } = await params;
  const slug = slugFromParams(parts);
  const page = getDocPage(slug);
  if (!page) notFound();

  void allDocSlugs;

  return (
    <div className="mx-auto grid w-full max-w-[1100px] gap-10 xl:grid-cols-[minmax(0,1fr)_200px]">
      <article className="min-w-0">
        <div className="xl:hidden">
          <DocsToc headings={page.headings} />
        </div>
        <p className="font-mono text-[11px] uppercase tracking-[0.18em] text-[var(--trim-subtle)]">
          Documentation
        </p>
        <h1 className="font-display mt-3 text-3xl font-semibold tracking-tight text-[var(--trim-fg)] sm:text-4xl">
          {page.title}
        </h1>
        <p className="mt-3 max-w-3xl text-base leading-relaxed text-[var(--trim-muted)]">
          {page.description}
        </p>
        <div className="mt-10">
          <DocBody blocks={page.blocks} />
        </div>
        <DocsPager slug={slug} />
      </article>
      <aside className="hidden self-start xl:sticky xl:top-24 xl:block xl:max-h-[calc(100vh-7rem)] xl:overflow-y-auto">
        <DocsToc headings={page.headings} />
      </aside>
    </div>
  );
}
