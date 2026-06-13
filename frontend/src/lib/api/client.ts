import type { ApiError } from "@/types";

/** Production: same-origin (/v1 nginx proxy). Dev: localhost:8080. */
export function resolveApiUrl(): string {
  if (typeof window !== "undefined") {
    const { hostname, origin } = window.location;
    if (hostname !== "localhost" && hostname !== "127.0.0.1") {
      return origin;
    }
  }
  return process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
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

let accessToken: string | null = null;
let refreshPromise: Promise<string | null> | null = null;

export function setAccessToken(token: string | null) {
  accessToken = token;
}

export function getAccessToken() {
  return accessToken;
}

async function refreshAccessToken(): Promise<string | null> {
  if (!refreshPromise) {
    refreshPromise = (async () => {
      try {
        const res = await fetch(`${resolveApiUrl()}/v1/auth/refresh`, {
          method: "POST",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({}),
        });
        if (!res.ok) return null;
        const data = await res.json();
        accessToken = data.access_token;
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

export async function apiFetch<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const { auth = false, headers, ...rest } = options;
  const reqHeaders = new Headers(headers);
  reqHeaders.set("Content-Type", "application/json");

  if (auth && accessToken) {
    reqHeaders.set("Authorization", `Bearer ${accessToken}`);
  }

  const base = resolveApiUrl();
  let res = await fetch(`${base}${path}`, {
    ...rest,
    headers: reqHeaders,
    credentials: auth ? "include" : rest.credentials,
  });

  if (auth && res.status === 401 && accessToken) {
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

export { API_URL, resolveApiUrl as API_URL_RESOLVER };
