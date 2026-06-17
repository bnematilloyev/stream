/** API, WHIP va sahifa URLlari — bitta domen (stream.vibrant.uz). */

function trimSlash(url: string): string {
  return url.replace(/\/+$/, "");
}

export function configuredApiUrl(): string | undefined {
  const raw = process.env.NEXT_PUBLIC_API_URL?.trim();
  if (!raw) return undefined;
  return trimSlash(raw.replace(/\/v1\/?$/, ""));
}

export function siteOrigin(): string {
  if (typeof window !== "undefined") {
    const { hostname, origin } = window.location;
    if (hostname !== "localhost" && hostname !== "127.0.0.1") {
      return origin;
    }
  }
  return configuredApiUrl() ?? "https://stream.vibrant.uz";
}

export function watchPageUrl(streamId: string): string {
  return `${siteOrigin()}/live/${streamId}`;
}

export function broadcastPageUrl(): string {
  return `${siteOrigin()}/studio/broadcast`;
}
