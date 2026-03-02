import "./globals.css";
import type { Metadata } from "next";
import { ReactNode } from "react";
import Link from "next/link";
import { cookies } from "next/headers";

// Metadata is used by Next.js to populate browser title/description.
export const metadata: Metadata = {
  title: "Sync Audio Platforms",
  description: "Sync playlists across platforms with secure credentials, job tracking, and approval-gated onboarding."
};

// RootLayout wraps every page with shared HTML/body and global styles.
export default async function RootLayout({ children }: { children: ReactNode }) {
  // We treat existence of backend session cookie as authenticated navigation state.
  const cookieStore = await cookies();
  const isLoggedIn = Boolean(cookieStore.get("session")?.value);

  return (
    <html lang="en">
      <body>
        <header className="topbar">
          <div className="topbar-inner">
            <Link href="/" className="brand">
              Sync Audio Platforms
            </Link>
            <nav className="topnav">
              {isLoggedIn ? <Link href="/dashboard">Dashboard</Link> : <Link href="/login">Login</Link>}
            </nav>
          </div>
        </header>
        <main>{children}</main>
      </body>
    </html>
  );
}

