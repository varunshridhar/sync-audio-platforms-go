"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { approveUser, listPendingUsers, logout, me, PendingUser, User } from "@/lib/api";

// AdminPage lets approved admin users review and approve pending signup requests.
export default function AdminPage() {
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [pendingUsers, setPendingUsers] = useState<PendingUser[]>([]);
  const [statusMsg, setStatusMsg] = useState("");
  const [errorMsg, setErrorMsg] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function init() {
      try {
        const user = await me();
        setCurrentUser(user);
        const pending = await listPendingUsers();
        setPendingUsers(pending);
      } catch (err) {
        setErrorMsg(err instanceof Error ? err.message : "Failed to load admin page");
      } finally {
        setLoading(false);
      }
    }
    void init();
  }, []);

  async function onApprove(userId: string) {
    setStatusMsg("");
    setErrorMsg("");
    try {
      const approved = await approveUser(userId);
      setPendingUsers((users) => users.filter((u) => u.id !== approved.id));
      setStatusMsg(`Approved ${approved.email}`);
    } catch (err) {
      setErrorMsg(err instanceof Error ? err.message : "Failed to approve user");
    }
  }

  async function onLogout() {
    await logout();
    window.location.href = "/login";
  }

  return (
    <section>
      <h1>Admin Approvals</h1>
      <p>
        <Link href="/dashboard">Back to dashboard</Link>
      </p>
      <p>Signed in as: {currentUser?.email ?? "..."}</p>
      <p>Account status: {currentUser?.status ?? "..."}</p>
      <button onClick={onLogout} type="button">
        Logout
      </button>
      {statusMsg ? <p>{statusMsg}</p> : null}
      {errorMsg ? <p>{errorMsg}</p> : null}

      {loading ? <p>Loading pending users...</p> : null}
      {!loading && pendingUsers.length === 0 ? <p>No pending signup requests.</p> : null}

      {pendingUsers.length > 0 ? (
        <table>
          <thead>
            <tr>
              <th>Email</th>
              <th>Requested At</th>
              <th>Action</th>
            </tr>
          </thead>
          <tbody>
            {pendingUsers.map((user) => (
              <tr key={user.id}>
                <td>{user.email}</td>
                <td>{new Date(user.createdAt).toLocaleString()}</td>
                <td>
                  <button onClick={() => void onApprove(user.id)} type="button">
                    Approve
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : null}
    </section>
  );
}
