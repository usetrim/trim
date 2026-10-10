import { Providers } from "@/components/providers";
import { buildWebRootMetadata, loadSiteChrome, siteViewport } from "@/lib/site-metadata";
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
  return buildWebRootMetadata();
}

export const viewport: Viewport = siteViewport;

export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const site = await loadSiteChrome();
  const lang = site?.html_lang?.trim() || undefined;

  return (
    <html {...(lang ? { lang } : {})} suppressHydrationWarning>
      <head>
        <script
          // Apply saved / system theme before paint (logos + UI + favicons).
          // biome-ignore lint/security/noDangerouslySetInnerHtml: static theme boot script, no user input
          dangerouslySetInnerHTML={{
            __html: `(function(){try{var k="trim-theme";var m=localStorage.getItem(k);var dark;if(m==="light")dark=false;else if(m==="dark")dark=true;else{dark=window.matchMedia("(prefers-color-scheme: dark)").matches;m=dark?"dark":"light";try{localStorage.setItem(k,m);}catch(e){}}var r=document.documentElement;r.classList.toggle("dark",dark);r.style.colorScheme=dark?"dark":"light";r.setAttribute("data-trim-theme",m);}catch(e){}})();`,
          }}
        />
      </head>
      <body className={`${plexSans.variable} ${plexMono.variable} font-sans`}>
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
