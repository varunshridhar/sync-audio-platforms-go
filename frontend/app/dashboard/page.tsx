"use client";

import { FormEvent, useEffect, useState } from "react";
import Link from "next/link";
import { connectProvider, createJob, listJobs, listProviders, logout, me, ProviderAccount, SyncJob, User } from "@/lib/api";

// DashboardPage is the control panel for:
// - showing logged-in user info,
// - connecting provider tokens,
// - creating sync jobs,
// - viewing recent job status.
export default function DashboardPage() {
  const [user, setUser] = useState<User | null>(null);
  const [providers, setProviders] = useState<ProviderAccount[]>([]);
  const [jobs, setJobs] = useState<SyncJob[]>([]);
  const [statusMsg, setStatusMsg] = useState("");
  const [errorMsg, setErrorMsg] = useState("");

  // Initial data load for current user, provider connections, and recent jobs.
  useEffect(() => {
    async function init() {
      try {
        const currentUser = await me();
        setUser(currentUser);
        if (currentUser.status === "approved") {
          setProviders(await listProviders());
          setJobs(await listJobs());
        }
      } catch (err) {
        setErrorMsg(err instanceof Error ? err.message : "Failed to load dashboard");
      }
    }
    void init();
  }, []);

  // Saves provider credentials/timing metadata via backend API.
  async function onConnectProvider(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const provider = form.get("provider");
    const accessToken = form.get("accessToken");
    const refreshToken = form.get("refreshToken");
    const expiryUnix = form.get("expiryUnix");
    if (
      (provider !== "spotify" && provider !== "youtube_music") ||
      typeof accessToken !== "string" ||
      typeof refreshToken !== "string" ||
      typeof expiryUnix !== "string"
    ) {
      setErrorMsg("Invalid provider form values");
      return;
    }
    try {
      await connectProvider({
        provider,
        accessToken,
        refreshToken,
        expiryUnix: Number(expiryUnix)
      });
      setProviders(await listProviders());
      setStatusMsg("Provider connected successfully");
      setErrorMsg("");
      event.currentTarget.reset();
    } catch (err) {
      setErrorMsg(err instanceof Error ? err.message : "Failed to connect provider");
    }
  }

  // Queues a new sync job from one provider to another.
  async function onCreateJob(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const source = form.get("source");
    const destination = form.get("destination");
    const playlistId = form.get("playlistId");
    if (
      (source !== "spotify" && source !== "youtube_music") ||
      (destination !== "spotify" && destination !== "youtube_music") ||
      typeof playlistId !== "string"
    ) {
      setErrorMsg("Invalid job form values");
      return;
    }
    try {
      await createJob({ source, destination, playlistId });
      setJobs(await listJobs());
      setStatusMsg("Sync job queued");
      setErrorMsg("");
      event.currentTarget.reset();
    } catch (err) {
      setErrorMsg(err instanceof Error ? err.message : "Failed to create sync job");
    }
  }

  // Clears backend session cookie and returns user to login page.
  async function onLogout() {
    await logout();
    window.location.href = "/login";
  }

  return (
    <>
      <section>
        <h1>Dashboard</h1>
        <p>Signed in as: {user?.email ?? "..."}</p>
        <p>Account status: {user?.status ?? "..."}</p>
        <button onClick={onLogout} type="button">
          Logout
        </button>
        <p>
          <Link href="/admin">Admin approvals</Link>
        </p>
        {statusMsg ? <p>{statusMsg}</p> : null}
        {errorMsg ? <p>{errorMsg}</p> : null}
      </section>
      {user?.status === "pending" ? (
        <section>
          <h2>Awaiting Approval</h2>
          <p>Your signup request is pending developer approval. You can sign out and return later.</p>
        </section>
      ) : null}

      <section hidden={user?.status !== "approved"}>
        <h2>Connect Provider</h2>
        <form onSubmit={onConnectProvider}>
          <p>
            <select name="provider" required defaultValue="spotify">
              <option value="spotify">Spotify</option>
              <option value="youtube_music">YouTube Music</option>
            </select>
          </p>
          <p>
            <input name="accessToken" required placeholder="Access token" />
          </p>
          <p>
            <input name="refreshToken" required placeholder="Refresh token" />
          </p>
          <p>
            <input name="expiryUnix" required type="number" placeholder="Expiry Unix timestamp" />
          </p>
          <button type="submit">Save Provider</button>
        </form>
      </section>

      <section hidden={user?.status !== "approved"}>
        <h2>Create Sync Job</h2>
        <form onSubmit={onCreateJob}>
          <p>
            <select name="source" required defaultValue="spotify">
              <option value="spotify">Spotify</option>
              <option value="youtube_music">YouTube Music</option>
            </select>
            <select name="destination" required defaultValue="youtube_music">
              <option value="youtube_music">YouTube Music</option>
              <option value="spotify">Spotify</option>
            </select>
          </p>
          <p>
            <input name="playlistId" required placeholder="Source playlist ID" />
            <button type="submit">Queue Sync</button>
          </p>
        </form>
      </section>

      <section hidden={user?.status !== "approved"}>
        <h2>Connected Providers</h2>
        <pre>{JSON.stringify(providers, null, 2)}</pre>
      </section>

      <section hidden={user?.status !== "approved"}>
        <h2>Recent Sync Jobs</h2>
        <pre>{JSON.stringify(jobs, null, 2)}</pre>
      </section>
    </>
  );
}

