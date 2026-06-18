import {
  apiFetch,
  clearStoredRefreshToken,
  resolveApiUrl,
  setAccessToken,
  setStoredRefreshToken,
} from "./client";
import { getStoredRefreshToken } from "@/lib/refresh-token";
import { useAuthStore } from "@/stores/authStore";
import type { AuthResponse, User } from "@/types";

function persistAuth(res: AuthResponse) {
  setStoredRefreshToken(res.refresh_token);
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

async function tryRefresh(body: string): Promise<Response> {
  return fetch(`${resolveApiUrl()}/v1/auth/refresh`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body,
  });
}

/** Cookie yoki sessionStorage refresh_token orqali sessiyani tiklash. */
export async function restoreSession(): Promise<boolean> {
  const refreshToken = getStoredRefreshToken();

  for (let attempt = 0; attempt < 3; attempt++) {
    try {
      let res = await tryRefresh(
        JSON.stringify(refreshToken ? { refresh_token: refreshToken } : {}),
      );
      if (!res.ok && refreshToken) {
        res = await tryRefresh("{}");
      }
      if (res.ok) {
        const data: AuthResponse = await res.json();
        persistAuth(data);
        setAccessToken(data.access_token);
        return true;
      }
      if (res.status === 401 || res.status === 403) {
        break;
      }
    } catch {
      /* tarmoq xatosi — qayta uriniladi */
    }
    if (attempt < 2) {
      await new Promise((r) => setTimeout(r, 400 * (attempt + 1)));
    }
  }

  useAuthStore.getState().clearAuth();
  clearStoredRefreshToken();
  return false;
}
