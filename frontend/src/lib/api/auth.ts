import { apiFetch, setAccessToken } from "./client";
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
  setAccessToken(res.access_token);
  return res;
}

export async function login(data: { email: string; password: string }) {
  const res = await apiFetch<AuthResponse>("/v1/auth/login", {
    method: "POST",
    body: JSON.stringify(data),
    credentials: "include",
  });
  setAccessToken(res.access_token);
  return res;
}

export async function logout() {
  await apiFetch("/v1/auth/logout", {
    method: "POST",
    auth: true,
    credentials: "include",
  });
  setAccessToken(null);
}

export async function getMe() {
  return apiFetch<User>("/v1/auth/me", { auth: true });
}
