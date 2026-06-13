import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import { QueryProvider } from "@/components/providers/QueryProvider";
import { AuthHydrator } from "@/components/providers/AuthHydrator";
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
  title: {
    default: "Sahiy Stream — Live Streaming",
    template: "%s | Sahiy Stream",
  },
  description:
    "Professional live streaming platform. Watch and broadcast in HD with adaptive bitrate.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="uz" className="dark">
      <body
        className={`${geistSans.variable} ${geistMono.variable} min-h-screen antialiased`}
      >
        <QueryProvider>
          <AuthHydrator>{children}</AuthHydrator>
        </QueryProvider>
      </body>
    </html>
  );
}
