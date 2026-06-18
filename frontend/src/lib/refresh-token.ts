const KEY = "sahiy-refresh-token";

/** Cookie ishlamasa fallback — sessionStorage (tab davomida). */
export function getStoredRefreshToken(): string | null {
  if (typeof window === "undefined") return null;
  try {
    return sessionStorage.getItem(KEY);
  } catch {
    return null;
  }
}

export function setStoredRefreshToken(token: string | null) {
  if (typeof window === "undefined") return;
  try {
    if (token) sessionStorage.setItem(KEY, token);
    else sessionStorage.removeItem(KEY);
  } catch {
    /* ignore */
  }
}

export function clearStoredRefreshToken() {
  setStoredRefreshToken(null);
}
