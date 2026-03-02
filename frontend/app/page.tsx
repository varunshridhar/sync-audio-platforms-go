import Link from "next/link";
import { cookies } from "next/headers";

const featureItems = [
  {
    title: "Cross-platform playlist sync",
    description: "Move and mirror playlists between Spotify and YouTube Music with one workflow."
  },
  {
    title: "Approval-gated onboarding",
    description: "Keep the app public while requiring developer approval before users can access sync features."
  },
  {
    title: "Encrypted provider credentials",
    description: "Access and refresh tokens are encrypted before being stored in the backend."
  },
  {
    title: "Live job visibility",
    description: "Track sync jobs and statuses from pending to complete right from your dashboard."
  }
];

// HomePage showcases what the platform can do with a fast, visual-first layout.
export default async function HomePage() {
  // Keep primary CTA aligned with auth state:
  // logged-out users see Login/Request access, logged-in users see Dashboard.
  const cookieStore = await cookies();
  const isLoggedIn = Boolean(cookieStore.get("session")?.value);

  return (
    <>
      <section className="hero">
        <p className="eyebrow">Playlist orchestration for modern listeners</p>
        <h1>Sync your music stack, not your stress.</h1>
        <p className="hero-copy">
          Sync Audio Platforms helps you move playlists across providers with secure auth, clear job tracking,
          and developer-approved access controls.
        </p>
        <div className="hero-cta">
          {isLoggedIn ? (
            <Link href="/dashboard" className="btn-primary">
              View dashboard
            </Link>
          ) : (
            <Link href="/login" className="btn-primary">
              Request access
            </Link>
          )}
        </div>
      </section>

      <section>
        <h2>What you can do</h2>
        <div className="feature-grid">
          {featureItems.map((item) => (
            <article className="feature-card" key={item.title}>
              <h3>{item.title}</h3>
              <p>{item.description}</p>
            </article>
          ))}
        </div>
      </section>

      <section className="workflow">
        <h2>How it works</h2>
        <ol>
          <li>Request access with your email.</li>
          <li>Developer approves your account.</li>
          <li>Connect provider tokens securely.</li>
          <li>Queue sync jobs and monitor status.</li>
        </ol>
      </section>

      <section>
        <h2>Built for fast, calm UX</h2>
        <p>
          The frontend uses lightweight styles, system fonts, and zero heavy media assets so pages load quickly,
          even on slower connections.
        </p>
      </section>
    </>
  );
}
