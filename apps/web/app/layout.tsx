import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import { Toaster } from "@/components/ui/sonner";
import { Providers } from "@/components/providers";
import { AppearanceProvider } from "@/components/appearance-provider";
import { FIRST_PAINT_SCRIPT } from "@/lib/appearance";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "sparstrowgen",
  description: "One chat window for every coding agent you have installed.",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    // suppressHydrationWarning because the appearance is written onto <html>
    // before React runs — by the script below for the surface and accent, and
    // by next-themes for the mode — so the server's markup deliberately differs
    // from the browser's first render.
    <html
      lang="en"
      suppressHydrationWarning
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
    >
      <head>
        {/* Runs before the first paint, so a person's own theme is what they
            see rather than a flash of somebody else's. */}
        <script dangerouslySetInnerHTML={{ __html: FIRST_PAINT_SCRIPT }} />
      </head>
      <body className="h-full overflow-hidden">
        <Providers>
          <AppearanceProvider>{children}</AppearanceProvider>
        </Providers>
        <Toaster position="bottom-right" />
      </body>
    </html>
  );
}
