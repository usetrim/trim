import { Providers } from "@/components/providers";
import { buildAdminRootMetadata, siteViewport } from "@/lib/site-metadata";
import type { Metadata, Viewport } from "next";
import { IBM_Plex_Mono, IBM_Plex_Sans } from "next/font/google";
import "./globals.css";

const plexSans = IBM_Plex_Sans({
  variable: "--font-plex-sans",
  subsets: ["latin"],
  weight: ["400", "500", "600", "700"],
});

const plexMono = IBM_Plex_Mono({
  variable: "--font-plex-mono",
  subsets: ["latin"],
  weight: ["400", "500", "600", "700"],
});

export async function generateMetadata(): Promise<Metadata> {
  return buildAdminRootMetadata();
}

export const viewport: Viewport = siteViewport;

/**
 * Auth is enforced by middleware + AdminGate (client). Do NOT force-dynamic the
 * root layout - that re-fetches the RSC tree on every soft-nav and remounts page
 * clients (looks like a full refresh when clicking sidebar links).
 */

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <script
          // Apply saved theme before paint so light/dark works without a flash.
          // biome-ignore lint/security/noDangerouslySetInnerHtml: static theme boot script, no user input
          dangerouslySetInnerHTML={{
            __html: `(function(){try{var k="trim-admin-theme";var m=localStorage.getItem(k);var dark;if(m==="light")dark=false;else if(m==="dark")dark=true;else{dark=window.matchMedia("(prefers-color-scheme: dark)").matches;m=dark?"dark":"light";try{localStorage.setItem(k,m);}catch(e){}}var r=document.documentElement;r.classList.toggle("dark",dark);r.style.colorScheme=dark?"dark":"light";r.setAttribute("data-trim-theme",m);}catch(e){}})();`,
          }}
        />
      </head>
      <body className={`${plexSans.variable} ${plexMono.variable} font-sans`}>
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
