import type { Metadata } from "next";
import { Suspense } from "react";
import { Inter, Anton } from "next/font/google";
import "./globals.css";
import Navbar from "@/components/Navbar";
import UIProvider from "@/components/UIProvider";
import AuthProvider from "@/components/AuthProvider";

const poppins = Inter({
  subsets: ["latin"],
  weight: ["300", "400", "500", "600", "700", "800"],
  variable: "--font-poppins",
  display: "swap",
  fallback: ["Segoe UI", "system-ui", "Arial", "sans-serif"],
});

const logoFont = Anton({
  subsets: ["latin"],
  weight: "400",
  variable: "--font-logo",
  display: "swap",
  fallback: ["Impact", "Arial Narrow", "sans-serif"],
});

import RouteGuard from "@/components/RouteGuard";

export const metadata: Metadata = {
  metadataBase: new URL(process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3000"),
  title: {
    default: "Waveflix | Stream Without Limits",
    template: "%s",
  },
  description: "Watch your favorite movies and TV series without limits.",
  openGraph: {
    title: "Waveflix | Stream Without Limits",
    description: "Watch your favorite movies and TV series without limits.",
    type: "website",
    siteName: "Waveflix",
  },
  twitter: {
    card: "summary_large_image",
    title: "Waveflix | Stream Without Limits",
    description: "Watch your favorite movies and TV series without limits.",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html
      lang="en"
      className={`${poppins.variable} ${logoFont.variable}`}
      suppressHydrationWarning
    >
      <body className="font-sans" suppressHydrationWarning>
        <AuthProvider>
          <UIProvider>
            <RouteGuard>
              <Suspense fallback={null}>
                <Navbar />
              </Suspense>
              {children}
              <footer className="py-10 text-center text-sm text-white/40 max-w-3xl mx-auto px-4">
                WAVEFLIX does not host, store, or distribute any media files. All content is automatically fetched from third-party providers on the internet.
              </footer>
            </RouteGuard>
          </UIProvider>
        </AuthProvider>
      </body>
    </html>
  );
}
