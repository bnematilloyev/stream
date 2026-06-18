import { configuredApiUrl } from "@/lib/urls";
import {
  clearStoredRefreshToken,
  getStoredRefreshToken,
  setStoredRefreshToken,
} from "@/lib/refresh-token";
import type { ApiError, AuthResponse } from "@/types";

/** Production: NEXT_PUBLIC_API_URL yoki brauzer origin. Dev: localhost:8080. */
export function resolveApiUrl(): string {
  const configured = configuredApiUrl();
  if (configured) {
    return configured;
  }

  if (typeof window !== "undefined") {
    const { hostname, origin } = window.location;
    if (hostname !== "localhost" && hostname !== "127.0.0.1") {
      return origin;
    }
  }
  return "http://localhost:8080";
}

const API_URL = resolveApiUrl();

export class ApiClientError extends Error {
  code: string;
  status: number;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

type RequestOptions = RequestInit & { auth?: boolean };

type TokenRefreshListener = (accessToken: string) => void;
type AuthRefreshListener = (data: AuthResponse) => void;
type AuthClearListener = () => void;

const SESSION_ACCESS_KEY = "sahiy-access-token";

function readSessionAccessToken(): string | null {
  if (typeof window === "undefined") return null;
  try {
    return sessionStorage.getItem(SESSION_ACCESS_KEY);
  } catch {
    return null;
  }
}

let accessToken: string | null = readSessionAccessToken();
let refreshPromise: Promise<string | null> | null = null;
const tokenRefreshListeners = new Set<TokenRefreshListener>();
const authRefreshListeners = new Set<AuthRefreshListener>();
const authClearListeners = new Set<AuthClearListener>();

export function setAccessToken(token: string | null) {
  accessToken = token;
  if (typeof window === "undefined") return;
  try {
    if (token) sessionStorage.setItem(SESSION_ACCESS_KEY, token);
    else sessionStorage.removeItem(SESSION_ACCESS_KEY);
  } catch {
    /* ignore */
  }
}

export function getAccessToken() {
  return accessToken;
}

export function onAccessTokenRefreshed(listener: TokenRefreshListener) {
  tokenRefreshListeners.add(listener);
  return () => tokenRefreshListeners.delete(listener);
}

export function onAuthRefreshed(listener: AuthRefreshListener) {
  authRefreshListeners.add(listener);
  return () => authRefreshListeners.delete(listener);
}

export function onAuthCleared(listener: AuthClearListener) {
  authClearListeners.add(listener);
  return () => authClearListeners.delete(listener);
}

function notifyTokenRefreshed(token: string) {
  tokenRefreshListeners.forEach((listener) => listener(token));
}

function notifyAuthRefreshed(data: AuthResponse) {
  authRefreshListeners.forEach((listener) => listener(data));
}

function notifyAuthCleared() {
  authClearListeners.forEach((listener) => listener());
}

function refreshRequestBody(useCookieOnly = false) {
  if (useCookieOnly) return "{}";
  const refreshToken = getStoredRefreshToken();
  return JSON.stringify(refreshToken ? { refresh_token: refreshToken } : {});
}

async function postRefresh(body: string) {
  return fetch(`${resolveApiUrl()}/v1/auth/refresh`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body,
  });
}

async function refreshAccessToken(): Promise<string | null> {
  if (!refreshPromise) {
    refreshPromise = (async () => {
      try {
        let res = await postRefresh(refreshRequestBody());
        if (!res.ok && getStoredRefreshToken()) {
          res = await postRefresh(refreshRequestBody(true));
        }
        if (!res.ok) {
          if (res.status === 401 || res.status === 403) {
            notifyAuthCleared();
          }
          return null;
        }
        const data: AuthResponse = await res.json();
        accessToken = data.access_token;
        if (data.refresh_token) setStoredRefreshToken(data.refresh_token);
        if (data.user) {
          notifyAuthRefreshed(data);
        } else {
          notifyTokenRefreshed(data.access_token);
        }
        return accessToken;
      } catch {
        return null;
      } finally {
        refreshPromise = null;
      }
    })();
  }
  return refreshPromise;
}

export { refreshAccessToken };

export async function apiFetch<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const { auth = false, headers, ...rest } = options;
  const reqHeaders = new Headers(headers);
  reqHeaders.set("Content-Type", "application/json");

  if (auth && !accessToken) {
    await refreshAccessToken();
  }

  if (auth && accessToken) {
    reqHeaders.set("Authorization", `Bearer ${accessToken}`);
  }

  const base = resolveApiUrl();
  let res = await fetch(`${base}${path}`, {
    ...rest,
    headers: reqHeaders,
    credentials: auth ? "include" : rest.credentials,
  });

  if (auth && res.status === 401) {
    const newToken = await refreshAccessToken();
    if (newToken) {
      reqHeaders.set("Authorization", `Bearer ${newToken}`);
      res = await fetch(`${base}${path}`, {
        ...rest,
        headers: reqHeaders,
        credentials: "include",
      });
    }
  }

  if (!res.ok) {
    let body: ApiError | null = null;
    try {
      body = await res.json();
    } catch {
      /* empty */
    }
    throw new ApiClientError(
      res.status,
      body?.error?.code ?? "UNKNOWN",
      body?.error?.message ?? res.statusText,
    );
  }

  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

export { clearStoredRefreshToken, setStoredRefreshToken };
export { API_URL, resolveApiUrl as API_URL_RESOLVER };
