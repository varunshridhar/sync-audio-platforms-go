const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

// Shared API types mirror backend JSON response contracts.
export type User = {
  id: string;
  email: string;
  status: "pending" | "approved" | "rejected";
  approvedBy?: string;
  approvedAt?: string;
  createdAt: string;
};

export type ProviderAccount = {
  userId: string;
  provider: "spotify" | "youtube_music";
  tokenExpiryUnix: number;
  lastSyncCheckpoint: string;
  connectedAt: string;
};

export type SyncJob = {
  id: string;
  userId: string;
  source: "spotify" | "youtube_music";
  destination: "spotify" | "youtube_music";
  playlistId: string;
  status: "pending" | "running" | "complete" | "failed";
  error?: string;
  createdAt: string;
  updatedAt: string;
};

export type PendingUser = User;

async function parseJSON<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const maybeError = await response.json().catch(() => ({ error: "request failed" }));
    throw new Error(maybeError.error ?? "request failed");
  }
  return response.json() as Promise<T>;
}

// login creates server session cookie (credentials: include is required).
export async function login(email: string, captchaToken: string, accessCode: string): Promise<User> {
  const response = await fetch(`${API_BASE_URL}/v1/auth/login`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, captchaToken, accessCode, website: "" })
  });
  return parseJSON<User>(response);
}

// logout clears server session cookie.
export async function logout(): Promise<void> {
  await fetch(`${API_BASE_URL}/v1/auth/logout`, {
    method: "POST",
    credentials: "include"
  });
}

// me fetches currently authenticated user from backend.
export async function me(): Promise<User> {
  const response = await fetch(`${API_BASE_URL}/v1/me`, { credentials: "include" });
  return parseJSON<User>(response);
}

// listProviders returns connected provider metadata without exposing tokens.
export async function listProviders(): Promise<ProviderAccount[]> {
  const response = await fetch(`${API_BASE_URL}/v1/providers`, { credentials: "include" });
  return parseJSON<ProviderAccount[]>(response);
}

// connectProvider posts provider credential payload to backend.
export async function connectProvider(payload: {
  provider: "spotify" | "youtube_music";
  accessToken: string;
  refreshToken: string;
  expiryUnix: number;
}): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/v1/providers/connect`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
  if (!response.ok) {
    const maybeError = await response.json().catch(() => ({ error: "request failed" }));
    throw new Error(maybeError.error ?? "request failed");
  }
}

// listJobs returns latest sync job status list for dashboard rendering.
export async function listJobs(): Promise<SyncJob[]> {
  const response = await fetch(`${API_BASE_URL}/v1/sync/jobs`, { credentials: "include" });
  return parseJSON<SyncJob[]>(response);
}

// createJob queues a new sync job request.
export async function createJob(payload: {
  source: "spotify" | "youtube_music";
  destination: "spotify" | "youtube_music";
  playlistId: string;
}): Promise<SyncJob> {
  const response = await fetch(`${API_BASE_URL}/v1/sync/jobs`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
  return parseJSON<SyncJob>(response);
}

// listPendingUsers returns users waiting for developer approval.
export async function listPendingUsers(): Promise<PendingUser[]> {
  const response = await fetch(`${API_BASE_URL}/v1/admin/users/pending`, { credentials: "include" });
  return parseJSON<PendingUser[]>(response);
}

// approveUser approves a pending user account.
export async function approveUser(userId: string): Promise<User> {
  const response = await fetch(`${API_BASE_URL}/v1/admin/users/${encodeURIComponent(userId)}/approve`, {
    method: "POST",
    credentials: "include"
  });
  return parseJSON<User>(response);
}
