"use client";

import { FormEvent, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Script from "next/script";
import { login } from "@/lib/api";

// LoginPage provides request-access flow with optional access-code instant approval.
export default function LoginPage() {
  const turnstileSiteKey = process.env.NEXT_PUBLIC_TURNSTILE_SITE_KEY ?? "";
  const [email, setEmail] = useState("");
  const [accessCode, setAccessCode] = useState("");
  const [captchaToken, setCaptchaToken] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const router = useRouter();

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    if (!captchaToken) {
      setError("Please complete the captcha challenge.");
      return;
    }
    setLoading(true);
    try {
      await login(email.trim(), captchaToken, accessCode.trim());
      router.push("/dashboard");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <section>
      {turnstileSiteKey ? (
        <Script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer />
      ) : null}
      <h1>Request Access</h1>
      <small>Use your email. A valid access code can instantly approve your account (codes may be limited-use).</small>
      <form onSubmit={onSubmit}>
        <p>
          <input
            type="email"
            autoComplete="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="you@example.com"
          />
        </p>
        <p>
          <input
            type="text"
            value={accessCode}
            onChange={(e) => setAccessCode(e.target.value)}
            placeholder="Access code (optional)"
          />
        </p>
        <p>
          <button disabled={loading} type="submit">
            {loading ? "Signing in..." : "Continue"}
          </button>
        </p>
        <input
          type="text"
          name="website"
          tabIndex={-1}
          autoComplete="off"
          value=""
          onChange={() => undefined}
          style={{ display: "none" }}
          aria-hidden="true"
        />
        {turnstileSiteKey ? (
          <div
            className="cf-turnstile"
            data-sitekey={turnstileSiteKey}
            data-callback="onTurnstileSuccess"
            data-expired-callback="onTurnstileExpired"
          />
        ) : (
          <p>Missing NEXT_PUBLIC_TURNSTILE_SITE_KEY in frontend env.</p>
        )}
      </form>
      {error ? <p>{error}</p> : null}
      <Script id="turnstile-callbacks">
        {`window.onTurnstileSuccess = (token) => window.dispatchEvent(new CustomEvent("turnstile-token", { detail: token }));
window.onTurnstileExpired = () => window.dispatchEvent(new CustomEvent("turnstile-token", { detail: "" }));`}
      </Script>
      <TurnstileTokenBridge onToken={setCaptchaToken} />
    </section>
  );
}

function TurnstileTokenBridge({ onToken }: { onToken: (token: string) => void }) {
  useEffect(() => {
    function onTurnstileToken(event: Event) {
      const custom = event as CustomEvent<string>;
      onToken(custom.detail ?? "");
    }
    window.addEventListener("turnstile-token", onTurnstileToken);
    return () => {
      window.removeEventListener("turnstile-token", onTurnstileToken);
    };
  }, [onToken]);

  return null;
}
