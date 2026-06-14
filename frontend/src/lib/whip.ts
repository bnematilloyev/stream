import { configuredApiUrl } from "./urls";

const WHIP_PORT = 8889;

function pageOrigin(): string {
  if (typeof window === "undefined") return "http://localhost:8889";
  return window.location.origin;
}

/** WHIP server — productionda api domeni (NEXT_PUBLIC_WHIP_BASE_URL yoki API URL). */
export function resolveWhipBase(configured?: string): string {
  const explicit = configured?.trim() || process.env.NEXT_PUBLIC_WHIP_BASE_URL?.trim();
  if (explicit) {
    try {
      return new URL(explicit).origin;
    } catch {
      /* fall through */
    }
  }

  const api = configuredApiUrl();
  if (api) {
    return api;
  }

  if (typeof window !== "undefined" && window.location.protocol === "https:") {
    return pageOrigin();
  }

  if (typeof window !== "undefined") {
    return `http://${window.location.hostname}:${WHIP_PORT}`;
  }
  return `http://localhost:${WHIP_PORT}`;
}

export function whipEndpoint(streamId: string, configuredBase?: string): string {
  return `${resolveWhipBase(configuredBase)}/${streamId}/whip`;
}

export { broadcastPageUrl, watchPageUrl } from "./urls";
