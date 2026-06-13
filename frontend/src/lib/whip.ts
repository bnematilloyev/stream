const WHIP_PORT = 8889;

function pageOrigin(): string {
  if (typeof window === "undefined") return "http://localhost:8889";
  // HTTPS domen orqali — WHIP ham shu origin (nginx proxy)
  if (window.location.protocol === "https:") {
    return window.location.origin;
  }
  return `http://${window.location.hostname}:${WHIP_PORT}`;
}

/** WHIP server manzili — HTTPS da same-origin, dev da hostname:8889. */
export function resolveWhipBase(configured?: string): string {
  const fallback = pageOrigin();

  if (!configured?.trim()) {
    return fallback;
  }

  try {
    const url = new URL(configured.trim());
    if (typeof window !== "undefined") {
      if (window.location.protocol === "https:") {
        return window.location.origin;
      }
      if (
        (url.hostname === "localhost" || url.hostname === "127.0.0.1") &&
        window.location.hostname !== "localhost" &&
        window.location.hostname !== "127.0.0.1"
      ) {
        return `http://${window.location.hostname}:${WHIP_PORT}`;
      }
    }
    return url.origin;
  } catch {
    return fallback;
  }
}

export function whipEndpoint(streamId: string, configuredBase?: string): string {
  const base = resolveWhipBase(
    configuredBase ?? process.env.NEXT_PUBLIC_WHIP_BASE_URL,
  );
  return `${base}/${streamId}/whip`;
}

export { broadcastPageUrl, watchPageUrl } from "./urls";
