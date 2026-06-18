import { apiFetch, resolveApiUrl, setAccessToken } from "./client";
import { useAuthStore } from "@/stores/authStore";
import type { AuthResponse, User } from "@/types";

export async function register(data: {
  email: string;
  username: string;
  display_name: string;
  password: string;
}) {
  const res = await apiFetch<AuthResponse>("/v1/auth/register", {
    method: "POST",
    body: JSON.stringify(data),
    credentials: "include",
  });
  useAuthStore.getState().setAuth(res.user, res.access_token);
  return res;
}

export async function login(data: { email: string; password: string }) {
  const res = await apiFetch<AuthResponse>("/v1/auth/login", {
    method: "POST",
    body: JSON.stringify(data),
    credentials: "include",
  });
  useAuthStore.getState().setAuth(res.user, res.access_token);
  return res;
}

export async function logout() {
  try {
    await apiFetch("/v1/auth/logout", {
      method: "POST",
      auth: true,
      credentials: "include",
    });
  } finally {
    useAuthStore.getState().clearAuth();
  }
}

export async function getMe() {
  return apiFetch<User>("/v1/auth/me", { auth: true });
}

/** HttpOnly refresh cookie orqali yangi access token olish (sahifa yuklanganda). */
export async function restoreSession(): Promise<boolean> {
  try {
    const res = await fetch(`${resolveApiUrl()}/v1/auth/refresh`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({}),
    });
    if (!res.ok) {
      useAuthStore.getState().clearAuth();
      return false;
    }
    const data: AuthResponse = await res.json();
    useAuthStore.getState().setAuth(data.user, data.access_token);
    setAccessToken(data.access_token);
    return true;
  } catch {
    useAuthStore.getState().clearAuth();
    return false;
  }
}
