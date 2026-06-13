export function siteOrigin(): string {
  if (typeof window !== "undefined") {
    const { hostname, origin } = window.location;
    if (hostname !== "localhost" && hostname !== "127.0.0.1") {
      return origin;
    }
  }
  const configured = process.env.NEXT_PUBLIC_API_URL?.replace(/\/v1\/?$/, "");
  return configured && !configured.includes("localhost")
    ? configured
    : "https://stream.shopla.uz";
}

export function watchPageUrl(streamId: string): string {
  return `${siteOrigin()}/live/${streamId}`;
}

export function broadcastPageUrl(): string {
  return `${siteOrigin()}/studio/broadcast`;
}
