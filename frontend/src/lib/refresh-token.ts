/** Eski deploylardan qolgan refresh token kalitlarini tozalash. */
const LEGACY_KEYS = ["sahiy-refresh-token", "sahiy-refresh-sync"] as const;

export function clearStoredRefreshToken() {
  if (typeof window === "undefined") return;
  try {
    for (const key of LEGACY_KEYS) {
      sessionStorage.removeItem(key);
      localStorage.removeItem(key);
    }
  } catch {
    /* ignore */
  }
}

/** Sahifa yuklanganda eski localStorage tokenlari cookie bilan ziddiyat qilmasin. */
export function clearLegacyRefreshTokens() {
  clearStoredRefreshToken();
}
