const KEY = "sahiy-refresh-token";
/** Cross-tab sync — sessionStorage is per-tab; localStorage syncs refresh rotation. */
const SYNC_KEY = "sahiy-refresh-sync";

function syncFromLocal() {
  if (typeof window === "undefined") return;
  try {
    const synced = localStorage.getItem(SYNC_KEY);
    if (synced) sessionStorage.setItem(KEY, synced);
  } catch {
    /* ignore */
  }
}

if (typeof window !== "undefined") {
  syncFromLocal();
  window.addEventListener("storage", (e) => {
    if (e.key !== SYNC_KEY) return;
    try {
      if (e.newValue) sessionStorage.setItem(KEY, e.newValue);
      else sessionStorage.removeItem(KEY);
    } catch {
      /* ignore */
    }
  });
}

/** Cookie ishlamasa fallback — sessionStorage (tab davomida). */
export function getStoredRefreshToken(): string | null {
  if (typeof window === "undefined") return null;
  try {
    return sessionStorage.getItem(KEY) || localStorage.getItem(SYNC_KEY);
  } catch {
    return null;
  }
}

export function setStoredRefreshToken(token: string | null) {
  if (typeof window === "undefined") return;
  try {
    if (token) {
      sessionStorage.setItem(KEY, token);
      localStorage.setItem(SYNC_KEY, token);
    } else {
      sessionStorage.removeItem(KEY);
      localStorage.removeItem(SYNC_KEY);
    }
  } catch {
    /* ignore */
  }
}

export function clearStoredRefreshToken() {
  setStoredRefreshToken(null);
}
