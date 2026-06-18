import {
  apiFetch,
  clearStoredRefreshToken,
  getAccessToken,
  refreshAccessToken,
  setAccessToken,
} from "./client";
import { useAuthStore } from "@/stores/authStore";
import type { AuthResponse, User } from "@/types";

function persistAuth(res: AuthResponse) {
  setAccessToken(res.access_token);
  useAuthStore.getState().setAuth(res.user, res.access_token);
}

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
  persistAuth(res);
  return res;
}

export async function login(data: { email: string; password: string }) {
  const res = await apiFetch<AuthResponse>("/v1/auth/login", {
    method: "POST",
    body: JSON.stringify(data),
    credentials: "include",
  });
  persistAuth(res);
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
    clearStoredRefreshToken();
    useAuthStore.getState().clearAuth();
  }
}

export async function getMe() {
  return apiFetch<User>("/v1/auth/me", { auth: true });
}

/** Sahifa yuklanganda — faqat access token yo'q bo'lsa refresh qiladi. */
export async function restoreSession(): Promise<boolean> {
  if (getAccessToken()) {
    return true;
  }

  const token = await refreshAccessToken();
  return token != null;
}
