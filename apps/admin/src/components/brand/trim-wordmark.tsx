import { cn } from "@/lib/utils";
import Image from "next/image";
import Link from "next/link";

/** Width / height of generated page marks (scripts/_gen_site_icons.py). */
export const TRIM_WORDMARK_ASPECT = 0.988281;

const SRC_BLACK = "/brand/trim-mark-black.png";
const SRC_WHITE = "/brand/trim-mark-white.png";

type Size = "xs" | "sm" | "md" | "lg" | "xl";

const HEIGHT: Record<Size, number> = {
  xs: 14,
  sm: 18,
  md: 22,
  lg: 28,
  xl: 36,
};

type TrimWordmarkProps = {
  className?: string;
  size?: Size;
  alt?: string;
  priority?: boolean;
  /**
   * `auto` follows painted theme (black in light, white in dark).
   * `black` forces the black mark - use for receipt print / light paper.
   */
  ink?: "auto" | "black";
};

/**
 * Brand mark (page logo). Follows painted UI (`html.dark`), same as page chrome -
 * so forced app light/dark never clashes with OS preference.
 */
export function TrimWordmark({
  className,
  size = "md",
  alt = "Trim",
  priority = false,
  ink = "auto",
}: TrimWordmarkProps) {
  const h = HEIGHT[size];
  const w = Math.max(1, Math.round(h * TRIM_WORDMARK_ASPECT));
  const decorative = !alt;
  const imgClass = "h-full w-auto";
  const forceBlack = ink === "black";

  return (
    <span
      className={cn("inline-flex shrink-0 items-center leading-none", className)}
      style={{ height: h }}
      {...(decorative ? { "aria-hidden": true } : {})}
    >
      <Image
        src={SRC_BLACK}
        alt={decorative ? "" : alt}
        width={w}
        height={h}
        priority={priority}
        className={cn(imgClass, !forceBlack && "dark:hidden")}
      />
      {!forceBlack ? (
        <Image
          src={SRC_WHITE}
          alt=""
          width={w}
          height={h}
          priority={priority}
          aria-hidden
          className={cn(imgClass, "hidden dark:block")}
        />
      ) : null}
    </span>
  );
}

type TrimWordmarkLinkProps = TrimWordmarkProps & {
  href: string;
  linkClassName?: string;
};

export function TrimWordmarkLink({
  href,
  linkClassName,
  className,
  size = "md",
  alt = "Trim",
  priority,
}: TrimWordmarkLinkProps) {
  return (
    <Link
      href={href}
      scroll={false}
      className={cn("inline-flex items-center transition hover:opacity-80", linkClassName)}
      aria-label={alt || "Trim"}
    >
      <TrimWordmark className={className} size={size} alt="" priority={priority} />
    </Link>
  );
}
